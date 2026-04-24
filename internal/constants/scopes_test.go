package constants

import (
	"testing"
)

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected bool
	}{
		{
			name:     "Valid scope - admin:cloudhub",
			scope:    ScopeAdminCloudHub,
			expected: true,
		},
		{
			name:     "Valid scope - manage:runtime_fabrics",
			scope:    ScopeManageRuntimeFabrics,
			expected: true,
		},
		{
			name:     "Valid scope - create:environment",
			scope:    ScopeCreateEnvironment,
			expected: true,
		},
		{
			name:     "Valid scope - read:api_query",
			scope:    ScopeReadAPIQuery,
			expected: true,
		},
		{
			name:     "Valid scope - view:monitoring",
			scope:    ScopeViewMonitoring,
			expected: true,
		},
		{
			name:     "Invalid scope - nonexistent:scope",
			scope:    "nonexistent:scope",
			expected: false,
		},
		{
			name:     "Invalid scope - empty string",
			scope:    "",
			expected: false,
		},
		{
			name:     "Invalid scope - malformed",
			scope:    "admin-cloudhub",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidScope(tt.scope)
			if result != tt.expected {
				t.Errorf("IsValidScope(%q) = %v, expected %v", tt.scope, result, tt.expected)
			}
		})
	}
}

func TestGetAllScopes(t *testing.T) {
	scopes := GetAllScopes()

	// Check that we have scopes
	if len(scopes) == 0 {
		t.Error("GetAllScopes() returned empty slice")
	}

	// Check that the count matches ValidScopes map
	if len(scopes) != len(ValidScopes) {
		t.Errorf("GetAllScopes() returned %d scopes, expected %d", len(scopes), len(ValidScopes))
	}

	// Check that all returned scopes are valid
	for _, scope := range scopes {
		if !IsValidScope(scope) {
			t.Errorf("GetAllScopes() returned invalid scope: %q", scope)
		}
	}
}

func TestValidScopesMapConsistency(t *testing.T) {
	// List all constants that should be in ValidScopes
	expectedScopes := []string{
		// Admin Scopes
		ScopeAdminAccessControls,
		ScopeAdminAngGovernanceProfiles,
		ScopeAdminAPIManager,
		ScopeAdminAPIQuery,
		ScopeAdminCloudHub,
		ScopeAdminDataExporterConfigurations,
		ScopeAdminDataExporterConnections,
		ScopeAdminOrgClientProviderClients,
		ScopeAdminOrgClientProviders,
		ScopeAdminOrgClients,
		ScopeAdminPartnerManager,

		// Administer Scopes
		ScopeAdministerDestinations,

		// AEH Admin
		ScopeAEHAdmin,

		// Clear Scopes
		ScopeClearDestinations,

		// Create Scopes
		ScopeCreateApplications,
		ScopeCreateClientApplications,
		ScopeCreateDesignCenter,
		ScopeCreateEnvironment,
		ScopeCreateExchange,
		ScopeCreateExchangeGenAI,
		ScopeCreateGenerations,
		ScopeCreateOrgClients,
		ScopeCreateSubOrgs,

		// Delete Scopes
		ScopeDeleteApplications,

		// Download Scopes
		ScopeDownloadApplications,

		// Edit Scopes
		ScopeEditAPICatalog,
		ScopeEditAPIQuery,
		ScopeEditDesignCenter,
		ScopeEditEnvironment,
		ScopeEditFlowDesigner,
		ScopeEditIdentityProviders,
		ScopeEditMonitoring,
		ScopeEditOrganization,
		ScopeEditOrgInvites,
		ScopeEditOrgUsers,
		ScopeEditRPA,
		ScopeEditVisualizer,

		// Execute Scopes
		ScopeExecuteDocumentActions,

		// Manage Scopes
		ScopeManageActivity,
		ScopeManageAPIAlerts,
		ScopeManageAPIConfiguration,
		ScopeManageAPIContracts,
		ScopeManageAPIGroups,
		ScopeManageAPIPolicies,
		ScopeManageAPIProxies,
		ScopeManageAPIQuery,
		ScopeManageAPIs,
		ScopeManageApplicationAlerts,
		ScopeManageApplicationData,
		ScopeManageApplicationFlows,
		ScopeManageApplicationQueues,
		ScopeManageApplicationSchedules,
		ScopeManageApplicationSettings,
		ScopeManageApplicationTenants,
		ScopeManageClients,
		ScopeManageCloudHubNetworking,
		ScopeManageDataGateway,
		ScopeManageEnvClientProviders,
		ScopeManageExchange,
		ScopeManageHost,
		ScopeManageIdentityProviders,
		ScopeManagePartners,
		ScopeManagePrivateSpaces,
		ScopeManageRuntimeFabrics,
		ScopeManageSecretGroups,
		ScopeManageSecrets,
		ScopeManageServers,
		ScopeManageStore,
		ScopeManageStoreClients,
		ScopeManageStoreData,

		// Promote Scopes
		ScopePromoteAPIQuery,

		// Publish Scopes
		ScopePublishDestinations,

		// Read Scopes
		ScopeReadActivity,
		ScopeReadAPIConfiguration,
		ScopeReadAPIContracts,
		ScopeReadAPIPolicies,
		ScopeReadAPIQuery,
		ScopeReadApplicationAlerts,
		ScopeReadApplications,
		ScopeReadAuditLogs,
		ScopeReadClientApplications,
		ScopeReadCloudHubNetworking,
		ScopeReadDataGateway,
		ScopeReadExchange,
		ScopeReadHostPartners,
		ScopeReadOrgClientProviderClients,
		ScopeReadOrgClientProviders,
		ScopeReadOrgClients,
		ScopeReadOrgConnApps,
		ScopeReadOrgEnvironments,
		ScopeReadOrgInvites,
		ScopeReadOrganization,
		ScopeReadOrgUsers,
		ScopeReadRuntimeFabrics,
		ScopeReadSecrets,
		ScopeReadSecretsMetadata,
		ScopeReadServers,
		ScopeReadStats,
		ScopeReadStore,
		ScopeReadStoreClients,
		ScopeReadStoreMetrics,

		// Restart Scopes
		ScopeRestartApplications,

		// Subscribe Scopes
		ScopeSubscribeDestinations,

		// View Scopes
		ScopeViewAccessControls,
		ScopeViewAngGovernanceProfiles,
		ScopeViewClients,
		ScopeViewDesignCenter,
		ScopeViewDestinations,
		ScopeViewEnvClientProviders,
		ScopeViewEnvironment,
		ScopeViewIdentityProviders,
		ScopeViewMetering,
		ScopeViewMonitoring,

		// Write Scopes
		ScopeWriteAuditLogSettings,
	}

	// Check that all expected scopes are in ValidScopes
	for _, scope := range expectedScopes {
		if !ValidScopes[scope] {
			t.Errorf("Scope constant %q is not in ValidScopes map", scope)
		}
	}

	// Check that ValidScopes doesn't have extra entries
	if len(ValidScopes) != len(expectedScopes) {
		t.Errorf("ValidScopes has %d entries, expected %d", len(ValidScopes), len(expectedScopes))
	}
}

func TestScopeConstants(t *testing.T) {
	// Test specific scope constant values
	tests := []struct {
		constant string
		expected string
	}{
		{ScopeAdminCloudHub, "admin:cloudhub"},
		{ScopeManageRuntimeFabrics, "manage:runtime_fabrics"},
		{ScopeCreateEnvironment, "create:environment"},
		{ScopeReadAPIQuery, "read:api_query"},
		{ScopeEditAPIQuery, "edit:api_query"},
		{ScopeManageAPIQuery, "manage:api_query"},
		{ScopePromoteAPIQuery, "promote:api_query"},
		{ScopeManageAPIs, "manage:apis"},
		{ScopeReadApplications, "read:applications"},
		{ScopeCreateApplications, "create:applications"},
		{ScopeDeleteApplications, "delete:applications"},
		{ScopeManageSecrets, "manage:secrets"},
		{ScopeReadSecrets, "read:secrets"},
		{ScopeEditEnvironment, "edit:environment"},
		{ScopeViewEnvironment, "view:environment"},
		{ScopeManageApplicationSettings, "manage:application_settings"},
		{ScopeCreateExchange, "create:exchange"},
		{ScopeReadExchange, "read:exchange"},
		{ScopeManageExchange, "manage:exchange"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("Constant value = %q, expected %q", tt.constant, tt.expected)
		}
	}
}
