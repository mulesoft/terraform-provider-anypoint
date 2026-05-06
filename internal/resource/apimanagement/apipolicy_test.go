package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAPIPolicyResource(t *testing.T) {
	r := NewAPIPolicyResource()
	if r == nil {
		t.Error("NewAPIPolicyResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("APIPolicyResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("APIPolicyResource should implement ResourceWithImportState")
	}
}

func TestAPIPolicyResource_Metadata(t *testing.T) {
	r := NewAPIPolicyResource()
	testutil.TestResourceMetadata(t, r, "_api_policy")
}

func TestAPIPolicyResource_Schema(t *testing.T) {
	r := NewAPIPolicyResource()
	requiredAttrs := []string{"environment_id", "api_instance_id", "configuration_data"}
	optionalAttrs := []string{"organization_id", "policy_type", "group_id", "asset_id", "asset_version", "label", "order", "disabled"}
	computedAttrs := []string{"id", "organization_id", "policy_template_id"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestAPIPolicyResource_Configure(t *testing.T) {
	res := NewAPIPolicyResource().(*APIPolicyResource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	testutil.TestResourceConfigure(t, res, providerData)
	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestAPIPolicyResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewAPIPolicyResource().(*APIPolicyResource)
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: "invalid"}
	resp := &resource.ConfigureResponse{}
	res.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestAPIPolicyResource_ImportState(t *testing.T) {
	r := NewAPIPolicyResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestAPIPolicyResourceModel_Validation(t *testing.T) {
	model := APIPolicyResourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.APIInstanceID
	_ = model.PolicyType
	_ = model.GroupID
	_ = model.AssetID
	_ = model.AssetVersion
	_ = model.Label
	_ = model.ConfigurationData
	_ = model.Order
	_ = model.Disabled
	_ = model.PolicyTemplateID
}

func TestAPIPolicyResource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/200"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":               200,
				"policyTemplateId": "test-template-id",
				"groupId":          "test-group-id",
				"assetId":          "test-asset-id",
				"assetVersion":     "1.0.0",
				"label":            "test-label",
				"order":            1,
				"disabled":         false,
				"configurationData": map[string]interface{}{},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAPIPolicyResource().(*APIPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "200"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "100"),
		"policy_type":        tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, "test-group-id"),
		"asset_id":           tftypes.NewValue(tftypes.String, "test-asset-id"),
		"asset_version":      tftypes.NewValue(tftypes.String, "1.0.0"),
		"label":              tftypes.NewValue(tftypes.String, "test-label"),
		"configuration_data": tftypes.NewValue(tftypes.String, "{}"),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, false),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIPolicyResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.AssetID.ValueString() != "test-asset-id" {
		t.Errorf("Expected AssetID 'test-asset-id', got %s", got.AssetID.ValueString())
	}
}

func TestAPIPolicyResource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/200"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAPIPolicyResource().(*APIPolicyResource)
	res.client = &apimgmtclient.APIPolicyClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "200"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id":    tftypes.NewValue(tftypes.String, "100"),
		"policy_type":        tftypes.NewValue(tftypes.String, nil),
		"group_id":           tftypes.NewValue(tftypes.String, "test-group-id"),
		"asset_id":           tftypes.NewValue(tftypes.String, "test-asset-id"),
		"asset_version":      tftypes.NewValue(tftypes.String, "1.0.0"),
		"label":              tftypes.NewValue(tftypes.String, nil),
		"configuration_data": tftypes.NewValue(tftypes.String, "{}"),
		"order":              tftypes.NewValue(tftypes.Number, nil),
		"disabled":           tftypes.NewValue(tftypes.Bool, false),
		"policy_template_id": tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}
