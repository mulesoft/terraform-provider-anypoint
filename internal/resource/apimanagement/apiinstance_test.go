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

func TestNewAPIInstanceResource(t *testing.T) {
	r := NewAPIInstanceResource()
	if r == nil {
		t.Error("NewAPIInstanceResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("APIInstanceResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("APIInstanceResource should implement ResourceWithImportState")
	}
}

func TestAPIInstanceResource_Metadata(t *testing.T) {
	r := NewAPIInstanceResource()
	testutil.TestResourceMetadata(t, r, "_api_instance")
}

func TestAPIInstanceResource_Schema(t *testing.T) {
	r := NewAPIInstanceResource()
	requiredAttrs := []string{"environment_id", "spec"}
	optionalAttrs := []string{"organization_id", "technology", "instance_label", "approval_method", "gateway_id", "endpoint", "deployment", "routing"}
	computedAttrs := []string{"id", "organization_id", "status", "product_version", "asset_id", "asset_version"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestAPIInstanceResource_Configure(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
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

func TestAPIInstanceResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
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

func TestAPIInstanceResource_ImportState(t *testing.T) {
	r := NewAPIInstanceResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestAPIInstanceResourceModel_Validation(t *testing.T) {
	model := APIInstanceResourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.Technology
	_ = model.ProviderID
	_ = model.InstanceLabel
	_ = model.ApprovalMethod
	_ = model.Status
	_ = model.AssetID
	_ = model.AssetVersion
	_ = model.ProductVersion
	_ = model.ConsumerEndpoint
	_ = model.UpstreamURI
	_ = model.GatewayID
	_ = model.Endpoint
	_ = model.Deployment
	_ = model.Routing
}

func TestAPIInstanceResource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/300"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             300,
				"assetId":        "test-api",
				"assetVersion":   "1.0.0",
				"technology":     "flexGateway",
				"status":         "Active",
				"instanceLabel":  "test-label",
				"approvalMethod": "AUTO",
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAPIInstanceResource().(*APIInstanceResource)
	res.client = &apimgmtclient.APIInstanceClient{
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
	endpointObjType := objType.AttributeTypes["endpoint"].(tftypes.Object)
	deploymentObjType := objType.AttributeTypes["deployment"].(tftypes.Object)
	routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType
	specObjType := objType.AttributeTypes["spec"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "300"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, "flexGateway"),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, "test-label"),
		"approval_method":   tftypes.NewValue(tftypes.String, "AUTO"),
		"status":            tftypes.NewValue(tftypes.String, "Active"),
		"asset_id":          tftypes.NewValue(tftypes.String, "test-api"),
		"asset_version":     tftypes.NewValue(tftypes.String, "1.0.0"),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":      tftypes.NewValue(tftypes.String, nil),
		"gateway_id":        tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(specObjType, nil),
		"endpoint":          tftypes.NewValue(endpointObjType, nil),
		"deployment":        tftypes.NewValue(deploymentObjType, nil),
		"routing":           tftypes.NewValue(tftypes.List{ElementType: routingElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIInstanceResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.AssetID.ValueString() != "test-api" {
		t.Errorf("Expected AssetID 'test-api', got %s", got.AssetID.ValueString())
	}
}

func TestAPIInstanceResource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/300"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAPIInstanceResource().(*APIInstanceResource)
	res.client = &apimgmtclient.APIInstanceClient{
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
	endpointObjType := objType.AttributeTypes["endpoint"].(tftypes.Object)
	deploymentObjType := objType.AttributeTypes["deployment"].(tftypes.Object)
	routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType
	specObjType := objType.AttributeTypes["spec"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "300"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, "flexGateway"),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, "test-label"),
		"approval_method":   tftypes.NewValue(tftypes.String, "AUTO"),
		"status":            tftypes.NewValue(tftypes.String, "Active"),
		"asset_id":          tftypes.NewValue(tftypes.String, "test-api"),
		"asset_version":     tftypes.NewValue(tftypes.String, "1.0.0"),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":      tftypes.NewValue(tftypes.String, nil),
		"gateway_id":        tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(specObjType, nil),
		"endpoint":          tftypes.NewValue(endpointObjType, nil),
		"deployment":        tftypes.NewValue(deploymentObjType, nil),
		"routing":           tftypes.NewValue(tftypes.List{ElementType: routingElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}
