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
	ScopeCreateOmniGenAI          = "create:omni_genai"
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
	ScopeEditExchange          = "edit:exchange"
	ScopeEditFlowDesigner      = "edit:flow_designer"
	ScopeEditIdentityProviders = "edit:identityproviders"
	ScopeEditMonitoring        = "edit:monitoring"
	ScopeEditOrganization      = "edit:organization"
	ScopeEditOrgInvites        = "edit:orginvites"
	ScopeEditOrgUsers          = "edit:orgusers"
	ScopeEditRPA               = "edit:rpa"
	ScopeEditVisualizer        = "edit:visualizer"

	// ScopeEmail provides email claim access (OpenID).
	ScopeEmail = "email"

	// ScopeExecutionRPA provides RPA process execution access.
	ScopeExecutionRPA = "execution:rpa"

	// ScopeExecuteDocumentActions provides document action execution access.
	ScopeExecuteDocumentActions = "execute:document_actions"

	// ScopeFull provides full unrestricted access.
	ScopeFull = "full"

	// ScopeManageActivity provides activity management access.
	ScopeManageActivity              = "manage:activity"
	ScopeManageAPIContractsAllEnvs   = "manage:api_contracts_all_envs"
	ScopeManageAPIAlerts             = "manage:api_alerts"
	ScopeManageAPIConfiguration      = "manage:api_configuration"
	ScopeManageAPIContracts          = "manage:api_contracts"
	ScopeManageAPIGroups             = "manage:api_groups"
	ScopeManageAPIPolicies           = "manage:api_policies"
	ScopeManageAPIProxies            = "manage:api_proxies"
	ScopeManageAPIQuery              = "manage:api_query"
	ScopeManageAPIs                  = "manage:apis"
	ScopeManageApplicationAlerts     = "manage:application_alerts"
	ScopeManageApplicationData       = "manage:application_data"
	ScopeManageApplicationFlows      = "manage:application_flows"
	ScopeManageApplicationQueues     = "manage:application_queues"
	ScopeManageApplicationSchedules  = "manage:application_schedules"
	ScopeManageApplicationSettings   = "manage:application_settings"
	ScopeManageApplicationTenants    = "manage:application_tenants"
	ScopeManageClients               = "manage:clients"
	ScopeManageCloudHubNetworking    = "manage:cloudhub_networking"
	ScopeManageDataGateway           = "manage:data_gateway"
	ScopeManageEnvClientProviders    = "manage:envclientproviders"
	ScopeManageExchange              = "manage:exchange"
	ScopeManageHost                  = "manage:host"
	ScopeManageIdentityProviders     = "manage:identityproviders"
	ScopeManagePartners              = "manage:partners"
	ScopeManagePrivateSpaces         = "manage:private_spaces"
	ScopeManageRuntimeFabrics        = "manage:runtime_fabrics"
	ScopeManageSecretGroups          = "manage:secret_groups"
	ScopeManageSecrets               = "manage:secrets"
	ScopeManageServers               = "manage:servers"
	ScopeManageClientApplication     = "manage:client_application"
	ScopeManageOrgClientApplications = "manage:org_client_applications"
	ScopeManageStore                 = "manage:store"
	ScopeManageStoreClients          = "manage:store_clients"
	ScopeManageStoreData             = "manage:store_data"

	// OpenID/OAuth scopes
	ScopeOfflineAccess = "offline_access"
	ScopeOpenID        = "openid"
	ScopeOpenIDGoogle  = "openid:google_wif"
	ScopeProfile       = "profile"

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
	ScopeReadAnypointSalesforceSso    = "read:anypoint_salesforce_sso"
	ScopeReadAPIAlerts                = "read:api_alerts"
	ScopeReadAPIPoliciesAllEnvs       = "read:api_policies_all_envs"
	ScopeReadFull                     = "read:full"
	ScopeReadMavenRepository          = "read:maven_repository"
	ScopeReadOrgClientApplications    = "read:org_client_applications"
	ScopeReadStoreClients             = "read:store_clients"
	ScopeReadStoreMetrics             = "read:store_metrics"

	// ScopeRestartApplications provides application restart access.
	ScopeRestartApplications = "restart:applications"

	// ScopeSubscribeDestinations provides destination subscription access.
	ScopeSubscribeDestinations = "subscribe:destinations"

	// ScopeViewAccessControls provides view access to access controls.
	ScopeViewAccessControls        = "view:access_controls"
	ScopeViewAllEnvs               = "view:all_envs"
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
	ScopeReadAnypointSalesforceSso:    true,
	ScopeReadAPIAlerts:                true,
	ScopeReadAPIPoliciesAllEnvs:       true,
	ScopeReadFull:                     true,
	ScopeReadMavenRepository:          true,
	ScopeReadOrgClientApplications:    true,

	// Restart Scopes
	ScopeRestartApplications: true,

	// Subscribe Scopes
	ScopeSubscribeDestinations: true,

	// View Scopes
	ScopeViewAccessControls:        true,
	ScopeViewAllEnvs:               true,
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

	// OpenID/OAuth Scopes
	ScopeOfflineAccess: true,
	ScopeOpenID:        true,
	ScopeOpenIDGoogle:  true,
	ScopeProfile:       true,
	ScopeEmail:         true,

	// Additional scopes
	ScopeEditExchange:                true,
	ScopeExecutionRPA:                true,
	ScopeFull:                        true,
	ScopeManageAPIContractsAllEnvs:   true,
	ScopeManageClientApplication:     true,
	ScopeManageOrgClientApplications: true,
	ScopeCreateOmniGenAI:             true,
}

// DisplayNameToScope maps the EXACT display names from the Anypoint Platform
// scopes catalog API (GET /accounts/api/cs/scopes) to scope identifiers.
// These names match what users see in the Anypoint UI when adding scopes.
// Source of truth: live catalog query against devx on 2026-07-08.
var DisplayNameToScope = map[string]string{
	"Access Controls Admin":                              ScopeAdminAccessControls,
	"Admin Client Management Provider Clients":           ScopeAdminOrgClientProviderClients,
	"Admin Client Management Providers":                  ScopeAdminOrgClientProviders,
	"Admin Particular Organization Clients":              ScopeAdminOrgClients,
	"Administer destinations":                            ScopeAdministerDestinations,
	"Anypoint Salesforce SSO":                            ScopeReadAnypointSalesforceSso,
	"API Catalog Contributor":                            ScopeEditAPICatalog,
	"API Experience Hub Admin":                           ScopeAEHAdmin,
	"API Group Administrator":                            ScopeManageAPIGroups,
	"API Manager All Environments Viewer":                ScopeViewAllEnvs,
	"API Manager Environment Administrator":              ScopeManageAPIs,
	"Application Creator":                                ScopeCreateClientApplications,
	"Application Owner":                                  ScopeManageClientApplication,
	"Application Viewer":                                 ScopeReadClientApplications,
	"Audit Log Config Manager":                           ScopeWriteAuditLogSettings,
	"Audit Log Viewer":                                   ScopeReadAuditLogs,
	"Background Access":                                  ScopeOfflineAccess,
	"Clear destinations":                                 ScopeClearDestinations,
	"Client Identity for Google's WIF":                   ScopeOpenIDGoogle,
	"Cloudhub Network Administrator":                     ScopeManageCloudHubNetworking,
	"Cloudhub Network Viewer":                            ScopeReadCloudHubNetworking,
	"Cloudhub Organization Admin":                        ScopeAdminCloudHub,
	"Consume":                                            ScopeReadAPIQuery,
	"Contribute":                                         ScopeEditAPIQuery,
	"Create Applications":                                ScopeCreateApplications,
	"Create BGs under a given org":                       ScopeCreateSubOrgs,
	"Create Environment":                                 ScopeCreateEnvironment,
	"Create Organization Clients":                        ScopeCreateOrgClients,
	"Data Gateway Administrator":                         ScopeManageDataGateway,
	"Data Gateway Viewer":                                ScopeReadDataGateway,
	"DataGraph Admin":                                    ScopeAdminAPIQuery,
	"Delete Applications":                                ScopeDeleteApplications,
	"Deploy API Proxies":                                 ScopeManageAPIProxies,
	"Design Center Creator":                              ScopeCreateDesignCenter,
	"Design Center Developer":                            ScopeEditDesignCenter,
	"Design Center Viewer":                               ScopeViewDesignCenter,
	"Destination publisher for given environment":        ScopePublishDestinations,
	"Destination subscriber for given environment":       ScopeSubscribeDestinations,
	"Download Applications":                              ScopeDownloadApplications,
	"Edit Environment":                                   ScopeEditEnvironment,
	"Edit Identity Management Providers":                 ScopeEditIdentityProviders,
	"Edit Organization":                                  ScopeEditOrganization,
	"Edit users in an organization":                      ScopeEditOrgUsers,
	"Email":                                              ScopeEmail,
	"Exchange Administrator":                             ScopeManageExchange,
	"Exchange Contributor":                               ScopeEditExchange,
	"Exchange Creator":                                   ScopeCreateExchange,
	"Exchange Viewer":                                    ScopeReadExchange,
	"Execute Published Actions":                          ScopeExecuteDocumentActions,
	"Flow Designer Developer":                            ScopeEditFlowDesigner,
	"Full Access":                                        ScopeFull,
	"Generate Asset Documentation with AI":               ScopeCreateExchangeGenAI,
	"Governance Administrator":                           ScopeAdminAngGovernanceProfiles,
	"Governance Viewer":                                  ScopeViewAngGovernanceProfiles,
	"Grant access to secrets":                            ScopeReadSecrets,
	"Identity":                                           ScopeOpenID,
	"Manage Activity":                                    ScopeManageActivity,
	"Manage Alerts":                                      ScopeManageApplicationAlerts,
	"Manage API Alerts":                                  ScopeManageAPIAlerts,
	"Manage APIs Configuration":                          ScopeManageAPIConfiguration,
	"Manage Application Data":                            ScopeManageApplicationData,
	"Manage Application Flows":                           ScopeManageApplicationFlows,
	"Manage Client Applications":                         ScopeManageOrgClientApplications,
	"Manage Contracts":                                   ScopeManageAPIContracts,
	"Manage Contracts All Environments":                  ScopeManageAPIContractsAllEnvs,
	"Manage Environment Client Management Providers":     ScopeManageEnvClientProviders,
	"Manage Host":                                        ScopeManageHost,
	"Manage Identity Management Providers":               ScopeManageIdentityProviders,
	"Manage Partners and Message Flows":                  ScopeManagePartners,
	"Manage Policies":                                    ScopeManageAPIPolicies,
	"Manage Queues":                                      ScopeManageApplicationQueues,
	"Manage Runtime Fabrics":                             ScopeManageRuntimeFabrics,
	"Manage Schedules":                                   ScopeManageApplicationSchedules,
	"Manage Servers":                                     ScopeManageServers,
	"Manage Settings":                                    ScopeManageApplicationSettings,
	"Manage Tenants":                                     ScopeManageApplicationTenants,
	"Manage clients":                                     ScopeManageClients,
	"Manage invites in an organization":                  ScopeEditOrgInvites,
	"Manage secret groups":                               ScopeManageSecretGroups,
	"Manage store clients":                               ScopeManageStoreClients,
	"Manage stores":                                      ScopeManageStore,
	"Manage stores data":                                 ScopeManageStoreData,
	"Maven Repository Reader":                            ScopeReadMavenRepository,
	"Monitoring Administrator":                           ScopeEditMonitoring,
	"Monitoring Viewer":                                  ScopeViewMonitoring,
	"Mule Developer Generative AI User":                  ScopeCreateGenerations,
	"Mulesoft Omni Agent for Anypoint":                   ScopeCreateOmniGenAI,
	"Operate":                                            ScopeManageAPIQuery,
	"Partner Manager Administrator":                      ScopeAdminPartnerManager,
	"Profile":                                            ScopeProfile,
	"Promote":                                            ScopePromoteAPIQuery,
	"RPA Integrator":                                     ScopeEditRPA,
	"RPA Invocable Process":                              ScopeExecutionRPA,
	"Read Alerts":                                        ScopeReadApplicationAlerts,
	"Read Applications":                                  ScopeReadApplications,
	"Read MQ stats":                                      ScopeReadStats,
	"Read Runtime Fabrics":                               ScopeReadRuntimeFabrics,
	"Read Servers":                                       ScopeReadServers,
	"Read secrets metadata":                              ScopeReadSecretsMetadata,
	"Read-Only Access":                                   ScopeReadFull,
	"Restart Applications":                               ScopeRestartApplications,
	"Store Metrics Viewer":                               ScopeReadStoreMetrics,
	"Telemetry Exporter Administrator":                   ScopeAdminDataExporterConnections,
	"Telemetry Exporter Configurations Manager":          ScopeAdminDataExporterConfigurations,
	"Usage Viewer":                                       ScopeViewMetering,
	"View API Alerts":                                    ScopeReadAPIAlerts,
	"View APIs Configuration":                            ScopeReadAPIConfiguration,
	"View Activity":                                      ScopeReadActivity,
	"View All Environments' Client Management Providers": ScopeViewEnvClientProviders,
	"View Client Applications":                           ScopeReadOrgClientApplications,
	"View Client Management Provider Clients":            ScopeReadOrgClientProviderClients,
	"View Client Management Providers":                   ScopeReadOrgClientProviders,
	"View Connected Applications":                        ScopeReadOrgConnApps,
	"View Contracts":                                     ScopeReadAPIContracts,
	"View Environment":                                   ScopeViewEnvironment,
	"View Environments in a particular organization":     ScopeReadOrgEnvironments,
	"View Host, Partners and Message Flows":              ScopeReadHostPartners,
	"View Identity Management Providers":                 ScopeViewIdentityProviders,
	"View Organization":                                  ScopeReadOrganization,
	"View Particular Organization Clients":               ScopeReadOrgClients,
	"View Policies":                                      ScopeReadAPIPolicies,
	"View Policies All Environments":                     ScopeReadAPIPoliciesAllEnvs,
	"View Users in a particular organization":            ScopeReadOrgUsers,
	"View clients":                                       ScopeViewClients,
	"View destinations":                                  ScopeViewDestinations,
	"View invites in an organization":                    ScopeReadOrgInvites,
	"View store clients":                                 ScopeReadStoreClients,
	"View stores":                                        ScopeReadStore,
	"Visualizer Editor":                                  ScopeEditVisualizer,
	"Write secrets":                                      ScopeManageSecrets,
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
