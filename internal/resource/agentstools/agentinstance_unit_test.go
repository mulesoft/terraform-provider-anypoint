package agentstools

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-framework/types"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// --- helper to build null-filled ValidateConfig request ---

func buildAgentInstanceConfigReq(t *testing.T, r *AgentInstanceResource, overrides map[string]tftypes.Value) resource.ValidateConfigRequest {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)

	vals := map[string]tftypes.Value{
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
	for k, v := range overrides {
		vals[k] = v
	}
	raw := tftypes.NewValue(stateType, vals)
	return resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
	}
}

// --- ValidateConfig ---

func TestAgentInstanceResource_ValidateConfig(t *testing.T) {
	r := NewAgentInstanceResource().(*AgentInstanceResource)
	ctx := context.Background()

	t.Run("no errors for clean config", func(t *testing.T) {
		req := buildAgentInstanceConfigReq(t, r, nil)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ValidateConfig() unexpected errors: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("response_timeout on omniGateway produces error", func(t *testing.T) {
		schemaResp2 := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, schemaResp2)
		stateType := schemaResp2.Schema.Type().TerraformType(ctx)
		objType := stateType.(tftypes.Object)
		epType := objType.AttributeTypes["endpoint"].(tftypes.Object)

		ep := tftypes.NewValue(epType, map[string]tftypes.Value{
			"deployment_type":  tftypes.NewValue(tftypes.String, "HY"),
			"type":             tftypes.NewValue(tftypes.String, "a2a"),
			"base_path":        tftypes.NewValue(tftypes.String, nil),
			"uri":              tftypes.NewValue(tftypes.String, nil),
			"response_timeout": tftypes.NewValue(tftypes.Number, 5000),
		})
		req := buildAgentInstanceConfigReq(t, r, map[string]tftypes.Value{
			"endpoint":   ep,
			"technology": tftypes.NewValue(tftypes.String, "omniGateway"),
		})
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Expected error for response_timeout on omniGateway")
		}
	})

	t.Run("upstream_uri and routing conflict produces error", func(t *testing.T) {
		schemaResp2 := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, schemaResp2)
		stateType := schemaResp2.Schema.Type().TerraformType(ctx)
		objType := stateType.(tftypes.Object)
		routingElemType := objType.AttributeTypes["routing"].(tftypes.List).ElementType

		routingVal := tftypes.NewValue(
			tftypes.List{ElementType: routingElemType},
			[]tftypes.Value{
				tftypes.NewValue(routingElemType, map[string]tftypes.Value{
					"label": tftypes.NewValue(tftypes.String, "r"),
					"rules": tftypes.NewValue(routingElemType.(tftypes.Object).AttributeTypes["rules"], nil),
					"upstreams": tftypes.NewValue(routingElemType.(tftypes.Object).AttributeTypes["upstreams"], nil),
				}),
			},
		)
		req := buildAgentInstanceConfigReq(t, r, map[string]tftypes.Value{
			"upstream_uri": tftypes.NewValue(tftypes.String, "https://backend.example.com"),
			"routing":      routingVal,
		})
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Expected error for upstream_uri + routing conflict")
		}
	})
}

// --- flattenInstance (AgentInstanceResource) ---

func TestAgentInstanceResource_flattenInstance(t *testing.T) {
	r := &AgentInstanceResource{}
	ctx := context.Background()

	t.Run("basic instance is flattened", func(t *testing.T) {
		provID := "prov-1"
		inst := &agentsclient.AgentInstance{
			ID:             42,
			Status:         "Active",
			AssetID:        "my-agent",
			AssetVersion:   "1.0.0",
			ProductVersion: "v1",
			Technology:     "flexGateway",
			InstanceLabel:  "my-label",
			ProviderID:     &provID,
			Spec: &agentsclient.AgentInstanceSpec{
				AssetID: "spec-asset",
				GroupID: "spec-group",
				Version: "1.0",
			},
		}
		data := &AgentInstanceResourceModel{OrganizationID: types.StringNull()}
		r.flattenInstance(ctx, inst, data, "org-1", "env-2")

		if data.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", data.ID.ValueString())
		}
		if data.Technology.ValueString() != "omniGateway" {
			t.Errorf("Technology = %q, want omniGateway", data.Technology.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", data.EnvironmentID.ValueString())
		}
		if data.InstanceLabel.ValueString() != "my-label" {
			t.Errorf("InstanceLabel = %q, want my-label", data.InstanceLabel.ValueString())
		}
		if data.ProviderID.ValueString() != "prov-1" {
			t.Errorf("ProviderID = %q, want prov-1", data.ProviderID.ValueString())
		}
		if data.Spec == nil || data.Spec.AssetID.ValueString() != "spec-asset" {
			t.Error("Spec should be populated")
		}
	})

	t.Run("endpoint with proxy URI sets base_path", func(t *testing.T) {
		proxyURI := "http://0.0.0.0:8081/my/path"
		inst := &agentsclient.AgentInstance{
			ID:         1,
			Technology: "flexGateway",
			Endpoint: &agentsclient.AgentInstanceEndpoint{
				DeploymentType: "HY",
				Type:           "a2a",
				ProxyURI:       &proxyURI,
			},
		}
		data := &AgentInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.Endpoint.IsNull() {
			t.Fatal("Endpoint should not be null")
		}
		attrs := data.Endpoint.Attributes()
		if attrs["base_path"].(types.String).ValueString() != "my/path" {
			t.Errorf("BasePath = %v, want my/path", attrs["base_path"])
		}
	})

	t.Run("nil endpoint sets null object", func(t *testing.T) {
		inst := &agentsclient.AgentInstance{ID: 2, Technology: "mule4"}
		data := &AgentInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if !data.Endpoint.IsNull() {
			t.Error("Endpoint should be null when nil from API")
		}
	})

	t.Run("consumer endpoint set when EndpointURI present", func(t *testing.T) {
		inst := &agentsclient.AgentInstance{
			ID:          3,
			Technology:  "flexGateway",
			EndpointURI: "https://consumer.example.com",
		}
		data := &AgentInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.ConsumerEndpoint.ValueString() != "https://consumer.example.com" {
			t.Errorf("ConsumerEndpoint = %q", data.ConsumerEndpoint.ValueString())
		}
	})

	t.Run("deployment object is set when present", func(t *testing.T) {
		inst := &agentsclient.AgentInstance{
			ID:         4,
			Technology: "flexGateway",
			Deployment: &agentsclient.AgentInstanceDeployment{
				EnvironmentID:  "env-dep",
				Type:           "HY",
				ExpectedStatus: "deployed",
				Overwrite:      false,
				TargetID:       "target-1",
				TargetName:     "GW",
				GatewayVersion: "1.6.0",
			},
		}
		data := &AgentInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.Deployment.IsNull() {
			t.Fatal("Deployment should not be null")
		}
		attrs := data.Deployment.Attributes()
		if attrs["environment_id"].(types.String).ValueString() != "env-dep" {
			t.Errorf("Deployment.EnvironmentID = %v, want env-dep", attrs["environment_id"])
		}
	})
}

// --- AgentInstanceResource.Read with error ---

func TestAgentInstanceResource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/42"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAgentInstanceResource().(*AgentInstanceResource)
	res.client = &agentsclient.AgentInstanceClient{
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

// --- AgentInstanceResource.ImportState ---

func TestAgentInstanceResource_ImportState_IDParsing(t *testing.T) {
	r := NewAgentInstanceResource().(*AgentInstanceResource)
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

	t.Run("valid 3-part ID", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/42"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got AgentInstanceResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", got.ID.ValueString())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
	})

	t.Run("invalid 2-part ID produces error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "only/two"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 2-part ID")
		}
	})
}
