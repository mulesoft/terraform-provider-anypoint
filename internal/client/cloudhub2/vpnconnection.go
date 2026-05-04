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

// VPNConnectionClient wraps the AnypointClient for VPN connection operations
type VPNConnectionClient struct {
	*client.AnypointClient
}

// NewVPNConnectionClient creates a new VPNConnectionClient
func NewVPNConnectionClient(config *client.Config) (*VPNConnectionClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &VPNConnectionClient{AnypointClient: anypointClient}, nil
}

// CreateVPNConnectionRequest represents the request to create a VPN connection
type CreateVPNConnectionRequest struct {
	Name string       `json:"name"`
	VPNs []VPNRequest `json:"vpns"`
}

// VPNRequest represents a VPN configuration in the request
type VPNRequest struct {
	Name            string                       `json:"name,omitempty"`
	LocalASN        string                       `json:"localAsn"`
	RemoteASN       string                       `json:"remoteAsn,omitempty"`
	RemoteIPAddress string                       `json:"remoteIpAddress"`
	StaticRoutes    []string                     `json:"staticRoutes,omitempty"`
	VPNTunnels      []VPNConnectionTunnelRequest `json:"vpnTunnels"`
}

// VPNConnectionTunnelRequest represents a VPN tunnel in the request
type VPNConnectionTunnelRequest struct {
	PSK           string `json:"psk"`
	PTPCidr       string `json:"ptpCidr"`
	StartupAction string `json:"startupAction"`
}

// VPNConnection is the response for a VPN connection
type VPNConnection struct {
	ID   string        `json:"id"`
	Name string        `json:"name"`
	VPNs []VPNResponse `json:"vpns"`
}

// VPNResponse represents a VPN configuration in the response
type VPNResponse struct {
	Name                string                        `json:"name"`
	VPNID               string                        `json:"vpnId"`
	ConnectionID        string                        `json:"connectionId"`
	ConnectionName      string                        `json:"connectionName"`
	VPNConnectionStatus string                        `json:"vpnConnectionStatus"`
	RemoteASN           int                           `json:"remoteAsn"`
	LocalASN            int                           `json:"localAsn"`
	RemoteIPAddress     string                        `json:"remoteIpAddress"`
	VPNTunnels          []VPNConnectionTunnelResponse `json:"vpnTunnels"`
}

// VPNConnectionTunnelResponse represents a VPN tunnel in the response
type VPNConnectionTunnelResponse struct {
	PSK           string `json:"psk"`
	PTPCidr       string `json:"ptpCidr"`
	StartupAction string `json:"startupAction"`
	IsLogsEnabled bool   `json:"isLogsEnabled"`
}

// CreateVPNConnection creates a new VPN connection
func (c *VPNConnectionClient) CreateVPNConnection(ctx context.Context, orgID, privateSpaceID string, request *CreateVPNConnectionRequest) (*VPNConnection, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VPN connection request: %w", err)
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
		return nil, fmt.Errorf("failed to create VPN connection with status %d: %s", resp.StatusCode, string(body))
	}

	var vpnConnection VPNConnection
	if err := json.NewDecoder(resp.Body).Decode(&vpnConnection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &vpnConnection, nil
}

// GetVPNConnection retrieves a specific VPN connection by its ID.
func (c *VPNConnectionClient) GetVPNConnection(ctx context.Context, orgID, privateSpaceID, connectionID string) (*VPNConnection, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", c.BaseURL, orgID, privateSpaceID, connectionID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get vpn connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("VPN connection")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get VPN connection with status %d: %s", resp.StatusCode, string(body))
	}

	var vpnConnection VPNConnection
	if err := json.NewDecoder(resp.Body).Decode(&vpnConnection); err != nil {
		return nil, fmt.Errorf("failed to decode get vpn connection response: %w", err)
	}

	return &vpnConnection, nil
}

// DeleteVPNConnection deletes an entire VPN connection by its ID.
func (c *VPNConnectionClient) DeleteVPNConnection(ctx context.Context, orgID, privateSpaceID, connectionID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", c.BaseURL, orgID, privateSpaceID, connectionID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete vpn connection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute delete vpn connection request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted is not an error
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete VPN connection with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteVPN deletes a specific VPN from a connection.
func (c *VPNConnectionClient) DeleteVPN(ctx context.Context, orgID, privateSpaceID, connectionID, vpnID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s/vpns/%s", c.BaseURL, orgID, privateSpaceID, connectionID, vpnID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete vpn request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute delete vpn request: %w", err)
	}
	defer resp.Body.Close()

	// A 204 No Content is a successful deletion.
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete VPN with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
