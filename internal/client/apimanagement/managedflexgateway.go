package apimanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// ManagedFlexGatewayClient wraps the AnypointClient for managed Flex Gateway operations
type ManagedFlexGatewayClient struct {
	*client.AnypointClient
}

// NewManagedFlexGatewayClient creates a new ManagedFlexGatewayClient
func NewManagedFlexGatewayClient(config *client.ClientConfig) (*ManagedFlexGatewayClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &ManagedFlexGatewayClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// ManagedFlexGateway represents a managed Flex Gateway instance
type ManagedFlexGateway struct {
	ID             string                   `json:"id"`
	Name           string                   `json:"name"`
	TargetID       string                   `json:"targetId"`
	TargetName     string                   `json:"targetName,omitempty"`
	TargetType     string                   `json:"targetType,omitempty"`
	RuntimeVersion string                   `json:"runtimeVersion"`
	ReleaseChannel string                   `json:"releaseChannel"`
	Size           string                   `json:"size"`
	Status         string                   `json:"status,omitempty"`
	DesiredStatus  string                   `json:"desiredStatus,omitempty"`
	StatusMessage  string                   `json:"statusMessage,omitempty"`
	DateCreated    string                   `json:"dateCreated,omitempty"`
	LastUpdated    string                   `json:"lastUpdated,omitempty"`
	APILimit       int                      `json:"apiLimit,omitempty"`
	Configuration  ManagedFlexGatewayConfig `json:"configuration"`
	PortConfig     *PortConfiguration       `json:"portConfiguration,omitempty"`
	OrganizationID string                   `json:"organizationId,omitempty"`
	EnvironmentID  string                   `json:"environmentId,omitempty"`
}

// PortEntry represents a single port/protocol pair in port configuration
type PortEntry struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// PortConfiguration holds the ingress and egress port configuration
type PortConfiguration struct {
	Ingress PortEntry `json:"ingress"`
	Egress  PortEntry `json:"egress"`
}

// ManagedFlexGatewayConfig holds the full configuration block
type ManagedFlexGatewayConfig struct {
	Ingress    IngressConfig    `json:"ingress"`
	Properties PropertiesConfig `json:"properties"`
	Logging    LoggingConfig    `json:"logging"`
	Tracing    TracingConfig    `json:"tracing"`
}

// IngressConfig represents the ingress settings for the gateway.
// PublicURL is the primary URL sent in the API request/response (JSON: "publicUrl").
// PublicURLs is populated by BuildIngressURLs for all domains; it is not serialized.
type IngressConfig struct {
	PublicURL         string   `json:"publicUrl,omitempty"`
	PublicURLs        []string `json:"-"`
	InternalURL       string   `json:"internalUrl,omitempty"`
	ForwardSSLSession bool     `json:"forwardSslSession"`
	LastMileSecurity  bool     `json:"lastMileSecurity"`
}

// DomainsResponse represents the response from the domains API
type DomainsResponse struct {
	Domains    []string `json:"domains"`
	AppUniqueID string  `json:"appUniqueId"`
}

// PropertiesConfig represents the runtime properties for the gateway
type PropertiesConfig struct {
	UpstreamResponseTimeout int `json:"upstreamResponseTimeout"`
	ConnectionIdleTimeout   int `json:"connectionIdleTimeout"`
}

// LoggingConfig represents logging settings for the gateway
type LoggingConfig struct {
	Level       string `json:"level"`
	ForwardLogs bool   `json:"forwardLogs"`
}

// TracingConfig represents tracing settings for the gateway
type TracingConfig struct {
	Enabled bool `json:"enabled"`
}

// --- Request / Response Models ---

// CreateManagedFlexGatewayRequest represents the payload to create a gateway
type CreateManagedFlexGatewayRequest struct {
	Name            string                   `json:"name"`
	TargetID        string                   `json:"targetId"`
	RuntimeVersion  string                   `json:"runtimeVersion"`
	ReleaseChannel  string                   `json:"releaseChannel"`
	Size            string                   `json:"size"`
	Configuration   ManagedFlexGatewayConfig `json:"configuration"`
}

// UpdateManagedFlexGatewayRequest represents the full PUT body to update a gateway.
// The update API is a full replacement (PUT), so all fields must be provided.
type UpdateManagedFlexGatewayRequest struct {
	Name           string                   `json:"name"`
	TargetID       string                   `json:"targetId"`
	RuntimeVersion string                   `json:"runtimeVersion"`
	ReleaseChannel string                   `json:"releaseChannel"`
	Size           string                   `json:"size"`
	Configuration  ManagedFlexGatewayConfig `json:"configuration"`
}

// --- CRUD Operations ---

// CreateManagedFlexGateway creates a new managed Flex Gateway
func (c *ManagedFlexGatewayClient) CreateManagedFlexGateway(ctx context.Context, orgID, envID string, request *CreateManagedFlexGatewayRequest) (*ManagedFlexGateway, error) {
	url := fmt.Sprintf("%s/gatewaymanager/api/v1/organizations/%s/environments/%s/gateways", c.BaseURL, orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gateway request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create managed flex gateway with status %d: %s", resp.StatusCode, string(body))
	}

	var gw ManagedFlexGateway
	if err := json.NewDecoder(resp.Body).Decode(&gw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &gw, nil
}

// GetManagedFlexGateway retrieves a managed Flex Gateway by ID
func (c *ManagedFlexGatewayClient) GetManagedFlexGateway(ctx context.Context, orgID, envID, gatewayID string) (*ManagedFlexGateway, error) {
	url := fmt.Sprintf("%s/gatewaymanager/api/v1/organizations/%s/environments/%s/gateways/%s", c.BaseURL, orgID, envID, gatewayID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("managed flex gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get managed flex gateway with status %d: %s", resp.StatusCode, string(body))
	}

	var gw ManagedFlexGateway
	if err := json.NewDecoder(resp.Body).Decode(&gw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &gw, nil
}

// UpdateManagedFlexGateway replaces an existing managed Flex Gateway (PUT).
func (c *ManagedFlexGatewayClient) UpdateManagedFlexGateway(ctx context.Context, orgID, envID, gatewayID string, request *UpdateManagedFlexGatewayRequest) (*ManagedFlexGateway, error) {
	url := fmt.Sprintf("%s/gatewaymanager/api/v1/organizations/%s/environments/%s/gateways/%s", c.BaseURL, orgID, envID, gatewayID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gateway update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("managed flex gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update managed flex gateway with status %d: %s", resp.StatusCode, string(body))
	}

	var gw ManagedFlexGateway
	if err := json.NewDecoder(resp.Body).Decode(&gw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &gw, nil
}

// GetDomains fetches the available domains for a target in an environment.
// The domains are used to construct the public and internal ingress URLs.
func (c *ManagedFlexGatewayClient) GetDomains(ctx context.Context, orgID, targetID, envID string) (*DomainsResponse, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/targets/%s/environments/%s/domains?sendAppUniqueId=true",
		c.BaseURL, orgID, targetID, envID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError(fmt.Sprintf("domains for target %s in environment %s", targetID, envID))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get domains with status %d: %s", resp.StatusCode, string(body))
	}

	var domains DomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&domains); err != nil {
		return nil, fmt.Errorf("failed to decode domains response: %w", err)
	}

	return &domains, nil
}

// BuildIngressURLs constructs the public and internal ingress URLs from
// the gateway name and the raw wildcard domains returned by GetDomains.
// Each domain like "*.hey4z8.usa-e2.stgx.cloudhub.io" produces:
//   - publicURL:   https://<gwName>-<baseDomain>
//   - internalURL: https://<gwName>-.internal-<baseDomain>   (first domain only)
func BuildIngressURLs(gwName string, domains []string) (publicURLs []string, internalURL string) {
	for i, d := range domains {
		base := strings.TrimPrefix(d, "*.")
		publicURLs = append(publicURLs, fmt.Sprintf("https://%s.%s", gwName, base))
		if i == 0 {
			internalURL = fmt.Sprintf("https://%s.internal-%s", gwName, base)
		}
	}
	return
}

// DeleteManagedFlexGateway deletes a managed Flex Gateway by ID
func (c *ManagedFlexGatewayClient) DeleteManagedFlexGateway(ctx context.Context, orgID, envID, gatewayID string) error {
	url := fmt.Sprintf("%s/gatewaymanager/api/v1/organizations/%s/environments/%s/gateways/%s", c.BaseURL, orgID, envID, gatewayID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete managed flex gateway with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// --- Gateway Versions ---

// GatewayVersionsResponse is the top-level response from the versions API
type GatewayVersionsResponse struct {
	Channels map[string]GatewayChannel `json:"channels"`
	Default  string                    `json:"default"`
}

// GatewayChannel represents a release channel (e.g. "edge", "lts")
type GatewayChannel struct {
	Name     string           `json:"name"`
	Versions []GatewayVersion `json:"versions"`
}

// GatewayVersion is a single version entry
type GatewayVersion struct {
	DisplayName string `json:"displayName"`
}

// GetGatewayVersions fetches available gateway runtime versions grouped by release channel
func (c *ManagedFlexGatewayClient) GetGatewayVersions(ctx context.Context) (*GatewayVersionsResponse, error) {
	url := fmt.Sprintf("%s/gatewaymanager/xapi/v1/gateway/versions", c.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get gateway versions with status %d: %s", resp.StatusCode, string(body))
	}

	var versionsResp GatewayVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode gateway versions response: %w", err)
	}

	return &versionsResp, nil
}

// LatestVersionForChannel returns the first (latest) version for a given channel.
// Returns empty string if the channel is not found or has no versions.
func (v *GatewayVersionsResponse) LatestVersionForChannel(channel string) string {
	ch, ok := v.Channels[channel]
	if !ok || len(ch.Versions) == 0 {
		return ""
	}
	return ch.Versions[0].DisplayName
}

// ManagedFlexGatewayListItem represents a single gateway entry from the api/v1 list endpoint.
// The list response has a different (lighter) shape than the single-item CRUD response.
type ManagedFlexGatewayListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TargetID    string `json:"targetId"`
	Status      string `json:"status"`
	DateCreated string `json:"dateCreated"`
	LastUpdated string `json:"lastUpdated"`
}

// ManagedFlexGatewayListResponse wraps the paginated list response from the Gateway Manager list endpoint.
type ManagedFlexGatewayListResponse struct {
	Content       []ManagedFlexGatewayListItem `json:"content"`
	PageSize      int                          `json:"pageSize"`
	PageNumber    int                          `json:"pageNumber"`
	TotalElements int                          `json:"totalElements"`
}

// ListManagedFlexGateways returns all managed Flex Gateways for the given org/environment.
func (c *ManagedFlexGatewayClient) ListManagedFlexGateways(ctx context.Context, orgID, envID string) ([]ManagedFlexGatewayListItem, error) {
	url := fmt.Sprintf("%s/gatewaymanager/api/v1/organizations/%s/environments/%s/gateways", c.BaseURL, orgID, envID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list managed flex gateways with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp ManagedFlexGatewayListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Content, nil
}
