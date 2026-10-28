package exchange

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestAssetResource_ModifyPlan_UpdatedDateOnNoOp: with categories/custom_fields
// omitted from config but set on the asset, core proposes null for them and the
// framework marks updated_date unknown; the list modifiers restore the lists, so
// updated_date was the only diff left, on every plan. It must be restored when that
// makes the plan a no-op, and stay unknown on a real change.
func TestAssetResource_ModifyPlan_UpdatedDateOnNoOp(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	obj := func(name string, updated tftypes.Value) tftypes.Value {
		return assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, "org"),
			"group_id":        tftypes.NewValue(tftypes.String, "grp"),
			"asset_id":        tftypes.NewValue(tftypes.String, "asset"),
			"version":         tftypes.NewValue(tftypes.String, "1.0.0"),
			"type":            tftypes.NewValue(tftypes.String, "custom"),
			"status":          tftypes.NewValue(tftypes.String, "published"),
			"name":            tftypes.NewValue(tftypes.String, name),
			"updated_date":    updated,
		})
	}
	state := obj("n", tftypes.NewValue(tftypes.String, "t1"))
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	for _, tc := range []struct {
		name, planName string
		wantNoOp       bool
	}{
		{"nothing else changed -> plan equals state", "n", true},
		{"real change keeps updated_date unknown", "n2", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: obj(tc.planName, unknown)}
			req := resource.ModifyPlanRequest{
				Plan:   plan,
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: state},
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw},
			}
			resp := &resource.ModifyPlanResponse{Plan: plan}
			res.ModifyPlan(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var got types.String
			resp.Plan.GetAttribute(ctx, path.Root("updated_date"), &got)
			if got.IsUnknown() == tc.wantNoOp {
				t.Errorf("updated_date = %s, want known=%v", got, tc.wantNoOp)
			}
			if tc.wantNoOp && !resp.Plan.Raw.Equal(state) {
				t.Errorf("expected a no-op plan, got %s", resp.Plan.Raw)
			}
		})
	}
}
