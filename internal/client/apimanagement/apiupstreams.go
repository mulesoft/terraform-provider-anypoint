package apimanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// APIUpstreamsClient wraps AnypointClient for upstream listing operations.
type APIUpstreamsClient struct {
	*client.AnypointClient
}

// NewAPIUpstreamsClient creates a new APIUpstreamsClient.
func NewAPIUpstreamsClient(config *client.Config) (*APIUpstreamsClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &APIUpstreamsClient{AnypointClient: anypointClient}, nil
}

// APIUpstream represents a single upstream returned by the API Manager upstreams endpoint.
type APIUpstream struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	URI   string `json:"uri"`
}

// apiUpstreamsResponse is the raw JSON envelope from the upstreams endpoint.
type apiUpstreamsResponse struct {
	Total     int           `json:"total"`
	Upstreams []APIUpstream `json:"upstreams"`
}

// ListUpstreams fetches all upstreams for the given API instance.
func (c *APIUpstreamsClient) ListUpstreams(ctx context.Context, orgID, envID, apiID string) ([]APIUpstream, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%s/upstreams",
		c.BaseURL, orgID, envID, apiID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from upstreams API", resp.StatusCode)
	}

	var envelope apiUpstreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return envelope.Upstreams, nil
}
