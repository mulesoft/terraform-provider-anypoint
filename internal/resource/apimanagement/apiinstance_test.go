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
	requiredAttrs := []string{"environment_id"}
	optionalAttrs := []string{"organization_id", "technology", "instance_label", "approval_method", "gateway_id", "spec", "endpoint", "deployment", "routing"}
	computedAttrs := []string{"id", "organization_id", "status", "product_version", "asset_id", "asset_version"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestAPIInstanceResource_Configure(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
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

func TestAPIInstanceResource_ValidateConfig_ResponseTimeoutOmniGateway(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	endpointObjType := objType.AttributeTypes["endpoint"].(tftypes.Object)
	deploymentObjType := objType.AttributeTypes["deployment"].(tftypes.Object)
	routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType
	specObjType := objType.AttributeTypes["spec"].(tftypes.Object)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, nil),
		"organization_id":   tftypes.NewValue(tftypes.String, nil),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, "omniGateway"),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":      tftypes.NewValue(tftypes.String, nil),
		"gateway_id":        tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(specObjType, nil),
		"endpoint": tftypes.NewValue(endpointObjType, map[string]tftypes.Value{
			"deployment_type":  tftypes.NewValue(tftypes.String, "HY"),
			"type":             tftypes.NewValue(tftypes.String, "http"),
			"base_path":        tftypes.NewValue(tftypes.String, nil),
			"response_timeout": tftypes.NewValue(tftypes.Number, 3000),
		}),
		"deployment": tftypes.NewValue(deploymentObjType, nil),
		"routing":    tftypes.NewValue(tftypes.List{ElementType: routingElemType}, nil),
	})

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &resource.ValidateConfigResponse{}
	res.ValidateConfig(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ValidateConfig() should error when response_timeout is set with technology='omniGateway'")
	}
}


func TestAPIInstanceResource_ImportState_Valid(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	endpointObjType := objType.AttributeTypes["endpoint"].(tftypes.Object)
	deploymentObjType := objType.AttributeTypes["deployment"].(tftypes.Object)
	routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType
	specObjType := objType.AttributeTypes["spec"].(tftypes.Object)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, nil),
		"organization_id":   tftypes.NewValue(tftypes.String, nil),
		"environment_id":    tftypes.NewValue(tftypes.String, nil),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":      tftypes.NewValue(tftypes.String, nil),
		"gateway_id":        tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(specObjType, nil),
		"endpoint":          tftypes.NewValue(endpointObjType, nil),
		"deployment":        tftypes.NewValue(deploymentObjType, nil),
		"routing":           tftypes.NewValue(tftypes.List{ElementType: routingElemType}, nil),
	})

	req := resource.ImportStateRequest{ID: "test-org/test-env/300"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got APIInstanceResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.OrganizationID.ValueString() != "test-org" {
		t.Errorf("expected organization_id 'test-org', got '%s'", got.OrganizationID.ValueString())
	}
	if got.EnvironmentID.ValueString() != "test-env" {
		t.Errorf("expected environment_id 'test-env', got '%s'", got.EnvironmentID.ValueString())
	}
	if got.ID.ValueString() != "300" {
		t.Errorf("expected id '300', got '%s'", got.ID.ValueString())
	}
}

func TestAPIInstanceResource_ImportState_SimpleID(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	endpointObjType := objType.AttributeTypes["endpoint"].(tftypes.Object)
	deploymentObjType := objType.AttributeTypes["deployment"].(tftypes.Object)
	routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType
	specObjType := objType.AttributeTypes["spec"].(tftypes.Object)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, nil),
		"organization_id":   tftypes.NewValue(tftypes.String, nil),
		"environment_id":    tftypes.NewValue(tftypes.String, nil),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":      tftypes.NewValue(tftypes.String, nil),
		"gateway_id":        tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(specObjType, nil),
		"endpoint":          tftypes.NewValue(endpointObjType, nil),
		"deployment":        tftypes.NewValue(deploymentObjType, nil),
		"routing":           tftypes.NewValue(tftypes.List{ElementType: routingElemType}, nil),
	})

	req := resource.ImportStateRequest{ID: "test-env/300"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with simple ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var got APIInstanceResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.EnvironmentID.ValueString() != "test-env" {
		t.Errorf("expected environment_id 'test-env', got '%s'", got.EnvironmentID.ValueString())
	}
	if got.ID.ValueString() != "300" {
		t.Errorf("expected id '300', got '%s'", got.ID.ValueString())
	}
	if !got.OrganizationID.IsNull() {
		t.Errorf("expected organization_id to be null for simple import, got '%s'", got.OrganizationID.ValueString())
	}
}

func TestAPIInstanceResource_ImportState_Invalid(t *testing.T) {
	res := NewAPIInstanceResource().(*APIInstanceResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	for _, id := range []string{"only-one-part", "four/parts/too/many"} {
		req := resource.ImportStateRequest{ID: id}
		resp := &resource.ImportStateResponse{
			State: tfsdk.State{Schema: schemaResp.Schema},
		}
		res.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("ImportState() with ID %q should produce errors", id)
		}
	}
}

func TestAPIInstanceResource_Read_ImportPath_EnrichesRouting(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/300"
	upstreamsPath := basePath + "/upstreams"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":           300,
				"assetId":      "test-api",
				"assetVersion": "1.0.0",
				"technology":   "flexGateway",
				"status":       "active",
				"routing": []map[string]interface{}{
					{
						"label": "read-traffic",
						"rules": map[string]interface{}{"methods": "GET"},
						"upstreams": []map[string]interface{}{
							{"id": "upstream-id-primary", "weight": 90},
							{"id": "upstream-id-secondary", "weight": 10},
						},
					},
				},
			})
		},
		upstreamsPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"total": 2,
				"upstreams": []map[string]interface{}{
					{"id": "upstream-id-primary", "label": "primary", "uri": "http://backend-primary.internal:8080"},
					{"id": "upstream-id-secondary", "label": "secondary", "uri": "http://backend-secondary.internal:8080"},
				},
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

	// Simulate the import path: routing is null (no prior state).
	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "300"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
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

	if got.Routing.IsNull() || got.Routing.IsUnknown() {
		t.Fatal("Routing should be populated after import-path Read")
	}

	var routes []RouteModel
	if diags := got.Routing.ElementsAs(ctx, &routes, false); diags.HasError() {
		t.Fatalf("ElementsAs errors: %v", diags.Errors())
	}
	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	var upstreams []UpstreamModel
	if diags := routes[0].Upstreams.ElementsAs(ctx, &upstreams, false); diags.HasError() {
		t.Fatalf("ElementsAs upstream errors: %v", diags.Errors())
	}
	if len(upstreams) != 2 {
		t.Fatalf("Expected 2 upstreams, got %d", len(upstreams))
	}
	if upstreams[0].URI.ValueString() != "http://backend-primary.internal:8080" {
		t.Errorf("upstream[0].URI = %q, want primary URI", upstreams[0].URI.ValueString())
	}
	if upstreams[0].Label.ValueString() != "primary" {
		t.Errorf("upstream[0].Label = %q, want primary", upstreams[0].Label.ValueString())
	}
	if upstreams[1].URI.ValueString() != "http://backend-secondary.internal:8080" {
		t.Errorf("upstream[1].URI = %q, want secondary URI", upstreams[1].URI.ValueString())
	}
}

func TestAPIInstanceResource_Read_ImportPath_PopulatesTLSContextID(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/301"
	upstreamsPath := basePath + "/upstreams"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":           301,
				"assetId":      "test-api",
				"assetVersion": "1.0.0",
				"technology":   "flexGateway",
				"status":       "active",
				"routing": []map[string]interface{}{
					{
						"upstreams": []map[string]interface{}{
							{"id": "upstream-tls-id", "weight": 100},
						},
					},
				},
			})
		},
		upstreamsPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"total": 1,
				"upstreams": []map[string]interface{}{
					{
						"id":         "upstream-tls-id",
						"label":      "secure",
						"uri":        "https://secure.example.com",
						"connection": nil,
						"tlsContext": map[string]interface{}{
							"secretGroupId": "sg-abc123",
							"tlsContextId":  "tls-def456",
						},
					},
				},
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
		"id":                tftypes.NewValue(tftypes.String, "301"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
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

	if got.Routing.IsNull() || got.Routing.IsUnknown() {
		t.Fatal("Routing should be populated after import-path Read")
	}

	var routes []RouteModel
	if diags := got.Routing.ElementsAs(ctx, &routes, false); diags.HasError() {
		t.Fatalf("ElementsAs errors: %v", diags.Errors())
	}
	if len(routes) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(routes))
	}

	var upstreams []UpstreamModel
	if diags := routes[0].Upstreams.ElementsAs(ctx, &upstreams, false); diags.HasError() {
		t.Fatalf("ElementsAs upstream errors: %v", diags.Errors())
	}
	if len(upstreams) != 1 {
		t.Fatalf("Expected 1 upstream, got %d", len(upstreams))
	}

	want := "sg-abc123/tls-def456"
	if upstreams[0].TLSContextID.ValueString() != want {
		t.Errorf("tls_context_id = %q, want %q", upstreams[0].TLSContextID.ValueString(), want)
	}
}

func TestAPIInstanceResource_Read_ImportPath_PopulatesGatewayID(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/302"
	upstreamsPath := basePath + "/upstreams"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":           302,
				"assetId":      "test-api",
				"assetVersion": "1.0.0",
				"technology":   "flexGateway",
				"status":       "active",
				"deployment": map[string]interface{}{
					"environmentId":  "test-env-id",
					"type":           "HY",
					"expectedStatus": "deployed",
					"targetId":       "gw-target-uuid",
					"targetName":     "my-gateway",
					"gatewayVersion": "1.12.5",
				},
				"routing": []map[string]interface{}{
					{
						"upstreams": []map[string]interface{}{
							{"id": "us-1", "weight": 100},
						},
					},
				},
			})
		},
		upstreamsPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"total": 1,
				"upstreams": []map[string]interface{}{
					{"id": "us-1", "label": "backend", "uri": "https://backend.example.com"},
				},
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
		"id":                tftypes.NewValue(tftypes.String, "302"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
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

	if got.GatewayID.ValueString() != "gw-target-uuid" {
		t.Errorf("gateway_id = %q, want gw-target-uuid (from deployment.target_id)", got.GatewayID.ValueString())
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
