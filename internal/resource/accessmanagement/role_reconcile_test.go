package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// --- Pure helper tests -------------------------------------------------------

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"Exchange Viewer":   "exchange viewer",
		"  Exchange Viewer": "exchange viewer",
		"EXCHANGE VIEWER  ": "exchange viewer",
		"alice":             "alice",
	}
	for in, want := range cases {
		if got := normalizeName(in); got != want {
			t.Errorf("normalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCanonicalContextParams(t *testing.T) {
	// nil and empty map both canonicalize to "{}".
	if got := canonicalContextParams(nil); got != "{}" {
		t.Errorf("canonicalContextParams(nil) = %q, want {}", got)
	}
	if got := canonicalContextParams(map[string]string{}); got != "{}" {
		t.Errorf("canonicalContextParams(empty) = %q, want {}", got)
	}
	// Key order is stable regardless of insertion order.
	a := canonicalContextParams(map[string]string{"org": "o1", "envId": "e1"})
	b := canonicalContextParams(map[string]string{"envId": "e1", "org": "o1"})
	if a != b {
		t.Errorf("canonicalContextParams not order-stable: %q vs %q", a, b)
	}
}

func TestStringMapToTypesMap_EmptyIsNull(t *testing.T) {
	if m := stringMapToTypesMap(map[string]string{}); !m.IsNull() {
		t.Errorf("expected empty map to become null, got %v", m)
	}
	if m := stringMapToTypesMap(nil); !m.IsNull() {
		t.Errorf("expected nil map to become null, got %v", m)
	}
	m := stringMapToTypesMap(map[string]string{"org": "o1"})
	if m.IsNull() {
		t.Errorf("expected non-empty map to be non-null")
	}
}

// --- Test harness ------------------------------------------------------------

// roleTestServerState captures the mutable server-side state so apply tests can
// assert which assignments/members exist after reconcile.
type roleTestServerState struct {
	mu          sync.Mutex
	assignments []accessmanagement.RoleAssignment
	members     []accessmanagement.RoleGroupUser
}

const (
	testOrgID = "test-org-id"
	testRGID  = "test-rg-id"
)

// newRoleTestResource spins up a mock server backing all three sub-clients and
// returns a RoleResource wired to it plus the mutable state for assertions.
func newRoleTestResource(t *testing.T, state *roleTestServerState) *RoleResource {
	t.Helper()

	rolesCatalog := []accessmanagement.AvailableRole{
		{RoleID: "role-exchange-viewer", Name: "Exchange Viewer"},
		{RoleID: "role-exchange-admin", Name: "Exchange Administrator"},
		{RoleID: "role-dup", Name: "Duplicate Role"},
		{RoleID: "role-dup-2", Name: "Duplicate Role"}, // ambiguous by name
	}
	orgUsers := []accessmanagement.OrgUser{
		{ID: "user-1", Username: "alice"},
		{ID: "user-2", Username: "bob"},
		{ID: "user-3", Username: "carol"},
	}

	rgRolesPath := "/accounts/api/organizations/" + testOrgID + "/rolegroups/" + testRGID + "/roles"
	rgUsersPath := "/accounts/api/organizations/" + testOrgID + "/rolegroups/" + testRGID + "/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": rolesCatalog, "total": len(rolesCatalog),
			})
		},
		"/accounts/api/organizations/" + testOrgID + "/users": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": orgUsers, "total": len(orgUsers),
			})
		},
		rgRolesPath: func(w http.ResponseWriter, r *http.Request) {
			state.mu.Lock()
			defer state.mu.Unlock()
			switch r.Method {
			case http.MethodGet:
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"data": state.assignments, "total": len(state.assignments),
				})
			case http.MethodPost:
				var body []accessmanagement.AssignRoleRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				for _, req := range body {
					state.assignments = append(state.assignments, accessmanagement.RoleAssignment{
						RoleGroupID:   testRGID,
						RoleID:        req.RoleID,
						OrgID:         testOrgID,
						ContextParams: req.ContextParams,
					})
				}
				testutil.JSONResponse(w, http.StatusOK, []accessmanagement.AssignRoleResponse{
					{RoleGroupID: testRGID, RoleID: body[0].RoleID, RoleGroupAssignmentID: "new-assign", ContextParams: body[0].ContextParams},
				})
			case http.MethodDelete:
				var body []accessmanagement.AssignRoleRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				for _, req := range body {
					kept := state.assignments[:0]
					for _, a := range state.assignments {
						if a.RoleID == req.RoleID && canonicalContextParams(a.ContextParams) == canonicalContextParams(req.ContextParams) {
							continue
						}
						kept = append(kept, a)
					}
					state.assignments = kept
				}
				w.WriteHeader(http.StatusOK)
			}
		},
		rgUsersPath: func(w http.ResponseWriter, r *http.Request) {
			state.mu.Lock()
			defer state.mu.Unlock()
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": state.members, "total": len(state.members),
			})
		},
	}

	// Per-user add/remove endpoints: register one handler per known user id.
	for _, u := range orgUsers {
		uid := u.ID
		uname := u.Username
		handlers[rgUsersPath+"/"+uid] = func(w http.ResponseWriter, r *http.Request) {
			state.mu.Lock()
			defer state.mu.Unlock()
			switch r.Method {
			case http.MethodPost:
				state.members = append(state.members, accessmanagement.RoleGroupUser{ID: uid, Username: uname})
				w.WriteHeader(http.StatusOK)
			case http.MethodDelete:
				kept := state.members[:0]
				for _, m := range state.members {
					if m.ID == uid {
						continue
					}
					kept = append(kept, m)
				}
				state.members = kept
				w.WriteHeader(http.StatusOK)
			}
		}
	}

	server := testutil.MockHTTPServer(t, handlers)
	userClient := &client.UserAnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      testOrgID,
	}
	return &RoleResource{
		client:      &accessmanagement.RoleClient{UserAnypointClient: userClient},
		permClient:  &accessmanagement.RolePermissionClient{UserAnypointClient: userClient},
		usersClient: &accessmanagement.RoleUsersClient{UserAnypointClient: userClient},
	}
}

// permSet builds a types.Set of permission objects from (name, context_params) pairs.
func permSet(t *testing.T, entries ...typedPerm) types.Set {
	t.Helper()
	objs := make([]attr.Value, 0, len(entries))
	for _, e := range entries {
		cp := e.cp
		if cp.IsNull() || cp.IsUnknown() {
			cp = types.MapNull(types.StringType)
		}
		obj, diags := types.ObjectValue(permissionObjectType.AttrTypes, map[string]attr.Value{
			"name":           e.name,
			"context_params": cp,
		})
		if diags.HasError() {
			t.Fatalf("permSet build error: %v", diags.Errors())
		}
		objs = append(objs, obj)
	}
	set, diags := types.SetValue(permissionObjectType, objs)
	if diags.HasError() {
		t.Fatalf("permSet set error: %v", diags.Errors())
	}
	return set
}

func strMap(kv map[string]string) types.Map {
	if len(kv) == 0 {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(kv))
	for k, v := range kv {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}

func memberSet(vals ...string) types.Set {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	return types.SetValueMust(types.StringType, elems)
}

// --- resolvePermissions ------------------------------------------------------

func TestResolvePermissions_Unmanaged(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	got, err := r.resolvePermissions(context.Background(), types.SetNull(permissionObjectType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unmanaged (null) set, got %v", got)
	}
}

func TestResolvePermissions_CaseInsensitive(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	set := permSet(t, typedPerm{name: types.StringValue("exCHANge VIEWer"), cp: strMap(map[string]string{"org": testOrgID})})
	got, err := r.resolvePermissions(context.Background(), set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].roleID != "role-exchange-viewer" {
		t.Fatalf("expected role-exchange-viewer, got %+v", got)
	}
	// typedName preserves exactly what the user wrote.
	if got[0].typedName != "exCHANge VIEWer" {
		t.Errorf("expected typedName preserved, got %q", got[0].typedName)
	}
}

func TestResolvePermissions_UnknownName(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	set := permSet(t, typedPerm{name: types.StringValue("No Such Role")})
	_, err := r.resolvePermissions(context.Background(), set)
	if err == nil {
		t.Fatal("expected error for unknown permission name")
	}
}

func TestResolvePermissions_AmbiguousName(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	set := permSet(t, typedPerm{name: types.StringValue("Duplicate Role")})
	_, err := r.resolvePermissions(context.Background(), set)
	if err == nil {
		t.Fatal("expected error for ambiguous permission name")
	}
}

// --- applyPermissions --------------------------------------------------------

func TestApplyPermissions_AddRemoveSkipInternal(t *testing.T) {
	state := &roleTestServerState{
		assignments: []accessmanagement.RoleAssignment{
			// Pre-existing desired assignment (should be kept, untouched).
			{RoleID: "role-exchange-viewer", ContextParams: map[string]string{"org": testOrgID}},
			// Extra non-internal assignment (should be removed).
			{RoleID: "role-exchange-admin", ContextParams: map[string]string{"org": testOrgID}},
			// Internal assignment (must never be touched).
			{RoleID: "role-internal-xyz", Internal: true, ContextParams: map[string]string{"org": testOrgID}},
		},
	}
	r := newRoleTestResource(t, state)

	desired := []desiredPermission{
		{roleID: "role-exchange-viewer", contextParams: map[string]string{"org": testOrgID}, contextKey: canonicalContextParams(map[string]string{"org": testOrgID})},
	}
	if err := r.applyPermissions(context.Background(), testOrgID, testRGID, desired); err != nil {
		t.Fatalf("applyPermissions error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var haveViewer, haveAdmin, haveInternal bool
	for _, a := range state.assignments {
		switch a.RoleID {
		case "role-exchange-viewer":
			haveViewer = true
		case "role-exchange-admin":
			haveAdmin = true
		case "role-internal-xyz":
			haveInternal = true
		}
	}
	if !haveViewer {
		t.Error("expected desired role-exchange-viewer to remain")
	}
	if haveAdmin {
		t.Error("expected extra role-exchange-admin to be removed")
	}
	if !haveInternal {
		t.Error("internal assignment must never be removed")
	}
}

// --- resolveMembers ----------------------------------------------------------

func TestResolveMembers_CaseInsensitive(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	got, err := r.resolveMembers(context.Background(), testOrgID, memberSet("ALICE", "Bob"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["user-1"] != "ALICE" || got["user-2"] != "Bob" {
		t.Errorf("expected user-1->ALICE, user-2->Bob, got %+v", got)
	}
}

func TestResolveMembers_UnknownUsername(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	_, err := r.resolveMembers(context.Background(), testOrgID, memberSet("nobody"))
	if err == nil {
		t.Fatal("expected error for unknown username")
	}
}

// --- applyMembers ------------------------------------------------------------

func TestApplyMembers_AddRemove(t *testing.T) {
	state := &roleTestServerState{
		members: []accessmanagement.RoleGroupUser{
			{ID: "user-1", Username: "alice"}, // keep
			{ID: "user-3", Username: "carol"}, // remove
		},
	}
	r := newRoleTestResource(t, state)

	// Desired: alice (keep) + bob (add). carol should be removed.
	desired := map[string]string{"user-1": "alice", "user-2": "bob"}
	if err := r.applyMembers(context.Background(), testOrgID, testRGID, desired); err != nil {
		t.Fatalf("applyMembers error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	got := map[string]bool{}
	for _, m := range state.members {
		got[m.ID] = true
	}
	if !got["user-1"] {
		t.Error("expected alice (user-1) to remain")
	}
	if !got["user-2"] {
		t.Error("expected bob (user-2) to be added")
	}
	if got["user-3"] {
		t.Error("expected carol (user-3) to be removed")
	}
}

// --- reconcile*IntoState (typed-casing preservation) -------------------------

func TestReconcilePermissionsIntoState_PreservesTypedCasing(t *testing.T) {
	state := &roleTestServerState{
		assignments: []accessmanagement.RoleAssignment{
			{RoleID: "role-exchange-viewer", Name: "Exchange Viewer", ContextParams: map[string]string{"org": testOrgID}},
			{RoleID: "role-internal-xyz", Name: "Internal", Internal: true},
		},
	}
	r := newRoleTestResource(t, state)

	// User originally typed a different casing; reconcile should preserve it.
	typed := permSet(t, typedPerm{name: types.StringValue("exchange viewer"), cp: strMap(map[string]string{"org": testOrgID})})
	got, err := r.reconcilePermissionsIntoState(context.Background(), testOrgID, testRGID, typed)
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	// Internal assignment excluded => exactly 1 entry.
	if len(got.Elements()) != 1 {
		t.Fatalf("expected 1 permission (internal excluded), got %d", len(got.Elements()))
	}
	obj := got.Elements()[0].(types.Object)
	name := obj.Attributes()["name"].(types.String).ValueString()
	if name != "exchange viewer" {
		t.Errorf("expected typed casing 'exchange viewer' preserved, got %q", name)
	}
}

func TestReconcileMembersIntoState_PreservesTypedCasing(t *testing.T) {
	state := &roleTestServerState{
		members: []accessmanagement.RoleGroupUser{
			{ID: "user-1", Username: "alice"},
		},
	}
	r := newRoleTestResource(t, state)

	typed := memberSet("ALICE")
	got, err := r.reconcileMembersIntoState(context.Background(), testOrgID, testRGID, typed)
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	vals := make([]string, 0, len(got.Elements()))
	for _, el := range got.Elements() {
		vals = append(vals, el.(types.String).ValueString())
	}
	sort.Strings(vals)
	if len(vals) != 1 || vals[0] != "ALICE" {
		t.Errorf("expected typed casing 'ALICE' preserved, got %v", vals)
	}
}

// --- EDGE CASES: deleted user / deleted permission, null-vs-empty ------------
//
// Parallel to the team-side edge cases: an unknown username or permission name at
// APPLY time (resolveMembers/resolvePermissions) is a hard error that names the
// offender; an empty (non-null) set is authoritative-empty (remove all), while a
// null set means unmanaged.

// TestResolveMembers_DeletedUser_ErrorNamesUser: a member deleted from the org but
// still referenced fails the apply with a message naming the username.
func TestResolveMembers_DeletedUser_ErrorNamesUser(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	_, err := r.resolveMembers(context.Background(), testOrgID, memberSet("alice", "dave"))
	if err == nil {
		t.Fatal("expected error when a referenced member no longer exists in the org")
	}
	if !strings.Contains(err.Error(), `"dave"`) {
		t.Errorf("error must name the offending username so the operator can fix the .tf, got: %v", err)
	}
	if !strings.Contains(err.Error(), "anypoint_users") {
		t.Errorf("error should point at the anypoint_users data source for discovery, got: %v", err)
	}
}

// TestResolvePermissions_DeletedPermission_ErrorNamesPermission: a permission
// removed from the catalog but still referenced fails the apply, naming it.
func TestResolvePermissions_DeletedPermission_ErrorNamesPermission(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	set := permSet(t,
		typedPerm{name: types.StringValue("Exchange Viewer"), cp: strMap(map[string]string{"org": testOrgID})},
		typedPerm{name: types.StringValue("Deleted Permission")},
	)
	_, err := r.resolvePermissions(context.Background(), set)
	if err == nil {
		t.Fatal("expected error when a referenced permission no longer exists in the catalog")
	}
	if !strings.Contains(err.Error(), `"Deleted Permission"`) {
		t.Errorf("error must name the offending permission, got: %v", err)
	}
}

// TestResolveMembers_EmptySetIsAuthoritativeEmpty: an empty (non-null) member set
// resolves to an empty (non-nil) map — "remove all managed members" — whereas a
// null set is unmanaged (nil). This boundary is what makes an explicit `members =
// []` wipe membership while omitting the attribute leaves it alone.
func TestResolveMembers_EmptySetIsAuthoritativeEmpty(t *testing.T) {
	r := newRoleTestResource(t, &roleTestServerState{})
	// unmanaged (null) -> nil
	got, err := r.resolveMembers(context.Background(), testOrgID, types.SetNull(types.StringType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("null set must be unmanaged (nil), got %v", got)
	}
	// authoritative-empty -> non-nil empty
	got, err = r.resolveMembers(context.Background(), testOrgID, memberSet())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("empty (non-null) set must NOT be unmanaged; expected non-nil empty map")
	}
	if len(got) != 0 {
		t.Errorf("empty set must resolve to zero desired members, got %d", len(got))
	}
}
