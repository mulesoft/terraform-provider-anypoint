package secretsmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

type SecretGroupClient struct {
	*client.AnypointClient
}

func NewSecretGroupClient(config *client.Config) (*SecretGroupClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &SecretGroupClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type SecretGroup struct {
	ID           string `json:"meta,omitempty"`
	Name         string `json:"name"`
	Downloadable bool   `json:"downloadable"`
	CurrentState string `json:"currentState,omitempty"`
	CreatedAt    string `json:"createdAt,omitempty"`
	ModifiedAt   string `json:"modifiedAt,omitempty"`
}

type SecretGroupMeta struct {
	ID string `json:"id"`
}

type SecretGroupResponse struct {
	Name         string          `json:"name"`
	Downloadable bool            `json:"downloadable"`
	Meta         SecretGroupMeta `json:"meta"`
	CurrentState string          `json:"currentState,omitempty"`
	CreatedAt    string          `json:"createdAt,omitempty"`
	ModifiedAt   string          `json:"modifiedAt,omitempty"`
}

// --- Request Models ---

type CreateSecretGroupRequest struct {
	Name         string `json:"name"`
	Downloadable bool   `json:"downloadable"`
}

type UpdateSecretGroupRequest struct {
	Name         string `json:"name,omitempty"`
	Downloadable *bool  `json:"downloadable,omitempty"`
}

// --- CRUD Operations ---

func (c *SecretGroupClient) basePath(orgID, envID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups",
		c.BaseURL, orgID, envID)
}

func (c *SecretGroupClient) CreateSecretGroup(ctx context.Context, orgID, envID string, request *CreateSecretGroupRequest) (*SecretGroupResponse, error) {
	url := c.basePath(orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret group request: %w", err)
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
		return nil, fmt.Errorf("failed to create secret group with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetSecretGroup(ctx, orgID, envID, createResp.ID)
}

func (c *SecretGroupClient) GetSecretGroup(ctx context.Context, orgID, envID, sgID string) (*SecretGroupResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID), sgID)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("secret group")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get secret group with status %d: %s", resp.StatusCode, string(body))
	}

	var sg SecretGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&sg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &sg, nil
}

func (c *SecretGroupClient) UpdateSecretGroup(ctx context.Context, orgID, envID, sgID string, request *UpdateSecretGroupRequest) (*SecretGroupResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID), sgID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret group update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
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
		return nil, client.NewNotFoundError("secret group")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update secret group with status %d: %s", resp.StatusCode, string(body))
	}

	return c.GetSecretGroup(ctx, orgID, envID, sgID)
}

func (c *SecretGroupClient) DeleteSecretGroup(ctx context.Context, orgID, envID, sgID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID), sgID)

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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete secret group with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListSecretGroups returns all secret groups for the given org/environment.
func (c *SecretGroupClient) ListSecretGroups(ctx context.Context, orgID, envID string) ([]SecretGroupResponse, error) {
	url := c.basePath(orgID, envID)

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
		return nil, fmt.Errorf("failed to list secret groups with status %d: %s", resp.StatusCode, string(body))
	}

	var groups []SecretGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return groups, nil
}
