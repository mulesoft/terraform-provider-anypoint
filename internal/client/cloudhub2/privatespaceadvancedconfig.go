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

// PrivateSpaceAdvancedConfigClient wraps the AnypointClient for private space advanced config operations
type PrivateSpaceAdvancedConfigClient struct {
	*client.AnypointClient
}

// NewPrivateSpaceAdvancedConfigClient creates a new PrivateSpaceAdvancedConfigClient
func NewPrivateSpaceAdvancedConfigClient(config *client.Config) (*PrivateSpaceAdvancedConfigClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &PrivateSpaceAdvancedConfigClient{AnypointClient: anypointClient}, nil
}

// PrivateSpaceAdvancedConfig represents the advanced configuration for a private space
type PrivateSpaceAdvancedConfig struct {
	IngressConfiguration IngressConfiguration `json:"ingressConfiguration"`
	EnableIAMRole        bool                 `json:"enableIAMRole"`
}

// IngressConfiguration represents the ingress configuration
type IngressConfiguration struct {
	ReadResponseTimeout string            `json:"readResponseTimeout"`
	Logs                IngressLogs       `json:"logs"`
	Protocol            string            `json:"protocol"`
	Deployment          IngressDeployment `json:"deployment"`
}

// IngressLogs represents the ingress logs configuration
type IngressLogs struct {
	Filters      []IngressLogFilter `json:"filters"`
	PortLogLevel string             `json:"portLogLevel"`
}

// IngressLogFilter represents a log filter for ingress
type IngressLogFilter struct {
	IP    string `json:"ip"`
	Level string `json:"level"`
}

// IngressDeployment represents the deployment configuration
type IngressDeployment struct {
	Status            string `json:"status"`
	LastSeenTimestamp int64  `json:"lastSeenTimestamp"`
}

// UpdatePrivateSpaceAdvancedConfigRequest represents the request to update advanced config
type UpdatePrivateSpaceAdvancedConfigRequest struct {
	IngressConfiguration IngressConfiguration `json:"ingressConfiguration"`
	EnableIAMRole        bool                 `json:"enableIAMRole"`
}

// UpdatePrivateSpaceAdvancedConfig updates the advanced configuration of a private space
func (c *PrivateSpaceAdvancedConfigClient) UpdatePrivateSpaceAdvancedConfig(ctx context.Context, orgID, privateSpaceID string, request *UpdatePrivateSpaceAdvancedConfigRequest) (*PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update private space advanced config with status %d: %s", resp.StatusCode, string(body))
	}

	var privateSpace PrivateSpace
	if err := json.Unmarshal(body, &privateSpace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &privateSpace, nil
}

// GetPrivateSpace gets a private space (reused from privatespaces client pattern)
func (c *PrivateSpaceAdvancedConfigClient) GetPrivateSpace(ctx context.Context, orgID, privateSpaceID string) (*PrivateSpace, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get private space with status %d: %s", resp.StatusCode, string(body))
	}

	var privateSpace PrivateSpace
	if err := json.Unmarshal(body, &privateSpace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &privateSpace, nil
}
