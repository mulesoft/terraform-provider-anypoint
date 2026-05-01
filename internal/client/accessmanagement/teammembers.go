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

// TeamMember represents a team member with membership details
type TeamMember struct {
	ID             string `json:"id"`
	MembershipType string `json:"membership_type"`
}

// TeamMemberResponse represents the API response for team members
type TeamMemberResponse struct {
	Data []TeamMemberDetails `json:"data"`
}

// TeamMemberDetails represents detailed information about a team member from API response
type TeamMemberDetails struct {
	ID             string `json:"id"`
	Username       string `json:"username,omitempty"`
	FirstName      string `json:"firstName,omitempty"`
	LastName       string `json:"lastName,omitempty"`
	Email          string `json:"email,omitempty"`
	MembershipType string `json:"membership_type"`
}

// TeamMembersClient wraps the AnypointClient for team member operations
type TeamMembersClient struct {
	*client.AnypointClient
}

// NewTeamMembersClient creates a new TeamMembersClient
func NewTeamMembersClient(config *client.Config) (*TeamMembersClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TeamMembersClient{AnypointClient: anypointClient}, nil
}

// AddMembersToTeam adds or updates members in a team using PATCH API
func (c *TeamMembersClient) AddMembersToTeam(ctx context.Context, orgID, teamID string, members []TeamMember) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/members", c.AnypointClient.BaseURL, orgID, teamID)

	reqBody, err := json.Marshal(members)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(reqBody))
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
		return fmt.Errorf("failed to add members to team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetTeamMembers retrieves members of a team
func (c *TeamMembersClient) GetTeamMembers(ctx context.Context, orgID, teamID string) ([]TeamMemberDetails, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/members", c.AnypointClient.BaseURL, orgID, teamID)

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
		return []TeamMemberDetails{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get members for team with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return []TeamMemberDetails{}, nil
	}

	// Try to unmarshal as wrapper object first
	var response TeamMemberResponse
	if err := json.Unmarshal(body, &response); err == nil && response.Data != nil {
		return response.Data, nil
	}

	// Try to unmarshal as direct array
	var membersArray []TeamMemberDetails
	if err := json.Unmarshal(body, &membersArray); err == nil {
		return membersArray, nil
	}

	// Try to unmarshal as single member
	var singleMember TeamMemberDetails
	if err := json.Unmarshal(body, &singleMember); err == nil && singleMember.ID != "" {
		return []TeamMemberDetails{singleMember}, nil
	}

	// Try to unmarshal as wrapper object with common field names
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response. Response body: %s, Error: %w", string(body), err)
	}

	// Check for common wrapper field names
	for _, fieldName := range []string{"data", "members", "items", "results"} {
		if field, exists := rawResponse[fieldName]; exists {
			if fieldBytes, err := json.Marshal(field); err == nil {
				var members []TeamMemberDetails
				if err := json.Unmarshal(fieldBytes, &members); err == nil {
					return members, nil
				}
			}
		}
	}

	// If all else fails, return empty array
	return []TeamMemberDetails{}, nil
}

// RemoveMembersFromTeam removes members from a team using DELETE API
func (c *TeamMembersClient) RemoveMembersFromTeam(ctx context.Context, orgID, teamID string, memberIDs []string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/members", c.AnypointClient.BaseURL, orgID, teamID)

	reqBody, err := json.Marshal(memberIDs)
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
		// Team or members not found, consider it successful
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove members from team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
