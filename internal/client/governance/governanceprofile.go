package governance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

type GovernanceProfileClient struct {
	*client.AnypointClient
}

func NewGovernanceProfileClient(config *client.ClientConfig) (*GovernanceProfileClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &GovernanceProfileClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type GovernanceProfile struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Description        string             `json:"description,omitempty"`
	Org                string             `json:"org,omitempty"`
	Rulesets           []string           `json:"rulesets,omitempty"`
	Filter             string             `json:"filter,omitempty"`
	Allowing           []string           `json:"allowing"`
	Denying            []string           `json:"denying"`
	NotificationConfig NotificationConfig `json:"notificationConfig"`
}

type NotificationConfig struct {
	Enabled       bool           `json:"enabled"`
	Notifications []Notification `json:"notifications,omitempty"`
}

type Notification struct {
	Enabled    bool               `json:"enabled"`
	Condition  string             `json:"condition,omitempty"`
	Recipients []NotifyRecipient  `json:"recipients,omitempty"`
}

type NotifyRecipient struct {
	ContactType      string `json:"contactType,omitempty"`
	NotificationType string `json:"notificationType,omitempty"`
	Value            string `json:"value"`
	Label            string `json:"label"`
}

// --- Request Models ---

type CreateGovernanceProfileRequest struct {
	Name               string             `json:"name"`
	Description        string             `json:"description,omitempty"`
	Org                string             `json:"org"`
	Rulesets           []string           `json:"rulesets"`
	Filter             string             `json:"filter,omitempty"`
	Allowing           []string           `json:"allowing"`
	Denying            []string           `json:"denying"`
	NotificationConfig NotificationConfig `json:"notificationConfig"`
}

type UpdateGovernanceProfileRequest = CreateGovernanceProfileRequest

// --- CRUD Operations ---

func (c *GovernanceProfileClient) basePath() string {
	return fmt.Sprintf("%s/governance/xapi/api/v1/profiles", c.BaseURL)
}

func (c *GovernanceProfileClient) CreateProfile(ctx context.Context, request *CreateGovernanceProfileRequest) (*GovernanceProfile, error) {
	url := c.basePath()

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal governance profile request: %w", err)
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
		return nil, fmt.Errorf("failed to create governance profile with status %d: %s", resp.StatusCode, string(body))
	}

	var profile GovernanceProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &profile, nil
}

func (c *GovernanceProfileClient) GetProfile(ctx context.Context, profileID string) (*GovernanceProfile, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(), profileID)

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
		return nil, client.NewNotFoundError("governance profile")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get governance profile with status %d: %s", resp.StatusCode, string(body))
	}

	var profile GovernanceProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &profile, nil
}

func (c *GovernanceProfileClient) UpdateProfile(ctx context.Context, profileID string, request *UpdateGovernanceProfileRequest) (*GovernanceProfile, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(), profileID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal governance profile update request: %w", err)
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
		return nil, client.NewNotFoundError("governance profile")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update governance profile with status %d: %s", resp.StatusCode, string(body))
	}

	var profile GovernanceProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &profile, nil
}

func (c *GovernanceProfileClient) DeleteProfile(ctx context.Context, profileID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(), profileID)

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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete governance profile with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
