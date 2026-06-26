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

// TeamMembersClient wraps the UserAnypointClient for team membership operations.
type TeamMembersClient struct {
	*client.UserAnypointClient
}

// NewTeamMembersClient creates a new TeamMembersClient
func NewTeamMembersClient(config *client.UserClientConfig) (*TeamMembersClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TeamMembersClient{UserAnypointClient: userAnypointClient}, nil
}

// TeamMember represents a user who is a member of a team
type TeamMember struct {
	ID                          string `json:"id"`
	IdentityType                string `json:"identity_type"`
	Name                        string `json:"name"`
	MembershipType              string `json:"membership_type"`
	IsAssignedViaExternalGroups bool   `json:"is_assigned_via_external_groups"`
	CreatedAt                   string `json:"created_at"`
	UpdatedAt                   string `json:"updated_at"`
}

// ListTeamMembersResponse wraps the GET response
type ListTeamMembersResponse struct {
	Data  []TeamMember `json:"data"`
	Total int          `json:"total"`
}

// AddTeamMemberRequest represents the request body to add a member to a team
type AddTeamMemberRequest struct {
	MembershipType string `json:"membership_type"`
}

// AddTeamMember adds a user to a team.
// API: PUT /accounts/api/organizations/{orgId}/teams/{teamId}/members/{userId}
// Body: {"membership_type": "member"|"maintainer"}
// Returns 204 No Content on success.
func (c *TeamMembersClient) AddTeamMember(ctx context.Context, orgID, teamID, userID, membershipType string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/members/%s", c.BaseURL, orgID, teamID, userID)

	if membershipType == "" {
		membershipType = "member"
	}

	reqBody := AddTeamMemberRequest{
		MembershipType: membershipType,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal add member request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("team or user")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add member to team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveTeamMember removes a user from a team.
// API: DELETE /accounts/api/organizations/{orgId}/teams/{teamId}/members/{userId}
func (c *TeamMembersClient) RemoveTeamMember(ctx context.Context, orgID, teamID, userID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/members/%s", c.BaseURL, orgID, teamID, userID)

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

	// Already removed — treat as success
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusConflict {
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove member from team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListTeamMembers lists all members of a team.
// API: GET /accounts/api/organizations/{orgId}/teams/{teamId}/members
func (c *TeamMembersClient) ListTeamMembers(ctx context.Context, orgID, teamID string) ([]TeamMember, error) {
	var allMembers []TeamMember
	limit := 100
	offset := 0

	for {
		url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/members?limit=%d&offset=%d", c.BaseURL, orgID, teamID, limit, offset)

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
			return nil, client.NewNotFoundError("team")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to list team members with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp ListTeamMembersResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		allMembers = append(allMembers, listResp.Data...)

		if len(listResp.Data) < limit || len(allMembers) >= listResp.Total {
			break
		}
		offset += limit
	}

	return allMembers, nil
}

// GetTeamMember checks if a specific user is a member of a team.
// Lists all members and filters by userID.
// Note: The API path GET /teams/{teamId}/members/{userId} does NOT support GET —
// it only supports PUT (add) and DELETE (remove). So we must list + filter.
func (c *TeamMembersClient) GetTeamMember(ctx context.Context, orgID, teamID, userID string) (*TeamMember, error) {
	members, err := c.ListTeamMembers(ctx, orgID, teamID)
	if err != nil {
		return nil, err
	}

	for _, m := range members {
		if m.ID == userID {
			return &m, nil
		}
	}

	return nil, client.NewNotFoundError("user in team")
}
