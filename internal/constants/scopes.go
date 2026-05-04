package constants

// Anypoint Platform Scope Names
// This file contains all valid scope names for Anypoint Connected Applications.
// These scopes control access to various platform services and features.

const (
	// ScopeAdminAccessControls provides full administrative access to access controls.
	ScopeAdminAccessControls             = "admin:access_controls"
	ScopeAdminAngGovernanceProfiles      = "admin:ang_governance_profiles"
	ScopeAdminAPIManager                 = "admin:api_manager"
	ScopeAdminAPIQuery                   = "admin:api_query"
	ScopeAdminCloudHub                   = "admin:cloudhub"
	ScopeAdminDataExporterConfigurations = "admin:data_exporter_configurations"
	ScopeAdminDataExporterConnections    = "admin:data_exporter_connections"
	ScopeAdminOrgClientProviderClients   = "admin:orgclientproviderclients"
	ScopeAdminOrgClientProviders         = "admin:orgclientproviders"
	ScopeAdminOrgClients                 = "admin:orgclients"
	ScopeAdminPartnerManager             = "admin:partner_manager"

	// ScopeAdministerDestinations provides administrative operations on destinations.
	ScopeAdministerDestinations = "administer:destinations"

	// ScopeAEHAdmin provides Anypoint Event Hub administrative access.
	ScopeAEHAdmin = "aeh_admin"

	// ScopeClearDestinations provides clear/reset operations on destinations.
	ScopeClearDestinations = "clear:destinations"

	// ScopeCreateApplications provides application creation access.
	ScopeCreateApplications       = "create:applications"
	ScopeCreateClientApplications = "create:client_applications"
	ScopeCreateDesignCenter       = "create:design_center"
	ScopeCreateEnvironment        = "create:environment"
	ScopeCreateExchange           = "create:exchange"
	ScopeCreateExchangeGenAI      = "create:exchange_genai"
	ScopeCreateGenerations        = "create:generations"
	ScopeCreateOrgClients         = "create:orgclients"
	ScopeCreateSubOrgs            = "create:suborgs"

	// ScopeDeleteApplications provides application deletion access.
	ScopeDeleteApplications = "delete:applications"

	// ScopeDownloadApplications provides application download access.
	ScopeDownloadApplications = "download:applications"

	// ScopeEditAPICatalog provides API catalog editing access.
	ScopeEditAPICatalog        = "edit:api_catalog"
	ScopeEditAPIQuery          = "edit:api_query"
	ScopeEditDesignCenter      = "edit:design_center"
	ScopeEditEnvironment       = "edit:environment"
	ScopeEditFlowDesigner      = "edit:flow_designer"
	ScopeEditIdentityProviders = "edit:identityproviders"
	ScopeEditMonitoring        = "edit:monitoring"
	ScopeEditOrganization      = "edit:organization"
	ScopeEditOrgInvites        = "edit:orginvites"
	ScopeEditOrgUsers          = "edit:orgusers"
	ScopeEditRPA               = "edit:rpa"
	ScopeEditVisualizer        = "edit:visualizer"

	// ScopeExecuteDocumentActions provides document action execution access.
	ScopeExecuteDocumentActions = "execute:document_actions"

	// ScopeManageActivity provides activity management access.
	ScopeManageActivity             = "manage:activity"
	ScopeManageAPIAlerts            = "manage:api_alerts"
	ScopeManageAPIConfiguration     = "manage:api_configuration"
	ScopeManageAPIContracts         = "manage:api_contracts"
	ScopeManageAPIGroups            = "manage:api_groups"
	ScopeManageAPIPolicies          = "manage:api_policies"
	ScopeManageAPIProxies           = "manage:api_proxies"
	ScopeManageAPIQuery             = "manage:api_query"
	ScopeManageAPIs                 = "manage:apis"
	ScopeManageApplicationAlerts    = "manage:application_alerts"
	ScopeManageApplicationData      = "manage:application_data"
	ScopeManageApplicationFlows     = "manage:application_flows"
	ScopeManageApplicationQueues    = "manage:application_queues"
	ScopeManageApplicationSchedules = "manage:application_schedules"
	ScopeManageApplicationSettings  = "manage:application_settings"
	ScopeManageApplicationTenants   = "manage:application_tenants"
	ScopeManageClients              = "manage:clients"
	ScopeManageCloudHubNetworking   = "manage:cloudhub_networking"
	ScopeManageDataGateway          = "manage:data_gateway"
	ScopeManageEnvClientProviders   = "manage:envclientproviders"
	ScopeManageExchange             = "manage:exchange"
	ScopeManageHost                 = "manage:host"
	ScopeManageIdentityProviders    = "manage:identityproviders"
	ScopeManagePartners             = "manage:partners"
	ScopeManagePrivateSpaces        = "manage:private_spaces"
	ScopeManageRuntimeFabrics       = "manage:runtime_fabrics"
	ScopeManageSecretGroups         = "manage:secret_groups"
	ScopeManageSecrets              = "manage:secrets"
	ScopeManageServers              = "manage:servers"
	ScopeManageStore                = "manage:store"
	ScopeManageStoreClients         = "manage:store_clients"
	ScopeManageStoreData            = "manage:store_data"

	// ScopePromoteAPIQuery provides API Query promotion access.
	ScopePromoteAPIQuery = "promote:api_query"

	// ScopePublishDestinations provides destination publishing access.
	ScopePublishDestinations = "publish:destinations"

	// ScopeReadActivity provides read-only access to activity data.
	ScopeReadActivity                 = "read:activity"
	ScopeReadAPIConfiguration         = "read:api_configuration"
	ScopeReadAPIContracts             = "read:api_contracts"
	ScopeReadAPIPolicies              = "read:api_policies"
	ScopeReadAPIQuery                 = "read:api_query"
	ScopeReadApplicationAlerts        = "read:application_alerts"
	ScopeReadApplications             = "read:applications"
	ScopeReadAuditLogs                = "read:audit_logs"
	ScopeReadClientApplications       = "read:client_applications"
	ScopeReadCloudHubNetworking       = "read:cloudhub_networking"
	ScopeReadDataGateway              = "read:data_gateway"
	ScopeReadExchange                 = "read:exchange"
	ScopeReadHostPartners             = "read:host_partners"
	ScopeReadOrgClientProviderClients = "read:orgclientproviderclients"
	ScopeReadOrgClientProviders       = "read:orgclientproviders"
	ScopeReadOrgClients               = "read:orgclients"
	ScopeReadOrgConnApps              = "read:orgconnapps"
	ScopeReadOrgEnvironments          = "read:orgenvironments"
	ScopeReadOrgInvites               = "read:orginvites"
	ScopeReadOrganization             = "read:organization"
	ScopeReadOrgUsers                 = "read:orgusers"
	ScopeReadRuntimeFabrics           = "read:runtime_fabrics"
	ScopeReadSecrets                  = "read:secrets"
	ScopeReadSecretsMetadata          = "read:secrets_metadata"
	ScopeReadServers                  = "read:servers"
	ScopeReadStats                    = "read:stats"
	ScopeReadStore                    = "read:store"
	ScopeReadStoreClients             = "read:store_clients"
	ScopeReadStoreMetrics             = "read:store_metrics"

	// ScopeRestartApplications provides application restart access.
	ScopeRestartApplications = "restart:applications"

	// ScopeSubscribeDestinations provides destination subscription access.
	ScopeSubscribeDestinations = "subscribe:destinations"

	// ScopeViewAccessControls provides view access to access controls.
	ScopeViewAccessControls        = "view:access_controls"
	ScopeViewAngGovernanceProfiles = "view:ang_governance_profiles"
	ScopeViewClients               = "view:clients"
	ScopeViewDesignCenter          = "view:design_center"
	ScopeViewDestinations          = "view:destinations"
	ScopeViewEnvClientProviders    = "view:envclientproviders"
	ScopeViewEnvironment           = "view:environment"
	ScopeViewIdentityProviders     = "view:identityproviders"
	ScopeViewMetering              = "view:metering"
	ScopeViewMonitoring            = "view:monitoring"

	// ScopeWriteAuditLogSettings provides write access to audit log settings.
	ScopeWriteAuditLogSettings = "write:audit_log_settings"
)

// ValidScopes is a set of all valid scope names for validation
var ValidScopes = map[string]bool{
	// Admin Scopes
	ScopeAdminAccessControls:             true,
	ScopeAdminAngGovernanceProfiles:      true,
	ScopeAdminAPIManager:                 true,
	ScopeAdminAPIQuery:                   true,
	ScopeAdminCloudHub:                   true,
	ScopeAdminDataExporterConfigurations: true,
	ScopeAdminDataExporterConnections:    true,
	ScopeAdminOrgClientProviderClients:   true,
	ScopeAdminOrgClientProviders:         true,
	ScopeAdminOrgClients:                 true,
	ScopeAdminPartnerManager:             true,

	// Administer Scopes
	ScopeAdministerDestinations: true,

	// AEH Admin
	ScopeAEHAdmin: true,

	// Clear Scopes
	ScopeClearDestinations: true,

	// Create Scopes
	ScopeCreateApplications:       true,
	ScopeCreateClientApplications: true,
	ScopeCreateDesignCenter:       true,
	ScopeCreateEnvironment:        true,
	ScopeCreateExchange:           true,
	ScopeCreateExchangeGenAI:      true,
	ScopeCreateGenerations:        true,
	ScopeCreateOrgClients:         true,
	ScopeCreateSubOrgs:            true,

	// Delete Scopes
	ScopeDeleteApplications: true,

	// Download Scopes
	ScopeDownloadApplications: true,

	// Edit Scopes
	ScopeEditAPICatalog:        true,
	ScopeEditAPIQuery:          true,
	ScopeEditDesignCenter:      true,
	ScopeEditEnvironment:       true,
	ScopeEditFlowDesigner:      true,
	ScopeEditIdentityProviders: true,
	ScopeEditMonitoring:        true,
	ScopeEditOrganization:      true,
	ScopeEditOrgInvites:        true,
	ScopeEditOrgUsers:          true,
	ScopeEditRPA:               true,
	ScopeEditVisualizer:        true,

	// Execute Scopes
	ScopeExecuteDocumentActions: true,

	// Manage Scopes
	ScopeManageActivity:             true,
	ScopeManageAPIAlerts:            true,
	ScopeManageAPIConfiguration:     true,
	ScopeManageAPIContracts:         true,
	ScopeManageAPIGroups:            true,
	ScopeManageAPIPolicies:          true,
	ScopeManageAPIProxies:           true,
	ScopeManageAPIQuery:             true,
	ScopeManageAPIs:                 true,
	ScopeManageApplicationAlerts:    true,
	ScopeManageApplicationData:      true,
	ScopeManageApplicationFlows:     true,
	ScopeManageApplicationQueues:    true,
	ScopeManageApplicationSchedules: true,
	ScopeManageApplicationSettings:  true,
	ScopeManageApplicationTenants:   true,
	ScopeManageClients:              true,
	ScopeManageCloudHubNetworking:   true,
	ScopeManageDataGateway:          true,
	ScopeManageEnvClientProviders:   true,
	ScopeManageExchange:             true,
	ScopeManageHost:                 true,
	ScopeManageIdentityProviders:    true,
	ScopeManagePartners:             true,
	ScopeManagePrivateSpaces:        true,
	ScopeManageRuntimeFabrics:       true,
	ScopeManageSecretGroups:         true,
	ScopeManageSecrets:              true,
	ScopeManageServers:              true,
	ScopeManageStore:                true,
	ScopeManageStoreClients:         true,
	ScopeManageStoreData:            true,

	// Promote Scopes
	ScopePromoteAPIQuery: true,

	// Publish Scopes
	ScopePublishDestinations: true,

	// Read Scopes
	ScopeReadActivity:                 true,
	ScopeReadAPIConfiguration:         true,
	ScopeReadAPIContracts:             true,
	ScopeReadAPIPolicies:              true,
	ScopeReadAPIQuery:                 true,
	ScopeReadApplicationAlerts:        true,
	ScopeReadApplications:             true,
	ScopeReadAuditLogs:                true,
	ScopeReadClientApplications:       true,
	ScopeReadCloudHubNetworking:       true,
	ScopeReadDataGateway:              true,
	ScopeReadExchange:                 true,
	ScopeReadHostPartners:             true,
	ScopeReadOrgClientProviderClients: true,
	ScopeReadOrgClientProviders:       true,
	ScopeReadOrgClients:               true,
	ScopeReadOrgConnApps:              true,
	ScopeReadOrgEnvironments:          true,
	ScopeReadOrgInvites:               true,
	ScopeReadOrganization:             true,
	ScopeReadOrgUsers:                 true,
	ScopeReadRuntimeFabrics:           true,
	ScopeReadSecrets:                  true,
	ScopeReadSecretsMetadata:          true,
	ScopeReadServers:                  true,
	ScopeReadStats:                    true,
	ScopeReadStore:                    true,
	ScopeReadStoreClients:             true,
	ScopeReadStoreMetrics:             true,

	// Restart Scopes
	ScopeRestartApplications: true,

	// Subscribe Scopes
	ScopeSubscribeDestinations: true,

	// View Scopes
	ScopeViewAccessControls:        true,
	ScopeViewAngGovernanceProfiles: true,
	ScopeViewClients:               true,
	ScopeViewDesignCenter:          true,
	ScopeViewDestinations:          true,
	ScopeViewEnvClientProviders:    true,
	ScopeViewEnvironment:           true,
	ScopeViewIdentityProviders:     true,
	ScopeViewMetering:              true,
	ScopeViewMonitoring:            true,

	// Write Scopes
	ScopeWriteAuditLogSettings: true,
}

// IsValidScope validates if a given scope name is valid
func IsValidScope(scope string) bool {
	return ValidScopes[scope]
}

// GetAllScopes returns a slice of all valid scope names
func GetAllScopes() []string {
	scopes := make([]string, 0, len(ValidScopes))
	for scope := range ValidScopes {
		scopes = append(scopes, scope)
	}
	return scopes
}
