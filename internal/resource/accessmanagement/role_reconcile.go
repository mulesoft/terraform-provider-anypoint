package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// permissionObjectType is the Terraform object type for a single permission entry
// in the anypoint_role resource's `permissions` set.
var permissionObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

// desiredPermission is a resolved permission ready to be sent to the API.
type desiredPermission struct {
	roleID        string
	typedName     string            // exactly what the user wrote (preserved in state)
	contextParams map[string]string // resolved context params
	contextKey    string            // canonical JSON of contextParams for matching
}

// typedPerm holds the user's original (typed) representation of a permission so
// Read can preserve casing / context_params form and avoid perpetual diffs.
type typedPerm struct {
	name types.String
	cp   types.Map
}

// canonicalContextParams returns a stable JSON encoding of a context params map.
// A nil map and an empty map both canonicalize to "{}" so they compare equal.
// json.Marshal already emits map keys in sorted order, giving us a canonical form.
func canonicalContextParams(m map[string]string) string {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// mapToStringMap converts a types.Map (possibly null/unknown) to a plain map.
func mapToStringMap(m types.Map) map[string]string {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	for k, v := range m.Elements() {
		if s, ok := v.(types.String); ok {
			out[k] = s.ValueString()
		}
	}
	return out
}

// stringMapToTypesMap converts a plain map to a types.Map. An empty map becomes
// a null map so it matches a user who omitted context_params.
func stringMapToTypesMap(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}

// normalizeName lowercases and trims a display name / username for case-insensitive matching.
func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// resolvePermissions resolves each permission's display name to a role_id using the
// available-roles catalog (case-insensitive). Returns an error listing the offending
// name if it is unknown or ambiguous. Returns (nil, nil) when the set is unmanaged.
func (r *RoleResource) resolvePermissions(ctx context.Context, permSet types.Set) ([]desiredPermission, error) {
	if permSet.IsNull() || permSet.IsUnknown() {
		return nil, nil
	}

	roles, err := r.permClient.ListAvailableRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list available roles for name resolution: %w", err)
	}

	// Build a case-insensitive name -> []role_id map so we can detect ambiguity.
	nameToIDs := make(map[string][]string, len(roles))
	for _, role := range roles {
		key := normalizeName(role.Name)
		nameToIDs[key] = append(nameToIDs[key], role.RoleID)
	}

	out := make([]desiredPermission, 0, len(permSet.Elements()))
	for _, el := range permSet.Elements() {
		obj := el.(types.Object)
		attrs := obj.Attributes()
		typedName := attrs["name"].(types.String).ValueString()
		cpMap := mapToStringMap(attrs["context_params"].(types.Map))

		ids := nameToIDs[normalizeName(typedName)]
		if len(ids) == 0 {
			return nil, fmt.Errorf(
				"permission %q is not a valid role name; use the anypoint_available_roles data source to discover valid permission names",
				typedName,
			)
		}
		if len(ids) > 1 {
			return nil, fmt.Errorf(
				"permission name %q is ambiguous (matches %d roles); the platform has multiple roles with this name so it cannot be resolved by name alone",
				typedName, len(ids),
			)
		}

		out = append(out, desiredPermission{
			roleID:        ids[0],
			typedName:     typedName,
			contextParams: cpMap,
			contextKey:    canonicalContextParams(cpMap),
		})
	}
	return out, nil
}

// applyPermissions reconciles the role group's assignments to exactly `desired`.
// It adds missing assignments and removes extras, skipping internal (system) ones.
func (r *RoleResource) applyPermissions(ctx context.Context, orgID, roleGroupID string, desired []desiredPermission) error {
	assignments, err := r.permClient.ListRoleAssignments(ctx, orgID, roleGroupID)
	if err != nil {
		return fmt.Errorf("could not list current role assignments: %w", err)
	}

	// Build the set of catalog (assignable) role IDs. Any assignment whose role_id
	// is not in the catalog is a platform-injected side-effect grant (e.g. the
	// org-scoped "Business Group Viewer" the platform auto-adds alongside an
	// env-scoped role) and must never be removed — the user can't express it in
	// config, so removing it (in the remove loop below) would fail or fight the
	// platform on every apply. Mirror applyTeamRoles. (Class A invariant.)
	catalog, err := r.permClient.ListAvailableRoles(ctx)
	if err != nil {
		return fmt.Errorf("could not list available roles for assignment reconciliation: %w", err)
	}
	catalogIDs := make(map[string]struct{}, len(catalog))
	for _, role := range catalog {
		catalogIDs[role.RoleID] = struct{}{}
	}

	actual := make(map[string]accessmanagement.RoleAssignment)
	for _, a := range assignments {
		if a.Internal {
			continue // never touch system-managed assignments
		}
		if _, inCatalog := catalogIDs[a.RoleID]; !inCatalog {
			continue // never touch platform-injected side-effect grants
		}
		actual[a.RoleID+"|"+canonicalContextParams(a.ContextParams)] = a
	}

	desiredByKey := make(map[string]desiredPermission, len(desired))
	for _, d := range desired {
		desiredByKey[d.roleID+"|"+d.contextKey] = d
	}

	// Add anything desired that isn't already present.
	for key, d := range desiredByKey {
		if _, ok := actual[key]; ok {
			continue
		}
		if _, err := r.permClient.AssignRole(ctx, orgID, roleGroupID, &accessmanagement.AssignRoleRequest{
			RoleID:        d.roleID,
			ContextParams: d.contextParams,
		}); err != nil {
			return fmt.Errorf("failed to assign permission %q: %w", d.typedName, err)
		}
	}

	// Remove anything present that is no longer desired.
	for key, a := range actual {
		if _, ok := desiredByKey[key]; ok {
			continue
		}
		if err := r.permClient.UnassignRole(ctx, orgID, roleGroupID, &accessmanagement.AssignRoleRequest{
			RoleID:        a.RoleID,
			ContextParams: a.ContextParams,
		}); err != nil {
			return fmt.Errorf("failed to unassign permission %q: %w", a.Name, err)
		}
	}

	return nil
}

// reconcilePermissionsIntoState reads the actual (non-internal) assignments from the
// API and builds a permissions set. For each assignment that matches an entry in
// typedSource, the user's original name/context_params are preserved (so casing
// differences don't cause perpetual diffs). Unmatched assignments (drift or import)
// are labeled with the canonical role name from the catalog.
func (r *RoleResource) reconcilePermissionsIntoState(ctx context.Context, orgID, roleGroupID string, typedSource types.Set) (types.Set, error) {
	assignments, err := r.permClient.ListRoleAssignments(ctx, orgID, roleGroupID)
	if err != nil {
		return types.SetNull(permissionObjectType), fmt.Errorf("could not list role assignments: %w", err)
	}

	roles, err := r.permClient.ListAvailableRoles(ctx)
	if err != nil {
		return types.SetNull(permissionObjectType), fmt.Errorf("could not list available roles: %w", err)
	}
	roleIDToName := make(map[string]string, len(roles))
	nameToID := make(map[string]string, len(roles))
	for _, role := range roles {
		roleIDToName[role.RoleID] = role.Name
		nameToID[normalizeName(role.Name)] = role.RoleID
	}

	// Index the user's typed permissions by (role_id | context) so we can preserve
	// their exact representation for matched assignments.
	typedByKey := map[string]typedPerm{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			obj := el.(types.Object)
			attrs := obj.Attributes()
			typedName := attrs["name"].(types.String)
			cpTypes := attrs["context_params"].(types.Map)
			rid := nameToID[normalizeName(typedName.ValueString())]
			key := rid + "|" + canonicalContextParams(mapToStringMap(cpTypes))
			typedByKey[key] = typedPerm{name: typedName, cp: cpTypes}
		}
	}

	objs := make([]attr.Value, 0, len(assignments))
	for _, a := range assignments {
		if a.Internal {
			continue
		}
		// Skip platform-injected side-effect grants that are not in the assignable
		// catalog (e.g. the org-scoped "Business Group Viewer" the platform auto-adds
		// alongside an env-scoped role). The user cannot express these in config —
		// resolvePermissions only accepts catalog names — so surfacing them would show
		// a phantom `name = ""` entry and a perpetual diff. Mirror the internal-skip
		// above and the identical filter in applyPermissions. (Class A invariant.)
		if _, inCatalog := roleIDToName[a.RoleID]; !inCatalog {
			continue
		}
		key := a.RoleID + "|" + canonicalContextParams(a.ContextParams)

		var nameVal types.String
		var cpVal types.Map
		if tp, ok := typedByKey[key]; ok {
			nameVal = tp.name
			cpVal = tp.cp
		} else {
			nameVal = types.StringValue(roleIDToName[a.RoleID])
			cpVal = stringMapToTypesMap(a.ContextParams)
		}

		obj, diags := types.ObjectValue(permissionObjectType.AttrTypes, map[string]attr.Value{
			"name":           nameVal,
			"context_params": cpVal,
		})
		if diags.HasError() {
			return types.SetNull(permissionObjectType), fmt.Errorf("failed to build permission object")
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(permissionObjectType, objs)
	if diags.HasError() {
		return types.SetNull(permissionObjectType), fmt.Errorf("failed to build permissions set")
	}
	return set, nil
}

// resolveMembers resolves each username to a user_id (case-insensitive) using the
// org user list. Returns a map of user_id -> typed username. Returns an error listing
// the offending username if it is unknown. Returns (nil, nil) when unmanaged.
func (r *RoleResource) resolveMembers(ctx context.Context, orgID string, memberSet types.Set) (map[string]string, error) {
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

	out := make(map[string]string)
	for _, el := range memberSet.Elements() {
		typed := el.(types.String).ValueString()
		id, ok := usernameToID[normalizeName(typed)]
		if !ok {
			return nil, fmt.Errorf(
				"member %q is not a valid username in organization %s; use the anypoint_users data source to discover usernames",
				typed, orgID,
			)
		}
		out[id] = typed
	}
	return out, nil
}

// applyMembers reconciles the role group's membership to exactly the desired user IDs.
func (r *RoleResource) applyMembers(ctx context.Context, orgID, roleGroupID string, desiredIDToUsername map[string]string) error {
	actual, err := r.usersClient.ListRoleGroupUsers(ctx, orgID, roleGroupID)
	if err != nil {
		return fmt.Errorf("could not list current role group members: %w", err)
	}
	actualIDs := make(map[string]bool, len(actual))
	for _, u := range actual {
		actualIDs[u.ID] = true
	}

	// Add missing members.
	for id, username := range desiredIDToUsername {
		if actualIDs[id] {
			continue
		}
		if err := r.usersClient.AddUserToRoleGroup(ctx, orgID, roleGroupID, id); err != nil {
			return fmt.Errorf("failed to add member %q: %w", username, err)
		}
	}

	// Remove members that are no longer desired.
	for id := range actualIDs {
		if _, ok := desiredIDToUsername[id]; ok {
			continue
		}
		if err := r.usersClient.RemoveUserFromRoleGroup(ctx, orgID, roleGroupID, id); err != nil {
			return fmt.Errorf("failed to remove member (user_id %s): %w", id, err)
		}
	}

	return nil
}

// reconcileMembersIntoState reads actual role group members and builds a members set,
// preserving the user's typed username casing for matched members.
func (r *RoleResource) reconcileMembersIntoState(ctx context.Context, orgID, roleGroupID string, typedSource types.Set) (types.Set, error) {
	actual, err := r.usersClient.ListRoleGroupUsers(ctx, orgID, roleGroupID)
	if err != nil {
		return types.SetNull(types.StringType), fmt.Errorf("could not list role group members: %w", err)
	}

	typedByLower := map[string]string{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			s := el.(types.String).ValueString()
			typedByLower[normalizeName(s)] = s
		}
	}

	vals := make([]attr.Value, 0, len(actual))
	for _, u := range actual {
		if typed, ok := typedByLower[normalizeName(u.Username)]; ok {
			vals = append(vals, types.StringValue(typed))
		} else {
			vals = append(vals, types.StringValue(u.Username))
		}
	}

	set, diags := types.SetValue(types.StringType, vals)
	if diags.HasError() {
		return types.SetNull(types.StringType), fmt.Errorf("failed to build members set")
	}
	return set, nil
}
