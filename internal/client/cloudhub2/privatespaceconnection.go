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

// PrivateSpaceConnectionClient wraps the AnypointClient for private space connection operations
type PrivateSpaceConnectionClient struct {
	*client.AnypointClient
}

// NewPrivateSpaceConnectionClient creates a new PrivateSpaceConnectionClient
func NewPrivateSpaceConnectionClient(config *client.Config) (*PrivateSpaceConnectionClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &PrivateSpaceConnectionClient{AnypointClient: anypointClient}, nil
}

// PrivateSpaceConnection represents an Anypoint Private Space Connection
type PrivateSpaceConnection struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	Status           string  `json:"status"`
	PrivateSpaceID   string  `json:"privateSpaceId"`
	OrganizationID   string  `json:"organizationId"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
	VPNs             []VPN   `json:"vpns"`
	Region           string  `json:"region"`
	CloudGatewayRef  *string `json:"cloudGatewayRef,omitempty"`
	DirectConnectRef *string `json:"directConnectRef,omitempty"`
}

// VPN represents a VPN configuration
type VPN struct {
	ID                   string      `json:"id,omitempty"`
	Name                 string      `json:"name"`
	RemoteAsn            int         `json:"remoteAsn"`
	RemoteIPAddress      string      `json:"remoteIpAddress"`
	SpecificationVersion string      `json:"specificationVersion"`
	VPNTunnels           []VPNTunnel `json:"vpnTunnels"`
}

// VPNTunnel represents a VPN tunnel configuration
type VPNTunnel struct {
	PreSharedKey            string   `json:"preSharedKey"`
	LocalExternalIPAddress  string   `json:"localExternalIpAddress"`
	LocalInternalIPAddress  string   `json:"localInternalIpAddress"`
	LocalBGPASN             string   `json:"localBgpAsn"`
	RemoteInternalIPAddress string   `json:"remoteInternalIpAddress"`
	RemoteBGPASN            string   `json:"remoteBgpAsn"`
	LocalPointToPointCIDR   string   `json:"localPointToPointCidr"`
	RemotePointToPointCIDR  string   `json:"remotePointToPointCidr"`
	LocalStaticRoutes       []string `json:"localStaticRoutes,omitempty"`
	RemoteStaticRoutes      []string `json:"remoteStaticRoutes,omitempty"`
	StatusCode              int      `json:"statusCode,omitempty"`
	StatusMessage           string   `json:"statusMessage,omitempty"`
}

// CreatePrivateSpaceConnectionRequest represents the request to create a private space connection
type CreatePrivateSpaceConnectionRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
	VPNs []VPN  `json:"vpns"`
}

// UpdatePrivateSpaceConnectionRequest represents the request to update a private space connection
type UpdatePrivateSpaceConnectionRequest struct {
	Name *string `json:"name,omitempty"`
	VPNs []VPN   `json:"vpns,omitempty"`
}

// CreatePrivateSpaceConnection creates a new private space connection in Anypoint
func (c *PrivateSpaceConnectionClient) CreatePrivateSpaceConnection(ctx context.Context, orgID, privateSpaceID string, connection *CreatePrivateSpaceConnectionRequest) (*PrivateSpaceConnection, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection data: %w", err)
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
		return nil, fmt.Errorf("failed to create private space connection with status %d: %s", resp.StatusCode, string(body))
	}

	var createdConnection PrivateSpaceConnection
	if err := json.NewDecoder(resp.Body).Decode(&createdConnection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdConnection, nil
}

// GetPrivateSpaceConnection retrieves a private space connection by ID
func (c *PrivateSpaceConnectionClient) GetPrivateSpaceConnection(ctx context.Context, orgID, privateSpaceID, connectionID string) (*PrivateSpaceConnection, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", c.BaseURL, orgID, privateSpaceID, connectionID)

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
		return nil, client.NewNotFoundError("private space connection")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get private space connection with status %d: %s", resp.StatusCode, string(body))
	}

	var connection PrivateSpaceConnection
	if err := json.NewDecoder(resp.Body).Decode(&connection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &connection, nil
}

// UpdatePrivateSpaceConnection updates an existing private space connection
func (c *PrivateSpaceConnectionClient) UpdatePrivateSpaceConnection(ctx context.Context, orgID, privateSpaceID, connectionID string, connection *UpdatePrivateSpaceConnectionRequest) (*PrivateSpaceConnection, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", c.BaseURL, orgID, privateSpaceID, connectionID)

	jsonData, err := json.Marshal(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connection data: %w", err)
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
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("private space connection")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update private space connection with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedConnection PrivateSpaceConnection
	if err := json.NewDecoder(resp.Body).Decode(&updatedConnection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedConnection, nil
}

// DeletePrivateSpaceConnection deletes a private space connection by ID
func (c *PrivateSpaceConnectionClient) DeletePrivateSpaceConnection(ctx context.Context, orgID, privateSpaceID, connectionID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", c.BaseURL, orgID, privateSpaceID, connectionID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete private space connection with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
