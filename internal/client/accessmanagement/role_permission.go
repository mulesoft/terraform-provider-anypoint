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

// RolePermissionClient wraps the UserAnypointClient for role permission (role-to-rolegroup assignment) operations.
type RolePermissionClient struct {
	*client.UserAnypointClient
}

// NewRolePermissionClient creates a new RolePermissionClient
func NewRolePermissionClient(config *client.UserClientConfig) (*RolePermissionClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &RolePermissionClient{UserAnypointClient: userAnypointClient}, nil
}

// RoleAssignment represents a single role (permission) assigned to a role group
type RoleAssignment struct {
	RoleGroupAssignmentID string            `json:"role_group_assignment_id"`
	RoleGroupID           string            `json:"role_group_id"`
	RoleID                string            `json:"role_id"`
	OrgID                 string            `json:"org_id"`
	Name                  string            `json:"name"`
	Description           string            `json:"description"`
	Internal              bool              `json:"internal"`
	ContextParams         map[string]string `json:"context_params"`
	CreatedAt             string            `json:"created_at"`
}

// AssignRoleRequest is a single item in the POST/DELETE body array
type AssignRoleRequest struct {
	RoleID        string            `json:"role_id"`
	ContextParams map[string]string `json:"context_params"`
}

// AssignRoleResponse is a single item in the POST response array
type AssignRoleResponse struct {
	RoleGroupID           string            `json:"role_group_id"`
	RoleID                string            `json:"role_id"`
	RoleGroupAssignmentID string            `json:"role_group_assignment_id"`
	ContextParams         map[string]string `json:"context_params"`
}

// ListRoleAssignmentsResponse wraps the GET response
type ListRoleAssignmentsResponse struct {
	Data  []RoleAssignment `json:"data"`
	Total int              `json:"total"`
}

// AssignRole assigns a role (permission) to a role group.
// Returns the assignment ID on success.
func (c *RolePermissionClient) AssignRole(ctx context.Context, orgID, roleGroupID string, req *AssignRoleRequest) (*AssignRoleResponse, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/roles", c.BaseURL, orgID, roleGroupID)

	// API expects an array
	body := []AssignRoleRequest{*req}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal assign role data: %w", err)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to assign role with status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Some roles return null or empty body on success (HTTP 200).
	// In that case, do a read-back to confirm the assignment and get the ID.
	trimmed := bytes.TrimSpace(respBody)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		assignment, err := c.GetRoleAssignment(ctx, orgID, roleGroupID, req.RoleID, req.ContextParams)
		if err != nil {
			return nil, fmt.Errorf("role assigned but failed to read back assignment: %w", err)
		}
		return &AssignRoleResponse{
			RoleGroupID:           assignment.RoleGroupID,
			RoleID:                assignment.RoleID,
			RoleGroupAssignmentID: assignment.RoleGroupAssignmentID,
			ContextParams:         assignment.ContextParams,
		}, nil
	}

	var assignments []AssignRoleResponse
	if err := json.Unmarshal(trimmed, &assignments); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(assignments) == 0 {
		return nil, fmt.Errorf("no assignment returned from API")
	}

	return &assignments[0], nil
}

// UnassignRole removes a role (permission) from a role group.
func (c *RolePermissionClient) UnassignRole(ctx context.Context, orgID, roleGroupID string, req *AssignRoleRequest) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/roles", c.BaseURL, orgID, roleGroupID)

	// API expects an array
	body := []AssignRoleRequest{*req}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal unassign role data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("role assignment")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unassign role with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetRoleAssignment retrieves a specific role assignment by listing all assignments
// for the role group and filtering by role_id and context_params.
func (c *RolePermissionClient) GetRoleAssignment(ctx context.Context, orgID, roleGroupID, roleID string, contextParams map[string]string) (*RoleAssignment, error) {
	assignments, err := c.ListRoleAssignments(ctx, orgID, roleGroupID)
	if err != nil {
		return nil, err
	}

	for _, a := range assignments {
		if a.RoleID == roleID && contextParamsMatch(a.ContextParams, contextParams) {
			return &a, nil
		}
	}

	return nil, client.NewNotFoundError("role assignment")
}

// GetRoleAssignmentByID retrieves a specific role assignment by its role_group_assignment_id
func (c *RolePermissionClient) GetRoleAssignmentByID(ctx context.Context, orgID, roleGroupID, assignmentID string) (*RoleAssignment, error) {
	assignments, err := c.ListRoleAssignments(ctx, orgID, roleGroupID)
	if err != nil {
		return nil, err
	}

	for _, a := range assignments {
		if a.RoleGroupAssignmentID == assignmentID {
			return &a, nil
		}
	}

	return nil, client.NewNotFoundError("role assignment")
}

// ListRoleAssignments lists all role assignments for a role group
func (c *RolePermissionClient) ListRoleAssignments(ctx context.Context, orgID, roleGroupID string) ([]RoleAssignment, error) {
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("role group")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list role assignments with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp ListRoleAssignmentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Data, nil
}

// AvailableRole represents a role (permission) that can be assigned to a role group
type AvailableRole struct {
	RoleID      string `json:"role_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Internal    bool   `json:"internal"`
}

// ListAvailableRolesResponse wraps the GET /accounts/api/roles response
type ListAvailableRolesResponse struct {
	Data  []AvailableRole `json:"data"`
	Total int             `json:"total"`
}

// ListAvailableRoles returns all roles (permissions) available for assignment.
// It paginates through results to fetch the complete catalog.
func (c *RolePermissionClient) ListAvailableRoles(ctx context.Context) ([]AvailableRole, error) {
	var allRoles []AvailableRole
	limit := 100
	offset := 0

	for {
		url := fmt.Sprintf("%s/accounts/api/roles?limit=%d&offset=%d", c.BaseURL, limit, offset)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Token)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to list available roles with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp ListAvailableRolesResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		allRoles = append(allRoles, listResp.Data...)

		// If we got fewer than limit or we've fetched all (total), stop
		if len(listResp.Data) < limit || len(allRoles) >= listResp.Total {
			break
		}
		offset += limit
	}

	return allRoles, nil
}

// contextParamsMatch checks if two context_params maps are equal
func contextParamsMatch(a, b map[string]string) bool {
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
