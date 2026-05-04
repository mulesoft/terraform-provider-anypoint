package apimanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// APIInstanceClient wraps the AnypointClient for API Instance (API Manager) operations
type APIInstanceClient struct {
	*client.AnypointClient
}

// NewAPIInstanceClient creates a new APIInstanceClient
func NewAPIInstanceClient(config *client.Config) (*APIInstanceClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &APIInstanceClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// APIInstance represents a managed API instance in API Manager
type APIInstance struct {
	ID                        int                    `json:"id"`
	AssetID                   string                 `json:"assetId,omitempty"`
	AssetVersion              string                 `json:"assetVersion,omitempty"`
	ProductVersion            string                 `json:"productVersion,omitempty"`
	GroupID                   string                 `json:"groupId,omitempty"`
	Technology                string                 `json:"technology,omitempty"`
	EndpointURI               string                 `json:"endpointUri,omitempty"`
	ProviderID                *string                `json:"providerId,omitempty"`
	InstanceLabel             string                 `json:"instanceLabel,omitempty"`
	ApprovalMethod            string                 `json:"approvalMethod,omitempty"`
	Status                    string                 `json:"status,omitempty"`
	EnvironmentID             string                 `json:"environmentId,omitempty"`
	AutodiscoveryInstanceName string                 `json:"autodiscoveryInstanceName,omitempty"`
	Endpoint                  *APIInstanceEndpoint   `json:"endpoint,omitempty"`
	Spec                      *APIInstanceSpec       `json:"spec,omitempty"`
	Routing                   []APIInstanceRoute     `json:"routing,omitempty"`
	Deployment                *APIInstanceDeployment `json:"deployment,omitempty"`
}

// APIInstanceEndpoint represents the endpoint/proxy configuration.
// Fields use pointer types without omitempty so they serialize as JSON null
// when not set — the API requires these properties to be present.
type APIInstanceEndpoint struct {
	DeploymentType       string                  `json:"deploymentType,omitempty"`
	MuleVersion4OrAbove  *bool                   `json:"muleVersion4OrAbove"`
	Type                 string                  `json:"type,omitempty"`
	IsCloudHub           *bool                   `json:"isCloudHub"`
	URI                  *string                 `json:"uri,omitempty"`
	ProxyURI             *string                 `json:"proxyUri"`
	ReferencesUserDomain *bool                   `json:"referencesUserDomain"`
	ResponseTimeout      *int                    `json:"responseTimeout"`
	TLSContexts          *APIInstanceTLSContexts `json:"tlsContexts"`
}

// APIInstanceTLSContexts holds TLS context references
type APIInstanceTLSContexts struct {
	Inbound *APIInstanceTLSContext `json:"inbound"`
}

// APIInstanceTLSContext represents a single TLS context reference
type APIInstanceTLSContext struct {
	SecretGroupID string `json:"secretGroupId,omitempty"`
	TLSID         string `json:"tlsContextId,omitempty"`
}

// APIInstanceSpec identifies the Exchange asset backing this API instance
type APIInstanceSpec struct {
	AssetID string `json:"assetId"`
	GroupID string `json:"groupId"`
	Version string `json:"version"`
}

// APIInstanceRoute represents a single routing rule with weighted upstreams
type APIInstanceRoute struct {
	Label     string                `json:"label,omitempty"`
	Upstreams []APIInstanceUpstream `json:"upstreams"`
	Rules     *APIInstanceRules     `json:"rules,omitempty"`
}

// APIInstanceUpstream is one backend in a weighted routing set
type APIInstanceUpstream struct {
	ID         string                  `json:"id,omitempty"`
	Weight     int                     `json:"weight"`
	URI        string                  `json:"uri"`
	Label      string                  `json:"label,omitempty"`
	TLSContext *APIInstanceUpstreamTLS `json:"tlsContext,omitempty"`
}

// APIInstanceUpstreamTLS holds TLS context for an upstream backend
type APIInstanceUpstreamTLS struct {
	SecretGroupID string `json:"secretGroupId"`
	TLSContextID  string `json:"tlsContextId"`
}

// APIInstanceRules are match conditions for a route
type APIInstanceRules struct {
	Methods string            `json:"methods,omitempty"`
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// APIInstanceDeployment describes where this API is deployed
type APIInstanceDeployment struct {
	EnvironmentID  string `json:"environmentId,omitempty"`
	Type           string `json:"type,omitempty"`
	ExpectedStatus string `json:"expectedStatus,omitempty"`
	Overwrite      bool   `json:"overwrite,omitempty"`
	TargetID       string `json:"targetId,omitempty"`
	TargetName     string `json:"targetName,omitempty"`
	GatewayVersion string `json:"gatewayVersion,omitempty"`
}

// --- Request Models ---

// CreateAPIInstanceRequest is the payload sent to create an API instance.
// Pointer fields without omitempty serialize as null when nil, matching the API contract.
type CreateAPIInstanceRequest struct {
	Technology     string                 `json:"technology"`
	ApprovalMethod *string                `json:"approvalMethod"`
	ProviderID     *string                `json:"providerId"`
	EndpointURI    *string                `json:"endpointUri"`
	InstanceLabel  string                 `json:"instanceLabel,omitempty"`
	Endpoint       *APIInstanceEndpoint   `json:"endpoint,omitempty"`
	Spec           *APIInstanceSpec       `json:"spec"`
	Routing        []APIInstanceRoute     `json:"routing,omitempty"`
	Deployment     *APIInstanceDeployment `json:"deployment,omitempty"`
}

// UpdateAPIInstanceRequest uses pointer fields so the caller can distinguish
// "not provided" from the zero value.
type UpdateAPIInstanceRequest struct {
	Technology    *string                `json:"technology,omitempty"`
	EndpointURI   *string                `json:"endpointUri,omitempty"`
	InstanceLabel *string                `json:"instanceLabel,omitempty"`
	Endpoint      *APIInstanceEndpoint   `json:"endpoint,omitempty"`
	Spec          *APIInstanceSpec       `json:"spec,omitempty"`
	Routing       []APIInstanceRoute     `json:"routing,omitempty"`
	Deployment    *APIInstanceDeployment `json:"deployment,omitempty"`
}

// GatewayInfo holds the subset of gateway details needed to auto-build
// an API instance deployment from a gateway_id.
type GatewayInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	RuntimeVersion string `json:"runtimeVersion"`
}

// GetGatewayInfo fetches gateway details from the Gateway Manager API.
func (c *APIInstanceClient) GetGatewayInfo(ctx context.Context, orgID, envID, gatewayID string) (*GatewayInfo, error) {
	url := fmt.Sprintf("%s/gatewaymanager/xapi/v1/organizations/%s/environments/%s/gateways/%s",
		c.BaseURL, orgID, envID, gatewayID)

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
		return nil, client.NewNotFoundError(fmt.Sprintf("gateway %s", gatewayID))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get gateway info with status %d: %s", resp.StatusCode, string(body))
	}

	var gw GatewayInfo
	if err := json.NewDecoder(resp.Body).Decode(&gw); err != nil {
		return nil, fmt.Errorf("failed to decode gateway response: %w", err)
	}

	return &gw, nil
}

// PromoteEntities controls which entities to copy during promotion.
type PromoteEntities struct {
	AllEntities bool `json:"allEntities"`
}

// PromoteAPIInstanceRequest is the payload for promoting an API instance
// from one environment to another.
type PromoteAPIInstanceRequest struct {
	InstanceLabel *string       `json:"instanceLabel"`
	Promote       PromoteConfig `json:"promote"`
}

// PromoteConfig holds the promotion source and entity selection.
type PromoteConfig struct {
	OriginAPIID int              `json:"originApiId"`
	Alerts      *PromoteEntities `json:"alerts,omitempty"`
	Policies    *PromoteEntities `json:"policies,omitempty"`
	Tiers       *PromoteEntities `json:"tiers,omitempty"`
}

// PromoteAPIInstance promotes an API instance into the target environment.
// It hits the same POST /apis endpoint but with a "promote" payload.
func (c *APIInstanceClient) PromoteAPIInstance(ctx context.Context, orgID, envID string, request *PromoteAPIInstanceRequest) (*APIInstance, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal promote request: %w", err)
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
		return nil, fmt.Errorf("failed to promote API instance with status %d: %s", resp.StatusCode, string(body))
	}

	var instance APIInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instance, nil
}

// --- CRUD Operations ---

// CreateAPIInstance creates a new API instance in API Manager.
// Retries up to 5 times with 20s backoff on GatewayNotReadyError,
// which occurs when the gateway is still starting up.
func (c *APIInstanceClient) CreateAPIInstance(ctx context.Context, orgID, envID string, request *CreateAPIInstanceRequest) (*APIInstance, error) {
	const maxRetries = 5
	const retryDelay = 20 * time.Second

	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API instance request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
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

		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			var instance APIInstance
			if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			_ = resp.Body.Close()
			return &instance, nil
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("failed to create API instance with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "GatewayNotReadyError") {
			continue
		}

		return nil, lastErr
	}

	return nil, fmt.Errorf("gateway not ready after %d retries: %w", maxRetries, lastErr)
}

// GetAPIInstance retrieves an API instance by its numeric ID
func (c *APIInstanceClient) GetAPIInstance(ctx context.Context, orgID, envID string, apiID int) (*APIInstance, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, apiID)

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
		return nil, client.NewNotFoundError("API instance")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get API instance with status %d: %s", resp.StatusCode, string(body))
	}

	var instance APIInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instance, nil
}

// UpdateAPIInstance patches an existing API instance
func (c *APIInstanceClient) UpdateAPIInstance(ctx context.Context, orgID, envID string, apiID int, request *UpdateAPIInstanceRequest) (*APIInstance, error) {
	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, apiID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API instance update request: %w", err)
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
		return nil, client.NewNotFoundError("API instance")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update API instance with status %d: %s", resp.StatusCode, string(body))
	}

	var instance APIInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instance, nil
}

// DeleteAPIInstance deletes an API instance by its numeric ID
func (c *APIInstanceClient) DeleteAPIInstance(ctx context.Context, orgID, envID string, apiID int) error {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, apiID)

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
		return fmt.Errorf("failed to delete API instance with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// APIInstanceListResponse wraps the response from the xapi list endpoint.
type APIInstanceListResponse struct {
	Instances []APIInstance `json:"instances"`
	Total     int           `json:"total"`
}

// ListAPIInstances returns all API instances for the given org/environment.
func (c *APIInstanceClient) ListAPIInstances(ctx context.Context, orgID, envID string) ([]APIInstance, error) {
	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

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
		return nil, fmt.Errorf("failed to list API instances with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp APIInstanceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Instances, nil
}
