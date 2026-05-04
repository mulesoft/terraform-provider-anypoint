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

// endpointAttrTypes maps EndpointModel field names to their Terraform types.
// endpoint is Optional+Computed; when omitted from config with no prior state the
// framework marks the planned value as unknown. types.Object can hold an unknown
// value — *EndpointModel cannot, which causes a Value Conversion Error at apply.
var endpointAttrTypes = map[string]attr.Type{
	"deployment_type":  types.StringType,
	"type":             types.StringType,
	"base_path":        types.StringType,
	"uri":              types.StringType,
	"response_timeout": types.Int64Type,
}

// endpointFromObject extracts an EndpointModel from a types.Object.
// Returns nil when the object is null or unknown.
func endpointFromObject(obj types.Object) *EndpointModel {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	attrs := obj.Attributes()
	return &EndpointModel{
		DeploymentType:  attrs["deployment_type"].(types.String),
		Type:            attrs["type"].(types.String),
		BasePath:        attrs["base_path"].(types.String),
		URI:             attrs["uri"].(types.String),
		ResponseTimeout: attrs["response_timeout"].(types.Int64),
	}
}

// endpointToObject converts an EndpointModel into a types.Object for state storage.
// Returns a null object when ep is nil.
func endpointToObject(ep *EndpointModel) types.Object {
	if ep == nil {
		return types.ObjectNull(endpointAttrTypes)
	}
	obj, diags := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
		"deployment_type":  ep.DeploymentType,
		"type":             ep.Type,
		"base_path":        ep.BasePath,
		"uri":              ep.URI,
		"response_timeout": ep.ResponseTimeout,
	})
	if diags.HasError() {
		return types.ObjectNull(endpointAttrTypes)
	}
	return obj
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

// mergeDeploymentObjects returns apiDep with any known (non-null, non-unknown)
// attributes from plannedDep applied on top. This prevents Computed sub-attributes
// that were unknown in the plan (e.g. environment_id) from surviving in state as
// unknown after apply — they are filled from the API response instead.
func mergeDeploymentObjects(apiDep, plannedDep types.Object) types.Object {
	if plannedDep.IsNull() || plannedDep.IsUnknown() {
		return apiDep
	}
	if apiDep.IsNull() || apiDep.IsUnknown() {
		return plannedDep
	}
	planned := deploymentFromObject(plannedDep)
	api := deploymentFromObject(apiDep)
	if planned == nil {
		return apiDep
	}
	if api == nil {
		return plannedDep
	}
	merged := *api
	if !planned.EnvironmentID.IsNull() && !planned.EnvironmentID.IsUnknown() {
		merged.EnvironmentID = planned.EnvironmentID
	}
	if !planned.Type.IsNull() && !planned.Type.IsUnknown() {
		merged.Type = planned.Type
	}
	if !planned.ExpectedStatus.IsNull() && !planned.ExpectedStatus.IsUnknown() {
		merged.ExpectedStatus = planned.ExpectedStatus
	}
	if !planned.Overwrite.IsNull() && !planned.Overwrite.IsUnknown() {
		merged.Overwrite = planned.Overwrite
	}
	if !planned.TargetID.IsNull() && !planned.TargetID.IsUnknown() {
		merged.TargetID = planned.TargetID
	}
	if !planned.TargetName.IsNull() && !planned.TargetName.IsUnknown() {
		merged.TargetName = planned.TargetName
	}
	if !planned.GatewayVersion.IsNull() && !planned.GatewayVersion.IsUnknown() {
		merged.GatewayVersion = planned.GatewayVersion
	}
	return deploymentToObject(&merged)
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
