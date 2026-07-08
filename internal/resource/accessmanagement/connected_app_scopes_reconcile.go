package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/constants"
)

// This file implements the inline, authoritative-when-set `scopes` attribute on
// anypoint_connected_app. It mirrors the team roles/members reconcile pattern:
//   - resolve the user's typed scopes (display name OR identifier) to canonical identifiers,
//   - apply them authoritatively (PUT replaces the whole list; empty => DELETE all non-system),
//   - reconcile the live list back into state while (a) skipping platform-injected system scopes
//     (e.g. the undeletable "profile") and (b) preserving the user's typed representation to
//     avoid perpetual diffs.

// typedScope holds the user's original (typed) representation of a scope so Read can preserve the
// exact form (display name vs identifier, and the context_params map) and avoid perpetual diffs.
type typedScope struct {
	name types.String
	cp   types.Map
}

// validateAndResolveScopes validates every scope name and returns the resolved API scopes.
// Accepts identifiers ("read:applications") and display names ("Cloudhub Organization Admin"). Rejects
// system scopes (e.g. "profile") — they are platform-managed and undeletable, so listing them
// would create a perpetual diff. Returns (nil, nil) when the set is null/unknown (unmanaged).
func validateAndResolveScopes(scopeSet types.Set) ([]accessmanagement.Scope, diag.Diagnostics) {
	var diags diag.Diagnostics
	if scopeSet.IsNull() || scopeSet.IsUnknown() {
		return nil, diags
	}

	out := make([]accessmanagement.Scope, 0, len(scopeSet.Elements()))
	for i, el := range scopeSet.Elements() {
		attrs := el.(types.Object).Attributes()
		typed := attrs["scope"].(types.String).ValueString()

		if systemConnectedAppScopes[typed] {
			diags.AddError(
				"System Scope Not Manageable",
				fmt.Sprintf("The scope %q at index %d is automatically assigned by the platform and cannot be "+
					"managed by Terraform. Remove it from the scopes list.", typed, i),
			)
			continue
		}

		resolved, ok := constants.ResolveScopeIdentifier(typed)
		if !ok {
			diags.AddError(
				"Invalid Scope Name",
				fmt.Sprintf("The scope %q at index %d is not a valid Anypoint Platform scope. Use either the scope "+
					"identifier (e.g. 'read:applications') or the display name (e.g. 'Cloudhub Organization Admin'). Use the "+
					"anypoint_scopes_catalog data source to discover valid scopes.", typed, i),
			)
			continue
		}
		if systemConnectedAppScopes[resolved] {
			diags.AddError(
				"System Scope Not Manageable",
				fmt.Sprintf("The scope %q (resolves to %q) at index %d is platform-managed and cannot be set.", typed, resolved, i),
			)
			continue
		}

		cp := map[string]interface{}{}
		for k, v := range mapToStringMap(attrs["context_params"].(types.Map)) {
			cp[k] = v
		}
		out = append(out, accessmanagement.Scope{Scope: resolved, ContextParams: cp})
	}
	return out, diags
}

// applyScopes reconciles the connected app's scopes to exactly `scopeSet` (authoritative).
//   - non-empty desired => PUT (replaces the whole list; the platform keeps its injected "profile").
//   - empty desired ([]) => DELETE every current non-system scope (PUT cannot send an empty list).
func (r *ConnectedAppResource) applyScopes(ctx context.Context, clientID string, scopeSet types.Set) diag.Diagnostics {
	desired, diags := validateAndResolveScopes(scopeSet)
	if diags.HasError() {
		return diags
	}

	if len(desired) > 0 {
		if _, err := r.scopesClient.ReplaceConnectedAppScopes(ctx, clientID, desired); err != nil {
			diags.AddError("Error setting connected app scopes", "Could not replace connected app scopes: "+err.Error())
		}
		return diags
	}

	// Authoritative-empty: remove every currently-assigned non-system scope.
	current, err := r.scopesClient.GetConnectedAppScopes(ctx, clientID)
	if err != nil {
		diags.AddError("Error reading connected app scopes", "Could not read current scopes before clearing: "+err.Error())
		return diags
	}
	toRemove := make([]accessmanagement.Scope, 0, len(current.Scopes))
	for _, s := range current.Scopes {
		if systemConnectedAppScopes[s.Scope] {
			continue
		}
		toRemove = append(toRemove, s)
	}
	if len(toRemove) > 0 {
		if err := r.scopesClient.RemoveConnectedAppScopes(ctx, clientID, toRemove); err != nil {
			diags.AddError("Error clearing connected app scopes", "Could not remove connected app scopes: "+err.Error())
		}
	}
	return diags
}

// reconcileScopesIntoState reads the live scopes and returns a set suitable for state, skipping
// platform-injected system scopes and preserving the user's typed representation for matched
// entries. `typedSource` is the prior/config scopes set (may be null => unmanaged).
func (r *ConnectedAppResource) reconcileScopesIntoState(ctx context.Context, clientID string, typedSource types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	scopes, err := r.scopesClient.GetConnectedAppScopes(ctx, clientID)
	if err != nil {
		diags.AddError("Error reading connected app scopes", "Could not list connected app scopes: "+err.Error())
		return types.SetNull(connectedAppScopeObjectType), diags
	}

	// Index the user's typed scopes by (resolved_id | context) so we can preserve their exact
	// representation (display name vs id, context_params form) for matched assignments.
	typedByKey := map[string]typedScope{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			attrs := el.(types.Object).Attributes()
			nameVal := attrs["scope"].(types.String)
			cpVal := attrs["context_params"].(types.Map)
			resolved, _ := constants.ResolveScopeIdentifier(nameVal.ValueString())
			key := resolved + "|" + canonicalContextParams(mapToStringMap(cpVal))
			typedByKey[key] = typedScope{name: nameVal, cp: cpVal}
		}
	}

	objs := make([]attr.Value, 0, len(scopes.Scopes))
	for _, s := range scopes.Scopes {
		// Skip platform-injected, undeletable system scopes (e.g. "profile"). The user cannot
		// express these in config, so treating them as managed would surface a phantom entry and a
		// perpetual diff — same class as the team "Business Group Viewer" side-effect.
		if systemConnectedAppScopes[s.Scope] {
			continue
		}

		key := s.Scope + "|" + canonicalContextParamsIface(s.ContextParams)

		var nameVal types.String
		var cpVal types.Map
		if ts, ok := typedByKey[key]; ok {
			nameVal = ts.name
			cpVal = ts.cp
		} else {
			nameVal = types.StringValue(s.Scope)
			cpVal = stringMapToTypesMap(ifaceMapToStringMap(s.ContextParams))
		}

		obj, d := types.ObjectValue(connectedAppScopeObjectType.AttrTypes, map[string]attr.Value{
			"scope":          nameVal,
			"context_params": cpVal,
		})
		if d.HasError() {
			diags.Append(d...)
			return types.SetNull(connectedAppScopeObjectType), diags
		}
		objs = append(objs, obj)
	}

	set, d := types.SetValue(connectedAppScopeObjectType, objs)
	if d.HasError() {
		diags.Append(d...)
		return types.SetNull(connectedAppScopeObjectType), diags
	}
	return set, diags
}

// canonicalContextParamsIface is canonicalContextParams for an API scope's map[string]interface{}.
func canonicalContextParamsIface(m map[string]interface{}) string {
	return canonicalContextParams(ifaceMapToStringMap(m))
}

// ifaceMapToStringMap flattens the API's map[string]interface{} context_params to map[string]string.
func ifaceMapToStringMap(m map[string]interface{}) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		} else {
			out[k] = fmt.Sprintf("%v", v)
		}
	}
	return out
}
