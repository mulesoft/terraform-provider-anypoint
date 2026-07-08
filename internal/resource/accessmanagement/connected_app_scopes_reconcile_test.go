package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	accessmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// --- validateAndResolveScopes ---

func TestValidateAndResolveScopes_NullIsUnmanaged(t *testing.T) {
	got, diags := validateAndResolveScopes(types.SetNull(connectedAppScopeObjectType))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if got != nil {
		t.Fatalf("expected nil for null set, got %v", got)
	}
}

func TestValidateAndResolveScopes_ResolvesIdentifiersAndDisplayNames(t *testing.T) {
	set := makeScopeSet(t, []struct {
		scope  string
		params map[string]attr.Value
	}{
		{scope: "read:applications", params: map[string]attr.Value{"org": types.StringValue("o1")}},
		{scope: "Cloudhub Organization Admin", params: map[string]attr.Value{"org": types.StringValue("o1")}}, // display name
	})

	got, diags := validateAndResolveScopes(set)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if len(got) != 2 {
		t.Fatalf("got %d scopes, want 2", len(got))
	}
	byScope := map[string]bool{}
	for _, s := range got {
		byScope[s.Scope] = true
	}
	if !byScope["read:applications"] {
		t.Errorf("identifier read:applications missing: %+v", got)
	}
	if !byScope["admin:cloudhub"] {
		t.Errorf("display name 'Cloudhub Organization Admin' not resolved to admin:cloudhub: %+v", got)
	}
}

func TestValidateAndResolveScopes_RejectsInvalid(t *testing.T) {
	set := makeScopeSet(t, []struct {
		scope  string
		params map[string]attr.Value
	}{
		{scope: "not-a-real-scope", params: nil},
	})
	_, diags := validateAndResolveScopes(set)
	if !diags.HasError() {
		t.Fatalf("expected error for invalid scope, got none")
	}
}

// TestValidateAndResolveScopes_RejectsSystemProfile ensures the platform-injected, undeletable
// "profile" scope cannot be listed in config — it is not user-manageable and would create a
// perpetual diff (the read path always drops it).
func TestValidateAndResolveScopes_RejectsSystemProfile(t *testing.T) {
	set := makeScopeSet(t, []struct {
		scope  string
		params map[string]attr.Value
	}{
		{scope: "profile", params: nil},
	})
	_, diags := validateAndResolveScopes(set)
	if !diags.HasError() {
		t.Fatalf("expected error for system scope 'profile', got none")
	}
}

// --- reconcileScopesIntoState ---

// newTestScopesResource wires a ConnectedAppResource with a scopes client pointed at `server`.
func newTestScopesResource(t *testing.T, server string) *ConnectedAppResource {
	t.Helper()
	sc := &accessmgmt.ConnectedAppScopesClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	return &ConnectedAppResource{scopesClient: sc}
}

// TestReconcileScopesIntoState_SkipsProfileAndPreservesTyped verifies the two behaviors that keep
// the inline scopes attribute idempotent:
//  1. the platform-injected "profile" scope is dropped from state (never surfaced), and
//  2. a matched scope keeps the user's typed representation (display name) instead of the resolved
//     identifier the API returns.
func TestReconcileScopesIntoState_SkipsProfileAndPreservesTyped(t *testing.T) {
	// API returns: profile (system), admin:cloudhub (matches typed "Cloudhub Organization Admin"),
	// and read:applications (not in typed source -> emitted as identifier).
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/connectedApplications/app-1/scopes": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, accessmgmt.ConnectedAppScopes{
				Scopes: []accessmgmt.Scope{
					{Scope: "profile", ContextParams: map[string]interface{}{}},
					{Scope: "admin:cloudhub", ContextParams: map[string]interface{}{"org": "o1"}},
					{Scope: "read:applications", ContextParams: map[string]interface{}{"org": "o1"}},
				},
				Total: 3,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	r := newTestScopesResource(t, server.URL)

	// Typed source: user wrote the display name "Cloudhub Organization Admin".
	typed := makeScopeSet(t, []struct {
		scope  string
		params map[string]attr.Value
	}{
		{scope: "Cloudhub Organization Admin", params: map[string]attr.Value{"org": types.StringValue("o1")}},
	})

	got, diags := r.reconcileScopesIntoState(context.Background(), "app-1", typed)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}

	// profile must be dropped -> 2 entries remain.
	if len(got.Elements()) != 2 {
		t.Fatalf("expected 2 scopes in state (profile dropped), got %d", len(got.Elements()))
	}

	names := map[string]bool{}
	for _, el := range got.Elements() {
		names[el.(types.Object).Attributes()["scope"].(types.String).ValueString()] = true
	}
	if names["profile"] {
		t.Errorf("system scope 'profile' must not appear in state")
	}
	if !names["Cloudhub Organization Admin"] {
		t.Errorf("typed display name 'Cloudhub Organization Admin' should be preserved, got %v", names)
	}
	if !names["read:applications"] {
		t.Errorf("unmatched API scope should be emitted as identifier, got %v", names)
	}
}

// TestReconcileScopesIntoState_OnlyProfileYieldsEmptySet verifies that a fresh app (only the
// injected profile scope) reconciles to an EMPTY set — not null, not a phantom entry — so a plan
// with scopes=[] is idempotent.
func TestReconcileScopesIntoState_OnlyProfileYieldsEmptySet(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/connectedApplications/app-2/scopes": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, accessmgmt.ConnectedAppScopes{
				Scopes: []accessmgmt.Scope{{Scope: "profile", ContextParams: map[string]interface{}{}}},
				Total:  1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	r := newTestScopesResource(t, server.URL)

	// Managed with an empty set.
	empty, _ := types.SetValue(connectedAppScopeObjectType, []attr.Value{})
	got, diags := r.reconcileScopesIntoState(context.Background(), "app-2", empty)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if got.IsNull() {
		t.Fatalf("expected empty (non-null) set")
	}
	if len(got.Elements()) != 0 {
		t.Fatalf("expected 0 scopes (only profile present, which is skipped), got %d", len(got.Elements()))
	}
}
