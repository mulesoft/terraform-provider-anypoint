package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/constants"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppScopesResource(t *testing.T) {
	r := NewConnectedAppScopesResource()

	if r == nil {
		t.Error("NewConnectedAppScopesResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("ConnectedAppScopesResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("ConnectedAppScopesResource should implement ResourceWithImportState")
	}
}

func TestConnectedAppScopesResource_Metadata(t *testing.T) {
	r := NewConnectedAppScopesResource()
	testutil.TestResourceMetadata(t, r, "_connected_app_scopes")
}

func TestConnectedAppScopesResource_Schema(t *testing.T) {
	res := NewConnectedAppScopesResource()

	requiredAttrs := []string{"connected_app_id", "scopes"}
	optionalAttrs := []string{}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestConnectedAppScopesResource_Configure(t *testing.T) {
	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.ClientConfig{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestConnectedAppScopesResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)

	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data",
	}
	resp := &resource.ConfigureResponse{}

	res.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}

	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestConnectedAppScopesResource_ImportState(t *testing.T) {
	r := NewConnectedAppScopesResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestConnectedAppScopesResourceModel_Validation(t *testing.T) {
	model := ConnectedAppScopesResourceModel{}
	_ = model.ID
}

func BenchmarkConnectedAppScopesResource_Schema(b *testing.B) {
	res := NewConnectedAppScopesResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}

func TestConnectedAppScopesResource_validateScopes_ValidScopes(t *testing.T) {
	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)
	ctx := context.Background()

	validScopes := []attr.Value{
		createScopeObject(t, constants.ScopeAdminCloudHub, map[string]string{"org": "test-org-id"}),
		createScopeObject(t, constants.ScopeManageRuntimeFabrics, map[string]string{"org": "test-org-id"}),
		createScopeObject(t, constants.ScopeCreateEnvironment, map[string]string{"org": "test-org-id", "env": "test-env-id"}),
		createScopeObject(t, constants.ScopeManagePrivateSpaces, map[string]string{"org": "test-org-id"}),
	}

	scopesSet, diags := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"scope":          types.StringType,
			"context_params": types.MapType{ElemType: types.StringType},
		},
	}, validScopes)

	if diags.HasError() {
		t.Fatalf("Failed to create scopes set: %v", diags)
	}

	validationDiags := res.validateScopes(ctx, scopesSet)

	if validationDiags.HasError() {
		t.Errorf("validateScopes() should not have errors for valid scopes, got: %v", validationDiags.Errors())
	}
}

func TestConnectedAppScopesResource_validateScopes_InvalidScopes(t *testing.T) {
	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)
	ctx := context.Background()

	tests := []struct {
		name          string
		invalidScopes []string
		wantErrors    int
	}{
		{
			name:          "Single invalid scope",
			invalidScopes: []string{"invalid:scope"},
			wantErrors:    1,
		},
		{
			name:          "Multiple invalid scopes",
			invalidScopes: []string{"invalid:scope1", "bad:scope2", "wrong:scope3"},
			wantErrors:    3,
		},
		{
			name:          "Typo in scope name",
			invalidScopes: []string{"admin:cloudhb"},
			wantErrors:    1,
		},
		{
			name:          "Wrong format",
			invalidScopes: []string{"admin-cloudhub"},
			wantErrors:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopeValues := make([]attr.Value, len(tt.invalidScopes))
			for i, scope := range tt.invalidScopes {
				scopeValues[i] = createScopeObject(t, scope, map[string]string{"org": "test-org-id"})
			}

			scopesSet, diags := types.SetValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"scope":          types.StringType,
					"context_params": types.MapType{ElemType: types.StringType},
				},
			}, scopeValues)

			if diags.HasError() {
				t.Fatalf("Failed to create scopes set: %v", diags)
			}

			validationDiags := res.validateScopes(ctx, scopesSet)

			if !validationDiags.HasError() {
				t.Error("validateScopes() should have errors for invalid scopes")
			}

			if len(validationDiags.Errors()) != tt.wantErrors {
				t.Errorf("validateScopes() error count = %d, want %d", len(validationDiags.Errors()), tt.wantErrors)
			}

			for i, err := range validationDiags.Errors() {
				if i < len(tt.invalidScopes) {
					expectedScope := tt.invalidScopes[i]
					errMsg := err.Summary() + " " + err.Detail()
					if !contains(errMsg, expectedScope) {
						t.Errorf("Error message should contain scope '%s', got: %s", expectedScope, errMsg)
					}
				}
			}
		})
	}
}

func TestConnectedAppScopesResource_validateScopes_MixedScopes(t *testing.T) {
	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)
	ctx := context.Background()

	mixedScopes := []attr.Value{
		createScopeObject(t, constants.ScopeAdminCloudHub, map[string]string{"org": "test-org-id"}),
		createScopeObject(t, "invalid:scope", map[string]string{"org": "test-org-id"}),
		createScopeObject(t, constants.ScopeManageRuntimeFabrics, map[string]string{"org": "test-org-id"}),
		createScopeObject(t, "another:invalid", map[string]string{"org": "test-org-id"}),
	}

	scopesSet, diags := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"scope":          types.StringType,
			"context_params": types.MapType{ElemType: types.StringType},
		},
	}, mixedScopes)

	if diags.HasError() {
		t.Fatalf("Failed to create scopes set: %v", diags)
	}

	validationDiags := res.validateScopes(ctx, scopesSet)

	if !validationDiags.HasError() {
		t.Error("validateScopes() should have errors for mixed scopes")
	}

	if len(validationDiags.Errors()) != 2 {
		t.Errorf("validateScopes() error count = %d, want 2", len(validationDiags.Errors()))
	}
}

func TestConnectedAppScopesResource_validateScopes_AllKnownScopes(t *testing.T) {
	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)
	ctx := context.Background()

	allScopes := []string{
		constants.ScopeAdminCloudHub,
		constants.ScopeManageRuntimeFabrics,
		constants.ScopeCreateEnvironment,
		constants.ScopeManagePrivateSpaces,
		constants.ScopeAdminAPIManager,
		constants.ScopeReadAPIQuery,
		constants.ScopeEditAPIQuery,
		constants.ScopeManageAPIQuery,
		constants.ScopePromoteAPIQuery,
		constants.ScopeAdminAPIQuery,
	}

	scopeValues := make([]attr.Value, len(allScopes))
	for i, scope := range allScopes {
		scopeValues[i] = createScopeObject(t, scope, map[string]string{"org": "test-org-id"})
	}

	scopesSet, diags := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"scope":          types.StringType,
			"context_params": types.MapType{ElemType: types.StringType},
		},
	}, scopeValues)

	if diags.HasError() {
		t.Fatalf("Failed to create scopes set: %v", diags)
	}

	validationDiags := res.validateScopes(ctx, scopesSet)

	if validationDiags.HasError() {
		t.Errorf("validateScopes() should not have errors for known scopes, got: %v", validationDiags.Errors())
	}
}

func createScopeObject(t *testing.T, scopeName string, contextParams map[string]string) attr.Value {
	contextParamsMap := make(map[string]attr.Value)
	for k, v := range contextParams {
		contextParamsMap[k] = types.StringValue(v)
	}

	contextParamsValue, diags := types.MapValue(types.StringType, contextParamsMap)
	if diags.HasError() {
		t.Fatalf("Failed to create context params map: %v", diags)
	}

	scopeAttrs := map[string]attr.Value{
		"scope":          types.StringValue(scopeName),
		"context_params": contextParamsValue,
	}

	scopeObject, diags := types.ObjectValue(map[string]attr.Type{
		"scope":          types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	}, scopeAttrs)

	if diags.HasError() {
		t.Fatalf("Failed to create scope object: %v", diags)
	}

	return scopeObject
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
