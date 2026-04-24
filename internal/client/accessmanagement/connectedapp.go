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

// ConnectedAppClient wraps the AnypointClient for connected app operations
type ConnectedAppClient struct {
	*client.AnypointClient
}

// NewConnectedAppClient creates a new ConnectedAppClient
func NewConnectedAppClient(config *client.ClientConfig) (*ConnectedAppClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &ConnectedAppClient{AnypointClient: anypointClient}, nil
}

// ConnectedApp represents an Anypoint Connected Application
type ConnectedApp struct {
	ClientID                     string   `json:"client_id"`
	OwnerOrgID                   string   `json:"owner_org_id"`
	ClientName                   string   `json:"client_name"`
	ClientSecret                 string   `json:"client_secret"`
	PublicKeys                   []string `json:"public_keys"`
	RedirectURIs                 []string `json:"redirect_uris"`
	GrantTypes                   []string `json:"grant_types"`
	Scopes                       []string `json:"scopes"`
	Enabled                      bool     `json:"enabled"`
	Audience                     string   `json:"audience"`
	GenerateIssClaimWithoutToken bool     `json:"generate_iss_claim_without_token"`
}

// CreateConnectedAppRequest represents the request to create a connected app
type CreateConnectedAppRequest struct {
	ClientID                     string   `json:"client_id"`
	OwnerOrgID                   string   `json:"owner_org_id"`
	ClientName                   string   `json:"client_name"`
	ClientSecret                 string   `json:"client_secret"`
	PublicKeys                   []string `json:"public_keys,omitempty"`
	RedirectURIs                 []string `json:"redirect_uris,omitempty"`
	GrantTypes                   []string `json:"grant_types"`
	Scopes                       []string `json:"scopes,omitempty"`
	Enabled                      bool     `json:"enabled"`
	Audience                     string   `json:"audience"`
	GenerateIssClaimWithoutToken bool     `json:"generate_iss_claim_without_token"`
}

// CreateConnectedApp creates a new connected application
func (c *ConnectedAppClient) CreateConnectedApp(ctx context.Context, req *CreateConnectedAppRequest) (*ConnectedApp, error) {
	// Marshal the request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/accounts/api/connectedApplications", c.BaseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	// Execute the request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: %s, response: %s", resp.Status, string(respBody))
	}

	// Unmarshal the response
	var connectedApp ConnectedApp
	if err := json.Unmarshal(respBody, &connectedApp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &connectedApp, nil
}

// GetConnectedApp retrieves a connected application by client_id
func (c *ConnectedAppClient) GetConnectedApp(ctx context.Context, clientID string) (*ConnectedApp, error) {
	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/accounts/api/connectedApplications/%s", c.BaseURL, clientID), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	// Execute the request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode == 404 {
		return nil, client.NewNotFoundError("connected app")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: %s, response: %s", resp.Status, string(respBody))
	}

	// Unmarshal the response
	var connectedApp ConnectedApp
	if err := json.Unmarshal(respBody, &connectedApp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &connectedApp, nil
}

// DeleteConnectedApp deletes a connected application by client_id
func (c *ConnectedAppClient) DeleteConnectedApp(ctx context.Context, clientID string) error {
	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/accounts/api/connectedApplications/%s", c.BaseURL, clientID), nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	// Execute the request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s, response: %s", resp.Status, string(respBody))
	}

	return nil
}
