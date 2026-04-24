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

// TeamRoleAssignment represents a role assignment for a team
type TeamRoleAssignment struct {
	RoleID        string                 `json:"role_id"`
	ContextParams map[string]interface{} `json:"context_params"`
}

// TeamRolesClient wraps the AnypointClient for team roles operations
type TeamRolesClient struct {
	*client.AnypointClient
}

// NewTeamRolesClient creates a new TeamRolesClient
func NewTeamRolesClient(config *client.ClientConfig) (*TeamRolesClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TeamRolesClient{AnypointClient: anypointClient}, nil
}

// AssignRolesToTeam assigns roles to a team
func (c *TeamRolesClient) AssignRolesToTeam(ctx context.Context, orgID, teamID string, roles []TeamRoleAssignment) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/roles", c.AnypointClient.BaseURL, orgID, teamID)

	reqBody, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AnypointClient.Token)

	resp, err := c.AnypointClient.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign roles to team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetTeamRoles retrieves roles assigned to a team
func (c *TeamRolesClient) GetTeamRoles(ctx context.Context, orgID, teamID string) ([]TeamRoleAssignment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/roles", c.AnypointClient.BaseURL, orgID, teamID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AnypointClient.Token)

	resp, err := c.AnypointClient.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []TeamRoleAssignment{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get roles for team with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return []TeamRoleAssignment{}, nil
	}

	// Try to unmarshal as direct array first
	var rolesArray []TeamRoleAssignment
	if err := json.Unmarshal(body, &rolesArray); err == nil {
		return rolesArray, nil
	}

	// Try to unmarshal as single role
	var singleRole TeamRoleAssignment
	if err := json.Unmarshal(body, &singleRole); err == nil && singleRole.RoleID != "" {
		return []TeamRoleAssignment{singleRole}, nil
	}

	// Try to unmarshal as wrapper object with common field names
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response. Response body: %s, Error: %w", string(body), err)
	}

	// Check for common wrapper field names
	for _, fieldName := range []string{"data", "roles", "items", "results"} {
		if field, exists := rawResponse[fieldName]; exists {
			if fieldBytes, err := json.Marshal(field); err == nil {
				var roles []TeamRoleAssignment
				if err := json.Unmarshal(fieldBytes, &roles); err == nil {
					return roles, nil
				}
			}
		}
	}

	// If all else fails, return empty array
	return []TeamRoleAssignment{}, nil
}

// RemoveRolesFromTeam removes roles from a team
func (c *TeamRolesClient) RemoveRolesFromTeam(ctx context.Context, orgID, teamID string, roles []TeamRoleAssignment) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/roles", c.AnypointClient.BaseURL, orgID, teamID)

	reqBody, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AnypointClient.Token)

	resp, err := c.AnypointClient.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Team or roles not found, consider it successful
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove roles from team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
