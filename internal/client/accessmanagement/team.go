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

// TeamClient wraps the AnypointClient for team operations
type TeamClient struct {
	*client.AnypointClient
}

// NewTeamClient creates a new TeamClient
func NewTeamClient(config *client.Config) (*TeamClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TeamClient{AnypointClient: anypointClient}, nil
}

// Team represents an Anypoint team
type Team struct {
	ID        string `json:"team_id"`
	TeamName  string `json:"team_name"`
	OrgID     string `json:"org_id"`
	TeamType  string `json:"team_type"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateTeamRequest represents the request to create a team
type CreateTeamRequest struct {
	TeamName     string `json:"team_name"`
	ParentTeamID string `json:"parent_team_id,omitempty"`
	TeamType     string `json:"team_type"`
}

// UpdateTeamRequest represents the request to update a team (partial update)
// Note: The PATCH API currently only supports updating team_name
type UpdateTeamRequest struct {
	TeamName *string `json:"team_name,omitempty"`
}

// UpdateTeamParentRequest represents the request to update a team's parent
type UpdateTeamParentRequest struct {
	ParentTeamID string `json:"parent_team_id"`
}

// CreateTeam creates a new team in Anypoint
func (c *TeamClient) CreateTeam(ctx context.Context, orgID string, team *CreateTeamRequest) (*Team, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams", c.BaseURL, orgID)

	jsonData, err := json.Marshal(team)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal team data: %w", err)
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
		return nil, fmt.Errorf("failed to create team with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var createdTeam Team
	if err := json.Unmarshal(body, &createdTeam); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdTeam, nil
}

// GetTeam retrieves a team by ID
func (c *TeamClient) GetTeam(ctx context.Context, orgID, teamID string) (*Team, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s", c.BaseURL, orgID, teamID)

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
		return nil, client.NewNotFoundError("team")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get team with status %d: %s", resp.StatusCode, string(body))
	}

	var team Team
	if err := json.NewDecoder(resp.Body).Decode(&team); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &team, nil
}

// UpdateTeam updates an existing team using PATCH
func (c *TeamClient) UpdateTeam(ctx context.Context, orgID, teamID string, team *UpdateTeamRequest) (*Team, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s", c.BaseURL, orgID, teamID)

	jsonData, err := json.Marshal(team)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal team data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("team")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update team with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedTeam Team
	if err := json.NewDecoder(resp.Body).Decode(&updatedTeam); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedTeam, nil
}

// UpdateTeamParent updates a team's parent using the PUT parent API
func (c *TeamClient) UpdateTeamParent(ctx context.Context, orgID, teamID string, req *UpdateTeamParentRequest) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s/parent", c.BaseURL, orgID, teamID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal team parent data: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("team")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update team parent with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteTeam deletes a team by ID
func (c *TeamClient) DeleteTeam(ctx context.Context, orgID, teamID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/teams/%s", c.BaseURL, orgID, teamID)

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

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete team with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
