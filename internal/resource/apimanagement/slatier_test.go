package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewSLATierResource(t *testing.T) {
	r := NewSLATierResource()
	if r == nil {
		t.Error("NewSLATierResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("SLATierResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("SLATierResource should implement ResourceWithImportState")
	}
}

func TestSLATierResource_Metadata(t *testing.T) {
	r := NewSLATierResource()
	testutil.TestResourceMetadata(t, r, "_api_instance_sla_tier")
}

func TestSLATierResource_Schema(t *testing.T) {
	r := NewSLATierResource()
	requiredAttrs := []string{"environment_id", "api_instance_id", "name", "limits"}
	optionalAttrs := []string{"organization_id", "description", "auto_approve", "status"}
	computedAttrs := []string{"id", "organization_id", "status"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestSLATierResource_Configure(t *testing.T) {
	res := NewSLATierResource().(*SLATierResource)
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

func TestSLATierResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewSLATierResource().(*SLATierResource)
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

func TestSLATierResource_ImportState(t *testing.T) {
	r := NewSLATierResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestSLATierResourceModel_Validation(t *testing.T) {
	model := SLATierResourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.APIInstanceID
	_ = model.Name
	_ = model.Description
	_ = model.AutoApprove
	_ = model.Status
	_ = model.Limits
}

func TestSLALimitModel_Validation(t *testing.T) {
	model := SLALimitModel{}
	_ = model.TimePeriodInMilliseconds
	_ = model.MaximumRequests
	_ = model.Visible
}

func TestSLATierResource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/tiers/200"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":          200,
				"name":        "test-tier",
				"description": "test description",
				"autoApprove": false,
				"status":      "ACTIVE",
				"limits":      []interface{}{},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSLATierResource().(*SLATierResource)
	res.client = &apimgmtclient.SLATierClient{
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
	objType := stateType.(tftypes.Object)
	limitsElemType := objType.AttributeTypes["limits"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "200"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "100"),
		"name":            tftypes.NewValue(tftypes.String, "test-tier"),
		"description":     tftypes.NewValue(tftypes.String, nil),
		"auto_approve":    tftypes.NewValue(tftypes.Bool, false),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"limits":          tftypes.NewValue(tftypes.List{ElementType: limitsElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestSLATierResource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/tiers/200"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSLATierResource().(*SLATierResource)
	res.client = &apimgmtclient.SLATierClient{
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
	objType := stateType.(tftypes.Object)
	limitsElemType := objType.AttributeTypes["limits"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "200"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"api_instance_id": tftypes.NewValue(tftypes.String, "100"),
		"name":            tftypes.NewValue(tftypes.String, "test-tier"),
		"description":     tftypes.NewValue(tftypes.String, nil),
		"auto_approve":    tftypes.NewValue(tftypes.Bool, false),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"limits":          tftypes.NewValue(tftypes.List{ElementType: limitsElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}
