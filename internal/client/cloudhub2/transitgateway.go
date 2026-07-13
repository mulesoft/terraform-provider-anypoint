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

// TransitGatewayClient wraps the AnypointClient for transit gateway operations.
type TransitGatewayClient struct {
	*client.AnypointClient
}

// NewTransitGatewayClient creates a new TransitGatewayClient.
func NewTransitGatewayClient(config *client.Config) (*TransitGatewayClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TransitGatewayClient{AnypointClient: anypointClient}, nil
}

// TransitGateway represents a transit gateway attachment in a Private Space.
type TransitGateway struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Spec   TransitGatewaySpec   `json:"spec"`
	Status TransitGatewayStatus `json:"status"`
}

// TransitGatewaySpec holds the configuration details of a transit gateway.
type TransitGatewaySpec struct {
	ResourceShare ResourceShare `json:"resourceShare"`
	Region        string        `json:"region"`
	SpaceName     string        `json:"spaceName,omitempty"`
}

// ResourceShare represents the AWS RAM resource share details.
type ResourceShare struct {
	ID      string `json:"id"`
	Account string `json:"account"`
}

// TransitGatewayStatus holds the runtime status of a transit gateway.
type TransitGatewayStatus struct {
	Gateway     string   `json:"gateway"`
	Attachment  string   `json:"attachment"`
	TgwResource string   `json:"tgwResource"`
	Routes      []string `json:"routes"`
}

// CreateTransitGatewayRequest represents the request to create a transit gateway attachment.
// The platform discovers the AWS Transit Gateway from the resource share automatically.
type CreateTransitGatewayRequest struct {
	Name                 string   `json:"name"`
	ResourceShareID      string   `json:"resourceShareId"`      // AWS RAM resource share ID (UUID format)
	ResourceShareAccount string   `json:"resourceShareAccount"` // AWS account ID that owns the TGW (12 digits)
	Routes               []string `json:"routes"`               // Initial CIDR routes (at least one required)
}

// UpdateTransitGatewayRequest represents the request to update a transit gateway.
// Only the name field can be updated.
type UpdateTransitGatewayRequest struct {
	Name string `json:"name"`
}

// CreateTransitGateway creates a new transit gateway attachment in a Private Space.
// The API returns 201 with no body, so we list transit gateways afterward to find
// the created resource (matched by the AWS TGW ID).
// API: POST /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways
func (c *TransitGatewayClient) CreateTransitGateway(ctx context.Context, orgID, privateSpaceID string, req *CreateTransitGatewayRequest) (*TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transit gateway request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	// Create returns 201 with no body. List transit gateways to find the new one.
	tgws, err := c.ListTransitGateways(ctx, orgID, privateSpaceID)
	if err != nil {
		return nil, fmt.Errorf("transit gateway created but failed to list: %w", err)
	}

	// Find the TGW by name (ID is assigned by the platform)
	for i := range tgws {
		if tgws[i].Name == req.Name {
			return &tgws[i], nil
		}
	}

	return nil, fmt.Errorf("transit gateway created but could not be found in list")
}

// GetTransitGateway retrieves a specific transit gateway by its ID.
// API: GET /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) GetTransitGateway(ctx context.Context, orgID, privateSpaceID, transitGatewayID string) (*TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

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
		return nil, client.NewNotFoundError("transit gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	var tgw TransitGateway
	if err := json.NewDecoder(resp.Body).Decode(&tgw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tgw, nil
}

// UpdateTransitGateway updates a transit gateway (only name can be changed).
// API: PATCH /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) UpdateTransitGateway(ctx context.Context, orgID, privateSpaceID, transitGatewayID string, req *UpdateTransitGatewayRequest) (*TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transit gateway update request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("transit gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	var tgw TransitGateway
	if err := json.NewDecoder(resp.Body).Decode(&tgw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tgw, nil
}

// DeleteTransitGateway deletes a transit gateway attachment.
// API: DELETE /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) DeleteTransitGateway(ctx context.Context, orgID, privateSpaceID, transitGatewayID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListTransitGateways lists all transit gateways in a Private Space.
// API: GET /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways
func (c *TransitGatewayClient) ListTransitGateways(ctx context.Context, orgID, privateSpaceID string) ([]TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways", c.BaseURL, orgID, privateSpaceID)

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
		return nil, fmt.Errorf("failed to list transit gateways with status %d: %s", resp.StatusCode, string(body))
	}

	var tgws []TransitGateway
	if err := json.NewDecoder(resp.Body).Decode(&tgws); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tgws, nil
}

// GetTransitGatewayRoutes retrieves the routes for a transit gateway.
// API: GET /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}/routes
func (c *TransitGatewayClient) GetTransitGatewayRoutes(ctx context.Context, orgID, privateSpaceID, transitGatewayID string) ([]string, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s/routes", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

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
		return nil, fmt.Errorf("failed to get transit gateway routes with status %d: %s", resp.StatusCode, string(body))
	}

	var routes []string
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return routes, nil
}

// UpdateTransitGatewayRoutes replaces the routes for a transit gateway.
// Routes are managed as a full replacement (PATCH with the complete list).
// API: PATCH /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}/routes
func (c *TransitGatewayClient) UpdateTransitGatewayRoutes(ctx context.Context, orgID, privateSpaceID, transitGatewayID string, routes []string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s/routes", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

	jsonData, err := json.Marshal(routes)
	if err != nil {
		return fmt.Errorf("failed to marshal routes request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update transit gateway routes with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
