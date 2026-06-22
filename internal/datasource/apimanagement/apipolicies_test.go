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

func TestNewAPIPoliciesDataSource(t *testing.T) {
	ds := NewAPIPoliciesDataSource()
	if ds == nil {
		t.Error("NewAPIPoliciesDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("APIPoliciesDataSource should implement DataSourceWithConfigure")
	}
}

func TestAPIPoliciesDataSource_Metadata(t *testing.T) {
	ds := NewAPIPoliciesDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_api_policies" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_api_policies")
	}
}

func TestAPIPoliciesDataSource_Schema(t *testing.T) {
	ds := NewAPIPoliciesDataSource()
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
	computed := []string{"id", "policies"}
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

func TestAPIPoliciesDataSource_Configure(t *testing.T) {
	ds := NewAPIPoliciesDataSource().(*APIPoliciesDataSource)
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

func TestAPIPoliciesDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewAPIPoliciesDataSource().(*APIPoliciesDataSource)
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

func TestAPIPoliciesDataSourceModel_Validation(t *testing.T) {
	model := APIPoliciesDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.APIInstanceID
	_ = model.Policies
}

func TestAPIPolicyItemModel_Validation(t *testing.T) {
	model := APIPolicyItemModel{}
	_ = model.ID
	_ = model.PolicyTemplateID
	_ = model.GroupID
	_ = model.AssetID
	_ = model.AssetVersion
	_ = model.ConfigurationJSON
	_ = model.Order
	_ = model.Disabled
	_ = model.PointcutJSON
}

func TestAPIPoliciesDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123/policies"

	// The list endpoint returns an envelope {"policies": [...]} with compact entries.
	// policyTemplateId is json.Number in the decoder so we use numeric IDs here.
	mockPoliciesJSON := `{"policies": [
		{
			"id": 456,
			"policyTemplateId": 101,
			"groupId": "test-group",
			"assetId": "rate-limit-policy",
			"assetVersion": "1.0.0",
			"configurationData": {"limit": 100},
			"order": 1,
			"disabled": false
		},
		{
			"id": 789,
			"policyTemplateId": 202,
			"groupId": "test-group",
			"assetId": "cors-policy",
			"assetVersion": "2.0.0",
			"configurationData": {"allowedOrigins": ["*"]},
			"order": 2,
			"disabled": true
		}
	]}`

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockPoliciesJSON))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIPoliciesDataSource().(*APIPoliciesDataSource)
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["policies"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "123"),
		"policies":        tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIPoliciesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Policies) != 2 {
		t.Fatalf("Expected 2 policies, got %d", len(got.Policies))
	}
	if got.Policies[0].ID.ValueString() != "456" {
		t.Errorf("Expected first policy ID '456', got %q", got.Policies[0].ID.ValueString())
	}
	if got.Policies[0].PolicyTemplateID.ValueString() != "101" {
		t.Errorf("Expected first policy template '101', got %q", got.Policies[0].PolicyTemplateID.ValueString())
	}
	if got.Policies[1].ID.ValueString() != "789" {
		t.Errorf("Expected second policy ID '789', got %q", got.Policies[1].ID.ValueString())
	}
	if got.Policies[1].Disabled.ValueBool() != true {
		t.Errorf("Expected second policy disabled true, got %v", got.Policies[1].Disabled.ValueBool())
	}
}

func TestAPIPoliciesDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123/policies"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIPoliciesDataSource().(*APIPoliciesDataSource)
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["policies"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "123"),
		"policies":        tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func TestAPIPoliciesDataSource_Read_InvalidAPIID(t *testing.T) {
	ds := NewAPIPoliciesDataSource().(*APIPoliciesDataSource)
	ds.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			OrgID: "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["policies"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "invalid"),
		"policies":        tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors with invalid api_instance_id")
	}
}

func TestMapAPIPolicyToItemModel(t *testing.T) {
	policy := apimgmtclient.APIPolicy{
		ID:                123,
		PolicyTemplateID:  "rate-limiting",
		GroupID:           "test-group",
		AssetID:           "rate-limit-policy",
		AssetVersion:      "1.0.0",
		ConfigurationData: map[string]interface{}{"limit": float64(100)},
		Order:             1,
		Disabled:          false,
		PointcutData:      map[string]interface{}{"methodRegex": "GET"},
	}

	result := mapAPIPolicyToItemModel(policy)

	if result.ID.ValueString() != "123" {
		t.Errorf("Expected ID '123', got %q", result.ID.ValueString())
	}
	if result.PolicyTemplateID.ValueString() != "rate-limiting" {
		t.Errorf("Expected PolicyTemplateID 'rate-limiting', got %q", result.PolicyTemplateID.ValueString())
	}
	if result.GroupID.ValueString() != "test-group" {
		t.Errorf("Expected GroupID 'test-group', got %q", result.GroupID.ValueString())
	}
	if result.Order.ValueInt64() != 1 {
		t.Errorf("Expected Order 1, got %d", result.Order.ValueInt64())
	}
	if result.Disabled.ValueBool() != false {
		t.Errorf("Expected Disabled false, got %v", result.Disabled.ValueBool())
	}
	if result.ConfigurationJSON.ValueString() != `{"limit":100}` {
		t.Errorf("Expected ConfigurationJSON '{\"limit\":100}', got %q", result.ConfigurationJSON.ValueString())
	}
}

func TestMapAPIPolicyToItemModel_NullJSON(t *testing.T) {
	policy := apimgmtclient.APIPolicy{
		ID:                123,
		PolicyTemplateID:  "rate-limiting",
		GroupID:           "test-group",
		AssetID:           "rate-limit-policy",
		AssetVersion:      "1.0.0",
		ConfigurationData: nil,
		Order:             1,
		Disabled:          false,
		PointcutData:      nil,
	}

	result := mapAPIPolicyToItemModel(policy)

	if !result.ConfigurationJSON.IsNull() {
		t.Error("Expected ConfigurationJSON to be null")
	}
	if !result.PointcutJSON.IsNull() {
		t.Error("Expected PointcutJSON to be null")
	}
}

func BenchmarkAPIPoliciesDataSource_Schema(b *testing.B) {
	ds := NewAPIPoliciesDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		ds.Schema(ctx, req, resp)
	}
}
