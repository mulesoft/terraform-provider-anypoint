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

// EnvironmentClient wraps the UserAnypointClient for environment operations
type EnvironmentClient struct {
	*client.UserAnypointClient
}

// NewEnvironmentClient creates a new EnvironmentClient using UserAnypointClient
func NewEnvironmentClient(config *client.UserClientConfig) (*EnvironmentClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &EnvironmentClient{UserAnypointClient: userAnypointClient}, nil
}

// Environment represents an Anypoint Environment
type Environment struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	IsProduction   bool    `json:"isProduction"`
	OrganizationID string  `json:"organizationId,omitempty"`
	ClientID       string  `json:"clientId,omitempty"`
	ArcNamespace   *string `json:"arcNamespace,omitempty"`
}

// CreateEnvironmentRequest represents the request to create an environment
type CreateEnvironmentRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	IsProduction bool   `json:"isProduction,omitempty"`
}

// UpdateEnvironmentRequest represents the request to update an environment
type UpdateEnvironmentRequest struct {
	Name         *string `json:"name,omitempty"`
	Type         *string `json:"type,omitempty"`
	IsProduction *bool   `json:"isProduction,omitempty"`
}

// CreateEnvironment creates a new environment in Anypoint
func (c *EnvironmentClient) CreateEnvironment(ctx context.Context, orgID string, environment *CreateEnvironmentRequest) (*Environment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/environments", c.BaseURL, orgID)

	jsonData, err := json.Marshal(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment data: %w", err)
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
		return nil, fmt.Errorf("failed to create environment with status %d: %s", resp.StatusCode, string(body))
	}

	var createdEnvironment Environment
	if err := json.NewDecoder(resp.Body).Decode(&createdEnvironment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdEnvironment, nil
}

// GetEnvironment retrieves an environment by ID
func (c *EnvironmentClient) GetEnvironment(ctx context.Context, orgID, environmentID string) (*Environment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/environments/%s", c.BaseURL, orgID, environmentID)

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
		return nil, client.NewNotFoundError("environment")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get environment with status %d: %s", resp.StatusCode, string(body))
	}

	var environment Environment
	if err := json.NewDecoder(resp.Body).Decode(&environment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &environment, nil
}

// UpdateEnvironment updates an existing environment
func (c *EnvironmentClient) UpdateEnvironment(ctx context.Context, orgID, environmentID string, environment *UpdateEnvironmentRequest) (*Environment, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/environments/%s", c.BaseURL, orgID, environmentID)

	jsonData, err := json.Marshal(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("environment")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update environment with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedEnvironment Environment
	if err := json.NewDecoder(resp.Body).Decode(&updatedEnvironment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedEnvironment, nil
}

// DeleteEnvironment deletes an environment by ID
func (c *EnvironmentClient) DeleteEnvironment(ctx context.Context, orgID, environmentID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/environments/%s", c.BaseURL, orgID, environmentID)

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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete environment with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
