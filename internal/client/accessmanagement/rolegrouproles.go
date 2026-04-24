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

// RoleGroupRolesClient wraps the AnypointClient for role group roles operations
type RoleGroupRolesClient struct {
	*client.AnypointClient
}

// NewRoleGroupRolesClient creates a new RoleGroupRolesClient
func NewRoleGroupRolesClient(config *client.ClientConfig) (*RoleGroupRolesClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &RoleGroupRolesClient{AnypointClient: anypointClient}, nil
}

// RoleAssignment represents a role assignment for a role group
type RoleAssignment struct {
	RoleID        string                 `json:"role_id"`
	ContextParams map[string]interface{} `json:"context_params"`
}

// AssignRolesToRoleGroup assigns roles to a role group (bulk replacement)
func (c *RoleGroupRolesClient) AssignRolesToRoleGroup(ctx context.Context, orgID, roleGroupID string, roles []RoleAssignment) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/roles", c.BaseURL, orgID, roleGroupID)

	jsonData, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("failed to marshal roles data: %w", err)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign roles to role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetRoleGroupRoles retrieves the roles assigned to a role group
func (c *RoleGroupRolesClient) GetRoleGroupRoles(ctx context.Context, orgID, roleGroupID string) ([]RoleAssignment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/roles", c.BaseURL, orgID, roleGroupID)

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
		// Role group not found or no roles assigned
		return []RoleAssignment{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get roles for role group with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body first for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle empty response
	if len(body) == 0 {
		return []RoleAssignment{}, nil
	}

	// Try to decode as array first (original expected format)
	var rolesArray []RoleAssignment
	if err := json.Unmarshal(body, &rolesArray); err == nil {
		return rolesArray, nil
	}

	// Try to decode as a single role assignment object
	var singleRole RoleAssignment
	if err := json.Unmarshal(body, &singleRole); err == nil && singleRole.RoleID != "" {
		return []RoleAssignment{singleRole}, nil
	}

	// Try to decode as wrapper object with flexible structure
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response. Response body: %s, Error: %w", string(body), err)
	}

	// Try to extract roles from common field names
	for _, fieldName := range []string{"data", "roles", "items", "results"} {
		if field, exists := rawResponse[fieldName]; exists {
			if fieldBytes, err := json.Marshal(field); err == nil {
				var roles []RoleAssignment
				if err := json.Unmarshal(fieldBytes, &roles); err == nil {
					return roles, nil
				}
			}
		}
	}

	// If we can't find roles anywhere, return empty array
	return []RoleAssignment{}, nil
}

// RemoveRolesFromRoleGroup removes roles from a role group
func (c *RoleGroupRolesClient) RemoveRolesFromRoleGroup(ctx context.Context, orgID, roleGroupID string, roles []RoleAssignment) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/roles", c.BaseURL, orgID, roleGroupID)

	jsonData, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("failed to marshal roles data: %w", err)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove roles from role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
