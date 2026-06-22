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

func TestNewSLATierDataSource(t *testing.T) {
	ds := NewSLATierDataSource()
	if ds == nil {
		t.Error("NewSLATierDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("SLATierDataSource should implement DataSourceWithConfigure")
	}
}

func TestSLATierDataSource_Metadata(t *testing.T) {
	ds := NewSLATierDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_api_instance_sla_tier" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_api_instance_sla_tier")
	}
}

func TestSLATierDataSource_Schema(t *testing.T) {
	ds := NewSLATierDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	required := []string{"environment_id", "api_instance_id", "tier_id"}
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
	computed := []string{"id", "name", "description", "auto_approve", "status", "limits"}
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

func TestSLATierDataSource_Configure(t *testing.T) {
	ds := NewSLATierDataSource().(*SLATierDataSource)
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

func TestSLATierDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewSLATierDataSource().(*SLATierDataSource)
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

func TestSLATierDataSourceModel_Validation(t *testing.T) {
	model := SLATierDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.APIInstanceID
	_ = model.TierID
	_ = model.Name
	_ = model.Description
	_ = model.AutoApprove
	_ = model.Status
	_ = model.Limits
}

func TestSLATierDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123/tiers"

	mockTier := apimgmtclient.SLATier{
		ID:          456,
		Name:        "Gold",
		Description: "Gold tier with higher limits",
		AutoApprove: true,
		Status:      "ACTIVE",
		Limits: []apimgmtclient.SLALimit{
			{
				TimePeriodInMilliseconds: 60000,
				MaximumRequests:          1000,
				Visible:                  true,
			},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{"tiers": []apimgmtclient.SLATier{mockTier}})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewSLATierDataSource().(*SLATierDataSource)
	ds.client = &apimgmtclient.SLATierClient{
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
	limitsElemType := objType.AttributeTypes["limits"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":   tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":  tftypes.NewValue(tftypes.String, "123"),
		"tier_id":          tftypes.NewValue(tftypes.String, "456"),
		"name":             tftypes.NewValue(tftypes.String, nil),
		"description":      tftypes.NewValue(tftypes.String, nil),
		"auto_approve":     tftypes.NewValue(tftypes.Bool, nil),
		"status":           tftypes.NewValue(tftypes.String, nil),
		"limits":           tftypes.NewValue(tftypes.List{ElementType: limitsElemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got SLATierDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "Gold" {
		t.Errorf("Expected name 'Gold', got %q", got.Name.ValueString())
	}
	if len(got.Limits) != 1 {
		t.Fatalf("Expected 1 limit, got %d", len(got.Limits))
	}
}

func TestSLATierDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123/tiers"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "tier not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewSLATierDataSource().(*SLATierDataSource)
	ds.client = &apimgmtclient.SLATierClient{
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
	limitsElemType := objType.AttributeTypes["limits"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":   tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":  tftypes.NewValue(tftypes.String, "123"),
		"tier_id":          tftypes.NewValue(tftypes.String, "456"),
		"name":             tftypes.NewValue(tftypes.String, nil),
		"description":      tftypes.NewValue(tftypes.String, nil),
		"auto_approve":     tftypes.NewValue(tftypes.Bool, nil),
		"status":           tftypes.NewValue(tftypes.String, nil),
		"limits":           tftypes.NewValue(tftypes.List{ElementType: limitsElemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func BenchmarkSLATierDataSource_Schema(b *testing.B) {
	ds := NewSLATierDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		ds.Schema(ctx, req, resp)
	}
}
