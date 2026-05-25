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

func buildMCPServerConfigReq(t *testing.T, r *MCPServerResource, overrides map[string]tftypes.Value) resource.ValidateConfigRequest {
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
		"upstream_id":      tftypes.NewValue(tftypes.String, nil),
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

// --- MCPServerResource.ValidateConfig ---

func TestMCPServerResource_ValidateConfig(t *testing.T) {
	r := NewMCPServerResource().(*MCPServerResource)
	ctx := context.Background()

	t.Run("no errors for clean config", func(t *testing.T) {
		req := buildMCPServerConfigReq(t, r, nil)
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
			"type":             tftypes.NewValue(tftypes.String, "mcp"),
			"base_path":        tftypes.NewValue(tftypes.String, nil),
			"uri":              tftypes.NewValue(tftypes.String, nil),
			"response_timeout": tftypes.NewValue(tftypes.Number, 5000),
		})
		req := buildMCPServerConfigReq(t, r, map[string]tftypes.Value{
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
		req := buildMCPServerConfigReq(t, r, map[string]tftypes.Value{
			"upstream_uri": tftypes.NewValue(tftypes.String, "https://mcp.example.com"),
			"routing":      routingVal,
		})
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Expected error for upstream_uri + routing conflict")
		}
	})
}

// --- MCPServerResource.flattenInstance ---

func TestMCPServerResource_flattenInstance(t *testing.T) {
	r := &MCPServerResource{}
	ctx := context.Background()

	t.Run("basic instance flattened", func(t *testing.T) {
		inst := &agentsclient.MCPServer{
			ID:           10,
			Status:       "Active",
			AssetID:      "my-mcp",
			AssetVersion: "1.0.0",
			Technology:   "flexGateway",
			Spec: &agentsclient.MCPServerSpec{
				AssetID: "mcp-spec",
				GroupID: "com.example",
				Version: "2.0",
			},
		}
		data := &MCPServerResourceModel{OrganizationID: types.StringNull()}
		r.flattenInstance(ctx, inst, data, "org-1", "env-2")

		if data.ID.ValueString() != "10" {
			t.Errorf("ID = %q, want 10", data.ID.ValueString())
		}
		if data.Technology.ValueString() != "omniGateway" {
			t.Errorf("Technology = %q, want omniGateway", data.Technology.ValueString())
		}
		if data.Spec == nil || data.Spec.AssetID.ValueString() != "mcp-spec" {
			t.Error("Spec should be populated")
		}
	})

	t.Run("endpoint with proxy URI sets base_path", func(t *testing.T) {
		proxyURI := "http://0.0.0.0:8081/mcp/v1"
		inst := &agentsclient.MCPServer{
			ID:         1,
			Technology: "flexGateway",
			Endpoint: &agentsclient.MCPServerEndpoint{
				DeploymentType: "HY",
				Type:           "mcp",
				ProxyURI:       &proxyURI,
			},
		}
		data := &MCPServerResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.Endpoint.IsNull() {
			t.Fatal("Endpoint should not be null")
		}
		attrs := data.Endpoint.Attributes()
		if attrs["base_path"].(types.String).ValueString() != "mcp/v1" {
			t.Errorf("BasePath = %v, want mcp/v1", attrs["base_path"])
		}
	})

	t.Run("nil endpoint produces null object", func(t *testing.T) {
		inst := &agentsclient.MCPServer{ID: 2, Technology: "mule4"}
		data := &MCPServerResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if !data.Endpoint.IsNull() {
			t.Error("Endpoint should be null when nil from API")
		}
	})

	t.Run("deployment set when present", func(t *testing.T) {
		inst := &agentsclient.MCPServer{
			ID:         3,
			Technology: "flexGateway",
			Deployment: &agentsclient.MCPServerDeployment{
				EnvironmentID:  "env-dep",
				Type:           "HY",
				ExpectedStatus: "deployed",
				TargetID:       "target-1",
				TargetName:     "GW",
				GatewayVersion: "1.6.0",
			},
		}
		data := &MCPServerResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.Deployment.IsNull() {
			t.Fatal("Deployment should not be null")
		}
	})
}

// --- MCPServerResource.Read with error ---

func TestMCPServerResource_Read_Error(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/10"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewMCPServerResource().(*MCPServerResource)
	res.client = &agentsclient.MCPServerClient{
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
		"id":               tftypes.NewValue(tftypes.String, "10"),
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
		"upstream_id":      tftypes.NewValue(tftypes.String, nil),
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

// --- MCPServerResource.ImportState ---

func TestMCPServerResource_ImportState_IDParsing(t *testing.T) {
	r := NewMCPServerResource().(*MCPServerResource)
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
		"upstream_id":      tftypes.NewValue(tftypes.String, nil),
		"gateway_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":             tftypes.NewValue(objType.AttributeTypes["spec"], nil),
		"endpoint":         tftypes.NewValue(objType.AttributeTypes["endpoint"], nil),
		"deployment":       tftypes.NewValue(objType.AttributeTypes["deployment"], nil),
		"routing":          tftypes.NewValue(objType.AttributeTypes["routing"], nil),
	})

	t.Run("valid 3-part ID", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/10"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got MCPServerResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.ID.ValueString() != "10" {
			t.Errorf("ID = %q, want 10", got.ID.ValueString())
		}
	})

	t.Run("invalid 2-part ID", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "only/two"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 2-part ID")
		}
	})
}
