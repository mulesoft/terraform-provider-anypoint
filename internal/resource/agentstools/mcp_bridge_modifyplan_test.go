package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// bridgePlanValue builds a whole-resource value with one tool, so ModifyPlan can be
// driven end to end. consumerEndpoint nil models "not declared in config".
func bridgePlanValue(t *testing.T, r *MCPBridgeResource, toolName string, consumerEndpoint *string) (tftypes.Value, tftypes.Type) {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	srcList := objType.AttributeTypes["source_apis"].(tftypes.List)
	srcElem := srcList.ElementType.(tftypes.Object)
	toolList := srcElem.AttributeTypes["tools"].(tftypes.List)
	toolElem := toolList.ElementType.(tftypes.Object)

	tool := tftypes.NewValue(toolElem, map[string]tftypes.Value{
		"method":        tftypes.NewValue(tftypes.String, "GET"),
		"path":          tftypes.NewValue(tftypes.String, "/pets"),
		"name":          tftypes.NewValue(tftypes.String, toolName),
		"description":   tftypes.NewValue(tftypes.String, nil),
		"query_params":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"header_params": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"has_body":      tftypes.NewValue(tftypes.Bool, false),
		"input_schema":  tftypes.NewValue(tftypes.String, nil),
		"http_mapping":  tftypes.NewValue(toolElem.AttributeTypes["http_mapping"], nil),
	})

	src := tftypes.NewValue(srcElem, map[string]tftypes.Value{
		"label":          tftypes.NewValue(tftypes.String, "petstore"),
		"upstream_uri":   tftypes.NewValue(tftypes.String, "https://backend.example.com"),
		"tls_context_id": tftypes.NewValue(tftypes.String, nil),
		"asset_id":       tftypes.NewValue(tftypes.String, "pets"),
		"group_id":       tftypes.NewValue(tftypes.String, nil),
		"version":        tftypes.NewValue(tftypes.String, "1.0.0"),
		"tools":          tftypes.NewValue(toolList, []tftypes.Value{tool}),
	})

	endpoint := tftypes.NewValue(tftypes.String, nil)
	if consumerEndpoint != nil {
		endpoint = tftypes.NewValue(tftypes.String, *consumerEndpoint)
	}

	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "123"),
		"organization_id":   tftypes.NewValue(tftypes.String, "org-1"),
		"environment_id":    tftypes.NewValue(tftypes.String, "env-1"),
		"gateway_id":        tftypes.NewValue(tftypes.String, "gw-1"),
		"mcp_asset_name":    tftypes.NewValue(tftypes.String, "bridge"),
		"port":              tftypes.NewValue(tftypes.Number, 8081),
		"base_path":         tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, "bridge"),
		"asset_version":     tftypes.NewValue(tftypes.String, "1.0.0"),
		"product_version":   tftypes.NewValue(tftypes.String, "v1.0"),
		"consumer_endpoint": endpoint,
		"status":            tftypes.NewValue(tftypes.String, "active"),
		"technology":        tftypes.NewValue(tftypes.String, "flexGateway"),
		"deployment":        tftypes.NewValue(objType.AttributeTypes["deployment"], nil),
		"source_apis":       tftypes.NewValue(srcList, []tftypes.Value{src}),
	}), objType
}

func runBridgeModifyPlan(t *testing.T, consumerEndpoint *string) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()
	r := &MCPBridgeResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	// The tool name differs between plan and state, so the tool signature changes and
	// ModifyPlan takes its "re-plan the computed fields" branch.
	planVal, _ := bridgePlanValue(t, r, "list_pets_v2", consumerEndpoint)
	stateVal, _ := bridgePlanValue(t, r, "list_pets", consumerEndpoint)

	resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal}}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)
	return resp
}

// consumer_endpoint became Optional as well as Computed. Terraform rejects a plan that
// replaces a configured value with unknown ("planned value cty.UnknownVal does not match
// config value"), so a declared endpoint must survive ModifyPlan untouched.
func TestModifyPlan_KeepsConfiguredConsumerEndpoint(t *testing.T) {
	endpoint := "https://gw.example.com/mcp"
	resp := runBridgeModifyPlan(t, &endpoint)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
	}

	var got types.String
	resp.Plan.GetAttribute(context.Background(), pathConsumerEndpoint(), &got)
	if got.IsUnknown() {
		t.Error("a consumer_endpoint declared in config must not be re-planned as unknown; " +
			"Terraform rejects that as an invalid plan")
	}
	if got.ValueString() != endpoint {
		t.Errorf("consumer_endpoint = %q, want %q", got.ValueString(), endpoint)
	}
}

// When the platform supplies it, it still has to be re-planned as unknown: the value is
// read back after the asset version move and cannot be predicted.
func TestModifyPlan_UnknownsPlatformSuppliedConsumerEndpoint(t *testing.T) {
	resp := runBridgeModifyPlan(t, nil)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
	}

	var got types.String
	resp.Plan.GetAttribute(context.Background(), pathConsumerEndpoint(), &got)
	if !got.IsUnknown() {
		t.Errorf("consumer_endpoint = %v, want unknown when it is not declared in config", got)
	}
}

// asset_version is always platform-supplied and must always be re-planned as unknown on
// a tool change, or the apply fails as inconsistent once the version is bumped.
func TestModifyPlan_UnknownsAssetVersionOnToolChange(t *testing.T) {
	endpoint := "https://gw.example.com/mcp"
	resp := runBridgeModifyPlan(t, &endpoint)

	var got types.String
	resp.Plan.GetAttribute(context.Background(), pathAssetVersion(), &got)
	if !got.IsUnknown() {
		t.Errorf("asset_version = %v, want unknown after a tool change", got)
	}
}

func pathConsumerEndpoint() path.Path { return path.Root("consumer_endpoint") }
func pathAssetVersion() path.Path     { return path.Root("asset_version") }
