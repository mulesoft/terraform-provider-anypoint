package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// These are REPRODUCE-FIRST regression tests for the two role bug classes the user
// hit during manual testing of sibling resources (team) and predicted would recur
// for role. Each fails with the pre-fix code and passes with the fix, so a future
// regression (dropping either filter/modifier) is caught before manual testing.
//
// The shared harness (newRoleTestResource) builds a role catalog that intentionally
// does NOT contain "Business Group Viewer" — so an assignment whose role_id is
// "role-business-group-viewer" models the platform-injected, org-scoped side-effect
// grant the platform auto-adds whenever an env-scoped role is granted. The user can
// never express it in config (resolvePermissions only accepts catalog names), so it
// must be treated identically to an internal assignment across EVERY path.

const phantomRoleID = "role-business-group-viewer" // NOT in the test catalog

// --- Class A: apply path (Create + Update write) -----------------------------

// TestApplyPermissions_SkipsNonCatalogPhantom proves the reconcile WRITE path never
// tries to unassign a platform-injected side-effect grant.
//
// Reproduce-first: with the pre-fix applyPermissions (which built `actual` skipping
// only Internal), the phantom — being non-internal and absent from `desired` —
// lands in the remove loop and is unassigned. That both fights the platform on every
// apply and, on the live API, errors (the grant can't be removed directly). With the
// catalog-skip in place, the phantom is excluded from `actual`, so it is never
// touched and survives the apply.
func TestApplyPermissions_SkipsNonCatalogPhantom(t *testing.T) {
	state := &roleTestServerState{
		assignments: []accessmanagement.RoleAssignment{
			// Desired assignment the user manages.
			{RoleID: "role-exchange-viewer", ContextParams: map[string]string{"org": testOrgID}},
			// Platform-injected side-effect grant: non-internal, but NOT in the catalog.
			{RoleID: phantomRoleID, Internal: false, ContextParams: map[string]string{"org": testOrgID}},
		},
	}
	r := newRoleTestResource(t, state)

	// User's config lists ONLY the exchange-viewer permission.
	desired := []desiredPermission{
		{roleID: "role-exchange-viewer", contextParams: map[string]string{"org": testOrgID}, contextKey: canonicalContextParams(map[string]string{"org": testOrgID})},
	}
	if err := r.applyPermissions(context.Background(), testOrgID, testRGID, desired); err != nil {
		t.Fatalf("applyPermissions error: %v", err)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var haveViewer, havePhantom bool
	for _, a := range state.assignments {
		switch a.RoleID {
		case "role-exchange-viewer":
			haveViewer = true
		case phantomRoleID:
			havePhantom = true
		}
	}
	if !haveViewer {
		t.Error("expected desired role-exchange-viewer to remain")
	}
	if !havePhantom {
		t.Error("platform-injected non-catalog phantom was unassigned — it must be skipped like an internal assignment (Class A regression)")
	}
}

// --- Class A: read path (Read + Import refresh) ------------------------------

// TestReconcilePermissionsIntoState_SkipsNonCatalogPhantom proves the READ path never
// surfaces the phantom as a managed permission with an empty name.
//
// Reproduce-first: with the pre-fix reconcile (skipping only Internal), the phantom
// assignment is emitted with name = roleIDToName[phantomRoleID] = "" (not in the
// catalog map), producing a phantom `{name = ""}` entry and a perpetual diff. With
// the catalog-skip, only the real, in-catalog permission survives.
func TestReconcilePermissionsIntoState_SkipsNonCatalogPhantom(t *testing.T) {
	state := &roleTestServerState{
		assignments: []accessmanagement.RoleAssignment{
			{RoleID: "role-exchange-viewer", Name: "Exchange Viewer", ContextParams: map[string]string{"org": testOrgID}},
			{RoleID: phantomRoleID, Name: "Business Group Viewer", Internal: false, ContextParams: map[string]string{"org": testOrgID}},
			{RoleID: "role-internal-xyz", Name: "Internal", Internal: true},
		},
	}
	r := newRoleTestResource(t, state)

	typed := permSet(t, typedPerm{name: types.StringValue("Exchange Viewer"), cp: strMap(map[string]string{"org": testOrgID})})
	got, err := r.reconcilePermissionsIntoState(context.Background(), testOrgID, testRGID, typed)
	if err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	if n := len(got.Elements()); n != 1 {
		t.Fatalf("expected exactly 1 permission (internal + non-catalog phantom excluded), got %d", n)
	}
	for _, el := range got.Elements() {
		obj := el.(types.Object)
		name := obj.Attributes()["name"].(types.String).ValueString()
		if name == "" {
			t.Error("a permission surfaced with an empty name — the Business Group Viewer phantom leaked through the read filter (Class A regression)")
		}
		if name != "Exchange Viewer" {
			t.Errorf("expected surviving permission 'Exchange Viewer', got %q", name)
		}
	}
}

// --- Class B: description must not be wiped on a name-only change -------------

// TestRoleSchema_DescriptionHasUseStateForUnknown proves the description attribute
// carries a plan modifier so an omitted (Optional+Computed) description is never
// re-marked "(known after apply)" and wiped to "" during an unrelated update.
//
// Reproduce-first: removing the UseStateForUnknown modifier (the pre-fix state)
// leaves PlanModifiers empty and this test fails; with the modifier it passes. The
// live-lifecycle self-test additionally proves the end-to-end symptom (omit
// description, change name, confirm the server-side description survives).
func TestRoleSchema_DescriptionHasUseStateForUnknown(t *testing.T) {
	res := NewRoleResource().(*RoleResource)
	schemaResp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() errors: %v", schemaResp.Diagnostics.Errors())
	}

	attr, ok := schemaResp.Schema.Attributes["description"]
	if !ok {
		t.Fatal("schema missing 'description' attribute")
	}
	sa, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("'description' is not a StringAttribute, got %T", attr)
	}
	// Sanity: description is the Optional+Computed shape that is vulnerable to the wipe.
	if !sa.Optional || !sa.Computed {
		t.Errorf("expected description Optional+Computed, got Optional=%v Computed=%v", sa.Optional, sa.Computed)
	}
	if len(sa.PlanModifiers) == 0 {
		t.Error("description has no plan modifier — an omitted description will be re-marked unknown and WIPED on a name-only update; it needs UseStateForUnknown (Class B regression)")
	}
}
