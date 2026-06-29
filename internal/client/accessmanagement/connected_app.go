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

// ConnectedAppClient wraps the UserAnypointClient for connected app operations.
type ConnectedAppClient struct {
	*client.UserAnypointClient
}

// NewConnectedAppClient creates a new ConnectedAppClient
func NewConnectedAppClient(config *client.UserClientConfig) (*ConnectedAppClient, error) {
	userClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &ConnectedAppClient{UserAnypointClient: userClient}, nil
}

// ConnectedApp represents a connected application in Anypoint Platform.
type ConnectedApp struct {
	ClientID                   string   `json:"client_id"`
	ClientSecret               string   `json:"client_secret,omitempty"`
	ClientName                 string   `json:"client_name"`
	OrgID                      string   `json:"org_id"`
	OwnerOrgID                 string   `json:"owner_org_id"`
	OwnerUserID                string   `json:"owner_user_id"`
	GrantTypes                 []string `json:"grant_types"`
	RedirectURIs               []string `json:"redirect_uris"`
	PublicKeys                 []string `json:"public_keys"`
	Audience                   string   `json:"audience"`
	ClientURI                  string   `json:"client_uri"`
	Enabled                    bool     `json:"enabled"`
	PolicyURI                  *string  `json:"policy_uri"`
	TosURI                     *string  `json:"tos_uri"`
	CertExpiry                 *string  `json:"cert_expiry"`
	GenerateIssClaimWithoutToken bool   `json:"generate_iss_claim_without_token"`
	IDProviderID               *string  `json:"idprovider_id"`
	IsDynamic                  bool     `json:"is_dynamic"`
	IPAllowlistExcluded        bool     `json:"ip_allowlist_excluded"`
	PKCEEnforcement            string   `json:"pkce_enforcement"`
	SkipUserConsent            bool     `json:"skip_user_consent"`
	CreatedAt                  string   `json:"created_at"`
	UpdatedAt                  string   `json:"updated_at"`
	ClientSecretUpdatedAt      string   `json:"client_secret_updated_at"`
}

// CreateConnectedAppRequest represents the request body to create a connected app.
// Note: redirect_uris MUST always be present (even as []) — the API's RAML validation rejects requests without it.
type CreateConnectedAppRequest struct {
	ClientName   string   `json:"client_name"`
	GrantTypes   []string `json:"grant_types"`
	RedirectURIs []string `json:"redirect_uris"`
	PublicKeys   []string `json:"public_keys"`
	Audience     string   `json:"audience,omitempty"`
	ClientURI    string   `json:"client_uri,omitempty"`
	Scopes       []Scope  `json:"scopes,omitempty"`
}

// UpdateConnectedAppRequest represents the request body to update a connected app.
type UpdateConnectedAppRequest struct {
	ClientName   *string  `json:"client_name,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
	RedirectURIs []string `json:"redirect_uris,omitempty"`
	PublicKeys   []string `json:"public_keys,omitempty"`
	Audience     *string  `json:"audience,omitempty"`
	ClientURI    *string  `json:"client_uri,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// ListConnectedAppsResponse wraps the GET list response
type ListConnectedAppsResponse struct {
	Data  []ConnectedApp `json:"data"`
	Total int            `json:"total"`
}

// CreateConnectedApp creates a new connected app.
// API: POST /accounts/api/organizations/{orgId}/connectedApplications
// Returns 201 Created with the app object (including client_secret).
func (c *ConnectedAppClient) CreateConnectedApp(ctx context.Context, orgID string, req *CreateConnectedAppRequest) (*ConnectedApp, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/connectedApplications", c.BaseURL, orgID)

	// Ensure arrays are never nil (API requires them present, even empty)
	if req.RedirectURIs == nil {
		req.RedirectURIs = []string{}
	}
	if req.PublicKeys == nil {
		req.PublicKeys = []string{}
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create connected app with status %d: %s", resp.StatusCode, string(body))
	}

	var app ConnectedApp
	if err := json.Unmarshal(body, &app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// GetConnectedApp retrieves a single connected app by ID.
// API: GET /accounts/api/organizations/{orgId}/connectedApplications/{clientId}
func (c *ConnectedAppClient) GetConnectedApp(ctx context.Context, orgID, clientID string) (*ConnectedApp, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/connectedApplications/%s", c.BaseURL, orgID, clientID)

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
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("connected app")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get connected app with status %d: %s", resp.StatusCode, string(body))
	}

	var app ConnectedApp
	if err := json.Unmarshal(body, &app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// UpdateConnectedApp updates a connected app.
// API: PATCH /accounts/api/organizations/{orgId}/connectedApplications/{clientId}
// Returns 200 OK with the updated app object.
func (c *ConnectedAppClient) UpdateConnectedApp(ctx context.Context, orgID, clientID string, req *UpdateConnectedAppRequest) (*ConnectedApp, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/connectedApplications/%s", c.BaseURL, orgID, clientID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("connected app")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update connected app with status %d: %s", resp.StatusCode, string(body))
	}

	var app ConnectedApp
	if err := json.Unmarshal(body, &app); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &app, nil
}

// DeleteConnectedApp deletes a connected app.
// API: DELETE /accounts/api/organizations/{orgId}/connectedApplications/{clientId}
func (c *ConnectedAppClient) DeleteConnectedApp(ctx context.Context, orgID, clientID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/connectedApplications/%s", c.BaseURL, orgID, clientID)

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

	// Already deleted — treat as success
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete connected app with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListConnectedApps lists all connected apps in an organization.
// API: GET /accounts/api/organizations/{orgId}/connectedApplications?limit=100&offset=0&includeUsage=true
func (c *ConnectedAppClient) ListConnectedApps(ctx context.Context, orgID string) ([]ConnectedApp, error) {
	var allApps []ConnectedApp
	limit := 100
	offset := 0

	for {
		url := fmt.Sprintf("%s/accounts/api/organizations/%s/connectedApplications?limit=%d&offset=%d&includeUsage=true",
			c.BaseURL, orgID, limit, offset)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Token)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to list connected apps with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp ListConnectedAppsResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		allApps = append(allApps, listResp.Data...)

		if len(listResp.Data) < limit || len(allApps) >= listResp.Total {
			break
		}
		offset += limit
	}

	return allApps, nil
}
