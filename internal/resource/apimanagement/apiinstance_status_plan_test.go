package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestAPIInstanceModifyPlan_StatusOnNoOp pins the fix for a perpetual
// `status = "active" -> (known after apply)` diff: core proposes null for an omitted
// deployment block, the framework then marks computed attributes unknown, and status
// (no UseStateForUnknown) was the only one left unknown. It must be restored when
// that makes the plan a no-op, and stay unknown on a real change.
func TestAPIInstanceModifyPlan_StatusOnNoOp(t *testing.T) {
	ctx := context.Background()
	sresp := &resource.SchemaResponse{}
	NewAPIInstanceResource().Schema(ctx, resource.SchemaRequest{}, sresp)
	sch := sresp.Schema

	build := func(label string, status types.String) tftypes.Value {
		st := tfsdk.State{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(ctx), nil)}
		for p, v := range map[string]interface{}{
			"id": types.StringValue("300"), "environment_id": types.StringValue("env"),
			"instance_label": types.StringValue(label), "gateway_id": types.StringValue("gw"),
			"status": status,
		} {
			if d := st.SetAttribute(ctx, path.Root(p), v); d.HasError() {
				t.Fatalf("set %s: %v", p, d)
			}
		}
		return st.Raw
	}
	state := build("a", types.StringValue("active"))

	for _, tc := range []struct {
		name, label string
		wantKnown   bool
	}{
		{"nothing else changed -> plan equals state", "a", true},
		{"real change keeps status unknown", "b", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := tfsdk.Plan{Schema: sch, Raw: build(tc.label, types.StringUnknown())}
			req := resource.ModifyPlanRequest{
				State: tfsdk.State{Schema: sch, Raw: state}, Plan: plan,
				Config: tfsdk.Config{Schema: sch, Raw: plan.Raw},
			}
			resp := &resource.ModifyPlanResponse{Plan: plan}
			NewAPIInstanceResource().(*APIInstanceResource).ModifyPlan(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var got types.String
			resp.Plan.GetAttribute(ctx, path.Root("status"), &got)
			if got.IsUnknown() == tc.wantKnown {
				t.Errorf("status = %s, want known=%v", got, tc.wantKnown)
			}
			if tc.wantKnown && !resp.Plan.Raw.Equal(state) {
				t.Errorf("expected a no-op plan, got %s", resp.Plan.Raw)
			}
		})
	}
}
