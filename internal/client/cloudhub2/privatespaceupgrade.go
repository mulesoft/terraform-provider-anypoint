package cloudhub2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// PrivateSpaceUpgradeClient wraps the AnypointClient for private space upgrade operations
type PrivateSpaceUpgradeClient struct {
	*client.AnypointClient
}

// NewPrivateSpaceUpgradeClient creates a new PrivateSpaceUpgradeClient
func NewPrivateSpaceUpgradeClient(config *client.Config) (*PrivateSpaceUpgradeClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &PrivateSpaceUpgradeClient{AnypointClient: anypointClient}, nil
}

// PrivateSpaceUpgradeResponse represents the response from the upgrade API
type PrivateSpaceUpgradeResponse struct {
	ScheduledUpdateTime string `json:"scheduledUpdateTime"`
	Status              string `json:"status"`
}

// UpgradePrivateSpaceRequest represents the request parameters for upgrading a private space
type UpgradePrivateSpaceRequest struct {
	Date  string `json:"date"`
	OptIn bool   `json:"optIn"`
}

// UpgradePrivateSpace schedules an upgrade for a private space
func (c *PrivateSpaceUpgradeClient) UpgradePrivateSpace(ctx context.Context, orgID, privateSpaceID string, request *UpgradePrivateSpaceRequest) (*PrivateSpaceUpgradeResponse, error) {
	baseURL := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/upgrade", c.BaseURL, orgID, privateSpaceID)

	// Add query parameters
	params := url.Values{}
	params.Add("date", request.Date)
	params.Add("optIn", fmt.Sprintf("%t", request.OptIn))

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "PATCH", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to upgrade private space with status %d: %s", resp.StatusCode, string(body))
	}

	var upgradeResponse PrivateSpaceUpgradeResponse
	if err := json.NewDecoder(resp.Body).Decode(&upgradeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &upgradeResponse, nil
}

// GetPrivateSpaceUpgradeStatus retrieves the upgrade status for a private space
func (c *PrivateSpaceUpgradeClient) GetPrivateSpaceUpgradeStatus(ctx context.Context, orgID, privateSpaceID string) (*PrivateSpaceUpgradeResponse, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/upgradestatus", c.BaseURL, orgID, privateSpaceID)

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
		return nil, client.NewNotFoundError("private space upgrade status")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get private space upgrade status with status %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle empty response (no scheduled upgrade)
	if len(body) == 0 {
		return &PrivateSpaceUpgradeResponse{
			ScheduledUpdateTime: "",
			Status:              "NO_UPGRADE_SCHEDULED",
		}, nil
	}

	var upgradeResponse PrivateSpaceUpgradeResponse
	if err := json.Unmarshal(body, &upgradeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &upgradeResponse, nil
}

// DeletePrivateSpaceUpgrade deletes/cancels a scheduled upgrade for a private space
func (c *PrivateSpaceUpgradeClient) DeletePrivateSpaceUpgrade(ctx context.Context, orgID, privateSpaceID string) error {
	// Hardcoded success response for testing - skip actual API call
	// return nil

	// Original API call code (commented out for testing)

	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/upgrade", c.BaseURL, orgID, privateSpaceID)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted or doesn't exist
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete private space upgrade with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
