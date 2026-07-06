package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-framework/types"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// --- deploymentToObject ---

func TestDeploymentToObject(t *testing.T) {
	t.Run("nil returns null object", func(t *testing.T) {
		result := deploymentToObject(nil)
		if !result.IsNull() {
			t.Error("deploymentToObject(nil) should return null object")
		}
	})

	t.Run("valid deployment returns populated object", func(t *testing.T) {
		dep := &DeploymentModel{
			EnvironmentID:  types.StringValue("env-1"),
			Type:           types.StringValue("HY"),
			ExpectedStatus: types.StringValue("deployed"),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue("target-1"),
			TargetName:     types.StringValue("My Gateway"),
			GatewayVersion: types.StringValue("1.6.0"),
		}
		obj := deploymentToObject(dep)
		if obj.IsNull() || obj.IsUnknown() {
			t.Fatal("deploymentToObject() returned null/unknown for valid dep")
		}
		attrs := obj.Attributes()
		if attrs["environment_id"].(types.String).ValueString() != "env-1" {
			t.Errorf("environment_id = %v, want env-1", attrs["environment_id"])
		}
		if attrs["gateway_version"].(types.String).ValueString() != "1.6.0" {
			t.Errorf("gateway_version = %v, want 1.6.0", attrs["gateway_version"])
		}
	})
}

// --- mergeDeploymentObjects (apiinstance package) ---

func TestMergeDeploymentObjects_APIInstance(t *testing.T) {
	makeObj := func(envID, typ string) types.Object {
		dep := &DeploymentModel{
			EnvironmentID:  types.StringValue(envID),
			Type:           types.StringValue(typ),
			ExpectedStatus: types.StringValue("deployed"),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue("t"),
			TargetName:     types.StringValue("tn"),
			GatewayVersion: types.StringValue("1.0"),
		}
		return deploymentToObject(dep)
	}

	t.Run("null planned returns apiDep", func(t *testing.T) {
		api := makeObj("env-api", "HY")
		result := mergeDeploymentObjects(api, types.ObjectNull(deploymentAttrTypes))
		if !result.Equal(api) {
			t.Error("expected apiDep when plannedDep is null")
		}
	})

	t.Run("null api returns plannedDep", func(t *testing.T) {
		planned := makeObj("env-plan", "HY")
		result := mergeDeploymentObjects(types.ObjectNull(deploymentAttrTypes), planned)
		if !result.Equal(planned) {
			t.Error("expected plannedDep when apiDep is null")
		}
	})

	t.Run("planned values override api values", func(t *testing.T) {
		api := makeObj("env-api", "CH")
		planned := makeObj("env-plan", "HY")
		result := mergeDeploymentObjects(api, planned)
		if result.IsNull() {
			t.Fatal("merge result should not be null")
		}
		attrs := result.Attributes()
		if attrs["environment_id"].(types.String).ValueString() != "env-plan" {
			t.Errorf("environment_id = %v, want env-plan", attrs["environment_id"])
		}
		if attrs["type"].(types.String).ValueString() != "HY" {
			t.Errorf("type = %v, want HY", attrs["type"])
		}
	})

	t.Run("unknown api returns plannedDep", func(t *testing.T) {
		planned := makeObj("env-plan", "HY")
		result := mergeDeploymentObjects(types.ObjectUnknown(deploymentAttrTypes), planned)
		if !result.Equal(planned) {
			t.Error("expected plannedDep when apiDep is unknown")
		}
	})
}

// --- ValidateConfig ---

func buildAPIInstanceConfig(t *testing.T, r *APIInstanceResource, values map[string]tftypes.Value) resource.ValidateConfigRequest {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	raw := tftypes.NewValue(stateType, values)
	return resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
	}
}

func nullAPIInstanceState(t *testing.T, r *APIInstanceResource) map[string]tftypes.Value {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	return map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"organization_id":  tftypes.NewValue(tftypes.String, nil),
		"environment_id":   tftypes.NewValue(tftypes.String, "env-1"),
		"technology":       tftypes.NewValue(tftypes.String, "omniGateway"),
		"provider_id":      tftypes.NewValue(tftypes.String, nil),
		"instance_label":   tftypes.NewValue(tftypes.String, nil),
		"approval_method":  tftypes.NewValue(tftypes.String, nil),
		"status":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":         tftypes.NewValue(tftypes.String, nil),
		"asset_version":    tftypes.NewValue(tftypes.String, nil),
		"product_version":  tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":     tftypes.NewValue(tftypes.String, nil),
		"gateway_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":             tftypes.NewValue(objType.AttributeTypes["spec"], nil),
		"endpoint":         tftypes.NewValue(objType.AttributeTypes["endpoint"], nil),
		"deployment":       tftypes.NewValue(objType.AttributeTypes["deployment"], nil),
		"routing":          tftypes.NewValue(objType.AttributeTypes["routing"], nil),
	}
}

func TestAPIInstanceResource_ValidateConfig(t *testing.T) {
	r := NewAPIInstanceResource().(*APIInstanceResource)
	ctx := context.Background()

	t.Run("no errors for minimal valid config", func(t *testing.T) {
		vals := nullAPIInstanceState(t, r)
		req := buildAPIInstanceConfig(t, r, vals)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ValidateConfig() unexpected errors: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("response_timeout on omniGateway produces error", func(t *testing.T) {
		schemaResp := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		stateType := schemaResp.Schema.Type().TerraformType(ctx)
		objType := stateType.(tftypes.Object)
		epType := objType.AttributeTypes["endpoint"].(tftypes.Object)

		ep := tftypes.NewValue(epType, map[string]tftypes.Value{
			"deployment_type":  tftypes.NewValue(tftypes.String, "CH"),
			"type":             tftypes.NewValue(tftypes.String, "http"),
			"base_path":        tftypes.NewValue(tftypes.String, nil),
			"response_timeout": tftypes.NewValue(tftypes.Number, 5000),
		})

		vals := nullAPIInstanceState(t, r)
		vals["endpoint"] = ep
		vals["technology"] = tftypes.NewValue(tftypes.String, "omniGateway")

		req := buildAPIInstanceConfig(t, r, vals)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Expected error for response_timeout on omniGateway")
		}
	})

	t.Run("upstream_uri and routing conflict produces error", func(t *testing.T) {
		schemaResp := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		stateType := schemaResp.Schema.Type().TerraformType(ctx)
		objType := stateType.(tftypes.Object)
		routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType

		// Build a minimal routing list with one element
		routingVal := tftypes.NewValue(
			tftypes.List{ElementType: routingElemType},
			[]tftypes.Value{
				tftypes.NewValue(routingElemType, map[string]tftypes.Value{
					"label": tftypes.NewValue(tftypes.String, "route-1"),
					"rules": tftypes.NewValue(routingElemType.(tftypes.Object).AttributeTypes["rules"], nil),
					"upstreams": tftypes.NewValue(
						routingElemType.(tftypes.Object).AttributeTypes["upstreams"],
						nil,
					),
				}),
			},
		)

		vals := nullAPIInstanceState(t, r)
		vals["upstream_uri"] = tftypes.NewValue(tftypes.String, "https://backend.example.com")
		vals["routing"] = routingVal

		req := buildAPIInstanceConfig(t, r, vals)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Expected error for upstream_uri + routing conflict")
		}
	})
}

// --- Read with error ---

func TestAPIInstanceResource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/42"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, "42"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":   tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":       tftypes.NewValue(tftypes.String, "omniGateway"),
		"provider_id":      tftypes.NewValue(tftypes.String, nil),
		"instance_label":   tftypes.NewValue(tftypes.String, nil),
		"approval_method":  tftypes.NewValue(tftypes.String, nil),
		"status":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":         tftypes.NewValue(tftypes.String, nil),
		"asset_version":    tftypes.NewValue(tftypes.String, nil),
		"product_version":  tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":     tftypes.NewValue(tftypes.String, nil),
		"gateway_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":             tftypes.NewValue(objType.AttributeTypes["spec"], nil),
		"endpoint":         tftypes.NewValue(objType.AttributeTypes["endpoint"], nil),
		"deployment":       tftypes.NewValue(objType.AttributeTypes["deployment"], nil),
		"routing":          tftypes.NewValue(objType.AttributeTypes["routing"], nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// --- expandUpdateRequest ---

func TestAPIInstanceResource_expandUpdateRequest(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("technology is mapped", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(routeListElemType),
			InstanceLabel:    types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandUpdateRequest(ctx, data)
		if req.Technology == nil || *req.Technology != "flexGateway" {
			t.Errorf("Technology = %v, want flexGateway", req.Technology)
		}
	})

	t.Run("instance_label set when non-null", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(routeListElemType),
			InstanceLabel:    types.StringValue("my-label"),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandUpdateRequest(ctx, data)
		if req.InstanceLabel == nil || *req.InstanceLabel != "my-label" {
			t.Errorf("InstanceLabel = %v, want my-label", req.InstanceLabel)
		}
	})

	t.Run("upstream_uri sets top-level upstream (routing set by Update from server)", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringValue("https://backend.example.com"),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(routeListElemType),
			InstanceLabel:    types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandUpdateRequest(ctx, data)
		// Routing is NOT set by expandUpdateRequest — it is fetched from the server
		// and injected by the Update function. Only Upstreams is set here.
		if len(req.Routing) != 0 {
			t.Errorf("Routing should be empty from expandUpdateRequest, got len=%d", len(req.Routing))
		}
		if len(req.Upstreams) != 1 || req.Upstreams[0].URI != "https://backend.example.com" {
			t.Errorf("Upstreams[0].URI = %q, want https://backend.example.com", req.Upstreams[0].URI)
		}
	})

	t.Run("deployment included when set", func(t *testing.T) {
		depObj, _ := types.ObjectValue(deploymentAttrTypes, map[string]attr.Value{
			"environment_id":  types.StringValue("env-1"),
			"type":            types.StringValue("HY"),
			"expected_status": types.StringValue("deployed"),
			"overwrite":       types.BoolValue(false),
			"target_id":       types.StringValue("target-1"),
			"target_name":     types.StringValue("GW"),
			"gateway_version": types.StringValue("1.6.0"),
		})
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       depObj,
			Routing:          types.ListNull(routeListElemType),
			InstanceLabel:    types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandUpdateRequest(ctx, data)
		if req.Deployment == nil {
			t.Fatal("Deployment should not be nil")
		}
		if req.Deployment.EnvironmentID != "env-1" {
			t.Errorf("EnvironmentID = %q, want env-1", req.Deployment.EnvironmentID)
		}
	})
}

// --- flattenInstance (APIInstanceResource) – additional cases ---

func TestAPIInstanceResource_flattenInstance_Extra(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("consumer endpoint is flattened", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			ID:          1,
			Technology:  "flexGateway",
			EndpointURI: "https://consumer.example.com",
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.ConsumerEndpoint.ValueString() != "https://consumer.example.com" {
			t.Errorf("ConsumerEndpoint = %q, want https://consumer.example.com", data.ConsumerEndpoint.ValueString())
		}
	})

	t.Run("nil consumer endpoint is null", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			ID:          2,
			Technology:  "mule4",
			EndpointURI: "",
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if !data.ConsumerEndpoint.IsNull() {
			t.Errorf("ConsumerEndpoint should be null when EndpointURI is empty")
		}
	})

	t.Run("deployment is populated from API", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			ID:         3,
			Technology: "flexGateway",
			Deployment: &apimgmtclient.APIInstanceDeployment{
				EnvironmentID:  "env-deploy",
				Type:           "HY",
				ExpectedStatus: "deployed",
				Overwrite:      true,
				TargetID:       "target-1",
				TargetName:     "GW",
				GatewayVersion: "1.6.0",
			},
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.Deployment.IsNull() {
			t.Fatal("Deployment should not be null when API returns deployment")
		}
		attrs := data.Deployment.Attributes()
		if attrs["environment_id"].(types.String).ValueString() != "env-deploy" {
			t.Errorf("Deployment.EnvironmentID = %v, want env-deploy", attrs["environment_id"])
		}
	})
}

// --- ImportState – ID parsing ---

func TestAPIInstanceResource_ImportState_IDParsing(t *testing.T) {
	r := NewAPIInstanceResource().(*APIInstanceResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	rawState := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"organization_id":  tftypes.NewValue(tftypes.String, nil),
		"environment_id":   tftypes.NewValue(tftypes.String, nil),
		"technology":       tftypes.NewValue(tftypes.String, nil),
		"provider_id":      tftypes.NewValue(tftypes.String, nil),
		"instance_label":   tftypes.NewValue(tftypes.String, nil),
		"approval_method":  tftypes.NewValue(tftypes.String, nil),
		"status":           tftypes.NewValue(tftypes.String, nil),
		"asset_id":         tftypes.NewValue(tftypes.String, nil),
		"asset_version":    tftypes.NewValue(tftypes.String, nil),
		"product_version":  tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_uri":     tftypes.NewValue(tftypes.String, nil),
		"gateway_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":             tftypes.NewValue(objType.AttributeTypes["spec"], nil),
		"endpoint":         tftypes.NewValue(objType.AttributeTypes["endpoint"], nil),
		"deployment":       tftypes.NewValue(objType.AttributeTypes["deployment"], nil),
		"routing":          tftypes.NewValue(objType.AttributeTypes["routing"], nil),
	})

	t.Run("valid 3-part ID parses correctly", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/42"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got APIInstanceResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", got.ID.ValueString())
		}
	})

	t.Run("valid 2-part ID sets environment_id and id", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "env-2/42"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors for 2-part ID: %v", resp.Diagnostics.Errors())
		}
		var got APIInstanceResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", got.ID.ValueString())
		}
		if !got.OrganizationID.IsNull() {
			t.Errorf("OrganizationID should be null for 2-part import, got %q", got.OrganizationID.ValueString())
		}
	})

	t.Run("invalid ID format produces error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "only-one-part"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 1-part ID")
		}
	})
}

// TestExpandUpdateRequest_AssetVersion verifies that asset version is correctly
// populated at root level in PATCH requests (GUS W-23307847).
func TestExpandUpdateRequest_AssetVersion(t *testing.T) {
	mockServer := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	client, err := anypointclient.NewAnypointClient(&anypointclient.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      mockServer.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	apiClient := &apimgmtclient.APIInstanceClient{AnypointClient: client}
	r := &APIInstanceResource{client: apiClient}
	ctx := context.Background()

	t.Run("AssetVersion populated from spec.version", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Spec: &SpecModel{
				AssetID: types.StringValue("test-api"),
				GroupID: types.StringValue("test-group"),
				Version: types.StringValue("2.0.0"),
			},
		}

		req := r.expandUpdateRequest(ctx, data)

		if req.AssetVersion == nil {
			t.Fatal("AssetVersion should not be nil when spec.version is provided")
		}

		if *req.AssetVersion != "2.0.0" {
			t.Errorf("AssetVersion = %q, want 2.0.0", *req.AssetVersion)
		}
	})

	t.Run("AssetVersion not populated when spec is nil", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Spec: nil,
		}

		req := r.expandUpdateRequest(ctx, data)

		if req.AssetVersion != nil {
			t.Errorf("AssetVersion should be nil when spec is nil, got %v", *req.AssetVersion)
		}
	})

	t.Run("AssetVersion not populated when version is null", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Spec: &SpecModel{
				AssetID: types.StringValue("test-api"),
				GroupID: types.StringValue("test-group"),
				Version: types.StringNull(),
			},
		}

		req := r.expandUpdateRequest(ctx, data)

		if req.AssetVersion != nil {
			t.Errorf("AssetVersion should be nil when spec.version is null, got %v", *req.AssetVersion)
		}
	})

	t.Run("AssetVersion not populated when version is unknown", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Spec: &SpecModel{
				AssetID: types.StringValue("test-api"),
				GroupID: types.StringValue("test-group"),
				Version: types.StringUnknown(),
			},
		}

		req := r.expandUpdateRequest(ctx, data)

		if req.AssetVersion != nil {
			t.Errorf("AssetVersion should be nil when spec.version is unknown, got %v", *req.AssetVersion)
		}
	})
}
