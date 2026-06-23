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

// RoleClient wraps the UserAnypointClient for role group operations.
// The Anypoint Role Groups API requires user-level (password grant) authentication.
type RoleClient struct {
	*client.UserAnypointClient
}

// NewRoleClient creates a new RoleClient
func NewRoleClient(config *client.UserClientConfig) (*RoleClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &RoleClient{UserAnypointClient: userAnypointClient}, nil
}

// RoleGroup represents an Anypoint role group
type RoleGroup struct {
	ID            string   `json:"role_group_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	OrgID         string   `json:"org_id"`
	Editable      bool     `json:"editable"`
	ExternalNames []string `json:"external_names"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// CreateRoleGroupRequest represents the request to create a role group
type CreateRoleGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateRoleGroupRequest represents the request to update a role group.
// NOTE: Do NOT include external_names — the API returns 400 "Invalid external_names format".
type UpdateRoleGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ListRoleGroupsResponse wraps the API list response
type ListRoleGroupsResponse struct {
	Data []RoleGroup `json:"data"`
}

// CreateRoleGroup creates a new role group in Anypoint
func (c *RoleClient) CreateRoleGroup(ctx context.Context, orgID string, req *CreateRoleGroupRequest) (*RoleGroup, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups", c.BaseURL, orgID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role group data: %w", err)
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
		return nil, fmt.Errorf("failed to create role group with status %d: %s", resp.StatusCode, string(body))
	}

	var roleGroup RoleGroup
	if err := json.NewDecoder(resp.Body).Decode(&roleGroup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &roleGroup, nil
}

// GetRoleGroup retrieves a role group by ID
func (c *RoleClient) GetRoleGroup(ctx context.Context, orgID, roleGroupID string) (*RoleGroup, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s", c.BaseURL, orgID, roleGroupID)

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
		return nil, client.NewNotFoundError("role group")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get role group with status %d: %s", resp.StatusCode, string(body))
	}

	var roleGroup RoleGroup
	if err := json.NewDecoder(resp.Body).Decode(&roleGroup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &roleGroup, nil
}

// UpdateRoleGroup updates an existing role group using PUT
func (c *RoleClient) UpdateRoleGroup(ctx context.Context, orgID, roleGroupID string, req *UpdateRoleGroupRequest) (*RoleGroup, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s", c.BaseURL, orgID, roleGroupID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role group data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
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
		return nil, client.NewNotFoundError("role group")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update role group with status %d: %s", resp.StatusCode, string(body))
	}

	var roleGroup RoleGroup
	if err := json.NewDecoder(resp.Body).Decode(&roleGroup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &roleGroup, nil
}

// DeleteRoleGroup deletes a role group by ID
func (c *RoleClient) DeleteRoleGroup(ctx context.Context, orgID, roleGroupID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s", c.BaseURL, orgID, roleGroupID)

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
		return client.NewNotFoundError("role group")
	}

	// DELETE returns 200 with an array of org IDs (not 204 like other resources)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListRoleGroups lists all role groups for an organization
func (c *RoleClient) ListRoleGroups(ctx context.Context, orgID string) ([]RoleGroup, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups", c.BaseURL, orgID)

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
		return nil, fmt.Errorf("failed to list role groups with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp ListRoleGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Data, nil
}
