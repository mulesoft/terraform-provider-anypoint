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

// RoleGroupUsersClient wraps the AnypointClient for role group users operations
type RoleGroupUsersClient struct {
	*client.AnypointClient
}

// NewRoleGroupUsersClient creates a new RoleGroupUsersClient
func NewRoleGroupUsersClient(config *client.ClientConfig) (*RoleGroupUsersClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &RoleGroupUsersClient{AnypointClient: anypointClient}, nil
}

// UserAssignment represents a user assigned to a role group
type UserAssignment struct {
	ID             string                 `json:"id"`
	Username       string                 `json:"username"`
	FirstName      string                 `json:"firstName"`
	LastName       string                 `json:"lastName"`
	Email          string                 `json:"email"`
	OrganizationID string                 `json:"organizationId"`
	Enabled        bool                   `json:"enabled"`
	IDProviderID   string                 `json:"idprovider_id"`
	ContextParams  map[string]interface{} `json:"context_params"`
}

// RoleGroupUsersResponse represents the API response for getting role group users
type RoleGroupUsersResponse struct {
	Data  []UserAssignment `json:"data"`
	Total int              `json:"total"`
}

// AssignUsersToRoleGroup assigns users to a role group (bulk replacement)
func (c *RoleGroupUsersClient) AssignUsersToRoleGroup(ctx context.Context, orgID, roleGroupID string, userIDs []string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/users", c.BaseURL, orgID, roleGroupID)

	jsonData, err := json.Marshal(userIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal user IDs: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign users to role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetRoleGroupUsers retrieves the users assigned to a role group
func (c *RoleGroupUsersClient) GetRoleGroupUsers(ctx context.Context, orgID, roleGroupID string) ([]UserAssignment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/users", c.BaseURL, orgID, roleGroupID)

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
		// Role group not found or no users assigned
		return []UserAssignment{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get users for role group with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle empty response
	if len(body) == 0 {
		return []UserAssignment{}, nil
	}

	var response RoleGroupUsersResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response. Response body: %s, Error: %w", string(body), err)
	}

	return response.Data, nil
}

// RemoveUsersFromRoleGroup removes users from a role group
func (c *RoleGroupUsersClient) RemoveUsersFromRoleGroup(ctx context.Context, orgID, roleGroupID string, userIDs []string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/users", c.BaseURL, orgID, roleGroupID)

	jsonData, err := json.Marshal(userIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal user IDs: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already removed or role group doesn't exist
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove users from role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
