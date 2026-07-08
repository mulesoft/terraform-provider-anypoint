package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// testOrgID, strMap, normalizeName, canonicalContextParams and stringMapToTypesMap
// are shared with role_reconcile_test.go / role_reconcile.go (same package).
const testTeamID = "test-team-id"

// childTeamID is a non-user identity that the platform surfaces in a team's member
// list once the team has a child team. It is NOT an org user, and its DELETE on the
// members endpoint is rejected with 405 (see the harness handler). Used to reproduce
// the "Cannot delete a child team using this endpoint" apply failure.
const childTeamID = "child-team-id"

// --- Test harness ------------------------------------------------------------

// teamTestServerState captures the mutable server-side state so apply tests can
// assert which role assignments/members exist after reconcile.
type teamTestServerState struct {
	mu          sync.Mutex
	assignments []accessmanagement.TeamRoleAssignment
	members     []accessmanagement.TeamMember
}

// newTeamTestResource spins up a mock server backing all five sub-clients and
// returns a TeamResource wired to it plus the mutable state for assertions.
func newTeamTestResource(t *testing.T, state *teamTestServerState) *TeamResource {
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

	teamRolesPath := "/accounts/api/organizations/" + testOrgID + "/teams/" + testTeamID + "/roles"
	teamMembersPath := "/accounts/api/organizations/" + testOrgID + "/teams/" + testTeamID + "/members"

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
		teamRolesPath: func(w http.ResponseWriter, r *http.Request) {
			state.mu.Lock()
			defer state.mu.Unlock()
			switch r.Method {
			case http.MethodGet:
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"data": state.assignments, "total": len(state.assignments),
				})
			case http.MethodPost:
				var body []accessmanagement.AssignTeamRoleRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				for _, req := range body {
					state.assignments = append(state.assignments, accessmanagement.TeamRoleAssignment{
						RoleID:        req.RoleID,
						OrgID:         testOrgID,
						ContextParams: req.ContextParams,
					})
				}
				// Empty body -> AssignTeamRole reads back via a subsequent GET.
				w.WriteHeader(http.StatusNoContent)
			case http.MethodDelete:
				var body []accessmanagement.AssignTeamRoleRequest
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
				w.WriteHeader(http.StatusNoContent)
			}
		},
		teamMembersPath: func(w http.ResponseWriter, r *http.Request) {
			state.mu.Lock()
			defer state.mu.Unlock()
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": state.members, "total": len(state.members),
			})
		},
	}

	// Child-team identity endpoint: the platform rejects DELETE of a child team via
	// the members endpoint with 405. applyTeamMembers must never call this; if it
	// does (the bug), the reconcile fails exactly as it did in the real apply.
	handlers[teamMembersPath+"/"+childTeamID] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Cannot delete a child team using this endpoint")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}

	// Per-user add(PUT)/remove(DELETE) endpoints: one handler per known user id.
	for _, u := range orgUsers {
		uid := u.ID
		handlers[teamMembersPath+"/"+uid] = func(w http.ResponseWriter, r *http.Request) {
			state.mu.Lock()
			defer state.mu.Unlock()
			switch r.Method {
			case http.MethodPut:
				var body accessmanagement.AddTeamMemberRequest
				_ = json.NewDecoder(r.Body).Decode(&body)
				mt := body.MembershipType
				if mt == "" {
					mt = "member"
				}
				// Update in place if already present, else append.
				found := false
				for i := range state.members {
					if state.members[i].ID == uid {
						state.members[i].MembershipType = mt
						found = true
						break
					}
				}
				if !found {
					state.members = append(state.members, accessmanagement.TeamMember{ID: uid, MembershipType: mt})
				}
				w.WriteHeader(http.StatusNoContent)
			case http.MethodDelete:
				kept := state.members[:0]
				for _, m := range state.members {
					if m.ID == uid {
						continue
					}
					kept = append(kept, m)
				}
				state.members = kept
				w.WriteHeader(http.StatusNoContent)
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
	return &TeamResource{
		client:        &accessmanagement.TeamClient{UserAnypointClient: userClient},
		rolesClient:   &accessmanagement.TeamRolesClient{UserAnypointClient: userClient},
		membersClient: &accessmanagement.TeamMembersClient{UserAnypointClient: userClient},
		usersClient:   &accessmanagement.RoleUsersClient{UserAnypointClient: userClient},
		catalogClient: &accessmanagement.RolePermissionClient{UserAnypointClient: userClient},
	}
}

// teamRoleSet builds a types.Set of team-role objects from typed entries.
func teamRoleSet(t *testing.T, entries ...typedTeamRole) types.Set {
	t.Helper()
	objs := make([]attr.Value, 0, len(entries))
	for _, e := range entries {
		cp := e.cp
		if cp.IsNull() || cp.IsUnknown() {
			cp = types.MapNull(types.StringType)
		}
		obj, diags := types.ObjectValue(teamRoleObjectType.AttrTypes, map[string]attr.Value{
			"name":           e.name,
			"context_params": cp,
		})
		if diags.HasError() {
			t.Fatalf("teamRoleSet build error: %v", diags.Errors())
		}
		objs = append(objs, obj)
	}
	set, diags := types.SetValue(teamRoleObjectType, objs)
	if diags.HasError() {
		t.Fatalf("teamRoleSet set error: %v", diags.Errors())
	}
	return set
}

// teamMemberSet builds a types.Set of team-member objects from typed entries.
func teamMemberSet(t *testing.T, entries ...typedTeamMember) types.Set {
	t.Helper()
	objs := make([]attr.Value, 0, len(entries))
	for _, e := range entries {
		obj, diags := types.ObjectValue(teamMemberObjectType.AttrTypes, map[string]attr.Value{
			"username":        e.username,
			"membership_type": e.membershipType,
		})
		if diags.HasError() {
			t.Fatalf("teamMemberSet build error: %v", diags.Errors())
		}
		objs = append(objs, obj)
	}
	set, diags := types.SetValue(teamMemberObjectType, objs)
	if diags.HasError() {
		t.Fatalf("teamMemberSet set error: %v", diags.Errors())
	}
	return set
}

// --- resolveTeamRoles --------------------------------------------------------

func TestResolveTeamRoles_Unmanaged(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	got, err := r.resolveTeamRoles(context.Background(), types.SetNull(teamRoleObjectType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unmanaged (null) set, got %v", got)
	}
}

func TestResolveTeamRoles_CaseInsensitive(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	set := teamRoleSet(t, typedTeamRole{name: types.StringValue("exCHANge VIEWer"), cp: strMap(map[string]string{"org": testOrgID})})
	got, err := r.resolveTeamRoles(context.Background(), set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].roleID != "role-exchange-viewer" {
		t.Fatalf("expected role-exchange-viewer, got %+v", got)
	}
	if got[0].typedName != "exCHANge VIEWer" {
		t.Errorf("expected typedName preserved, got %q", got[0].typedName)
	}
}

func TestResolveTeamRoles_UnknownName(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	set := teamRoleSet(t, typedTeamRole{name: types.StringValue("No Such Role"), cp: types.MapNull(types.StringType)})
	_, err := r.resolveTeamRoles(context.Background(), set)
	if err == nil {
		t.Fatal("expected error for unknown role name")
	}
}

func TestResolveTeamRoles_AmbiguousName(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	set := teamRoleSet(t, typedTeamRole{name: types.StringValue("Duplicate Role"), cp: types.MapNull(types.StringType)})
	_, err := r.resolveTeamRoles(context.Background(), set)
	if err == nil {
		t.Fatal("expected error for ambiguous role name")
	}
}

// --- applyTeamRoles ----------------------------------------------------------

func TestApplyTeamRoles_AddRemoveSkipInternal(t *testing.T) {
	state := &teamTestServerState{
		assignments: []accessmanagement.TeamRoleAssignment{
			// Pre-existing desired assignment (kept, untouched).
			{RoleID: "role-exchange-viewer", ContextParams: map[string]string{"org": testOrgID}},
			// Extra non-internal assignment (removed).
			{RoleID: "role-dup", ContextParams: map[string]string{"org": testOrgID}},
			// Internal assignment (never touched).
			{RoleID: "role-internal-xyz", Internal: true, ContextParams: map[string]string{"org": testOrgID}},
		},
	}
	r := newTeamTestResource(t, state)

	cp := map[string]string{"org": testOrgID}
	desired := []desiredTeamRole{
		{roleID: "role-exchange-viewer", contextParams: cp, contextKey: canonicalContextParams(cp)},
		{roleID: "role-exchange-admin", contextParams: cp, contextKey: canonicalContextParams(cp)}, // added
	}
	if err := r.applyTeamRoles(context.Background(), testOrgID, testTeamID, desired); err != nil {
		t.Fatalf("applyTeamRoles error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var haveViewer, haveAdmin, haveDup, haveInternal bool
	for _, a := range state.assignments {
		switch a.RoleID {
		case "role-exchange-viewer":
			haveViewer = true
		case "role-exchange-admin":
			haveAdmin = true
		case "role-dup":
			haveDup = true
		case "role-internal-xyz":
			haveInternal = true
		}
	}
	if !haveViewer {
		t.Error("expected desired role-exchange-viewer to remain")
	}
	if !haveAdmin {
		t.Error("expected role-exchange-admin to be added")
	}
	if haveDup {
		t.Error("expected extra role-dup to be removed")
	}
	if !haveInternal {
		t.Error("internal assignment must never be removed")
	}
}

// --- resolveTeamMembers ------------------------------------------------------

func TestResolveTeamMembers_Unmanaged(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	got, err := r.resolveTeamMembers(context.Background(), testOrgID, types.SetNull(teamMemberObjectType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unmanaged (null) set, got %v", got)
	}
}

func TestResolveTeamMembers_CaseInsensitiveAndDefaultType(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	set := teamMemberSet(t,
		typedTeamMember{username: types.StringValue("ALICE"), membershipType: types.StringNull()},            // default -> member
		typedTeamMember{username: types.StringValue("Bob"), membershipType: types.StringValue("maintainer")}, // explicit
	)
	got, err := r.resolveTeamMembers(context.Background(), testOrgID, set)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a, ok := got["user-1"]
	if !ok || a.typedUsername != "ALICE" || a.membershipType != "member" {
		t.Errorf("expected user-1 -> {ALICE, member}, got %+v", a)
	}
	b, ok := got["user-2"]
	if !ok || b.typedUsername != "Bob" || b.membershipType != "maintainer" {
		t.Errorf("expected user-2 -> {Bob, maintainer}, got %+v", b)
	}
}

func TestResolveTeamMembers_UnknownUsername(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	set := teamMemberSet(t, typedTeamMember{username: types.StringValue("nobody"), membershipType: types.StringNull()})
	_, err := r.resolveTeamMembers(context.Background(), testOrgID, set)
	if err == nil {
		t.Fatal("expected error for unknown username")
	}
}

// --- applyTeamMembers --------------------------------------------------------

func TestApplyTeamMembers_AddRemoveTypeChange(t *testing.T) {
	state := &teamTestServerState{
		members: []accessmanagement.TeamMember{
			{ID: "user-1", MembershipType: "member"}, // desired -> change to maintainer
			{ID: "user-3", MembershipType: "member"}, // not desired -> remove
		},
	}
	r := newTeamTestResource(t, state)

	desired := map[string]desiredTeamMember{
		"user-1": {userID: "user-1", typedUsername: "alice", membershipType: "maintainer"}, // type change
		"user-2": {userID: "user-2", typedUsername: "bob", membershipType: "member"},       // add
	}
	if err := r.applyTeamMembers(context.Background(), testOrgID, testTeamID, desired); err != nil {
		t.Fatalf("applyTeamMembers error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	byID := map[string]accessmanagement.TeamMember{}
	for _, m := range state.members {
		byID[m.ID] = m
	}
	if m, ok := byID["user-1"]; !ok || m.MembershipType != "maintainer" {
		t.Errorf("expected user-1 present with maintainer, got %+v (ok=%v)", m, ok)
	}
	if _, ok := byID["user-2"]; !ok {
		t.Error("expected bob (user-2) to be added")
	}
	if _, ok := byID["user-3"]; ok {
		t.Error("expected carol (user-3) to be removed")
	}
}

func TestApplyTeamMembers_SkipsExternalGroup(t *testing.T) {
	state := &teamTestServerState{
		members: []accessmanagement.TeamMember{
			// External-group membership; not desired, but must never be removed.
			{ID: "user-2", MembershipType: "maintainer", IsAssignedViaExternalGroups: true},
		},
	}
	r := newTeamTestResource(t, state)

	// Empty desired set: reconcile to zero managed members.
	if err := r.applyTeamMembers(context.Background(), testOrgID, testTeamID, map[string]desiredTeamMember{}); err != nil {
		t.Fatalf("applyTeamMembers error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var haveExternal bool
	for _, m := range state.members {
		if m.ID == "user-2" {
			haveExternal = true
		}
	}
	if !haveExternal {
		t.Error("external-group member must never be removed")
	}
}

// TestApplyTeamMembers_SkipsChildTeamIdentity reproduces the real apply failure
// "failed to remove member ...: status 405: Cannot delete a child team using this
// endpoint". Once a team has a child team, the platform lists that child as a
// non-user member. When the user starts managing members (e.g. adds a member block
// to a previously-unmanaged, imported team), reconcile must NOT try to remove the
// child-team identity — it isn't an org user and its DELETE is a 405. The fix
// mirrors the read path: only members that resolve to an org user are managed.
// Without the isOrgUser skip in applyTeamMembers, this test fails with the 405.
func TestApplyTeamMembers_SkipsChildTeamIdentity(t *testing.T) {
	state := &teamTestServerState{
		members: []accessmanagement.TeamMember{
			// A child team the platform injected into this team's member list.
			{ID: childTeamID, MembershipType: "member"},
		},
	}
	r := newTeamTestResource(t, state)

	// User now manages members: desire exactly alice. The child-team identity is not
	// (and cannot be) in the desired set, so a naive reconcile would DELETE it -> 405.
	desired := map[string]desiredTeamMember{
		"user-1": {userID: "user-1", typedUsername: "alice", membershipType: "member"},
	}
	if err := r.applyTeamMembers(context.Background(), testOrgID, testTeamID, desired); err != nil {
		t.Fatalf("applyTeamMembers error (bug: tried to remove the child-team identity): %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var haveChild, haveAlice bool
	for _, m := range state.members {
		switch m.ID {
		case childTeamID:
			haveChild = true
		case "user-1":
			haveAlice = true
		}
	}
	if !haveChild {
		t.Error("child-team identity must never be removed (it is not a manageable org user)")
	}
	if !haveAlice {
		t.Error("desired org user alice should have been added")
	}
}

// --- reconcile*IntoState (typed-casing preservation) -------------------------

func TestReconcileTeamRolesIntoState_PreservesTypedCasing(t *testing.T) {
	state := &teamTestServerState{
		assignments: []accessmanagement.TeamRoleAssignment{
			{RoleID: "role-exchange-viewer", Name: "Exchange Viewer", ContextParams: map[string]string{"org": testOrgID}},
			{RoleID: "role-internal-xyz", Name: "Internal", Internal: true},
		},
	}
	r := newTeamTestResource(t, state)

	typed := teamRoleSet(t, typedTeamRole{name: types.StringValue("exchange viewer"), cp: strMap(map[string]string{"org": testOrgID})})
	got, err := r.reconcileTeamRolesIntoState(context.Background(), testOrgID, testTeamID, typed)
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	if len(got.Elements()) != 1 {
		t.Fatalf("expected 1 role (internal excluded), got %d", len(got.Elements()))
	}
	obj := got.Elements()[0].(types.Object)
	if name := obj.Attributes()["name"].(types.String).ValueString(); name != "exchange viewer" {
		t.Errorf("expected typed casing 'exchange viewer' preserved, got %q", name)
	}
}

func TestReconcileTeamMembersIntoState_PreservesTypedAndExcludes(t *testing.T) {
	state := &teamTestServerState{
		members: []accessmanagement.TeamMember{
			{ID: "user-1", MembershipType: "member"},                                        // org user alice -> kept
			{ID: "user-2", MembershipType: "maintainer", IsAssignedViaExternalGroups: true}, // external -> excluded
			{ID: "user-999", MembershipType: "member"},                                      // not an org user -> excluded
		},
	}
	r := newTeamTestResource(t, state)

	// User typed "ALICE" and omitted membership_type (null): both preserved.
	typed := teamMemberSet(t, typedTeamMember{username: types.StringValue("ALICE"), membershipType: types.StringNull()})
	got, err := r.reconcileTeamMembersIntoState(context.Background(), testOrgID, testTeamID, typed)
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	if len(got.Elements()) != 1 {
		t.Fatalf("expected 1 member (external + non-user excluded), got %d", len(got.Elements()))
	}
	obj := got.Elements()[0].(types.Object)
	if uname := obj.Attributes()["username"].(types.String).ValueString(); uname != "ALICE" {
		t.Errorf("expected typed casing 'ALICE' preserved, got %q", uname)
	}
	if mt := obj.Attributes()["membership_type"].(types.String); !mt.IsNull() {
		t.Errorf("expected typed null membership_type preserved, got %v", mt)
	}
}

// --- EDGE CASES: deleted user / deleted role, null-vs-empty, apply-vs-read ----
//
// These tests document the design decision for "what happens when a referenced
// identity disappears from the platform": at APPLY time (Create/Update ->
// resolveTeam*), an unknown username/role is a hard error that names the offender
// so the operator can edit the .tf. At READ (refresh) time, a member/role that
// vanished server-side is silently dropped from state (drift), which Terraform
// then re-plans as an add on the next apply. The tests below pin both halves.

// TestResolveTeamMembers_DeletedUser_ErrorNamesUser mirrors the real "a user was
// deleted from the org but is still referenced in the .tf" scenario. The apply
// must fail with an actionable message identifying the exact username, so the
// operator knows which line to remove.
func TestResolveTeamMembers_DeletedUser_ErrorNamesUser(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	// "dave" is NOT in the org user list (alice/bob/carol) — simulates a deleted user.
	set := teamMemberSet(t,
		typedTeamMember{username: types.StringValue("alice"), membershipType: types.StringNull()},
		typedTeamMember{username: types.StringValue("dave"), membershipType: types.StringValue("maintainer")},
	)
	_, err := r.resolveTeamMembers(context.Background(), testOrgID, set)
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

// TestResolveTeamRoles_DeletedRole_ErrorNamesRole is the role-side analogue: a
// role that was removed from the platform catalog but is still referenced fails
// the apply with a message naming the role.
func TestResolveTeamRoles_DeletedRole_ErrorNamesRole(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	set := teamRoleSet(t,
		typedTeamRole{name: types.StringValue("Exchange Viewer"), cp: strMap(map[string]string{"org": testOrgID})},
		typedTeamRole{name: types.StringValue("Deleted Role"), cp: types.MapNull(types.StringType)},
	)
	_, err := r.resolveTeamRoles(context.Background(), set)
	if err == nil {
		t.Fatal("expected error when a referenced role no longer exists in the catalog")
	}
	if !strings.Contains(err.Error(), `"Deleted Role"`) {
		t.Errorf("error must name the offending role, got: %v", err)
	}
}

// TestResolveTeamMembers_EmptySetIsAuthoritativeEmpty verifies the null-vs-empty
// boundary that the whole "authoritative-when-set" model rests on. An EMPTY (but
// non-null) set is NOT "unmanaged" — it resolves to zero desired members, which
// applyTeamMembers then treats as "remove all managed members". Only a NULL set
// means "leave membership alone". Getting this wrong would either strand members
// or wipe them unexpectedly.
func TestResolveTeamMembers_EmptySetIsAuthoritativeEmpty(t *testing.T) {
	r := newTeamTestResource(t, &teamTestServerState{})
	empty, diags := types.SetValue(teamMemberObjectType, []attr.Value{})
	if diags.HasError() {
		t.Fatalf("failed to build empty set: %v", diags.Errors())
	}
	got, err := r.resolveTeamMembers(context.Background(), testOrgID, empty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("empty (non-null) set must NOT be treated as unmanaged; expected non-nil empty map")
	}
	if len(got) != 0 {
		t.Errorf("empty set must resolve to zero desired members, got %d", len(got))
	}
}

// TestReconcileTeamMembersIntoState_DeletedUserDropsFromState is the READ-path
// counterpart to the apply-time error: if a member is deleted server-side (its ID
// no longer maps to any org user), refresh silently drops it from state rather
// than erroring. Terraform then shows it as a diff (an add) on the next plan.
func TestReconcileTeamMembersIntoState_DeletedUserDropsFromState(t *testing.T) {
	state := &teamTestServerState{
		members: []accessmanagement.TeamMember{
			{ID: "user-1", MembershipType: "member"},        // alice, still exists -> kept
			{ID: "user-1000", MembershipType: "maintainer"}, // deleted user id -> dropped
		},
	}
	r := newTeamTestResource(t, state)

	typed := teamMemberSet(t,
		typedTeamMember{username: types.StringValue("alice"), membershipType: types.StringNull()},
		typedTeamMember{username: types.StringValue("ghost"), membershipType: types.StringValue("maintainer")},
	)
	got, err := r.reconcileTeamMembersIntoState(context.Background(), testOrgID, testTeamID, typed)
	if err != nil {
		t.Fatalf("reconcile must not error on a deleted member; got: %v", err)
	}
	if len(got.Elements()) != 1 {
		t.Fatalf("expected only the surviving member in state, got %d", len(got.Elements()))
	}
	obj := got.Elements()[0].(types.Object)
	if uname := obj.Attributes()["username"].(types.String).ValueString(); uname != "alice" {
		t.Errorf("expected surviving member 'alice', got %q", uname)
	}
}

// --- REGRESSION: platform-injected side-effect role grant (Business Group Viewer)
//
// Discovered via live E2E against devx: assigning ANY env-scoped role to a team
// makes the platform auto-grant an org-scoped "Business Group Viewer" role. That
// grant is (a) NOT in the assignable catalog (GET /accounts/api/roles) and (b) NOT
// flagged internal. Before the fix, reconcileTeamRolesIntoState surfaced it as a
// phantom `name = ""` entry and applyTeamRoles tried to remove it, so the plan was
// never idempotent after the first apply. Both tests below pin the fix: a
// non-catalog, non-internal assignment must be IGNORED for both state and removal.

// roleIDInjected is a role_id present as a live team assignment but absent from the
// catalog in newTeamTestResource — i.e. a platform-injected side-effect grant.
const roleIDInjected = "role-business-group-viewer-injected"

func TestReconcileTeamRolesIntoState_IgnoresInjectedNonCatalogRole(t *testing.T) {
	state := &teamTestServerState{
		assignments: []accessmanagement.TeamRoleAssignment{
			{RoleID: "role-exchange-viewer", Name: "Exchange Viewer", ContextParams: map[string]string{"org": testOrgID, "envId": "env-1"}},
			// Side-effect grant: org-scoped, not internal, not in the catalog.
			{RoleID: roleIDInjected, Name: "Business Group Viewer", ContextParams: map[string]string{"org": testOrgID}},
		},
	}
	r := newTeamTestResource(t, state)

	typed := teamRoleSet(t, typedTeamRole{name: types.StringValue("Exchange Viewer"), cp: strMap(map[string]string{"org": testOrgID, "envId": "env-1"})})
	got, err := r.reconcileTeamRolesIntoState(context.Background(), testOrgID, testTeamID, typed)
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	if len(got.Elements()) != 1 {
		t.Fatalf("expected only the catalog role in state (injected grant excluded), got %d", len(got.Elements()))
	}
	obj := got.Elements()[0].(types.Object)
	if name := obj.Attributes()["name"].(types.String).ValueString(); name != "Exchange Viewer" {
		t.Errorf("expected 'Exchange Viewer' preserved, got %q (a phantom name=\"\" here means the injected grant leaked into state)", name)
	}
}

func TestApplyTeamRoles_NeverRemovesInjectedNonCatalogRole(t *testing.T) {
	state := &teamTestServerState{
		assignments: []accessmanagement.TeamRoleAssignment{
			{RoleID: "role-exchange-viewer", ContextParams: map[string]string{"org": testOrgID, "envId": "env-1"}},
			// Side-effect grant the platform added; desired set below does NOT include it.
			{RoleID: roleIDInjected, ContextParams: map[string]string{"org": testOrgID}},
		},
	}
	r := newTeamTestResource(t, state)

	cp := map[string]string{"org": testOrgID, "envId": "env-1"}
	desired := []desiredTeamRole{
		{roleID: "role-exchange-viewer", contextParams: cp, contextKey: canonicalContextParams(cp)},
	}
	if err := r.applyTeamRoles(context.Background(), testOrgID, testTeamID, desired); err != nil {
		t.Fatalf("applyTeamRoles error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var haveViewer, haveInjected bool
	for _, a := range state.assignments {
		switch a.RoleID {
		case "role-exchange-viewer":
			haveViewer = true
		case roleIDInjected:
			haveInjected = true
		}
	}
	if !haveViewer {
		t.Error("expected desired role-exchange-viewer to remain")
	}
	if !haveInjected {
		t.Error("platform-injected non-catalog grant must NEVER be removed (would fight the platform every apply)")
	}
}
