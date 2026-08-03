package agentstools

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewMCPBridgesDataSource(t *testing.T) {
	ds := NewMCPBridgesDataSource()
	if ds == nil {
		t.Fatal("NewMCPBridgesDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("MCPBridgesDataSource should implement DataSourceWithConfigure")
	}
}

func TestMCPBridgesDataSource_Metadata(t *testing.T) {
	ds := NewMCPBridgesDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "test"}, resp)
	if resp.TypeName != "test_mcp_bridges" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_mcp_bridges")
	}
}

func TestMCPBridgesDataSource_Schema(t *testing.T) {
	ds := NewMCPBridgesDataSource()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	if a, ok := resp.Schema.Attributes["environment_id"]; !ok || !a.IsRequired() {
		t.Error("environment_id should be a required attribute")
	}
	if a, ok := resp.Schema.Attributes["bridges"]; !ok || !a.IsComputed() {
		t.Error("bridges should be a computed attribute")
	}
}

func TestMCPBridgesDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewMCPBridgesDataSource().(*MCPBridgesDataSource)
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "invalid"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should error")
	}
}

func TestMCPBridgesDataSource_Read(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis"

	mockItems := agentsclient.MCPBridgeListResponse{
		Instances: []agentsclient.MCPBridge{
			// A real bridge (should be included).
			{ID: 10, AssetID: "b1-asset", AssetVersion: "1.0.0", Status: "active", EndpointURI: "https://mcp.example.com/b1", Metadata: map[string]string{"generatedBy": "mcp_bridge"}},
			// A plain MCP server / other API (should be filtered out by the client).
			{ID: 11, AssetID: "not-a-bridge", Metadata: map[string]string{"generatedBy": "mcp_server"}},
		},
	}

	// The list endpoint omits the endpoint block, so proxy_uri is enriched via a per-bridge
	// GET. Register that single-GET path so the enrichment resolves a real proxy_uri.
	singlePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/10"
	server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockItems)
		},
		singlePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, agentsclient.MCPBridge{
				ID:       10,
				Endpoint: &agentsclient.MCPBridgeEndpoint{ProxyURI: testutil.StringPtr("http://0.0.0.0:8087/b1")},
			})
		},
	})

	ds := NewMCPBridgesDataSource().(*MCPBridgesDataSource)
	ds.client = &agentsclient.MCPBridgeClient{
		AnypointClient: &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id"},
	}

	got := readBridgesDS(t, ds)
	if len(got.Bridges) != 1 {
		t.Fatalf("expected 1 bridge (mcp_server filtered out), got %d", len(got.Bridges))
	}
	b := got.Bridges[0]
	if b.ID.ValueString() != "10" || b.AssetID.ValueString() != "b1-asset" || b.Status.ValueString() != "active" {
		t.Errorf("unexpected bridge item: %+v", b)
	}
	if b.Technology.ValueString() != "flexGateway" {
		t.Errorf("technology = %q, want flexGateway", b.Technology.ValueString())
	}
	if b.ProxyURI.ValueString() != "http://0.0.0.0:8087/b1" {
		t.Errorf("proxy_uri = %q, want http://0.0.0.0:8087/b1 (enriched via single GET)", b.ProxyURI.ValueString())
	}
}

func TestMCPBridgesDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis"
	server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
		},
	})

	ds := NewMCPBridgesDataSource().(*MCPBridgesDataSource)
	ds.client = &agentsclient.MCPBridgeClient{
		AnypointClient: &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id"},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	elemType := stateType.(tftypes.Object).AttributeTypes["bridges"].(tftypes.List).ElementType
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"bridges":         tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error when the list endpoint fails")
	}
}

func readBridgesDS(t *testing.T, ds *MCPBridgesDataSource) MCPBridgesDataSourceModel {
	t.Helper()
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	elemType := stateType.(tftypes.Object).AttributeTypes["bridges"].(tftypes.List).ElementType
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"bridges":         tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got MCPBridgesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	return got
}
