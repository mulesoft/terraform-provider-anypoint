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

type AlertClient struct {
	*client.AnypointClient
}

func NewAlertClient(config *client.Config) (*AlertClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &AlertClient{AnypointClient: anypointClient}, nil
}

// --- DeploymentType Enum ---

type DeploymentType string

// The Monitoring Alerts API expects UPPERCASE enum names.
const (
	DeploymentTypeCloudHub      DeploymentType = "CLOUDHUB"
	DeploymentTypeCloudHub2     DeploymentType = "CLOUDHUB2"
	DeploymentTypeHybrid        DeploymentType = "HYBRID"
	DeploymentTypeRuntimeFabric DeploymentType = "RUNTIMEFABRIC"
	DeploymentTypeServiceMesh   DeploymentType = "SERVICEMESH"
)

// deploymentTypeMap maps shortcodes, camelCase, and uppercase names to the
// UPPERCASE enum values the Monitoring Alerts API expects.
var deploymentTypeMap = map[string]DeploymentType{
	"CH":            DeploymentTypeCloudHub,
	"CH2":           DeploymentTypeCloudHub2,
	"HY":            DeploymentTypeHybrid,
	"RF":            DeploymentTypeRuntimeFabric,
	"SM":            DeploymentTypeServiceMesh,
	"cloudHub":      DeploymentTypeCloudHub,
	"cloudHub2":     DeploymentTypeCloudHub2,
	"hybrid":        DeploymentTypeHybrid,
	"runtimeFabric": DeploymentTypeRuntimeFabric,
	"serviceMesh":   DeploymentTypeServiceMesh,
	"CLOUDHUB":      DeploymentTypeCloudHub,
	"CLOUDHUB2":     DeploymentTypeCloudHub2,
	"HYBRID":        DeploymentTypeHybrid,
	"RUNTIMEFABRIC": DeploymentTypeRuntimeFabric,
	"SERVICEMESH":   DeploymentTypeServiceMesh,
}

// NormalizeDeploymentType converts any recognized form of a deployment type
// (shortcode, camelCase, or UPPERCASE) into the UPPERCASE enum the Alerts API expects.
func NormalizeDeploymentType(raw string) DeploymentType {
	if dt, ok := deploymentTypeMap[raw]; ok {
		return dt
	}
	return DeploymentType(raw)
}

// --- Domain Models ---

type Alert struct {
	ID                   string              `json:"alertId"`
	Name                 string              `json:"name"`
	Type                 string              `json:"alertType,omitempty"`
	Severity             string              `json:"severity,omitempty"`
	OrganizationID       string              `json:"organizationId,omitempty"`
	MasterOrganizationID string              `json:"masterOrganizationId,omitempty"`
	EnvironmentID        string              `json:"environmentId,omitempty"`
	ResourceType         string              `json:"resourceType,omitempty"`
	DeploymentType       DeploymentType      `json:"deploymentType,omitempty"`
	MetricType           string              `json:"metricType,omitempty"`
	Resources            []AlertResource     `json:"resources,omitempty"`
	Condition            AlertCondition      `json:"condition"`
	WildcardAlert        bool                `json:"wildcardAlert"`
	Notifications        []AlertNotification `json:"notifications,omitempty"`
	Enabled              *bool               `json:"enabled,omitempty"`
}

type AlertResource struct {
	APIVersionID   string         `json:"apiVersionId"`
	APIID          string         `json:"apiId,omitempty"`
	Type           string         `json:"type,omitempty"`
	DeploymentType DeploymentType `json:"deploymentType,omitempty"`
}

type AlertCondition struct {
	Operator  string `json:"operator"`
	Threshold int    `json:"threshold"`
	Interval  int    `json:"interval"`
}

type AlertNotification struct {
	Type       string   `json:"type"`
	Recipients []string `json:"recipients,omitempty"`
	Subject    string   `json:"subject,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// --- Request Models ---

type CreateAlertRequest struct {
	Name                 string              `json:"name"`
	Type                 string              `json:"type"`
	Severity             string              `json:"severity"`
	MasterOrganizationID string              `json:"masterOrganizationId"`
	OrganizationID       string              `json:"organizationId"`
	EnvironmentID        string              `json:"environmentId"`
	ResourceType         string              `json:"resourceType"`
	DeploymentType       DeploymentType      `json:"deploymentType"`
	MetricType           string              `json:"metricType"`
	Resources            []AlertResource     `json:"resources"`
	Condition            AlertCondition      `json:"condition"`
	WildcardAlert        bool                `json:"wildcardAlert"`
	Notifications        []AlertNotification `json:"notifications"`
}

type UpdateAlertRequest = CreateAlertRequest

// --- CRUD Operations ---

func (c *AlertClient) basePath(orgID, envID string) string {
	return fmt.Sprintf("%s/monitoring/api/alerts/api/v2/organizations/%s/environments/%s/alerts",
		c.BaseURL, orgID, envID)
}

func (c *AlertClient) CreateAlert(ctx context.Context, orgID, envID string, request *CreateAlertRequest) (*Alert, error) {
	url := c.basePath(orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alert request: %w", err)
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
		return nil, fmt.Errorf("failed to create alert with status %d: %s", resp.StatusCode, string(body))
	}

	var alert Alert
	if err := json.NewDecoder(resp.Body).Decode(&alert); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &alert, nil
}

func (c *AlertClient) GetAlert(ctx context.Context, orgID, envID, alertID string) (*Alert, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID), alertID)

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
		return nil, client.NewNotFoundError("alert")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get alert with status %d: %s", resp.StatusCode, string(body))
	}

	var alert Alert
	if err := json.NewDecoder(resp.Body).Decode(&alert); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &alert, nil
}

func (c *AlertClient) UpdateAlert(ctx context.Context, orgID, envID, alertID string, request *UpdateAlertRequest) (*Alert, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID), alertID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alert update request: %w", err)
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
		return nil, client.NewNotFoundError("alert")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update alert with status %d: %s", resp.StatusCode, string(body))
	}

	var alert Alert
	if err := json.NewDecoder(resp.Body).Decode(&alert); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &alert, nil
}

func (c *AlertClient) DeleteAlert(ctx context.Context, orgID, envID, alertID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID), alertID)

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
		return fmt.Errorf("failed to delete alert with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
