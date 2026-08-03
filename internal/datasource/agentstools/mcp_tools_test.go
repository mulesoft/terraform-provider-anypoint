package agentstools

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const dsOAS = `{
  "openapi": "3.0.0",
  "info": {"title": "Petstore", "version": "1.0.0"},
  "paths": {
    "/pets": {
      "get": {"operationId": "listPets", "parameters": [{"name": "limit", "in": "query", "schema": {"type": "string"}}]},
      "post": {"operationId": "createPet", "requestBody": {"content": {"application/json": {"schema": {"type": "object"}}}}}
    },
    "/pets/{petId}": {
      "get": {"operationId": "getPet"},
      "delete": {"operationId": "deletePet"}
    }
  }
}`

func oasZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create("petstore.json")
	_, _ = f.Write([]byte(dsOAS))
	_ = zw.Close()
	return buf.Bytes()
}

func TestMCPToolsDataSource_Metadata(t *testing.T) {
	ds := NewMCPToolsDataSource()
	metaResp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "anypoint"}, metaResp)
	if metaResp.TypeName != "anypoint_mcp_tools" {
		t.Errorf("TypeName = %q, want anypoint_mcp_tools", metaResp.TypeName)
	}
}

func TestMCPToolsDataSource_Read(t *testing.T) {
	org := "test-org-id"
	assetPath := "/exchange/api/v2/assets/" + org + "/petstore-api/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		assetPath: func(w http.ResponseWriter, r *http.Request) {
			// downloadURL points back at this same mock server.
			asset := map[string]interface{}{
				"groupId": org, "assetId": "petstore-api", "version": "1.0.0", "type": "rest-api",
				"files": []map[string]interface{}{
					{"classifier": "rest-api-metadata", "packaging": "json", "downloadURL": "http://" + r.Host + "/ignored"},
					{"classifier": "fat-oas", "packaging": "zip", "mainFile": "petstore.json", "downloadURL": "http://" + r.Host + "/download/spec"},
				},
			}
			testutil.JSONResponse(w, http.StatusOK, asset)
		},
		"/download/spec": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(oasZip(t))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewMCPToolsDataSource().(*MCPToolsDataSource)
	ds.client = &agentsclient.MCPToolsClient{
		AnypointClient: &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: org},
		Assets:         &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	obj := stateType.(tftypes.Object)

	// exclude_methods = ["DELETE"] to prove filtering.
	excludeMethods := tftypes.NewValue(obj.AttributeTypes["exclude_methods"], []tftypes.Value{
		tftypes.NewValue(tftypes.String, "DELETE"),
	})

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"organization_id":    tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":           tftypes.NewValue(tftypes.String, "petstore-api"),
		"version":            tftypes.NewValue(tftypes.String, "1.0.0"),
		"exclude_tool_names": tftypes.NewValue(obj.AttributeTypes["exclude_tool_names"], nil),
		"exclude_methods":    excludeMethods,
		"spec_type":          tftypes.NewValue(tftypes.String, nil),
		"tools":              tftypes.NewValue(obj.AttributeTypes["tools"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() errors: %v", resp.Diagnostics.Errors())
	}
	var got MCPToolsDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.SpecType.ValueString() != "oas3" {
		t.Errorf("spec_type = %q, want oas3", got.SpecType.ValueString())
	}
	if got.ID.ValueString() != org+"/petstore-api/1.0.0" {
		t.Errorf("id = %q", got.ID.ValueString())
	}
	var tools []MCPToolModelForTest
	got.Tools.ElementsAs(ctx, &tools, false)
	// 4 operations minus DELETE = 3.
	if len(tools) != 3 {
		t.Fatalf("want 3 tools (DELETE excluded), got %d", len(tools))
	}
	// Deterministic order: GET /pets, POST /pets, GET /pets/{petId}
	if tools[0].Method.ValueString() != "GET" || tools[0].Path.ValueString() != "/pets" {
		t.Errorf("tool[0] = %s %s", tools[0].Method.ValueString(), tools[0].Path.ValueString())
	}
	if tools[0].Name.ValueString() != "listPets" {
		t.Errorf("tool[0].name = %q, want listPets", tools[0].Name.ValueString())
	}
	if !tools[1].HasBody.ValueBool() {
		t.Error("POST /pets should have body")
	}
	for _, tl := range tools {
		if tl.Method.ValueString() == "DELETE" {
			t.Error("DELETE tool should have been excluded")
		}
	}
}

func TestMCPToolsDataSource_Read_NoSpec(t *testing.T) {
	org := "test-org-id"
	assetPath := "/exchange/api/v2/assets/" + org + "/no-spec/1.0.0"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		assetPath: func(w http.ResponseWriter, r *http.Request) {
			asset := map[string]interface{}{
				"groupId": org, "assetId": "no-spec", "version": "1.0.0",
				"files": []map[string]interface{}{{"classifier": "rest-api-metadata", "packaging": "json"}},
			}
			testutil.JSONResponse(w, http.StatusOK, asset)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	ds := NewMCPToolsDataSource().(*MCPToolsDataSource)
	ds.client = &agentsclient.MCPToolsClient{
		AnypointClient: &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: org},
		Assets:         &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}},
	}
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	obj := stateType.(tftypes.Object)
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"organization_id":    tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":           tftypes.NewValue(tftypes.String, "no-spec"),
		"version":            tftypes.NewValue(tftypes.String, "1.0.0"),
		"exclude_tool_names": tftypes.NewValue(obj.AttributeTypes["exclude_tool_names"], nil),
		"exclude_methods":    tftypes.NewValue(obj.AttributeTypes["exclude_methods"], nil),
		"spec_type":          tftypes.NewValue(tftypes.String, nil),
		"tools":              tftypes.NewValue(obj.AttributeTypes["tools"], nil),
	})
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error when the asset has no parseable spec")
	}
}

// MCPToolModelForTest mirrors the tools element object for decoding in tests.
type MCPToolModelForTest struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Method       types.String `tfsdk:"method"`
	Path         types.String `tfsdk:"path"`
	QueryParams  types.List   `tfsdk:"query_params"`
	HeaderParams types.List   `tfsdk:"header_params"`
	HasBody      types.Bool   `tfsdk:"has_body"`
}
