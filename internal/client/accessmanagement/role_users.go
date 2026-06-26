package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// RoleUsersClient wraps the UserAnypointClient for role group user membership operations.
type RoleUsersClient struct {
	*client.UserAnypointClient
}

// NewRoleUsersClient creates a new RoleUsersClient
func NewRoleUsersClient(config *client.UserClientConfig) (*RoleUsersClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &RoleUsersClient{UserAnypointClient: userAnypointClient}, nil
}

// RoleGroupUser represents a user assigned to a role group
type RoleGroupUser struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Email          string `json:"email"`
	OrganizationID string `json:"organizationId"`
	Enabled        bool   `json:"enabled"`
	Type           string `json:"type"`
}

// ListRoleGroupUsersResponse wraps the GET response
type ListRoleGroupUsersResponse struct {
	Data  []RoleGroupUser `json:"data"`
	Total int             `json:"total"`
}

// OrgUser represents a user in the organization
type OrgUser struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Email          string `json:"email"`
	OrganizationID string `json:"organizationId"`
	Enabled        bool   `json:"enabled"`
	Type           string `json:"type"`
	CreatedAt      string `json:"createdAt"`
}

// ListOrgUsersResponse wraps the GET /organizations/{orgId}/users response
type ListOrgUsersResponse struct {
	Data  []OrgUser `json:"data"`
	Total int       `json:"total"`
}

// AddUserToRoleGroup adds a user to a role group.
// API: POST /accounts/api/organizations/{orgId}/rolegroups/{rgId}/users/{userId}
// Returns null body on HTTP 200 success.
func (c *RoleUsersClient) AddUserToRoleGroup(ctx context.Context, orgID, roleGroupID, userID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/users/%s", c.BaseURL, orgID, roleGroupID, userID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
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
		return client.NewNotFoundError("role group or user")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add user to role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveUserFromRoleGroup removes a user from a role group.
// API: DELETE /accounts/api/organizations/{orgId}/rolegroups/{rgId}/users/{userId}
// Returns null body on HTTP 200 success.
// Returns HTTP 409 if user is not in the role group.
func (c *RoleUsersClient) RemoveUserFromRoleGroup(ctx context.Context, orgID, roleGroupID, userID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/users/%s", c.BaseURL, orgID, roleGroupID, userID)

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

	// 409 means user not in group — treat as already removed (idempotent)
	if resp.StatusCode == http.StatusConflict {
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove user from role group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListRoleGroupUsers lists all users assigned to a role group.
// API: GET /accounts/api/organizations/{orgId}/rolegroups/{rgId}/users
func (c *RoleUsersClient) ListRoleGroupUsers(ctx context.Context, orgID, roleGroupID string) ([]RoleGroupUser, error) {
	var allUsers []RoleGroupUser
	limit := 100
	offset := 0

	for {
		url := fmt.Sprintf("%s/accounts/api/organizations/%s/rolegroups/%s/users?limit=%d&offset=%d", c.BaseURL, orgID, roleGroupID, limit, offset)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Token)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()
			return nil, client.NewNotFoundError("role group")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to list role group users with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp ListRoleGroupUsersResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		allUsers = append(allUsers, listResp.Data...)

		if len(listResp.Data) < limit || len(allUsers) >= listResp.Total {
			break
		}
		offset += limit
	}

	return allUsers, nil
}

// GetRoleGroupUser checks if a specific user is in a role group.
// Returns the user details if found, NotFoundError if not.
func (c *RoleUsersClient) GetRoleGroupUser(ctx context.Context, orgID, roleGroupID, userID string) (*RoleGroupUser, error) {
	users, err := c.ListRoleGroupUsers(ctx, orgID, roleGroupID)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.ID == userID {
			return &u, nil
		}
	}

	return nil, client.NewNotFoundError("user in role group")
}

// ListOrgUsers lists all users in the organization.
// API: GET /accounts/api/organizations/{orgId}/users
func (c *RoleUsersClient) ListOrgUsers(ctx context.Context, orgID string) ([]OrgUser, error) {
	var allUsers []OrgUser
	limit := 100
	offset := 0

	for {
		url := fmt.Sprintf("%s/accounts/api/organizations/%s/users?limit=%d&offset=%d", c.BaseURL, orgID, limit, offset)

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
			return nil, fmt.Errorf("failed to list org users with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp ListOrgUsersResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		allUsers = append(allUsers, listResp.Data...)

		if len(listResp.Data) < limit || len(allUsers) >= listResp.Total {
			break
		}
		offset += limit
	}

	return allUsers, nil
}
