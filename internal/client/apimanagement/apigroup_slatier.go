package apimanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// GroupSLATierClient handles SLA tier operations for API Group instances.
// The API path differs from regular API instance tiers:
//
//	/apimanager/api/v1/organizations/{orgId}/environments/{envId}/groupInstances/{groupInstanceId}/tiers
type GroupSLATierClient struct {
	*client.AnypointClient
}

// NewGroupSLATierClient creates a new GroupSLATierClient.
func NewGroupSLATierClient(config *client.ClientConfig) (*GroupSLATierClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &GroupSLATierClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// GroupSLATier is an SLA tier belonging to a group instance. The limits field
// is named "defaultLimits" in the group instance tier API.
type GroupSLATier struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	AutoApprove   bool       `json:"autoApprove"`
	Status        string     `json:"status,omitempty"`
	DefaultLimits []SLALimit `json:"defaultLimits,omitempty"`
}

// --- Request Models ---

// CreateGroupSLATierRequest is the payload for creating a group instance SLA tier.
type CreateGroupSLATierRequest struct {
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	AutoApprove   bool       `json:"autoApprove"`
	Status        string     `json:"status,omitempty"`
	DefaultLimits []SLALimit `json:"defaultLimits"`
}

// UpdateGroupSLATierRequest is the payload for updating a group instance SLA tier.
type UpdateGroupSLATierRequest struct {
	Name          string     `json:"name,omitempty"`
	Description   string     `json:"description,omitempty"`
	AutoApprove   *bool      `json:"autoApprove,omitempty"`
	Status        string     `json:"status,omitempty"`
	DefaultLimits []SLALimit `json:"defaultLimits,omitempty"`
}

// --- CRUD Operations ---

func (c *GroupSLATierClient) basePath(orgID, envID string, groupInstanceID int) string {
	return fmt.Sprintf(
		"%s/apimanager/api/v1/organizations/%s/environments/%s/groupInstances/%d/tiers",
		c.BaseURL, orgID, envID, groupInstanceID,
	)
}

func (c *GroupSLATierClient) setHeaders(req *http.Request, orgID, envID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)
}

// CreateGroupSLATier creates a new SLA tier on the given group instance.
func (c *GroupSLATierClient) CreateGroupSLATier(ctx context.Context, orgID, envID string, groupInstanceID int, request *CreateGroupSLATierRequest) (*GroupSLATier, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal group SLA tier request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.basePath(orgID, envID, groupInstanceID), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID, envID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create group SLA tier with status %d: %s", resp.StatusCode, string(b))
	}

	var tier GroupSLATier
	if err := json.NewDecoder(resp.Body).Decode(&tier); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &tier, nil
}

// ListGroupSLATiers returns all SLA tiers for the given group instance.
func (c *GroupSLATierClient) ListGroupSLATiers(ctx context.Context, orgID, envID string, groupInstanceID int) ([]GroupSLATier, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.basePath(orgID, envID, groupInstanceID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID, envID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list group SLA tiers with status %d: %s", resp.StatusCode, string(b))
	}

	// Try wrapping response first; fall back to plain array
	var wrapped struct {
		Tiers []GroupSLATier `json:"tiers"`
	}
	rawBody, _ := io.ReadAll(resp.Body)
	if jsonErr := json.Unmarshal(rawBody, &wrapped); jsonErr == nil && wrapped.Tiers != nil {
		return wrapped.Tiers, nil
	}

	var tiers []GroupSLATier
	if jsonErr := json.Unmarshal(rawBody, &tiers); jsonErr != nil {
		return nil, fmt.Errorf("failed to decode response: %w", jsonErr)
	}
	return tiers, nil
}

// GetGroupSLATier fetches a single SLA tier by iterating the list.
func (c *GroupSLATierClient) GetGroupSLATier(ctx context.Context, orgID, envID string, groupInstanceID, tierID int) (*GroupSLATier, error) {
	tiers, err := c.ListGroupSLATiers(ctx, orgID, envID, groupInstanceID)
	if err != nil {
		return nil, err
	}
	for i := range tiers {
		if tiers[i].ID == tierID {
			return &tiers[i], nil
		}
	}
	return nil, client.NewNotFoundError("group SLA tier")
}

// UpdateGroupSLATier replaces a group instance SLA tier (PUT).
func (c *GroupSLATierClient) UpdateGroupSLATier(ctx context.Context, orgID, envID string, groupInstanceID, tierID int, request *UpdateGroupSLATierRequest) (*GroupSLATier, error) {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID, envID, groupInstanceID), tierID)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal group SLA tier update request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID, envID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("group SLA tier")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update group SLA tier with status %d: %s", resp.StatusCode, string(b))
	}

	var tier GroupSLATier
	if err := json.NewDecoder(resp.Body).Decode(&tier); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &tier, nil
}

// DeleteGroupSLATier deletes a group instance SLA tier.
func (c *GroupSLATierClient) DeleteGroupSLATier(ctx context.Context, orgID, envID string, groupInstanceID, tierID int) error {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID, envID, groupInstanceID), tierID)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID, envID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete group SLA tier with status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
