package agentstools

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Common models shared between Agent Instance and MCP Server resources

type SpecModel struct {
	AssetID types.String `tfsdk:"asset_id"`
	GroupID types.String `tfsdk:"group_id"`
	Version types.String `tfsdk:"version"`
}

type EndpointModel struct {
	DeploymentType  types.String `tfsdk:"deployment_type"`
	Type            types.String `tfsdk:"type"`
	BasePath        types.String `tfsdk:"base_path"`
	URI             types.String `tfsdk:"uri"`
	ResponseTimeout types.Int64  `tfsdk:"response_timeout"`
}

type DeploymentModel struct {
	EnvironmentID  types.String `tfsdk:"environment_id"`
	Type           types.String `tfsdk:"type"`
	ExpectedStatus types.String `tfsdk:"expected_status"`
	Overwrite      types.Bool   `tfsdk:"overwrite"`
	TargetID       types.String `tfsdk:"target_id"`
	TargetName     types.String `tfsdk:"target_name"`
	GatewayVersion types.String `tfsdk:"gateway_version"`
}

// deploymentAttrTypes maps DeploymentModel field names to their Terraform types.
var deploymentAttrTypes = map[string]attr.Type{
	"environment_id":  types.StringType,
	"type":            types.StringType,
	"expected_status": types.StringType,
	"overwrite":       types.BoolType,
	"target_id":       types.StringType,
	"target_name":     types.StringType,
	"gateway_version": types.StringType,
}

// deploymentFromObject extracts a DeploymentModel from a types.Object.
func deploymentFromObject(obj types.Object) *DeploymentModel {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	attrs := obj.Attributes()
	return &DeploymentModel{
		EnvironmentID:  attrs["environment_id"].(types.String),
		Type:           attrs["type"].(types.String),
		ExpectedStatus: attrs["expected_status"].(types.String),
		Overwrite:      attrs["overwrite"].(types.Bool),
		TargetID:       attrs["target_id"].(types.String),
		TargetName:     attrs["target_name"].(types.String),
		GatewayVersion: attrs["gateway_version"].(types.String),
	}
}

// deploymentToObject converts a DeploymentModel into a types.Object for state storage.
func deploymentToObject(dep *DeploymentModel) types.Object {
	if dep == nil {
		return types.ObjectNull(deploymentAttrTypes)
	}
	obj, diags := types.ObjectValue(deploymentAttrTypes, map[string]attr.Value{
		"environment_id":  dep.EnvironmentID,
		"type":            dep.Type,
		"expected_status": dep.ExpectedStatus,
		"overwrite":       dep.Overwrite,
		"target_id":       dep.TargetID,
		"target_name":     dep.TargetName,
		"gateway_version": dep.GatewayVersion,
	})
	if diags.HasError() {
		return types.ObjectNull(deploymentAttrTypes)
	}
	return obj
}

type RouteModel struct {
	Label     types.String `tfsdk:"label"`
	Rules     *RulesModel  `tfsdk:"rules"`
	Upstreams types.List   `tfsdk:"upstreams"`
}

type RulesModel struct {
	Methods types.String `tfsdk:"methods"`
	Path    types.String `tfsdk:"path"`
	Host    types.String `tfsdk:"host"`
	Headers types.Map    `tfsdk:"headers"`
}

type UpstreamModel struct {
	Weight       types.Int64  `tfsdk:"weight"`
	URI          types.String `tfsdk:"uri"`
	Label        types.String `tfsdk:"label"`
	TLSContextID types.String `tfsdk:"tls_context_id"`
}
