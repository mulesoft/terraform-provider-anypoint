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

// DisplayNameToScope maps human-readable display names to scope identifiers.
// Users can specify either the display name or the identifier in their Terraform config.
var DisplayNameToScope = map[string]string{
	// Admin scopes
	"Access Controls Admin":              ScopeAdminAccessControls,
	"ANG Governance Profiles Admin":      ScopeAdminAngGovernanceProfiles,
	"API Manager Admin":                  ScopeAdminAPIManager,
	"API Query Admin":                    ScopeAdminAPIQuery,
	"CloudHub Admin":                     ScopeAdminCloudHub,
	"Data Exporter Configurations Admin": ScopeAdminDataExporterConfigurations,
	"Data Exporter Connections Admin":    ScopeAdminDataExporterConnections,
	"Org Client Provider Clients Admin":  ScopeAdminOrgClientProviderClients,
	"Org Client Providers Admin":         ScopeAdminOrgClientProviders,
	"Org Clients Admin":                  ScopeAdminOrgClients,
	"Partner Manager Admin":              ScopeAdminPartnerManager,

	// Administer scopes
	"Administer Destinations": ScopeAdministerDestinations,

	// AEH
	"Anypoint Event Hub Admin": ScopeAEHAdmin,

	// Clear scopes
	"Clear Destinations": ScopeClearDestinations,

	// Create scopes
	"Create Applications":        ScopeCreateApplications,
	"Create Client Applications": ScopeCreateClientApplications,
	"Design Center Creator":      ScopeCreateDesignCenter,
	"Create Environment":         ScopeCreateEnvironment,
	"Exchange Creator":           ScopeCreateExchange,
	"Exchange GenAI Creator":     ScopeCreateExchangeGenAI,
	"Generative AI User":         ScopeCreateGenerations,
	"Create Org Clients":         ScopeCreateOrgClients,
	"Create Sub Orgs":            ScopeCreateSubOrgs,

	// Delete scopes
	"Delete Applications": ScopeDeleteApplications,

	// Download scopes
	"Download Applications": ScopeDownloadApplications,

	// Edit scopes
	"API Catalog Editor":        ScopeEditAPICatalog,
	"API Query Editor":          ScopeEditAPIQuery,
	"Design Center Editor":      ScopeEditDesignCenter,
	"Edit Environment":          ScopeEditEnvironment,
	"Flow Designer Editor":      ScopeEditFlowDesigner,
	"Identity Providers Editor": ScopeEditIdentityProviders,
	"Monitoring Editor":         ScopeEditMonitoring,
	"Organization Editor":       ScopeEditOrganization,
	"Org Invites Editor":        ScopeEditOrgInvites,
	"Org Users Editor":          ScopeEditOrgUsers,
	"RPA Editor":                ScopeEditRPA,
	"Visualizer Editor":         ScopeEditVisualizer,

	// Execute scopes
	"Execute Document Actions": ScopeExecuteDocumentActions,

	// Manage scopes
	"Manage Activity":              ScopeManageActivity,
	"Manage API Alerts":            ScopeManageAPIAlerts,
	"Manage API Configuration":     ScopeManageAPIConfiguration,
	"Manage API Contracts":         ScopeManageAPIContracts,
	"Manage API Groups":            ScopeManageAPIGroups,
	"Manage API Policies":          ScopeManageAPIPolicies,
	"Manage API Proxies":           ScopeManageAPIProxies,
	"Manage API Query":             ScopeManageAPIQuery,
	"Manage APIs":                  ScopeManageAPIs,
	"Manage Application Alerts":    ScopeManageApplicationAlerts,
	"Manage Application Data":      ScopeManageApplicationData,
	"Manage Application Flows":     ScopeManageApplicationFlows,
	"Manage Application Queues":    ScopeManageApplicationQueues,
	"Manage Application Schedules": ScopeManageApplicationSchedules,
	"Manage Application Settings":  ScopeManageApplicationSettings,
	"Manage Application Tenants":   ScopeManageApplicationTenants,
	"Manage Clients":               ScopeManageClients,
	"Manage CloudHub Networking":   ScopeManageCloudHubNetworking,
	"Manage Data Gateway":          ScopeManageDataGateway,
	"Manage Env Client Providers":  ScopeManageEnvClientProviders,
	"Manage Exchange":              ScopeManageExchange,
	"Manage Host":                  ScopeManageHost,
	"Manage Identity Providers":    ScopeManageIdentityProviders,
	"Manage Partners":              ScopeManagePartners,
	"Manage Private Spaces":        ScopeManagePrivateSpaces,
	"Manage Runtime Fabrics":       ScopeManageRuntimeFabrics,
	"Manage Secret Groups":         ScopeManageSecretGroups,
	"Manage Secrets":               ScopeManageSecrets,
	"Manage Servers":               ScopeManageServers,
	"Manage Store":                 ScopeManageStore,
	"Manage Store Clients":         ScopeManageStoreClients,
	"Manage Store Data":            ScopeManageStoreData,

	// Promote scopes
	"Promote API Query": ScopePromoteAPIQuery,

	// Publish scopes
	"Publish Destinations": ScopePublishDestinations,

	// Read scopes
	"Read Activity":                    ScopeReadActivity,
	"Read API Configuration":           ScopeReadAPIConfiguration,
	"Read API Contracts":               ScopeReadAPIContracts,
	"Read API Policies":                ScopeReadAPIPolicies,
	"Read API Query":                   ScopeReadAPIQuery,
	"Read Application Alerts":          ScopeReadApplicationAlerts,
	"Read Applications":                ScopeReadApplications,
	"Audit Log Viewer":                 ScopeReadAuditLogs,
	"Read Client Applications":         ScopeReadClientApplications,
	"Read CloudHub Networking":         ScopeReadCloudHubNetworking,
	"Read Data Gateway":                ScopeReadDataGateway,
	"Exchange Viewer":                  ScopeReadExchange,
	"Read Host Partners":               ScopeReadHostPartners,
	"Read Org Client Provider Clients": ScopeReadOrgClientProviderClients,
	"Read Org Client Providers":        ScopeReadOrgClientProviders,
	"Read Org Clients":                 ScopeReadOrgClients,
	"Read Org Connected Apps":          ScopeReadOrgConnApps,
	"Read Org Environments":            ScopeReadOrgEnvironments,
	"Read Org Invites":                 ScopeReadOrgInvites,
	"Read Organization":                ScopeReadOrganization,
	"Read Org Users":                   ScopeReadOrgUsers,
	"Read Runtime Fabrics":             ScopeReadRuntimeFabrics,
	"Read Secrets":                     ScopeReadSecrets,
	"Read Secrets Metadata":            ScopeReadSecretsMetadata,
	"Read Servers":                     ScopeReadServers,
	"Read Stats":                       ScopeReadStats,
	"Read Store":                       ScopeReadStore,
	"Read Store Clients":               ScopeReadStoreClients,
	"Read Store Metrics":               ScopeReadStoreMetrics,

	// Restart scopes
	"Restart Applications": ScopeRestartApplications,

	// Subscribe scopes
	"Subscribe Destinations": ScopeSubscribeDestinations,

	// View scopes
	"View Access Controls":         ScopeViewAccessControls,
	"View ANG Governance Profiles": ScopeViewAngGovernanceProfiles,
	"View Clients":                 ScopeViewClients,
	"View Design Center":           ScopeViewDesignCenter,
	"View Destinations":            ScopeViewDestinations,
	"View Env Client Providers":    ScopeViewEnvClientProviders,
	"View Environment":             ScopeViewEnvironment,
	"View Identity Providers":      ScopeViewIdentityProviders,
	"View Metering":                ScopeViewMetering,
	"View Monitoring":              ScopeViewMonitoring,

	// Write scopes
	"Write Audit Log Settings": ScopeWriteAuditLogSettings,
}

// scopeToDisplayName is the reverse mapping (identifier → display name), built at init.
var scopeToDisplayName map[string]string

func init() {
	scopeToDisplayName = make(map[string]string, len(DisplayNameToScope))
	for displayName, scope := range DisplayNameToScope {
		scopeToDisplayName[scope] = displayName
	}
}

// IsValidScope validates if a given scope name is valid.
// Accepts both identifiers (e.g. "read:exchange") and display names (e.g. "Exchange Viewer").
func IsValidScope(scope string) bool {
	if ValidScopes[scope] {
		return true
	}
	_, ok := DisplayNameToScope[scope]
	return ok
}

// ResolveScopeIdentifier resolves a scope value to its identifier.
// If the input is already a valid identifier, it is returned as-is.
// If the input is a display name, the corresponding identifier is returned.
// Returns the identifier and true if resolved, or the original input and false if not found.
func ResolveScopeIdentifier(scope string) (string, bool) {
	// Check if it's already a valid identifier
	if ValidScopes[scope] {
		return scope, true
	}
	// Check if it's a display name
	if id, ok := DisplayNameToScope[scope]; ok {
		return id, true
	}
	return scope, false
}

// GetDisplayName returns the display name for a scope identifier.
// Returns the display name and true if found, or empty string and false if not.
func GetDisplayName(scope string) (string, bool) {
	dn, ok := scopeToDisplayName[scope]
	return dn, ok
}

// GetAllScopes returns a slice of all valid scope names
func GetAllScopes() []string {
	scopes := make([]string, 0, len(ValidScopes))
	for scope := range ValidScopes {
		scopes = append(scopes, scope)
	}
	return scopes
}
