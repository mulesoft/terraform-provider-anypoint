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

type SharedSecretClient struct {
	*client.AnypointClient
}

func NewSharedSecretClient(config *client.ClientConfig) (*SharedSecretClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &SharedSecretClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type SharedSecret struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	ExpirationDate string `json:"expirationDate,omitempty"`

	// UsernamePassword
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// S3Credential
	AccessKeyID    string `json:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`

	// SymmetricKey
	Key string `json:"key,omitempty"`

	// Blob
	Content string `json:"content,omitempty"`
}

type SharedSecretResponse struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Meta           SecretGroupMeta `json:"meta"`
	ExpirationDate string          `json:"expirationDate,omitempty"`
	Username       string          `json:"username,omitempty"`
	AccessKeyID    string          `json:"accessKeyId,omitempty"`
}

// --- CRUD Operations ---

func (c *SharedSecretClient) basePath(orgID, envID, sgID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups/%s/sharedSecrets",
		c.BaseURL, orgID, envID, sgID)
}

func (c *SharedSecretClient) CreateSharedSecret(ctx context.Context, orgID, envID, sgID string, request *SharedSecret) (*SharedSecretResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shared secret request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("failed to create shared secret with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetSharedSecret(ctx, orgID, envID, sgID, createResp.ID)
}

func (c *SharedSecretClient) GetSharedSecret(ctx context.Context, orgID, envID, sgID, ssID string) (*SharedSecretResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), ssID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("shared secret")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get shared secret with status %d: %s", resp.StatusCode, string(body))
	}

	var ss SharedSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&ss); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &ss, nil
}

func (c *SharedSecretClient) UpdateSharedSecret(ctx context.Context, orgID, envID, sgID, ssID string, request *SharedSecret) (*SharedSecretResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), ssID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shared secret update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update shared secret with status %d: %s", resp.StatusCode, string(body))
	}

	return c.GetSharedSecret(ctx, orgID, envID, sgID, ssID)
}

func (c *SharedSecretClient) DeleteSharedSecret(ctx context.Context, orgID, envID, sgID, ssID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), ssID)

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
		return fmt.Errorf("failed to delete shared secret with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListSharedSecrets returns all shared secrets in the given secret group.
func (c *SharedSecretClient) ListSharedSecrets(ctx context.Context, orgID, envID, sgID string) ([]SharedSecretResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("failed to list shared secrets with status %d: %s", resp.StatusCode, string(body))
	}

	var items []SharedSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return items, nil
}
