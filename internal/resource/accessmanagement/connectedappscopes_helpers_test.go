package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	accessmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// scopeObjectType is the Terraform object type for a single scope entry.
var scopeObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"scope":          types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

func makeScopeSet(t *testing.T, scopes []struct {
	scope  string
	params map[string]attr.Value
}) types.Set {
	t.Helper()
	ctx := context.Background()
	objs := make([]attr.Value, 0, len(scopes))
	for _, s := range scopes {
		var cp types.Map
		if len(s.params) > 0 {
			m, diags := types.MapValue(types.StringType, s.params)
			if diags.HasError() {
				t.Fatalf("failed to create map: %v", diags.Errors())
			}
			cp = m
		} else {
			cp = types.MapNull(types.StringType)
		}
		obj, diags := types.ObjectValue(scopeObjectType.AttrTypes, map[string]attr.Value{
			"scope":          types.StringValue(s.scope),
			"context_params": cp,
		})
		if diags.HasError() {
			t.Fatalf("failed to create object: %v", diags.Errors())
		}
		objs = append(objs, obj)
	}
	set, diags := types.SetValue(scopeObjectType, objs)
	if diags.HasError() {
		t.Fatalf("failed to create set: %v", diags.Errors())
	}
	_ = ctx
	return set
}

// --- convertScopesToAPI ---

func TestConnectedAppScopesResource_convertScopesToAPI(t *testing.T) {
	r := &ConnectedAppScopesResource{}
	ctx := context.Background()

	t.Run("empty set returns empty slice", func(t *testing.T) {
		set, _ := types.SetValue(scopeObjectType, []attr.Value{})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 0 {
			t.Errorf("convertScopesToAPI() len = %d, want 0", len(result))
		}
	})

	t.Run("single scope without context params", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"read:data", nil},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 1 {
			t.Fatalf("convertScopesToAPI() len = %d, want 1", len(result))
		}
		if result[0].Scope != "read:data" {
			t.Errorf("Scope = %q, want read:data", result[0].Scope)
		}
	})

	t.Run("scope with context params is preserved", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"manage:environment", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 1 {
			t.Fatalf("convertScopesToAPI() len = %d, want 1", len(result))
		}
		if result[0].ContextParams["org"] != "org-1" {
			t.Errorf("ContextParams[org] = %v, want org-1", result[0].ContextParams["org"])
		}
	})

	t.Run("multiple scopes are all converted", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"read:data", nil},
			{"write:data", nil},
			{"admin:all", nil},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 3 {
			t.Fatalf("convertScopesToAPI() len = %d, want 3", len(result))
		}
	})
}

// --- convertScopesToAPI with display names ---

func TestConnectedAppScopesResource_convertScopesToAPI_DisplayNames(t *testing.T) {
	r := &ConnectedAppScopesResource{}
	ctx := context.Background()

	t.Run("display name is resolved to identifier", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"Exchange Viewer", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 1 {
			t.Fatalf("convertScopesToAPI() len = %d, want 1", len(result))
		}
		if result[0].Scope != "read:exchange" {
			t.Errorf("Scope = %q, want read:exchange (resolved from display name)", result[0].Scope)
		}
	})

	t.Run("multiple display names are resolved", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"Cloudhub Organization Admin", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Audit Log Viewer", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Manage Runtime Fabrics", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 3 {
			t.Fatalf("convertScopesToAPI() len = %d, want 3", len(result))
		}
		// Check each resolved correctly (order in set is not guaranteed, check by content)
		found := map[string]bool{}
		for _, s := range result {
			found[s.Scope] = true
		}
		expectedScopes := []string{"admin:cloudhub", "read:audit_logs", "manage:runtime_fabrics"}
		for _, expected := range expectedScopes {
			if !found[expected] {
				t.Errorf("Expected resolved scope %q not found in result", expected)
			}
		}
	})

	t.Run("mix of identifiers and display names", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"read:exchange", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Cloudhub Organization Admin", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 2 {
			t.Fatalf("convertScopesToAPI() len = %d, want 2", len(result))
		}
		found := map[string]bool{}
		for _, s := range result {
			found[s.Scope] = true
		}
		if !found["read:exchange"] {
			t.Error("Expected read:exchange in result")
		}
		if !found["admin:cloudhub"] {
			t.Error("Expected admin:cloudhub (resolved from Cloudhub Organization Admin) in result")
		}
	})

	t.Run("identifier passthrough when already valid", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"admin:cloudhub", nil},
		})
		result := r.convertScopesToAPI(ctx, set)
		if len(result) != 1 {
			t.Fatalf("convertScopesToAPI() len = %d, want 1", len(result))
		}
		if result[0].Scope != "admin:cloudhub" {
			t.Errorf("Scope = %q, want admin:cloudhub (passthrough)", result[0].Scope)
		}
	})
}

// --- normalizeScopesToIdentifiers ---

func TestConnectedAppScopesResource_normalizeScopesToIdentifiers(t *testing.T) {
	r := &ConnectedAppScopesResource{}
	ctx := context.Background()

	t.Run("display names are normalized to identifiers", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"Exchange Viewer", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Audit Log Viewer", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})

		normalized := r.normalizeScopesToIdentifiers(ctx, set)
		elements := normalized.Elements()
		if len(elements) != 2 {
			t.Fatalf("normalized len = %d, want 2", len(elements))
		}

		// Extract scope values from the normalized set
		found := map[string]bool{}
		for _, elem := range elements {
			obj := elem.(types.Object)
			scopeVal := obj.Attributes()["scope"].(types.String).ValueString()
			found[scopeVal] = true
		}
		if !found["read:exchange"] {
			t.Error("Expected read:exchange in normalized set")
		}
		if !found["read:audit_logs"] {
			t.Error("Expected read:audit_logs in normalized set")
		}
	})

	t.Run("identifiers pass through unchanged", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"read:exchange", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})

		normalized := r.normalizeScopesToIdentifiers(ctx, set)
		elements := normalized.Elements()
		if len(elements) != 1 {
			t.Fatalf("normalized len = %d, want 1", len(elements))
		}
		obj := elements[0].(types.Object)
		scopeVal := obj.Attributes()["scope"].(types.String).ValueString()
		if scopeVal != "read:exchange" {
			t.Errorf("Scope = %q, want read:exchange", scopeVal)
		}
	})
}

// --- validateScopes with display names ---

func TestConnectedAppScopesResource_validateScopes_DisplayNames(t *testing.T) {
	r := &ConnectedAppScopesResource{}
	ctx := context.Background()

	t.Run("display names are valid", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"Exchange Viewer", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Cloudhub Organization Admin", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Audit Log Viewer", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		diags := r.validateScopes(ctx, set)
		if diags.HasError() {
			t.Errorf("validateScopes() should accept display names, got errors: %v", diags.Errors())
		}
	})

	t.Run("invalid display name is rejected", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"Not A Real Scope Name", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		diags := r.validateScopes(ctx, set)
		if !diags.HasError() {
			t.Error("validateScopes() should reject invalid display names")
		}
	})

	t.Run("mix of identifiers and display names is valid", func(t *testing.T) {
		set := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{
			{"read:exchange", map[string]attr.Value{"org": types.StringValue("org-1")}},
			{"Cloudhub Organization Admin", map[string]attr.Value{"org": types.StringValue("org-1")}},
		})
		diags := r.validateScopes(ctx, set)
		if diags.HasError() {
			t.Errorf("validateScopes() should accept mix of identifiers and display names, got: %v", diags.Errors())
		}
	})
}

// --- updateStateFromAPI ---

func TestConnectedAppScopesResource_updateStateFromAPI(t *testing.T) {
	r := &ConnectedAppScopesResource{}
	ctx := context.Background()

	t.Run("empty scopes from API clears state set", func(t *testing.T) {
		apiScopes := &accessmgmt.ConnectedAppScopes{
			Scopes: []accessmgmt.Scope{},
		}
		data := &ConnectedAppScopesResourceModel{}
		err := r.updateStateFromAPI(ctx, data, apiScopes)
		if err != nil {
			t.Fatalf("updateStateFromAPI() unexpected error: %v", err)
		}
		if !data.Scopes.IsNull() && len(data.Scopes.Elements()) != 0 {
			t.Errorf("Scopes should be empty, got %v", data.Scopes)
		}
	})

	t.Run("single scope without context params is set in state", func(t *testing.T) {
		apiScopes := &accessmgmt.ConnectedAppScopes{
			Scopes: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{}},
			},
		}
		data := &ConnectedAppScopesResourceModel{}
		err := r.updateStateFromAPI(ctx, data, apiScopes)
		if err != nil {
			t.Fatalf("updateStateFromAPI() unexpected error: %v", err)
		}
		if len(data.Scopes.Elements()) != 1 {
			t.Fatalf("Scopes len = %d, want 1", len(data.Scopes.Elements()))
		}
	})

	t.Run("scope with string context params is set", func(t *testing.T) {
		apiScopes := &accessmgmt.ConnectedAppScopes{
			Scopes: []accessmgmt.Scope{
				{
					Scope:         "manage:environment",
					ContextParams: map[string]interface{}{"envId": "env-123"},
				},
			},
		}
		data := &ConnectedAppScopesResourceModel{}
		err := r.updateStateFromAPI(ctx, data, apiScopes)
		if err != nil {
			t.Fatalf("updateStateFromAPI() unexpected error: %v", err)
		}
		if len(data.Scopes.Elements()) != 1 {
			t.Fatalf("Scopes len = %d, want 1", len(data.Scopes.Elements()))
		}
	})

	t.Run("multiple scopes are all set in state", func(t *testing.T) {
		apiScopes := &accessmgmt.ConnectedAppScopes{
			Scopes: []accessmgmt.Scope{
				{Scope: "read:data"},
				{Scope: "write:data"},
				{Scope: "admin:all"},
			},
		}
		data := &ConnectedAppScopesResourceModel{}
		err := r.updateStateFromAPI(ctx, data, apiScopes)
		if err != nil {
			t.Fatalf("updateStateFromAPI() unexpected error: %v", err)
		}
		if len(data.Scopes.Elements()) != 3 {
			t.Fatalf("Scopes len = %d, want 3", len(data.Scopes.Elements()))
		}
	})
}
