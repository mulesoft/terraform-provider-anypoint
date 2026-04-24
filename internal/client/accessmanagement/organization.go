package accessmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// OrganizationClient wraps the UserAnypointClient for organization operations
type OrganizationClient struct {
	*client.UserAnypointClient
}

// NewOrganizationClient creates a new OrganizationClient using UserAnypointClient
func NewOrganizationClient(config *client.UserClientConfig) (*OrganizationClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &OrganizationClient{UserAnypointClient: userAnypointClient}, nil
}

// VCoreEntitlement represents a vCore-style entitlement structure
type VCoreEntitlement struct {
	Assigned   float64 `json:"assigned" tfsdk:"assigned"`
	Reassigned float64 `json:"reassigned,omitempty" tfsdk:"reassigned"`
}

type AssignedEntitlement struct {
	Assigned int `json:"assigned" tfsdk:"assigned"`
}

type EnabledEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type HybridEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type WorkerLoggingOverrideEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type MqEntitlement struct {
	Base  int `json:"base" tfsdk:"base"`
	AddOn int `json:"addOn" tfsdk:"add_on"`
}

type DesignCenterEntitlement struct {
	API    bool `json:"api" tfsdk:"api"`
	Mozart bool `json:"mozart" tfsdk:"mozart"`
}

type MonitoringCenterEntitlement struct {
	ProductSKU           int `json:"productSKU" tfsdk:"product_sku"`
	RawStorageOverrideGB int `json:"rawStorageOverrideGB" tfsdk:"raw_storage_override_gb"`
}

type ObjectStoreEntitlement struct {
	Base  int `json:"base" tfsdk:"base"`
	AddOn int `json:"addOn" tfsdk:"add_on"`
}

type PartnersEntitlement struct {
	Assigned int `json:"assigned" tfsdk:"assigned"`
}

type TradingPartnersEntitlement struct {
	Assigned int `json:"assigned" tfsdk:"assigned"`
}

type ApisEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type ApiMonitoringEntitlement struct {
	Schedules int `json:"schedules" tfsdk:"schedules"`
}

type ApiCommunityManagerEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type ApiExperienceHubEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type ApiQueryEntitlement struct {
	Enabled    bool `json:"enabled" tfsdk:"enabled"`
	Sandbox    int  `json:"sandbox" tfsdk:"sandbox"`
	ProductSKU int  `json:"productSKU" tfsdk:"product_sku"`
	Production int  `json:"production" tfsdk:"production"`
}

type ApiQueryC360Entitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type RpaEntitlement struct {
	Enabled         bool   `json:"enabled" tfsdk:"enabled"`
	Active          bool   `json:"active" tfsdk:"active"`
	ComposerVersion string `json:"composerVersion" tfsdk:"composer_version"`
}

type IdpEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type ApiGovernanceEntitlement struct {
	Enabled      bool `json:"enabled" tfsdk:"enabled"`
	ApisPerMonth int  `json:"apisPerMonth" tfsdk:"apis_per_month"`
}

type ApiGovernanceDomainEntitlement struct {
	Domain string `json:"domain" tfsdk:"domain"`
}

type CrowdEntitlement struct {
	HideApiManagerDesigner bool `json:"hideApiManagerDesigner" tfsdk:"hide_api_manager_designer"`
	HideFormerApiPlatform  bool `json:"hideFormerApiPlatform" tfsdk:"hide_former_api_platform"`
	Environments           bool `json:"environments" tfsdk:"environments"`
}

type CamEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type Exchange2Entitlement struct {
	Enabled                 bool `json:"enabled" tfsdk:"enabled"`
	AssetUsageAndEngagement struct {
		Enabled bool `json:"enabled" tfsdk:"enabled"`
	} `json:"assetUsageAndEngagement" tfsdk:"asset_usage_and_engagement"`
}

type CrowdSelfServiceMigrationEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type KpiDashboardEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type AnypointSecurityTokenizationEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type AnypointSecurityEdgePoliciesEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type RuntimeFabricCloudEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type ApiCatalogEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type ApiManagerEntitlement struct {
	Governance struct {
		Enabled bool `json:"enabled" tfsdk:"enabled"`
	} `json:"governance" tfsdk:"governance"`
	UsageBasedPricing struct {
		Api struct {
			Production struct {
				Amount int `json:"amount" tfsdk:"amount"`
			} `json:"production" tfsdk:"production"`
			Sandbox struct {
				Amount int `json:"amount" tfsdk:"amount"`
			} `json:"sandbox" tfsdk:"sandbox"`
		} `json:"api" tfsdk:"api"`
	} `json:"usageBasedPricing" tfsdk:"usage_based_pricing"`
}

type ComposerEntitlement struct {
	Enabled             bool `json:"enabled" tfsdk:"enabled"`
	TasksPerMonth       int  `json:"tasksPerMonth" tfsdk:"tasks_per_month"`
	MaxConnectors       int  `json:"maxConnectors" tfsdk:"max_connectors"`
	UnlimitedConnectors bool `json:"unlimitedConnectors" tfsdk:"unlimited_connectors"`
	IsHyperAutomation   bool `json:"isHyperAutomation" tfsdk:"is_hyper_automation"`
}

type MuleDxWebIdeEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type MuleDxGenAIEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type MuleDxEDAEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type UsageBasedPricingEntitlement struct {
	MuleRuntimeIntegration struct {
		Enabled    bool `json:"enabled" tfsdk:"enabled"`
		Flows      int  `json:"flows" tfsdk:"flows"`
		Messages   int  `json:"messages" tfsdk:"messages"`
		Throughput int  `json:"throughput" tfsdk:"throughput"`
	} `json:"muleRuntimeIntegration" tfsdk:"mule_runtime_integration"`
}

type UsageBasedPricingLimitsEntitlement struct {
	Cpu struct {
		Base  int `json:"base" tfsdk:"base"`
		Extra int `json:"extra" tfsdk:"extra"`
	} `json:"cpu" tfsdk:"cpu"`
	Memory struct {
		Base  int `json:"base" tfsdk:"base"`
		Extra int `json:"extra" tfsdk:"extra"`
	} `json:"memory" tfsdk:"memory"`
}

type HighAvailabilityEntitlement struct {
	Clustering bool `json:"clustering" tfsdk:"clustering"`
}

type Cloudhub1Entitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type MessagingEntitlement struct {
	Assigned int `json:"assigned" tfsdk:"assigned"`
}

type WorkerCloudsEntitlement struct {
	Assigned   int `json:"assigned" tfsdk:"assigned"`
	Reassigned int `json:"reassigned" tfsdk:"reassigned"`
}

type TelemetryExporterEntitlement struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

// Entitlements represents the entitlements structure for an organization
type Entitlements struct {
	CreateSubOrgs        bool              `json:"createSubOrgs"`
	CreateEnvironments   bool              `json:"createEnvironments"`
	GlobalDeployment     bool              `json:"globalDeployment"`
	VCoresProduction     *VCoreEntitlement `json:"vCoresProduction,omitempty"`
	VCoresSandbox        *VCoreEntitlement `json:"vCoresSandbox,omitempty"`
	VCoresDesign         *VCoreEntitlement `json:"vCoresDesign,omitempty"`
	StaticIps            *VCoreEntitlement `json:"staticIps,omitempty"`
	Vpcs                 *VCoreEntitlement `json:"vpcs,omitempty"`
	Vpns                 *VCoreEntitlement `json:"vpns,omitempty"`
	NetworkConnections   *VCoreEntitlement `json:"networkConnections,omitempty"`
	Hybrid               *HybridEntitlement                `json:"hybrid,omitempty"`
	RuntimeFabric        bool                              `json:"runtimeFabric"`
	FlexGateway          *EnabledEntitlement               `json:"flexGateway,omitempty"`
	WorkerLoggingOverride *WorkerLoggingOverrideEntitlement `json:"workerLoggingOverride,omitempty"`
	MqMessages           *MqEntitlement                    `json:"mqMessages,omitempty"`
	MqRequests           *MqEntitlement                    `json:"mqRequests,omitempty"`
	Gateways             *AssignedEntitlement              `json:"gateways,omitempty"`
	DesignCenter         *DesignCenterEntitlement          `json:"designCenter,omitempty"`
	LoadBalancer         *AssignedEntitlement              `json:"loadBalancer,omitempty"`
	ServiceMesh          *EnabledEntitlement               `json:"serviceMesh,omitempty"`
	ManagedGatewaySmall  *AssignedEntitlement              `json:"managedGatewaySmall,omitempty"`
	ManagedGatewayLarge  *AssignedEntitlement              `json:"managedGatewayLarge,omitempty"`
}

// Subscription represents the subscription details for an organization
type Subscription struct {
	Category      string  `json:"category" tfsdk:"category"`
	Type          string  `json:"type" tfsdk:"type"`
	Expiration    string  `json:"expiration" tfsdk:"expiration"`
	Justification *string `json:"justification" tfsdk:"justification"`
}

// OrgEnvironment represents an environment within an organization
type OrgEnvironment struct {
	ID             string  `json:"id" tfsdk:"id"`
	Name           string  `json:"name" tfsdk:"name"`
	OrganizationID string  `json:"organizationId" tfsdk:"organization_id"`
	IsProduction   bool    `json:"isProduction" tfsdk:"is_production"`
	Type           string  `json:"type" tfsdk:"type"`
	ClientID       string  `json:"clientId" tfsdk:"client_id"`
	ArcNamespace   *string `json:"arcNamespace" tfsdk:"arc_namespace"`
}

// Owner represents the owner details of an organization
type Owner struct {
	ID                      string  `json:"id" tfsdk:"id"`
	FirstName               string  `json:"firstName" tfsdk:"first_name"`
	LastName                string  `json:"lastName" tfsdk:"last_name"`
	Email                   string  `json:"email" tfsdk:"email"`
	Username                string  `json:"username" tfsdk:"username"`
	Enabled                 bool    `json:"enabled" tfsdk:"enabled"`
	CreatedAt               string  `json:"createdAt" tfsdk:"created_at"`
	UpdatedAt               string  `json:"updatedAt" tfsdk:"updated_at"`
	OrganizationID          string  `json:"organizationId" tfsdk:"organization_id"`
	PhoneNumber             string  `json:"phoneNumber" tfsdk:"phone_number"`
	IdproviderID            string  `json:"idprovider_id" tfsdk:"idprovider_id"`
	Deleted                 bool    `json:"deleted" tfsdk:"deleted"`
	LastLogin               string  `json:"lastLogin" tfsdk:"last_login"`
	MfaVerificationExcluded bool    `json:"mfaVerificationExcluded" tfsdk:"mfa_verification_excluded"`
	MfaVerifiersConfigured  string  `json:"mfaVerifiersConfigured" tfsdk:"mfa_verifiers_configured"`
	EmailVerifiedAt         *string `json:"emailVerifiedAt" tfsdk:"email_verified_at"`
	GdouID                  string  `json:"gdouId" tfsdk:"gdou_id"`
	PreviousLastLogin       string  `json:"previousLastLogin" tfsdk:"previous_last_login"`
	Type                    string  `json:"type" tfsdk:"type"`
}

// Organization represents an Anypoint Organization
type Organization struct {
	ID                              string                 `json:"id"`
	Name                            string                 `json:"name"`
	CreatedAt                       string                 `json:"createdAt"`
	UpdatedAt                       string                 `json:"updatedAt"`
	OwnerID                         string                 `json:"ownerId"`
	ClientID                        string                 `json:"clientId"`
	IdProviderID                    string                 `json:"idprovider_id"`
	IsFederated                     bool                   `json:"isFederated"`
	ParentOrganizationIds           []string               `json:"parentOrganizationIds"`
	SubOrganizationIds              []string               `json:"subOrganizationIds"`
	TenantOrganizationIds           []string               `json:"tenantOrganizationIds"`
	MfaRequired                     string                 `json:"mfaRequired"`
	IsAutomaticAdminPromotionExempt bool                   `json:"isAutomaticAdminPromotionExempt"`
	OrgType                         string                 `json:"orgType"`
	GdotID                          *string                `json:"gdotId"`
	DeletedAt                       *string                `json:"deletedAt"`
	Domain                          *string                `json:"domain"`
	IsRoot                          bool                   `json:"isRoot"`
	IsMaster                        bool                   `json:"isMaster"`
	Properties                      map[string]interface{} `json:"properties"`
	Entitlements                    Entitlements           `json:"entitlements"`
	Subscription                    Subscription           `json:"subscription"`
	Environments                    []OrgEnvironment       `json:"environments"`
	Owner                           Owner                  `json:"owner"`
	SessionTimeout                  int                    `json:"sessionTimeout"`
}

// CreateOrganizationRequest represents the request to create an organization
type CreateOrganizationRequest struct {
	Name                 string       `json:"name"`
	ParentOrganizationID string       `json:"parentOrganizationId"`
	OwnerID              string       `json:"ownerId"`
	Entitlements         Entitlements `json:"entitlements"`
}

// CreateOrganization creates a new organization in Anypoint
func (c *OrganizationClient) CreateOrganization(ctx context.Context, org *CreateOrganizationRequest) (*Organization, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations", c.BaseURL)

	jsonData, err := json.Marshal(org)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal organization data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create organization with status %d: %s", resp.StatusCode, string(body))
	}

	var createdOrg Organization
	if err := json.NewDecoder(resp.Body).Decode(&createdOrg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdOrg, nil
}

// GetOrganization retrieves an organization by ID
func (c *OrganizationClient) GetOrganization(ctx context.Context, organizationID string) (*Organization, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s", c.BaseURL, organizationID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("organization")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get organization with status %d: %s", resp.StatusCode, string(body))
	}

	var organization Organization
	if err := json.NewDecoder(resp.Body).Decode(&organization); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &organization, nil
}

// DeleteOrganizationRequest represents the request payload for deleting an organization
type DeleteOrganizationRequest struct {
	ID string `json:"id"`
}

// DeleteOrganization deletes an organization by ID
func (c *OrganizationClient) DeleteOrganization(ctx context.Context, organizationID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s", c.BaseURL, organizationID)

	reqBody := DeleteOrganizationRequest{
		ID: organizationID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("organization")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete organization with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// WaitForOrganizationDeletion waits for the organization to be fully deleted
// This is necessary to prevent "name already used" errors when recreating an organization
func (c *OrganizationClient) WaitForOrganizationDeletion(ctx context.Context, organizationID string, maxRetries int, retryInterval time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		_, err := c.GetOrganization(ctx, organizationID)
		if err != nil {
			// If we get an error containing "not found", the deletion is complete
			// This handles various error message formats (e.g., "organization not found", "not found", "404")
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "404") {
				return nil
			}
			// For other errors, continue retrying as the organization might still be deleting
		}

		// Organization still exists, wait before retrying
		if i < maxRetries-1 {
			time.Sleep(retryInterval)
		}
	}

	return fmt.Errorf("timeout waiting for organization deletion after %d retries", maxRetries)
}
