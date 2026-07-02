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

// TeamRolesClient wraps the UserAnypointClient for team role assignment operations.
type TeamRolesClient struct {
	*client.UserAnypointClient
}

// NewTeamRolesClient creates a new TeamRolesClient
func NewTeamRolesClient(config *client.UserClientConfig) (*TeamRolesClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TeamRolesClient{UserAnypointClient: userAnypointClient}, nil
}

// TeamRoleAssignment represents a role assigned to a team
type TeamRoleAssignment struct {
	RoleGroupAssignmentID string            `json:"role_group_assignment_id,omitempty"`
	RoleGroupID           string            `json:"role_group_id,omitempty"`
	RoleID                string            `json:"role_id"`
	Name                  string            `json:"name"`
	Description           string            `json:"description,omitempty"`
	Internal              bool              `json:"internal,omitempty"`
	OrgID                 string            `json:"org_id,omitempty"`
	ContextParams         map[string]string `json:"context_params"`
	CreatedAt             string            `json:"created_at,omitempty"`
}

// AssignTeamRoleRequest is the request body item for assigning a role to a team
type AssignTeamRoleRequest struct {
	RoleID        string            `json:"role_id"`
	ContextParams map[string]string `json:"context_params"`
}

// AssignTeamRoleResponse is a single item in the POST response
type AssignTeamRoleResponse struct {
	RoleGroupID           string            `json:"role_group_id"`
	RoleID                string            `json:"role_id"`
	RoleGroupAssignmentID string            `json:"role_group_assignment_id"`
	ContextParams         map[string]string `json:"context_params"`
}

// ListTeamRolesResponse wraps the GET response
type ListTeamRolesResponse struct {
	Data  []TeamRoleAssignment `json:"data"`
	Total int                  `json:"total"`
}

// AssignTeamRole assigns a role (permission) to a team.
// API: POST /accounts/api/organizations/{orgId}/teams/{teamId}/roles
func (c *TeamRolesClient) AssignTeamRole(ctx context.Context, orgID, teamID string, req *AssignTeamRoleRequest) (*AssignTeamRoleResponse, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/roles", c.BaseURL, orgID, teamID)

	// API expects an array
	body := []AssignTeamRoleRequest{*req}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal assign team role data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	// The team roles endpoint requires Origin header to pass CSRF protection
	httpReq.Header.Set("Origin", c.BaseURL)
	httpReq.Header.Set("Referer", c.BaseURL+"/")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to assign role to team with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle null/empty/204 response — read back to confirm
	trimmed := bytes.TrimSpace(respBody)
	if len(trimmed) == 0 || string(trimmed) == "null" || resp.StatusCode == http.StatusNoContent {
		assignment, err := c.GetTeamRoleAssignment(ctx, orgID, teamID, req.RoleID, req.ContextParams)
		if err != nil {
			return nil, fmt.Errorf("role assigned to team but failed to read back assignment: %w", err)
		}
		return &AssignTeamRoleResponse{
			RoleGroupID:           assignment.RoleGroupID,
			RoleID:                assignment.RoleID,
			RoleGroupAssignmentID: assignment.RoleGroupAssignmentID,
			ContextParams:         assignment.ContextParams,
		}, nil
	}

	var assignments []AssignTeamRoleResponse
	if err := json.Unmarshal(trimmed, &assignments); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(assignments) == 0 {
		return nil, fmt.Errorf("no assignment returned from API")
	}

	return &assignments[0], nil
}

// UnassignTeamRole removes a role (permission) from a team.
// API: DELETE /accounts/api/organizations/{orgId}/teams/{teamId}/roles
func (c *TeamRolesClient) UnassignTeamRole(ctx context.Context, orgID, teamID string, req *AssignTeamRoleRequest) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/roles", c.BaseURL, orgID, teamID)

	// API expects an array
	body := []AssignTeamRoleRequest{*req}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal unassign team role data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	// The team roles endpoint requires Origin header to pass CSRF protection
	httpReq.Header.Set("Origin", c.BaseURL)
	httpReq.Header.Set("Referer", c.BaseURL+"/")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already removed
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unassign role from team with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ListTeamRoles lists all roles assigned to a team.
// API: GET /accounts/api/organizations/{orgId}/teams/{teamId}/roles
func (c *TeamRolesClient) ListTeamRoles(ctx context.Context, orgID, teamID string) ([]TeamRoleAssignment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/roles", c.BaseURL, orgID, teamID)

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
		return nil, client.NewNotFoundError("team")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list team roles with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp ListTeamRolesResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Data, nil
}

// GetTeamRoleAssignment retrieves a specific role assignment by listing all
// and filtering by role_id + context_params.
func (c *TeamRolesClient) GetTeamRoleAssignment(ctx context.Context, orgID, teamID, roleID string, contextParams map[string]string) (*TeamRoleAssignment, error) {
	assignments, err := c.ListTeamRoles(ctx, orgID, teamID)
	if err != nil {
		return nil, err
	}

	for _, a := range assignments {
		if a.RoleID == roleID && teamContextParamsMatch(a.ContextParams, contextParams) {
			return &a, nil
		}
	}

	return nil, client.NewNotFoundError("role assignment on team")
}

// GetTeamRoleAssignmentByID retrieves a specific role assignment by its role_group_assignment_id
func (c *TeamRolesClient) GetTeamRoleAssignmentByID(ctx context.Context, orgID, teamID, assignmentID string) (*TeamRoleAssignment, error) {
	assignments, err := c.ListTeamRoles(ctx, orgID, teamID)
	if err != nil {
		return nil, err
	}

	for _, a := range assignments {
		if a.RoleGroupAssignmentID == assignmentID {
			return &a, nil
		}
	}

	return nil, client.NewNotFoundError("role assignment on team")
}

// teamContextParamsMatch checks if two context_params maps are equal
func teamContextParamsMatch(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
