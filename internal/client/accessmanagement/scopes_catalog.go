package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// ScopesCatalogClient wraps the AnypointClient for scopes catalog operations.
type ScopesCatalogClient struct {
	*client.AnypointClient
}

// NewScopesCatalogClient creates a new ScopesCatalogClient.
func NewScopesCatalogClient(config *client.Config) (*ScopesCatalogClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &ScopesCatalogClient{AnypointClient: anypointClient}, nil
}

// ScopeCatalogEntry represents a single scope in the platform catalog.
type ScopeCatalogEntry struct {
	Scope        string `json:"scope"`
	DisplayName  string `json:"displayName"`
	Description  string `json:"description"`
	ProductLabel string `json:"productLabel"`
	Internal     bool   `json:"internal"`
}

// ListScopesCatalog retrieves all available scopes from the platform catalog.
// API: GET /accounts/api/cs/scopes
func (c *ScopesCatalogClient) ListScopesCatalog(ctx context.Context) ([]ScopeCatalogEntry, error) {
	url := fmt.Sprintf("%s/accounts/api/cs/scopes", c.BaseURL)

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list scopes catalog with status %d: %s", resp.StatusCode, string(body))
	}

	var entries []ScopeCatalogEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return entries, nil
}
