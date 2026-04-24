package accessmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// ConnectedAppScopesClient wraps the UserAnypointClient for connected app scope operations
type ConnectedAppScopesClient struct {
	*client.UserAnypointClient
}

// NewConnectedAppScopesClient creates a new ConnectedAppScopesClient
func NewConnectedAppScopesClient(config *client.UserClientConfig) (*ConnectedAppScopesClient, error) {
	userClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &ConnectedAppScopesClient{UserAnypointClient: userClient}, nil
}

// Scope represents a connected app scope
type Scope struct {
	Scope         string                 `json:"scope"`
	ContextParams map[string]interface{} `json:"context_params,omitempty"`
}

// ConnectedAppScopes represents the full scope configuration for a connected app
type ConnectedAppScopes struct {
	Scopes []Scope `json:"scopes"`
}

// UpdateConnectedAppScopesRequest represents the request to update connected app scopes
type UpdateConnectedAppScopesRequest struct {
	Scopes []Scope `json:"scopes"`
}

// GetConnectedAppScopes retrieves the current scopes for a connected app
func (c *ConnectedAppScopesClient) GetConnectedAppScopes(ctx context.Context, connectedAppID string) (*ConnectedAppScopes, error) {
	url := fmt.Sprintf("%s/accounts/api/connectedApplications/%s/scopes", c.BaseURL, connectedAppID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var scopes ConnectedAppScopes
	if err := json.Unmarshal(body, &scopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &scopes, nil
}

// UpdateConnectedAppScopes updates the scopes for a connected app
func (c *ConnectedAppScopesClient) UpdateConnectedAppScopes(ctx context.Context, connectedAppID string, request *UpdateConnectedAppScopesRequest) (*ConnectedAppScopes, error) {
	url := fmt.Sprintf("%s/accounts/api/connectedApplications/%s/scopes", c.BaseURL, connectedAppID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For 204 No Content, we need to fetch the updated scopes separately
	return c.GetConnectedAppScopes(ctx, connectedAppID)
}

// DeleteConnectedAppScopes removes all scopes from a connected app (sets empty scopes list)
func (c *ConnectedAppScopesClient) DeleteConnectedAppScopes(ctx context.Context, connectedAppID string) error {
	request := &UpdateConnectedAppScopesRequest{
		Scopes: []Scope{}, // Empty scopes list
	}

	_, err := c.UpdateConnectedAppScopes(ctx, connectedAppID, request)
	return err
}
