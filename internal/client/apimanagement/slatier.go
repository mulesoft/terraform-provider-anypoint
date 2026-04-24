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

type SLATierClient struct {
	*client.AnypointClient
}

func NewSLATierClient(config *client.ClientConfig) (*SLATierClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &SLATierClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type SLATier struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	AutoApprove  bool       `json:"autoApprove"`
	Status       string     `json:"status,omitempty"`
	Limits       []SLALimit `json:"limits,omitempty"`
	APIVersionID string     `json:"apiVersionId,omitempty"`
	Audit        *SLAAudit  `json:"audit,omitempty"`
}

type SLALimit struct {
	TimePeriodInMilliseconds int  `json:"timePeriodInMilliseconds"`
	MaximumRequests          int  `json:"maximumRequests"`
	Visible                  bool `json:"visible"`
}

type SLAAudit struct {
	Created *SLAAuditEntry `json:"created,omitempty"`
	Updated *SLAAuditEntry `json:"updated,omitempty"`
}

type SLAAuditEntry struct {
	Date string `json:"date,omitempty"`
}

// --- Request Models ---

type CreateSLATierRequest struct {
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	AutoApprove  bool       `json:"autoApprove"`
	Limits       []SLALimit `json:"limits"`
	APIVersionID string     `json:"apiVersionId,omitempty"`
	Status       string     `json:"status,omitempty"`
}

type UpdateSLATierRequest struct {
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	AutoApprove *bool      `json:"autoApprove,omitempty"`
	Limits      []SLALimit `json:"limits,omitempty"`
	Status      string     `json:"status,omitempty"`
}

// --- CRUD Operations ---

func (c *SLATierClient) basePath(orgID, envID string, apiID int) string {
	return fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/tiers",
		c.BaseURL, orgID, envID, apiID)
}

func (c *SLATierClient) CreateSLATier(ctx context.Context, orgID, envID string, apiID int, request *CreateSLATierRequest) (*SLATier, error) {
	url := c.basePath(orgID, envID, apiID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SLA tier request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create SLA tier with status %d: %s", resp.StatusCode, string(body))
	}

	var tier SLATier
	if err := json.NewDecoder(resp.Body).Decode(&tier); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tier, nil
}

// ListSLATiers fetches all SLA tiers for the given API instance.
func (c *SLATierClient) ListSLATiers(ctx context.Context, orgID, envID string, apiID int) ([]SLATier, error) {
	url := c.basePath(orgID, envID, apiID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list SLA tiers with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tiers []SLATier `json:"tiers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Tiers, nil
}

// GetSLATier fetches a single SLA tier by listing all tiers and filtering by ID,
// because the API does not support GET on individual tier paths.
func (c *SLATierClient) GetSLATier(ctx context.Context, orgID, envID string, apiID, tierID int) (*SLATier, error) {
	tiers, err := c.ListSLATiers(ctx, orgID, envID, apiID)
	if err != nil {
		return nil, err
	}

	for i := range tiers {
		if tiers[i].ID == tierID {
			return &tiers[i], nil
		}
	}

	return nil, client.NewNotFoundError("SLA tier")
}

func (c *SLATierClient) UpdateSLATier(ctx context.Context, orgID, envID string, apiID, tierID int, request *UpdateSLATierRequest) (*SLATier, error) {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID, envID, apiID), tierID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SLA tier update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("SLA tier")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update SLA tier with status %d: %s", resp.StatusCode, string(body))
	}

	var tier SLATier
	if err := json.NewDecoder(resp.Body).Decode(&tier); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tier, nil
}

func (c *SLATierClient) DeleteSLATier(ctx context.Context, orgID, envID string, apiID, tierID int) error {
	url := fmt.Sprintf("%s/%d", c.basePath(orgID, envID, apiID), tierID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete SLA tier with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
