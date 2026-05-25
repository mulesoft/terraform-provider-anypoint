package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAPIUpstreamsDataSource(t *testing.T) {
	ds := NewAPIUpstreamsDataSource()
	if ds == nil {
		t.Error("NewAPIUpstreamsDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("APIUpstreamsDataSource should implement DataSourceWithConfigure")
	}
}

func TestAPIUpstreamsDataSource_Metadata(t *testing.T) {
	ds := NewAPIUpstreamsDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "anypoint_api_upstreams" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "anypoint_api_upstreams")
	}
}

func TestAPIUpstreamsDataSource_Schema(t *testing.T) {
	ds := NewAPIUpstreamsDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	required := []string{"environment_id", "api_instance_id"}
	for _, attr := range required {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing required attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Schema() attribute %q should be required", attr)
		}
	}
	computed := []string{"id", "upstreams", "total"}
	for _, attr := range computed {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Schema() attribute %q should be computed", attr)
		}
	}
}

func TestAPIUpstreamsDataSource_Configure(t *testing.T) {
	ds := NewAPIUpstreamsDataSource().(*APIUpstreamsDataSource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: providerData}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}
	if ds.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestAPIUpstreamsDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewAPIUpstreamsDataSource().(*APIUpstreamsDataSource)
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: nil}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() with nil provider data should not error: %v", resp.Diagnostics.Errors())
	}
}

func TestAPIUpstreamsDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewAPIUpstreamsDataSource().(*APIUpstreamsDataSource)
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: "invalid"}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if ds.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestAPIUpstreamsDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/api-42/upstreams"

	mockUpstreams := []apimgmtclient.APIUpstream{
		{ID: "up-1", Label: "Primary", URI: "https://primary.example.com"},
		{ID: "up-2", Label: "Secondary", URI: "https://secondary.example.com"},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"total":     len(mockUpstreams),
				"upstreams": mockUpstreams,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIUpstreamsDataSource().(*APIUpstreamsDataSource)
	ds.client = &apimgmtclient.APIUpstreamsClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["upstreams"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "api-42"),
		"total":           tftypes.NewValue(tftypes.Number, nil),
		"upstreams":       tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIUpstreamsDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Upstreams) != 2 {
		t.Fatalf("Expected 2 upstreams, got %d", len(got.Upstreams))
	}
	if got.Upstreams[0].ID.ValueString() != "up-1" {
		t.Errorf("Upstream[0].ID = %s, want up-1", got.Upstreams[0].ID.ValueString())
	}
	if got.Total.ValueInt64() != 2 {
		t.Errorf("Total = %d, want 2", got.Total.ValueInt64())
	}
}

func TestAPIUpstreamsDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/api-42/upstreams"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIUpstreamsDataSource().(*APIUpstreamsDataSource)
	ds.client = &apimgmtclient.APIUpstreamsClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["upstreams"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "api-42"),
		"total":           tftypes.NewValue(tftypes.Number, nil),
		"upstreams":       tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report errors on API failure")
	}
}
