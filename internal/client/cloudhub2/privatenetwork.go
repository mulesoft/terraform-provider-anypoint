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

// PrivateNetworkClient wraps the AnypointClient for private network operations
type PrivateNetworkClient struct {
	*client.AnypointClient
}

// NewPrivateNetworkClient creates a new PrivateNetworkClient
func NewPrivateNetworkClient(config *client.ClientConfig) (*PrivateNetworkClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &PrivateNetworkClient{AnypointClient: anypointClient}, nil
}

// NetworkConfiguration represents the network configuration for a private space
type NetworkConfiguration struct {
	Region        string   `json:"region"`
	CidrBlock     string   `json:"cidrBlock"`
	ReservedCIDRs []string `json:"reservedCidrs"`
}

// CreatePrivateNetworkRequest represents the request to create a private network
type CreatePrivateNetworkRequest struct {
	Network NetworkConfiguration `json:"network"`
}

// UpdatePrivateNetworkRequest represents the request to update a private network
type UpdatePrivateNetworkRequest struct {
	Network NetworkConfiguration `json:"network"`
}

// CreatePrivateNetwork creates a network configuration for a private space
func (c *PrivateNetworkClient) CreatePrivateNetwork(ctx context.Context, orgID, privateSpaceID string, networkConfig *CreatePrivateNetworkRequest) (*PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(networkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network configuration: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create private network with status %d: %s", resp.StatusCode, string(body))
	}

	var privateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&privateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &privateSpace, nil
}

// UpdatePrivateNetwork updates the network configuration for a private space
func (c *PrivateNetworkClient) UpdatePrivateNetwork(ctx context.Context, orgID, privateSpaceID string, networkConfig *UpdatePrivateNetworkRequest) (*PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(networkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network configuration: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update private network with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedPrivateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&updatedPrivateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedPrivateSpace, nil
}

// GetPrivateNetwork retrieves the network configuration for a private space
func (c *PrivateNetworkClient) GetPrivateNetwork(ctx context.Context, orgID, privateSpaceID string) (*PrivateSpace, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get private network with status %d: %s", resp.StatusCode, string(body))
	}

	var privateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&privateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &privateSpace, nil
}
