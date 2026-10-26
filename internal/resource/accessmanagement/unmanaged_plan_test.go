package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func teamFixture(name string, perms, members types.Set, updatedAt types.String) TeamResourceModel {
	return TeamResourceModel{
		ID:             types.StringValue("team-1"),
		Name:           types.StringValue(name),
		OrganizationID: types.StringValue("org-1"),
		ParentTeamID:   types.StringValue("root-1"),
		TeamType:       types.StringValue("internal"),
		Permissions:    perms,
		Members:        members,
		CreatedAt:      types.StringValue("t0"),
		UpdatedAt:      updatedAt,
	}
}

func teamPlanRequest(t *testing.T, cfgModel, stateModel, planModel TeamResourceModel) (resource.ModifyPlanRequest, *resource.ModifyPlanResponse) {
	t.Helper()
	ctx := context.Background()
	sresp := &resource.SchemaResponse{}
	NewTeamResource().Schema(ctx, resource.SchemaRequest{}, sresp)
	sch := sresp.Schema

	state := tfsdk.State{Schema: sch}
	if d := state.Set(ctx, &stateModel); d.HasError() {
		t.Fatalf("state: %v", d)
	}
	cfg := tfsdk.State{Schema: sch}
	if d := cfg.Set(ctx, &cfgModel); d.HasError() {
		t.Fatalf("config: %v", d)
	}
	plan := tfsdk.Plan{Schema: sch}
	if d := plan.Set(ctx, &planModel); d.HasError() {
		t.Fatalf("plan: %v", d)
	}
	req := resource.ModifyPlanRequest{Config: tfsdk.Config{Schema: sch, Raw: cfg.Raw}, State: state, Plan: plan}
	return req, &resource.ModifyPlanResponse{Plan: plan}
}

// What Terraform core + the framework hand ModifyPlan when a team that has
// permissions/members is planned with both attributes omitted: core proposes null,
// the framework then marks them (and updated_at) unknown.
func TestTeamModifyPlan_OmittedSetsAreANoOp(t *testing.T) {
	perm := types.ObjectValueMust(teamPermissionObjectType.AttrTypes, map[string]attr.Value{
		"name":           types.StringValue("View Organization"),
		"context_params": types.MapValueMust(types.StringType, map[string]attr.Value{"org": types.StringValue("org-1")}),
	})
	member := types.ObjectValueMust(teamMemberObjectType.AttrTypes, map[string]attr.Value{
		"username":        types.StringValue("alice"),
		"membership_type": types.StringValue("member"),
	})
	perms := types.SetValueMust(teamPermissionObjectType, []attr.Value{perm})
	members := types.SetValueMust(teamMemberObjectType, []attr.Value{member})
	nullP, nullM := types.SetNull(teamPermissionObjectType), types.SetNull(teamMemberObjectType)
	unkP, unkM := types.SetUnknown(teamPermissionObjectType), types.SetUnknown(teamMemberObjectType)
	stateModel := teamFixture("t", perms, members, types.StringValue("t1"))

	t.Run("nothing else changed -> plan equals state", func(t *testing.T) {
		req, resp := teamPlanRequest(t, teamFixture("t", nullP, nullM, types.StringNull()), stateModel,
			teamFixture("t", unkP, unkM, types.StringUnknown()))
		NewTeamResource().(*TeamResource).ModifyPlan(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if !resp.Plan.Raw.Equal(req.State.Raw) {
			t.Errorf("expected a no-op plan, got %s", resp.Plan.Raw)
		}
	})

	t.Run("real change keeps sets from state, updated_at unknown", func(t *testing.T) {
		req, resp := teamPlanRequest(t, teamFixture("t2", nullP, nullM, types.StringNull()), stateModel,
			teamFixture("t2", unkP, unkM, types.StringUnknown()))
		NewTeamResource().(*TeamResource).ModifyPlan(context.Background(), req, resp)
		var got TeamResourceModel
		resp.Plan.Get(context.Background(), &got)
		if !got.Permissions.Equal(perms) || !got.Members.Equal(members) {
			t.Errorf("omitted sets must keep prior values, got %s / %s", got.Permissions, got.Members)
		}
		if !got.UpdatedAt.IsUnknown() {
			t.Errorf("updated_at must stay unknown on a real change, got %s", got.UpdatedAt)
		}
	})

	t.Run("configured sets are left alone", func(t *testing.T) {
		req, resp := teamPlanRequest(t, teamFixture("t", nullP, members, types.StringNull()), stateModel,
			teamFixture("t", unkP, members, types.StringUnknown()))
		cfgEmpty := teamFixture("t", types.SetValueMust(teamPermissionObjectType, nil), nullM, types.StringNull())
		req2, resp2 := teamPlanRequest(t, cfgEmpty, stateModel,
			teamFixture("t", types.SetValueMust(teamPermissionObjectType, nil), unkM, types.StringUnknown()))
		NewTeamResource().(*TeamResource).ModifyPlan(context.Background(), req, resp)
		NewTeamResource().(*TeamResource).ModifyPlan(context.Background(), req2, resp2)
		var got, got2 TeamResourceModel
		resp.Plan.Get(context.Background(), &got)
		resp2.Plan.Get(context.Background(), &got2)
		if !got.Members.Equal(members) || !got.Permissions.Equal(perms) {
			t.Errorf("unexpected plan %s / %s", got.Permissions, got.Members)
		}
		if len(got2.Permissions.Elements()) != 0 || got2.Permissions.IsNull() {
			t.Errorf("permissions = [] in config must stay an empty (authoritative) set, got %s", got2.Permissions)
		}
	})
}
