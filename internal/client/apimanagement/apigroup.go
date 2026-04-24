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

// APIGroupClient wraps the AnypointClient for API Group operations.
type APIGroupClient struct {
	*client.AnypointClient
}

// NewAPIGroupClient creates a new APIGroupClient.
func NewAPIGroupClient(config *client.ClientConfig) (*APIGroupClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &APIGroupClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// APIGroup is the API Group resource returned by the Anypoint API.
type APIGroup struct {
	ID       int               `json:"id"`
	Name     string            `json:"name"`
	Versions []APIGroupVersion `json:"versions"`
}

// APIGroupVersion is a named version within an API Group.
type APIGroupVersion struct {
	ID        int                `json:"id,omitempty"`
	Name      string             `json:"name"`
	Instances []APIGroupInstance `json:"instances"`
}

// APIGroupInstance binds a set of API instance IDs (from one environment)
// to a group version.
type APIGroupInstance struct {
	EnvironmentID      string `json:"environmentId"`
	GroupInstanceLabel string `json:"groupInstanceLabel,omitempty"`
	APIInstances       []int  `json:"apiInstances"`
}

// --- Read-only response models ---
// The GET /groups/{id} endpoint returns apiInstances as objects {"id":…}
// rather than plain integers, so we use separate response structs and
// convert them to the canonical domain model after decoding.

type apiInstanceRef struct {
	ID int `json:"id"`
}

type apiGroupInstanceRead struct {
	EnvironmentID      string           `json:"environmentId"`
	GroupInstanceLabel string           `json:"groupInstanceLabel,omitempty"`
	APIInstances       []apiInstanceRef `json:"apiInstances"`
}

type apiGroupVersionRead struct {
	ID        int                    `json:"id,omitempty"`
	Name      string                 `json:"name"`
	Instances []apiGroupInstanceRead `json:"instances"`
}

type apiGroupRead struct {
	ID       int                   `json:"id"`
	Name     string                `json:"name"`
	Versions []apiGroupVersionRead `json:"versions"`
}

func (r *apiGroupRead) toAPIGroup() *APIGroup {
	g := &APIGroup{ID: r.ID, Name: r.Name}
	for _, v := range r.Versions {
		ver := APIGroupVersion{ID: v.ID, Name: v.Name}
		for _, inst := range v.Instances {
			ids := make([]int, 0, len(inst.APIInstances))
			for _, ref := range inst.APIInstances {
				ids = append(ids, ref.ID)
			}
			ver.Instances = append(ver.Instances, APIGroupInstance{
				EnvironmentID:      inst.EnvironmentID,
				GroupInstanceLabel: inst.GroupInstanceLabel,
				APIInstances:       ids,
			})
		}
		g.Versions = append(g.Versions, ver)
	}
	return g
}

// --- Request Models ---

// CreateAPIGroupRequest is the payload used for both create and full update.
type CreateAPIGroupRequest struct {
	Name     string            `json:"name"`
	Versions []APIGroupVersion `json:"versions"`
}

// --- CRUD Operations ---

func (c *APIGroupClient) basePath(orgID string) string {
	return fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/groups", c.BaseURL, orgID)
}

// CreateAPIGroup creates a new API Group.
func (c *APIGroupClient) CreateAPIGroup(ctx context.Context, orgID string, req *CreateAPIGroupRequest) (*APIGroup, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API group request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.basePath(orgID), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create API group with status %d: %s", resp.StatusCode, string(b))
	}

	var group APIGroup
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &group, nil
}

// GetAPIGroup retrieves an API Group by its numeric ID.
func (c *APIGroupClient) GetAPIGroup(ctx context.Context, orgID string, groupID int) (*APIGroup, error) {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID), groupID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("API group")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get API group with status %d: %s", resp.StatusCode, string(b))
	}

	var groupRead apiGroupRead
	if err := json.NewDecoder(resp.Body).Decode(&groupRead); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return groupRead.toAPIGroup(), nil
}

// UpdateAPIGroup replaces an API Group (full PUT).
func (c *APIGroupClient) UpdateAPIGroup(ctx context.Context, orgID string, groupID int, req *CreateAPIGroupRequest) (*APIGroup, error) {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID), groupID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API group update request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("API group")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update API group with status %d: %s", resp.StatusCode, string(b))
	}

	var groupRead apiGroupRead
	if err := json.NewDecoder(resp.Body).Decode(&groupRead); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return groupRead.toAPIGroup(), nil
}

// DeleteAPIGroup deletes an API Group by its numeric ID.
func (c *APIGroupClient) DeleteAPIGroup(ctx context.Context, orgID string, groupID int) error {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID), groupID)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	c.setHeaders(httpReq, orgID)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete API group with status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *APIGroupClient) setHeaders(req *http.Request, orgID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
}
