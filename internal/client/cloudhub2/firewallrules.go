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

// FirewallRulesClient wraps the AnypointClient for firewall rules operations
type FirewallRulesClient struct {
	*client.AnypointClient
}

// NewFirewallRulesClient creates a new FirewallRulesClient
func NewFirewallRulesClient(config *client.ClientConfig) (*FirewallRulesClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &FirewallRulesClient{AnypointClient: anypointClient}, nil
}

// UpdateFirewallRulesRequest represents the request to update firewall rules
type UpdateFirewallRulesRequest struct {
	ManagedFirewallRules []FirewallRule `json:"managedFirewallRules"`
}

// UpdateFirewallRules updates the firewall rules for a private space using PATCH
func (c *FirewallRulesClient) UpdateFirewallRules(ctx context.Context, orgID, privateSpaceID string, rules *UpdateFirewallRulesRequest) (*PrivateSpace, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal firewall rules data: %w", err)
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
		return nil, fmt.Errorf("failed to update firewall rules with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedPrivateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&updatedPrivateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedPrivateSpace, nil
}

// GetFirewallRules retrieves the firewall rules for a private space
func (c *FirewallRulesClient) GetFirewallRules(ctx context.Context, orgID, privateSpaceID string) (*PrivateSpace, error) {
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
		return nil, fmt.Errorf("failed to get firewall rules with status %d: %s", resp.StatusCode, string(body))
	}

	var privateSpace PrivateSpace
	if err := json.NewDecoder(resp.Body).Decode(&privateSpace); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &privateSpace, nil
}
