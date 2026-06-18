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

func TestNewAPIPolicyDataSource(t *testing.T) {
	ds := NewAPIPolicyDataSource()
	if ds == nil {
		t.Error("NewAPIPolicyDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("APIPolicyDataSource should implement DataSourceWithConfigure")
	}
}

func TestAPIPolicyDataSource_Metadata(t *testing.T) {
	ds := NewAPIPolicyDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_api_policy" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_api_policy")
	}
}

func TestAPIPolicyDataSource_Schema(t *testing.T) {
	ds := NewAPIPolicyDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	required := []string{"environment_id", "api_instance_id", "policy_id"}
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
	computed := []string{"id", "policy_template_id", "group_id", "asset_id", "asset_version",
		"configuration_json", "order", "disabled", "pointcut_json"}
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

func TestAPIPolicyDataSource_Configure(t *testing.T) {
	ds := NewAPIPolicyDataSource().(*APIPolicyDataSource)
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

func TestAPIPolicyDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewAPIPolicyDataSource().(*APIPolicyDataSource)
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

func TestAPIPolicyDataSourceModel_Validation(t *testing.T) {
	model := APIPolicyDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.APIInstanceID
	_ = model.PolicyID
	_ = model.PolicyTemplateID
	_ = model.GroupID
	_ = model.AssetID
	_ = model.AssetVersion
	_ = model.ConfigurationJSON
	_ = model.Order
	_ = model.Disabled
	_ = model.PointcutJSON
}

func TestAPIPolicyDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123/policies/456"

	mockPolicy := apimgmtclient.APIPolicy{
		ID:               456,
		PolicyTemplateID: "rate-limiting",
		GroupID:          "test-group",
		AssetID:          "rate-limit-policy",
		AssetVersion:     "1.0.0",
		ConfigurationData: map[string]interface{}{"limit": float64(100)},
		Order:            1,
		Disabled:         false,
		PointcutData:     map[string]interface{}{"methodRegex": "GET"},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockPolicy)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIPolicyDataSource().(*APIPolicyDataSource)
	ds.client = &apimgmtclient.APIPolicyClient{
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

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "123"),
		"policy_id":          tftypes.NewValue(tftypes.String, "456"),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_version":      tftypes.NewValue(tftypes.String, nil),
		"configuration_json": tftypes.NewValue(tftypes.String, nil),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, nil),
		"pointcut_json":      tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIPolicyDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.PolicyTemplateID.ValueString() != "rate-limiting" {
		t.Errorf("Expected policy_template_id 'rate-limiting', got %q", got.PolicyTemplateID.ValueString())
	}
	if got.Order.ValueInt64() != 1 {
		t.Errorf("Expected order 1, got %d", got.Order.ValueInt64())
	}
	if got.Disabled.ValueBool() != false {
		t.Errorf("Expected disabled false, got %v", got.Disabled.ValueBool())
	}
}

func TestAPIPolicyDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123/policies/456"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "policy not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIPolicyDataSource().(*APIPolicyDataSource)
	ds.client = &apimgmtclient.APIPolicyClient{
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

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "123"),
		"policy_id":          tftypes.NewValue(tftypes.String, "456"),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_version":      tftypes.NewValue(tftypes.String, nil),
		"configuration_json": tftypes.NewValue(tftypes.String, nil),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, nil),
		"pointcut_json":      tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func TestAPIPolicyDataSource_Read_InvalidAPIID(t *testing.T) {
	ds := NewAPIPolicyDataSource().(*APIPolicyDataSource)
	ds.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			OrgID: "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "invalid"),
		"policy_id":          tftypes.NewValue(tftypes.String, "456"),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":           tftypes.NewValue(tftypes.String, nil),
		"asset_version":      tftypes.NewValue(tftypes.String, nil),
		"configuration_json": tftypes.NewValue(tftypes.String, nil),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, nil),
		"pointcut_json":      tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors with invalid api_instance_id")
	}
}

func BenchmarkAPIPolicyDataSource_Schema(b *testing.B) {
	ds := NewAPIPolicyDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		ds.Schema(ctx, req, resp)
	}
}
