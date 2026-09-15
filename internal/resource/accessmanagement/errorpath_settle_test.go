package accessmanagement

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestSettleUnknownTeamCollections pins the fix for an error path users hit routinely
// rather than rarely.
//
// `permissions` and `members` are both Optional+Computed, so whichever the config omits
// plans as UNKNOWN on create. refreshManagedIntoStateBestEffort runs immediately before
// State.Set on the failure path, and it used to refresh only the MANAGED collection —
// leaving the other one unknown. Terraform then rejects the apply with "Provider
// returned invalid result object after apply", stacked on top of the real error.
//
// That matters because the real error here is usually a 403: assigning most team
// permissions is refused for a client_credentials principal (docs/index.md), so it is
// the documented, expected outcome — and the provider was burying it under a second,
// misleading failure.
//
// LIVE-PROVEN on prod: `permissions = ["Read Applications"]` with `members` omitted
// produced BOTH errors before the fix and only the 403 after it.
func TestSettleUnknownTeamCollections(t *testing.T) {
	cases := []struct {
		name        string
		permissions types.Set
		members     types.Set
	}{
		{
			name:        "permissions managed, members omitted",
			permissions: types.SetNull(teamPermissionObjectType),
			members:     types.SetUnknown(teamMemberObjectType),
		},
		{
			name:        "members managed, permissions omitted",
			permissions: types.SetUnknown(teamPermissionObjectType),
			members:     types.SetNull(teamMemberObjectType),
		},
		{
			name:        "both omitted",
			permissions: types.SetUnknown(teamPermissionObjectType),
			members:     types.SetUnknown(teamMemberObjectType),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := &TeamResourceModel{Permissions: tc.permissions, Members: tc.members}

			settleUnknownTeamCollections(data)

			if data.Permissions.IsUnknown() {
				t.Error("permissions left unknown: State.Set on the error path would fail the " +
					"apply with \"invalid result object after apply\", hiding the real error")
			}
			if data.Members.IsUnknown() {
				t.Error("members left unknown: State.Set on the error path would fail the " +
					"apply with \"invalid result object after apply\", hiding the real error")
			}
		})
	}
}

// TestSettleUnknownTeamCollections_LeavesKnownValuesAlone keeps the fix from becoming a
// blunt instrument: a collection the apply DID establish must survive untouched, so a
// successful partial apply is still recorded accurately.
func TestSettleUnknownTeamCollections_LeavesKnownValuesAlone(t *testing.T) {
	known := types.SetValueMust(teamMemberObjectType, nil) // known, empty
	data := &TeamResourceModel{
		Permissions: types.SetNull(teamPermissionObjectType),
		Members:     known,
	}

	settleUnknownTeamCollections(data)

	if data.Members.IsNull() {
		t.Error("a known-empty members set was overwritten with null; only UNKNOWN values " +
			"should be settled")
	}
	if !data.Permissions.IsNull() {
		t.Error("a null permissions set should stay null")
	}
}

// TestSettleUnknownRoleCollections is the anypoint_role sibling — same schema shape,
// same error path, same failure mode.
func TestSettleUnknownRoleCollections(t *testing.T) {
	cases := []struct {
		name        string
		permissions types.Set
		members     types.Set
	}{
		{
			name:        "permissions managed, members omitted",
			permissions: types.SetNull(permissionObjectType),
			members:     types.SetUnknown(types.StringType),
		},
		{
			name:        "members managed, permissions omitted",
			permissions: types.SetUnknown(permissionObjectType),
			members:     types.SetNull(types.StringType),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := &RoleResourceModel{Permissions: tc.permissions, Members: tc.members}

			settleUnknownRoleCollections(data)

			if data.Permissions.IsUnknown() {
				t.Error("permissions left unknown on the error path")
			}
			if data.Members.IsUnknown() {
				t.Error("members left unknown on the error path")
			}
		})
	}
}
