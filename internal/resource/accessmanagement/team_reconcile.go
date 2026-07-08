package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// teamRoleObjectType is the Terraform object type for a single role entry in the
// anypoint_team resource's `roles` set. It mirrors the role resource's permission
// entry (a display name plus optional context params).
var teamRoleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

// teamMemberObjectType is the Terraform object type for a single member entry in
// the anypoint_team resource's `members` set: a username and an optional
// membership_type ("member" or "maintainer").
var teamMemberObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"username":        types.StringType,
		"membership_type": types.StringType,
	},
}

// desiredTeamRole is a resolved role ready to be sent to the API.
type desiredTeamRole struct {
	roleID        string
	typedName     string            // exactly what the user wrote (preserved in state)
	contextParams map[string]string // resolved context params
	contextKey    string            // canonical JSON of contextParams for matching
}

// typedTeamRole holds the user's original (typed) representation of a role so Read
// can preserve casing / context_params form and avoid perpetual diffs.
type typedTeamRole struct {
	name types.String
	cp   types.Map
}

// desiredTeamMember is a resolved member ready to be sent to the API.
type desiredTeamMember struct {
	userID         string
	typedUsername  string // exactly what the user wrote (preserved in state)
	membershipType string // resolved membership type ("member" when omitted)
}

// typedTeamMember holds the user's original (typed) representation of a member so
// Read can preserve username casing and the (possibly null) membership_type.
type typedTeamMember struct {
	username       types.String
	membershipType types.String
}

// --- roles -------------------------------------------------------------------

// resolveTeamRoles resolves each role's display name to a role_id using the
// available-roles catalog (case-insensitive). Returns an error listing the
// offending name if it is unknown or ambiguous. Returns (nil, nil) when unmanaged.
func (r *TeamResource) resolveTeamRoles(ctx context.Context, roleSet types.Set) ([]desiredTeamRole, error) {
	if roleSet.IsNull() || roleSet.IsUnknown() {
		return nil, nil
	}

	roles, err := r.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list available roles for name resolution: %w", err)
	}

	// Build a case-insensitive name -> []role_id map so we can detect ambiguity.
	nameToIDs := make(map[string][]string, len(roles))
	for _, role := range roles {
		key := normalizeName(role.Name)
		nameToIDs[key] = append(nameToIDs[key], role.RoleID)
	}

	out := make([]desiredTeamRole, 0, len(roleSet.Elements()))
	for _, el := range roleSet.Elements() {
		obj := el.(types.Object)
		attrs := obj.Attributes()
		typedName := attrs["name"].(types.String).ValueString()
		cpMap := mapToStringMap(attrs["context_params"].(types.Map))

		ids := nameToIDs[normalizeName(typedName)]
		if len(ids) == 0 {
			return nil, fmt.Errorf(
				"role %q is not a valid role name; use the anypoint_available_roles data source to discover valid role names",
				typedName,
			)
		}
		if len(ids) > 1 {
			return nil, fmt.Errorf(
				"role name %q is ambiguous (matches %d roles); the platform has multiple roles with this name so it cannot be resolved by name alone",
				typedName, len(ids),
			)
		}

		out = append(out, desiredTeamRole{
			roleID:        ids[0],
			typedName:     typedName,
			contextParams: cpMap,
			contextKey:    canonicalContextParams(cpMap),
		})
	}
	return out, nil
}

// applyTeamRoles reconciles the team's role assignments to exactly `desired`.
// It adds missing assignments and removes extras, skipping internal (system) ones.
func (r *TeamResource) applyTeamRoles(ctx context.Context, orgID, teamID string, desired []desiredTeamRole) error {
	assignments, err := r.rolesClient.ListTeamRoles(ctx, orgID, teamID)
	if err != nil {
		return fmt.Errorf("could not list current team role assignments: %w", err)
	}

	// Build the set of catalog (assignable) role IDs. Any assignment whose role_id
	// is not in the catalog is a platform-injected side-effect grant (e.g. the
	// org-scoped "Business Group Viewer" auto-added alongside an env-scoped role)
	// and must never be removed — the user can't express it in config, so removing
	// it would fight the platform on every apply.
	catalog, err := r.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		return fmt.Errorf("could not list available roles for assignment reconciliation: %w", err)
	}
	catalogIDs := make(map[string]struct{}, len(catalog))
	for _, role := range catalog {
		catalogIDs[role.RoleID] = struct{}{}
	}

	actual := make(map[string]accessmanagement.TeamRoleAssignment)
	for _, a := range assignments {
		if a.Internal {
			continue // never touch system-managed assignments
		}
		if _, inCatalog := catalogIDs[a.RoleID]; !inCatalog {
			continue // never touch platform-injected side-effect grants
		}
		actual[a.RoleID+"|"+canonicalContextParams(a.ContextParams)] = a
	}

	desiredByKey := make(map[string]desiredTeamRole, len(desired))
	for _, d := range desired {
		desiredByKey[d.roleID+"|"+d.contextKey] = d
	}

	// Add anything desired that isn't already present.
	for key, d := range desiredByKey {
		if _, ok := actual[key]; ok {
			continue
		}
		if _, err := r.rolesClient.AssignTeamRole(ctx, orgID, teamID, &accessmanagement.AssignTeamRoleRequest{
			RoleID:        d.roleID,
			ContextParams: d.contextParams,
		}); err != nil {
			return fmt.Errorf("failed to assign role %q: %w", d.typedName, err)
		}
	}

	// Remove anything present that is no longer desired.
	for key, a := range actual {
		if _, ok := desiredByKey[key]; ok {
			continue
		}
		if err := r.rolesClient.UnassignTeamRole(ctx, orgID, teamID, &accessmanagement.AssignTeamRoleRequest{
			RoleID:        a.RoleID,
			ContextParams: a.ContextParams,
		}); err != nil {
			return fmt.Errorf("failed to unassign role %q: %w", a.Name, err)
		}
	}

	return nil
}

// reconcileTeamRolesIntoState reads the actual (non-internal) role assignments from
// the API and builds a roles set. For each assignment matching an entry in
// typedSource, the user's original name/context_params are preserved (so casing
// differences don't cause perpetual diffs). Unmatched assignments (drift or import)
// are labeled with the canonical role name from the catalog.
func (r *TeamResource) reconcileTeamRolesIntoState(ctx context.Context, orgID, teamID string, typedSource types.Set) (types.Set, error) {
	assignments, err := r.rolesClient.ListTeamRoles(ctx, orgID, teamID)
	if err != nil {
		return types.SetNull(teamRoleObjectType), fmt.Errorf("could not list team role assignments: %w", err)
	}

	roles, err := r.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		return types.SetNull(teamRoleObjectType), fmt.Errorf("could not list available roles: %w", err)
	}
	roleIDToName := make(map[string]string, len(roles))
	nameToID := make(map[string]string, len(roles))
	for _, role := range roles {
		roleIDToName[role.RoleID] = role.Name
		nameToID[normalizeName(role.Name)] = role.RoleID
	}

	// Index the user's typed roles by (role_id | context) so we can preserve their
	// exact representation for matched assignments.
	typedByKey := map[string]typedTeamRole{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			obj := el.(types.Object)
			attrs := obj.Attributes()
			typedName := attrs["name"].(types.String)
			cpTypes := attrs["context_params"].(types.Map)
			rid := nameToID[normalizeName(typedName.ValueString())]
			key := rid + "|" + canonicalContextParams(mapToStringMap(cpTypes))
			typedByKey[key] = typedTeamRole{name: typedName, cp: cpTypes}
		}
	}

	objs := make([]attr.Value, 0, len(assignments))
	for _, a := range assignments {
		if a.Internal {
			continue
		}
		// Skip platform-injected side-effect grants that are not in the assignable
		// catalog (e.g. the org-scoped "Business Group Viewer" the platform auto-adds
		// when any env-scoped role is assigned to a team). The user cannot express
		// these in config — resolveTeamRoles only accepts catalog role names — so
		// treating them as managed would surface a phantom `name = ""` entry and a
		// perpetual diff. Mirror the internal-skip above.
		if _, inCatalog := roleIDToName[a.RoleID]; !inCatalog {
			continue
		}
		key := a.RoleID + "|" + canonicalContextParams(a.ContextParams)

		var nameVal types.String
		var cpVal types.Map
		if tr, ok := typedByKey[key]; ok {
			nameVal = tr.name
			cpVal = tr.cp
		} else {
			nameVal = types.StringValue(roleIDToName[a.RoleID])
			cpVal = stringMapToTypesMap(a.ContextParams)
		}

		obj, diags := types.ObjectValue(teamRoleObjectType.AttrTypes, map[string]attr.Value{
			"name":           nameVal,
			"context_params": cpVal,
		})
		if diags.HasError() {
			return types.SetNull(teamRoleObjectType), fmt.Errorf("failed to build team role object")
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(teamRoleObjectType, objs)
	if diags.HasError() {
		return types.SetNull(teamRoleObjectType), fmt.Errorf("failed to build team roles set")
	}
	return set, nil
}

// --- members -----------------------------------------------------------------

// resolveTeamMembers resolves each username to a user_id (case-insensitive) using
// the org user list. Returns a map of user_id -> desiredTeamMember (carrying the
// typed username and resolved membership type). Returns an error listing the
// offending username if it is unknown. Returns (nil, nil) when unmanaged.
func (r *TeamResource) resolveTeamMembers(ctx context.Context, orgID string, memberSet types.Set) (map[string]desiredTeamMember, error) {
	if memberSet.IsNull() || memberSet.IsUnknown() {
		return nil, nil
	}

	users, err := r.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not list organization users for username resolution: %w", err)
	}
	usernameToID := make(map[string]string, len(users))
	for _, u := range users {
		usernameToID[normalizeName(u.Username)] = u.ID
	}

	out := make(map[string]desiredTeamMember)
	for _, el := range memberSet.Elements() {
		obj := el.(types.Object)
		attrs := obj.Attributes()
		typedUsername := attrs["username"].(types.String).ValueString()

		membershipType := "member"
		if mt, ok := attrs["membership_type"].(types.String); ok && !mt.IsNull() && !mt.IsUnknown() && mt.ValueString() != "" {
			membershipType = mt.ValueString()
		}

		id, ok := usernameToID[normalizeName(typedUsername)]
		if !ok {
			return nil, fmt.Errorf(
				"member %q is not a valid username in organization %s; use the anypoint_users data source to discover usernames",
				typedUsername, orgID,
			)
		}
		out[id] = desiredTeamMember{
			userID:         id,
			typedUsername:  typedUsername,
			membershipType: membershipType,
		}
	}
	return out, nil
}

// applyTeamMembers reconciles the team's membership to exactly the desired members.
// Members assigned via external groups (SAML/SCIM) are never modified. Members whose
// membership_type changed are re-PUT to update in place.
func (r *TeamResource) applyTeamMembers(ctx context.Context, orgID, teamID string, desired map[string]desiredTeamMember) error {
	actual, err := r.membersClient.ListTeamMembers(ctx, orgID, teamID)
	if err != nil {
		return fmt.Errorf("could not list current team members: %w", err)
	}

	// Build the set of real org-user IDs. The team-members endpoint also returns
	// non-user identities: once a team has a child team, that child surfaces here as
	// a member entry (and the platform may inject other group identities). Those are
	// NOT org users, so they can never appear in `desired` (which is built by
	// resolving usernames). They must NOT be removed — deleting a child team via the
	// members endpoint fails with 405 "Cannot delete a child team using this
	// endpoint". Mirroring the read path (reconcileTeamMembersIntoState), we only
	// manage members that resolve to an org user; every other identity is left
	// untouched, exactly like external-group memberships.
	users, err := r.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		return fmt.Errorf("could not list organization users: %w", err)
	}
	isOrgUser := make(map[string]bool, len(users))
	for _, u := range users {
		isOrgUser[u.ID] = true
	}

	actualByID := make(map[string]accessmanagement.TeamMember, len(actual))
	for _, m := range actual {
		if m.IsAssignedViaExternalGroups {
			continue // never touch externally-managed memberships
		}
		if !isOrgUser[m.ID] {
			continue // non-user identity (e.g. a child team) — not manageable here
		}
		actualByID[m.ID] = m
	}

	// Add missing members, or update those whose membership_type changed.
	for id, d := range desired {
		cur, present := actualByID[id]
		if present && cur.MembershipType == d.membershipType {
			continue
		}
		if err := r.membersClient.AddTeamMember(ctx, orgID, teamID, id, d.membershipType); err != nil {
			return fmt.Errorf("failed to add/update member %q: %w", d.typedUsername, err)
		}
	}

	// Remove members that are no longer desired.
	for id := range actualByID {
		if _, ok := desired[id]; ok {
			continue
		}
		if err := r.membersClient.RemoveTeamMember(ctx, orgID, teamID, id); err != nil {
			return fmt.Errorf("failed to remove member (user_id %s): %w", id, err)
		}
	}

	return nil
}

// reconcileTeamMembersIntoState reads actual team members and builds a members set,
// preserving the user's typed username casing and (possibly null) membership_type
// for matched members. External-group memberships and non-user identities that
// cannot be mapped to an org username are excluded.
func (r *TeamResource) reconcileTeamMembersIntoState(ctx context.Context, orgID, teamID string, typedSource types.Set) (types.Set, error) {
	actual, err := r.membersClient.ListTeamMembers(ctx, orgID, teamID)
	if err != nil {
		return types.SetNull(teamMemberObjectType), fmt.Errorf("could not list team members: %w", err)
	}

	users, err := r.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		return types.SetNull(teamMemberObjectType), fmt.Errorf("could not list organization users: %w", err)
	}
	userIDToUsername := make(map[string]string, len(users))
	for _, u := range users {
		userIDToUsername[u.ID] = u.Username
	}

	// Index the user's typed members by lowercased username so we can preserve their
	// exact casing and membership_type form for matched members.
	typedByLower := map[string]typedTeamMember{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			obj := el.(types.Object)
			attrs := obj.Attributes()
			uname := attrs["username"].(types.String)
			mt, _ := attrs["membership_type"].(types.String)
			typedByLower[normalizeName(uname.ValueString())] = typedTeamMember{username: uname, membershipType: mt}
		}
	}

	objs := make([]attr.Value, 0, len(actual))
	for _, m := range actual {
		if m.IsAssignedViaExternalGroups {
			continue
		}
		username, ok := userIDToUsername[m.ID]
		if !ok {
			// Not an org user (e.g. a group identity) — cannot represent by username.
			continue
		}

		var usernameVal types.String
		var mtVal types.String
		if tm, matched := typedByLower[normalizeName(username)]; matched {
			usernameVal = tm.username
			// Preserve the typed membership_type verbatim (including null when the
			// user omitted it) so state matches config and avoids perpetual diffs.
			mtVal = tm.membershipType
		} else {
			usernameVal = types.StringValue(username)
			if m.MembershipType != "" {
				mtVal = types.StringValue(m.MembershipType)
			} else {
				mtVal = types.StringValue("member")
			}
		}

		obj, diags := types.ObjectValue(teamMemberObjectType.AttrTypes, map[string]attr.Value{
			"username":        usernameVal,
			"membership_type": mtVal,
		})
		if diags.HasError() {
			return types.SetNull(teamMemberObjectType), fmt.Errorf("failed to build team member object")
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(teamMemberObjectType, objs)
	if diags.HasError() {
		return types.SetNull(teamMemberObjectType), fmt.Errorf("failed to build team members set")
	}
	return set, nil
}
