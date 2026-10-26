package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestTeamResource_Update_OmittedSetsKeepPlannedValue: with permissions/members omitted
// from config, ModifyPlan plans their prior value. Update must store exactly that value,
// not a fresh read: after an out-of-band change between plan and apply the read would
// differ from the plan and fail the apply with "inconsistent result". Refresh picks up
// the change on the next run.
func TestTeamResource_Update_OmittedSetsKeepPlannedValue(t *testing.T) {
	teamID := "team-1"
	basePath := "/accounts/api/organizations/test-org-id/teams/" + teamID
	setReads := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/teams": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"team_id": "root-team-id", "team_name": "Root Team", "org_id": "test-org-id", "ancestor_team_ids": []string{}},
				},
				"total": 1,
			})
		},
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id": teamID, "team_name": "renamed", "team_type": "internal", "org_id": "test-org-id",
				"ancestor_team_ids": []string{"root-team-id"},
				"created_at":        "2024-01-01T00:00:00Z", "updated_at": "2024-01-02T00:00:00Z",
			})
		},
		basePath + "/roles": func(w http.ResponseWriter, r *http.Request) {
			setReads++
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}, "total": 0})
		},
		basePath + "/members": func(w http.ResponseWriter, r *http.Request) {
			setReads++
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{"data": []interface{}{}, "total": 0})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestTeamResource(server.URL)
	ctx := context.Background()
	stateType := teamStateType(t, res)
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	withSets := func(v tftypes.Value, name string, nullSets bool) tftypes.Value {
		var attrs map[string]tftypes.Value
		if err := v.As(&attrs); err != nil {
			t.Fatal(err)
		}
		attrs["name"] = tftypes.NewValue(tftypes.String, name)
		objType := stateType.(tftypes.Object)
		permType := objType.AttributeTypes["permissions"].(tftypes.Set)
		memType := objType.AttributeTypes["members"].(tftypes.Set)
		if nullSets {
			attrs["permissions"] = tftypes.NewValue(permType, nil)
			attrs["members"] = tftypes.NewValue(memType, nil)
		} else {
			attrs["permissions"] = tftypes.NewValue(permType, []tftypes.Value{
				tftypes.NewValue(permType.ElementType, map[string]tftypes.Value{
					"name":           tftypes.NewValue(tftypes.String, "View Organization"),
					"context_params": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{"org": tftypes.NewValue(tftypes.String, "test-org-id")}),
				}),
			})
			attrs["members"] = tftypes.NewValue(memType, []tftypes.Value{
				tftypes.NewValue(memType.ElementType, map[string]tftypes.Value{
					"username":        tftypes.NewValue(tftypes.String, "someone"),
					"membership_type": tftypes.NewValue(tftypes.String, "member"),
				}),
			})
		}
		return tftypes.NewValue(stateType, attrs)
	}
	base := teamRawValue(stateType, teamID, "x", "internal", "root-team-id", false, "root-team-id")
	stateRaw := withSets(base, "old", false)
	planRaw := withSets(base, "renamed", false) // ModifyPlan copied the prior sets
	cfgRaw := withSets(base, "renamed", true)   // config omits both sets

	req := resource.UpdateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: cfgRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Update(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics.Errors())
	}
	if setReads != 0 {
		t.Errorf("omitted sets with a known plan were re-read from the API (%d reads); the live value can differ from the plan", setReads)
	}
	var gotPerms, wantPerms, gotMembers, wantMembers tftypes.Value
	for _, x := range []struct {
		raw  tftypes.Value
		attr string
		dst  *tftypes.Value
	}{{resp.State.Raw, "permissions", &gotPerms}, {planRaw, "permissions", &wantPerms}, {resp.State.Raw, "members", &gotMembers}, {planRaw, "members", &wantMembers}} {
		v, _, err := tftypes.WalkAttributePath(x.raw, tftypes.NewAttributePath().WithAttributeName(x.attr))
		if err != nil {
			t.Fatal(err)
		}
		*x.dst = v.(tftypes.Value)
	}
	if !gotPerms.Equal(wantPerms) || !gotMembers.Equal(wantMembers) {
		t.Errorf("state sets differ from the plan:\n perms   %s\n want    %s\n members %s\n want    %s", gotPerms, wantPerms, gotMembers, wantMembers)
	}
}
