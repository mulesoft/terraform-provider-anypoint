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

// RoleGroupClient wraps the AnypointClient for role group operations
type RoleGroupClient struct {
	*client.AnypointClient
}

// NewRoleGroupClient creates a new RoleGroupClient
func NewRoleGroupClient(config *client.ClientConfig) (*RoleGroupClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &RoleGroupClient{AnypointClient: anypointClient}, nil
}

// RoleGroup represents an Anypoint role group
type RoleGroup struct {
	RoleGroupID   string   `json:"role_group_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ExternalNames []string `json:"external_names"`
	OrgID         string   `json:"org_id"`
	Editable      bool     `json:"editable"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// ExternalNameRequest represents an external name entry in the request
type ExternalNameRequest struct {
	ExternalGroupName string `json:"external_group_name"`
	ProviderID        string `json:"provider_id"`
}

// CreateRoleGroupRequest represents the request to create a role group
type CreateRoleGroupRequest struct {
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	ExternalNames []ExternalNameRequest `json:"external_names,omitempty"`
}

// CreateRoleGroup creates a new role group in Anypoint
func (c *RoleGroupClient) CreateRoleGroup(ctx context.Context, orgID string, roleGroup *CreateRoleGroupRequest) (*RoleGroup, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups", c.BaseURL, orgID)

	jsonData, err := json.Marshal(roleGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role group data: %w", err)
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
		return nil, fmt.Errorf("failed to create role group with status %d: %s", resp.StatusCode, string(body))
	}

	var createdRoleGroup RoleGroup
	if err := json.NewDecoder(resp.Body).Decode(&createdRoleGroup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdRoleGroup, nil
}

// GetRoleGroup retrieves a role group by ID
func (c *RoleGroupClient) GetRoleGroup(ctx context.Context, orgID, roleGroupID string) (*RoleGroup, error) {
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
	defer resp.Body.Close()

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

// UpdateRoleGroupRequest represents the request to update a role group
type UpdateRoleGroupRequest struct {
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	ExternalNames []ExternalNameRequest `json:"external_names,omitempty"`
}

// UpdateRoleGroup updates an existing role group
func (c *RoleGroupClient) UpdateRoleGroup(ctx context.Context, orgID, roleGroupID string, roleGroup *UpdateRoleGroupRequest) (*RoleGroup, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s", c.BaseURL, orgID, roleGroupID)

	jsonData, err := json.Marshal(roleGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role group data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
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
		return nil, client.NewNotFoundError("role group")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update role group with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedRoleGroup RoleGroup
	if err := json.NewDecoder(resp.Body).Decode(&updatedRoleGroup); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedRoleGroup, nil
}

// DeleteRoleGroup deletes a role group by ID
func (c *RoleGroupClient) DeleteRoleGroup(ctx context.Context, orgID, roleGroupID string) error {
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
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
