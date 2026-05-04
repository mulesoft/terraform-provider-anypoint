package cloudhub2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// PrivateSpaceAssociationClient wraps the AnypointClient for private space association operations
type PrivateSpaceAssociationClient struct {
	*client.AnypointClient
}

// NewPrivateSpaceAssociationClient creates a new PrivateSpaceAssociationClient
func NewPrivateSpaceAssociationClient(config *client.Config) (*PrivateSpaceAssociationClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &PrivateSpaceAssociationClient{AnypointClient: anypointClient}, nil
}

// AssociationRequest represents a single association request
type AssociationRequest struct {
	OrganizationID string `json:"organizationId"`
	Environment    string `json:"environment"`
}

// CreatePrivateSpaceAssociationRequest represents the request to create private space associations
type CreatePrivateSpaceAssociationRequest struct {
	Associations []AssociationRequest `json:"associations"`
}

// PrivateSpaceAssociation represents a private space association
type PrivateSpaceAssociation struct {
	ID             string `json:"id"`
	EnvironmentID  string `json:"environmentId"`
	OrganizationID string `json:"organizationId"`
}

// CreatePrivateSpaceAssociations creates new private space associations
func (c *PrivateSpaceAssociationClient) CreatePrivateSpaceAssociations(ctx context.Context, orgID, privateSpaceID string, request *CreatePrivateSpaceAssociationRequest) ([]PrivateSpaceAssociation, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/associations", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private space association request: %w", err)
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
		return nil, fmt.Errorf("failed to create private space associations with status %d: %s", resp.StatusCode, string(body))
	}

	var associations []PrivateSpaceAssociation
	if err := json.NewDecoder(resp.Body).Decode(&associations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return associations, nil
}

// GetPrivateSpaceAssociations retrieves all associations for a private space
func (c *PrivateSpaceAssociationClient) GetPrivateSpaceAssociations(ctx context.Context, orgID, privateSpaceID string) ([]PrivateSpaceAssociation, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/associations", c.BaseURL, orgID, privateSpaceID)

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
		return []PrivateSpaceAssociation{}, nil // Return empty slice if not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get private space associations with status %d: %s", resp.StatusCode, string(body))
	}

	var associations []PrivateSpaceAssociation
	if err := json.NewDecoder(resp.Body).Decode(&associations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return associations, nil
}

// DeletePrivateSpaceAssociation deletes a specific association
func (c *PrivateSpaceAssociationClient) DeletePrivateSpaceAssociation(ctx context.Context, orgID, privateSpaceID, associationID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/associations/%s", c.BaseURL, orgID, privateSpaceID, associationID)

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
		// Association already deleted, consider it success
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete private space association with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
