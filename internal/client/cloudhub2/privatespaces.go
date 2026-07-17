package cloudhub2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// PrivateSpacesClient wraps the AnypointClient for private space operations
type PrivateSpacesClient struct {
	*client.AnypointClient
}

// NewPrivateSpacesClient creates a new PrivateSpacesClient
func NewPrivateSpacesClient(config *client.Config) (*PrivateSpacesClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &PrivateSpacesClient{AnypointClient: anypointClient}, nil
}

// PrivateSpace represents an Anypoint Private Space
type PrivateSpace struct {
	ID                      string                    `json:"id"`
	Name                    string                    `json:"name"`
	Version                 string                    `json:"version"`
	Status                  string                    `json:"status"`
	StatusMessage           string                    `json:"statusMessage"`
	Provisioning            PrivateSpaceProvisioning  `json:"provisioning"`
	Region                  string                    `json:"region"`
	OrganizationID          string                    `json:"organizationId"`
	RootOrganizationID      string                    `json:"rootOrganizationId"`
	ManagedFirewallRules    []FirewallRule            `json:"managedFirewallRules"`
	Environments            PrivateSpaceEnvironments  `json:"environments"`
	Network                 NetworkConfig             `json:"network"`
	FirewallRules           []FirewallRule            `json:"firewallRules"`
	LogForwarding           PrivateSpaceLogForwarding `json:"logForwarding"`
	IngressConfiguration    PrivateSpaceIngressConfig `json:"ingressConfiguration"`
	EnableIAMRole           bool                      `json:"enableIAMRole"`
	EnableEgress            bool                      `json:"enableEgress"`
	EnableNetworkIsolation  bool                      `json:"enableNetworkIsolation"`
	GlobalSpaceStatus       GlobalSpaceStatus         `json:"globalSpaceStatus"`
	MuleAppDeploymentCount  int                       `json:"muleAppDeploymentCount"`
	DaysLeftForRelaxedQuota int                       `json:"daysLeftForRelaxedQuota"`
	VPCMigrationInProgress  bool                      `json:"vpcMigrationInProgress"`
}

// PrivateSpaceProvisioning represents the provisioning status
type PrivateSpaceProvisioning struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// PrivateSpaceEnvironments represents the environments configuration
type PrivateSpaceEnvironments struct {
	Type           string   `json:"type"`
	BusinessGroups []string `json:"businessGroups"`
}

// PrivateSpaceLogForwarding represents the log forwarding configuration
type PrivateSpaceLogForwarding struct {
	AnypointMonitoring bool `json:"anypointMonitoring"`
}

// PrivateSpaceIngressConfig represents the ingress configuration
type PrivateSpaceIngressConfig struct {
	ReadResponseTimeout int                           `json:"readResponseTimeout"`
	Logs                PrivateSpaceIngressLogs       `json:"logs"`
	Protocol            string                        `json:"protocol"`
	Deployment          PrivateSpaceIngressDeployment `json:"deployment"`
}

// PrivateSpaceIngressLogs represents the ingress logs configuration
type PrivateSpaceIngressLogs struct {
	Filters      []PrivateSpaceLogFilter `json:"filters"`
	PortLogLevel string                  `json:"portLogLevel"`
}

// PrivateSpaceLogFilter represents a log filter
type PrivateSpaceLogFilter struct {
	IP    string `json:"ip"`
	Level string `json:"level"`
}

// PrivateSpaceIngressDeployment represents the ingress deployment status
type PrivateSpaceIngressDeployment struct {
	Status            string `json:"status"`
	LastSeenTimestamp int64  `json:"lastSeenTimestamp"`
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	CidrBlock string `json:"cidrBlock"`
	Protocol  string `json:"protocol"`
	FromPort  int    `json:"fromPort"`
	ToPort    int    `json:"toPort"`
	Type      string `json:"type"`
}

// GlobalSpaceStatus represents the global space status
type GlobalSpaceStatus struct {
	Space   GlobalSpaceInfo     `json:"space"`
	Cluster []GlobalClusterInfo `json:"cluster"`
	Network GlobalNetworkInfo   `json:"network"`
}

// GlobalSpaceInfo represents the space status information
type GlobalSpaceInfo struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	LastSeenTimeStamp int64  `json:"lastSeenTimeStamp"`
}

// GlobalClusterInfo represents the cluster status information
type GlobalClusterInfo struct {
	Name    string            `json:"name"`
	Infra   GlobalInfraInfo   `json:"infra"`
	Fabric  GlobalFabricInfo  `json:"fabric"`
	Ingress GlobalIngressInfo `json:"ingress"`
}

// GlobalInfraInfo represents the infrastructure status
type GlobalInfraInfo struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	LastSeenTimeStamp int64  `json:"lastSeenTimeStamp"`
}

// GlobalFabricInfo represents the fabric status
type GlobalFabricInfo struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	LastSeenTimeStamp int64  `json:"lastSeenTimeStamp"`
}

// GlobalIngressInfo represents the ingress status
type GlobalIngressInfo struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	LastSeenTimeStamp int64  `json:"lastSeenTimeStamp"`
}

// GlobalNetworkInfo represents the network status
type GlobalNetworkInfo struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	LastSeenTimeStamp int64  `json:"lastSeenTimeStamp"`
}

// NetworkConfig represents the network configuration within a private space
type NetworkConfig struct {
	Region                   string   `json:"region"`
	CidrBlock                string   `json:"cidrBlock"`
	ReservedCIDRs            []string `json:"reservedCidrs"`
	InboundStaticIPs         []string `json:"inboundStaticIps"`
	InboundInternalStaticIPs []string `json:"inboundInternalStaticIps"`
	OutboundStaticIPs        []string `json:"outboundStaticIps"`
	DNSTarget                string   `json:"dnsTarget"`
}

// CreatePrivateSpaceRequest represents the request to create a private space
type CreatePrivateSpaceRequest struct {
	Name          string `json:"name"`
	Region        string `json:"region,omitempty"`
	EnableIAMRole *bool  `json:"enableIAMRole,omitempty"`
	EnableEgress  *bool  `json:"enableEgress,omitempty"`
}

// UpdatePrivateSpaceRequest represents the request to update a private space
type UpdatePrivateSpaceRequest struct {
	Name          *string `json:"name,omitempty"`
	EnableIAMRole *bool   `json:"enableIAMRole,omitempty"`
	EnableEgress  *bool   `json:"enableEgress,omitempty"`
}

// CreatePrivateSpace creates a new private space in Anypoint
func (c *PrivateSpacesClient) CreatePrivateSpace(ctx context.Context, orgID string, privateSpace *CreatePrivateSpaceRequest) (*PrivateSpace, error) {
	if privateSpace == nil {
		return nil, fmt.Errorf("failed to marshal private space data: request cannot be nil")
	}

	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces", c.BaseURL, orgID)

	jsonData, err := json.Marshal(privateSpace)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private space data: %w", err)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create private space with status %d: %s", resp.StatusCode, string(body))
	}

	var createdPrivateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&createdPrivateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdPrivateSpace, nil
}

// GetPrivateSpace retrieves a private space by ID
func (c *PrivateSpacesClient) GetPrivateSpace(ctx context.Context, orgID, privateSpaceID string) (*PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get private space with status %d: %s", resp.StatusCode, string(body))
	}

	var privateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&privateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &privateSpace, nil
}

// ListPrivateSpaces lists all private spaces in an organization.
// The API returns the collection as a JSON OBJECT envelope wrapping the array
// (the array lives under a field such as "content"/"data"), not as a bare array
// like the transit-gateways list does. decodePrivateSpaceList tolerates both.
// API: GET /runtimefabric/api/organizations/{orgId}/privatespaces
func (c *PrivateSpacesClient) ListPrivateSpaces(ctx context.Context, orgID string) ([]PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces", c.BaseURL, orgID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list private spaces with status %d: %s", resp.StatusCode, string(body))
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return decodePrivateSpaceList(raw)
}

// decodePrivateSpaceList extracts the list of private spaces from the list
// endpoint's response body. The endpoint returns an object envelope (the array
// nested under a key like "content"/"data"), unlike the transit-gateways list
// which returns a bare array. To stay correct regardless of the exact envelope
// key, this tolerates a bare array, an object with a well-known array field, or
// any object with a single array-valued field.
func decodePrivateSpaceList(raw json.RawMessage) ([]PrivateSpace, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return []PrivateSpace{}, nil
	}

	// Bare array (forward-compatibility / matches other CH2 list endpoints).
	if trimmed[0] == '[' {
		var spaces []PrivateSpace
		if err := json.Unmarshal(trimmed, &spaces); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return spaces, nil
	}

	// Object envelope: decode fields lazily so a non-array field cannot fail us.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Prefer well-known collection keys.
	for _, key := range []string{"content", "data", "items", "value", "privateSpaces", "spaces", "results"} {
		if v, ok := envelope[key]; ok {
			vt := bytes.TrimSpace(v)
			if len(vt) > 0 && vt[0] == '[' {
				var spaces []PrivateSpace
				if err := json.Unmarshal(vt, &spaces); err != nil {
					return nil, fmt.Errorf("failed to decode response: %w", err)
				}
				return spaces, nil
			}
		}
	}

	// Fallback: if the envelope has exactly one array-valued field, use it.
	// (Map iteration order is random, so only auto-pick when unambiguous.)
	arrayKeys := make([]string, 0, len(envelope))
	for k, v := range envelope {
		vt := bytes.TrimSpace(v)
		if len(vt) > 0 && vt[0] == '[' {
			arrayKeys = append(arrayKeys, k)
		}
	}
	if len(arrayKeys) == 1 {
		var spaces []PrivateSpace
		if err := json.Unmarshal(bytes.TrimSpace(envelope[arrayKeys[0]]), &spaces); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return spaces, nil
	}

	keys := make([]string, 0, len(envelope))
	for k := range envelope {
		keys = append(keys, k)
	}
	if len(arrayKeys) > 1 {
		return nil, fmt.Errorf("failed to decode response: ambiguous list envelope, multiple array fields %v (object keys %v)", arrayKeys, keys)
	}
	return nil, fmt.Errorf("failed to decode response: no private space array found in object with keys %v", keys)
}

// UpdatePrivateSpace updates an existing private space
func (c *PrivateSpacesClient) UpdatePrivateSpace(ctx context.Context, orgID, privateSpaceID string, privateSpace *UpdatePrivateSpaceRequest) (*PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(privateSpace)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private space data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update private space with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedPrivateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&updatedPrivateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedPrivateSpace, nil
}

// DeletePrivateSpace deletes a private space by ID
func (c *PrivateSpacesClient) DeletePrivateSpace(ctx context.Context, orgID, privateSpaceID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete private space with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
