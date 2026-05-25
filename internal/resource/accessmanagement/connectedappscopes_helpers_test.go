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
