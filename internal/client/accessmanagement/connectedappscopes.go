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

// ConnectedAppScopes represents the full scope configuration for a connected app.
// The GET API returns a paginated response with "data" and "total" keys.
type ConnectedAppScopes struct {
	Scopes []Scope `json:"data"`
	Total  int     `json:"total"`
}

// UpdateConnectedAppScopesRequest represents the request to update connected app scopes
type UpdateConnectedAppScopesRequest struct {
	Scopes []Scope `json:"scopes"`
}

// GetConnectedAppScopes retrieves the current scopes for a connected app.
//
// The endpoint is offset-paginated and the accounts API defaults to a SMALL page size (25)
// when limit is omitted (RAML offset-paginated trait: limit default 25, max 500). So this MUST
// paginate: an app with more than one page of scopes would otherwise be silently truncated,
// corrupting the authoritative reconcile (perpetual diff for scopes past the first page, and an
// inability to remove them since they're invisible). Mirrors ListRoleAssignments / ListTeamRoles.
func (c *ConnectedAppScopesClient) GetConnectedAppScopes(ctx context.Context, connectedAppID string) (*ConnectedAppScopes, error) {
	var all []Scope
	total := 0
	limit := 100
	offset := 0

	for {
		url := fmt.Sprintf("%s/accounts/api/connectedApplications/%s/scopes?limit=%d&offset=%d", c.BaseURL, connectedAppID, limit, offset)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, client.NewNotFoundError("connected app scopes")
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var page ConnectedAppScopes
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		all = append(all, page.Scopes...)
		total = page.Total

		if len(page.Scopes) < limit || len(all) >= page.Total {
			break
		}
		offset += limit
	}

	return &ConnectedAppScopes{Scopes: all, Total: total}, nil
}

// ReplaceConnectedAppScopes authoritatively replaces the ENTIRE context-aware scope list for a
// connected app via PUT (verified on devx 2026-07-07: PUT replaces, not merges). Returns 204 and
// then re-GETs the resulting scopes.
//
// Notes verified live:
//   - The platform auto-injects an undeletable "profile" scope on every app; PUT preserves it, so
//     the caller must filter it out of desired/state (it is not user-manageable).
//   - PUT requires a non-empty list (RAML minItems:1). To clear all scopes, use DeleteConnectedAppScopes.
func (c *ConnectedAppScopesClient) ReplaceConnectedAppScopes(ctx context.Context, connectedAppID string, scopes []Scope) (*ConnectedAppScopes, error) {
	url := fmt.Sprintf("%s/accounts/api/connectedApplications/%s/scopes", c.BaseURL, connectedAppID)

	jsonData, err := json.Marshal(&UpdateConnectedAppScopesRequest{Scopes: scopes})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// PUT returns 204 No Content — fetch the resulting scopes.
	return c.GetConnectedAppScopes(ctx, connectedAppID)
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
	defer func() { _ = resp.Body.Close() }()

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

// RemoveConnectedAppScopes removes specific scopes from a connected app via DELETE.
// The API endpoint requires the org ID in the path.
func (c *ConnectedAppScopesClient) RemoveConnectedAppScopes(ctx context.Context, connectedAppID string, scopes []Scope) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/connectedApplications/%s/scopes", c.BaseURL, c.OrgID, connectedAppID)

	request := &UpdateConnectedAppScopesRequest{Scopes: scopes}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteConnectedAppScopes removes all scopes from a connected app.
func (c *ConnectedAppScopesClient) DeleteConnectedAppScopes(ctx context.Context, connectedAppID string) error {
	current, err := c.GetConnectedAppScopes(ctx, connectedAppID)
	if err != nil {
		return err
	}
	if len(current.Scopes) == 0 {
		return nil
	}
	return c.RemoveConnectedAppScopes(ctx, connectedAppID, current.Scopes)
}
