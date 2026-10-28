package accessmanagement

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// keepUnmanagedSetsFromState makes "omit the attribute to leave it unmanaged" plan
// cleanly for the Optional+Computed set attributes named in attrs.
//
// Terraform core proposes NULL (not the prior value) for an omitted Optional+Computed
// nested attribute whose prior value holds non-computed fields (objchange's
// optionalValueNotComputable: permissions[].name/context_params). The framework sees
// that as a change and marks every computed attribute unknown, so a team or role that
// has any permissions/members plans an update on every run. This copies the prior
// value back for omitted attributes and, when nothing else changes, also restores
// updated_at so the plan is a true no-op.
func keepUnmanagedSetsFromState(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, attrs ...string) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return // create or destroy
	}
	for _, name := range attrs {
		if configSets(ctx, req.Config, name) {
			continue
		}
		var prior types.Set
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(name), &prior)...)
		if resp.Diagnostics.HasError() || prior.IsNull() {
			continue
		}
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(name), prior)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	updatedAt := path.Root("updated_at")
	var planned, prior types.String
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, updatedAt, &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, updatedAt, &prior)...)
	if resp.Diagnostics.HasError() || !planned.IsUnknown() {
		return
	}
	probe := tfsdk.Plan{Schema: resp.Plan.Schema, Raw: resp.Plan.Raw.Copy()}
	if diags := probe.SetAttribute(ctx, updatedAt, prior); diags.HasError() {
		return
	}
	if probe.Raw.Equal(req.State.Raw) {
		resp.Plan = probe
	}
}

// configSets reports whether the configuration sets the named top-level attribute.
// Update uses it to decide whether a set is authoritative, so the prior value that
// keepUnmanagedSetsFromState copies into the plan never turns an omitted (unmanaged)
// attribute into a managed one. With no config available it answers true, which
// defers to the planned value exactly as before this check existed.
func configSets(ctx context.Context, config tfsdk.Config, name string) bool {
	if config.Schema == nil || config.Raw.IsNull() {
		return true
	}
	var v types.Set
	if diags := config.GetAttribute(ctx, path.Root(name), &v); diags.HasError() {
		return false
	}
	return !v.IsNull()
}
