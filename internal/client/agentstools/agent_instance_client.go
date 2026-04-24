package agentstools

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

// AgentInstanceClient wraps the AnypointClient for Agent Instance operations
type AgentInstanceClient struct {
	*client.AnypointClient
}

// NewAgentInstanceClient creates a new AgentInstanceClient
func NewAgentInstanceClient(config *client.ClientConfig) (*AgentInstanceClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &AgentInstanceClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// AgentInstance represents an Agent instance in API Manager (similar to API Instance)
type AgentInstance struct {
	ID                        int                      `json:"id"`
	AssetID                   string                   `json:"assetId,omitempty"`
	AssetVersion              string                   `json:"assetVersion,omitempty"`
	ProductVersion            string                   `json:"productVersion,omitempty"`
	GroupID                   string                   `json:"groupId,omitempty"`
	Technology                string                   `json:"technology,omitempty"`
	EndpointURI               string                   `json:"endpointUri,omitempty"`
	ProviderID                *string                  `json:"providerId,omitempty"`
	InstanceLabel             string                   `json:"instanceLabel,omitempty"`
	ApprovalMethod            string                   `json:"approvalMethod,omitempty"`
	Status                    string                   `json:"status,omitempty"`
	EnvironmentID             string                   `json:"environmentId,omitempty"`
	AutodiscoveryInstanceName string                   `json:"autodiscoveryInstanceName,omitempty"`
	Endpoint                  *AgentInstanceEndpoint   `json:"endpoint,omitempty"`
	Spec                      *AgentInstanceSpec       `json:"spec,omitempty"`
	Routing                   []AgentInstanceRoute     `json:"routing,omitempty"`
	Deployment                *AgentInstanceDeployment `json:"deployment,omitempty"`
}

// AgentInstanceEndpoint represents the endpoint/proxy configuration for agents
type AgentInstanceEndpoint struct {
	DeploymentType       string                    `json:"deploymentType,omitempty"`
	MuleVersion4OrAbove  *bool                     `json:"muleVersion4OrAbove"`
	Type                 string                    `json:"type,omitempty"`
	IsCloudHub           *bool                     `json:"isCloudHub"`
	URI                  *string                   `json:"uri,omitempty"`
	ProxyURI             *string                   `json:"proxyUri"`
	ReferencesUserDomain *bool                     `json:"referencesUserDomain"`
	ResponseTimeout      *int                      `json:"responseTimeout"`
	TLSContexts          *AgentInstanceTLSContexts `json:"tlsContexts"`
}

// AgentInstanceTLSContexts holds TLS context references
type AgentInstanceTLSContexts struct {
	Inbound *AgentInstanceTLSContext `json:"inbound"`
}

// AgentInstanceTLSContext represents a single TLS context reference
type AgentInstanceTLSContext struct {
	SecretGroupID string `json:"secretGroupId,omitempty"`
	TLSID         string `json:"tlsContextId,omitempty"`
}

// AgentInstanceSpec identifies the Exchange asset backing this agent instance
type AgentInstanceSpec struct {
	AssetID string `json:"assetId"`
	GroupID string `json:"groupId"`
	Version string `json:"version"`
}

// AgentInstanceRoute represents a single routing rule with weighted upstreams
type AgentInstanceRoute struct {
	Label     string                  `json:"label,omitempty"`
	Upstreams []AgentInstanceUpstream `json:"upstreams"`
	Rules     *AgentInstanceRules     `json:"rules,omitempty"`
}

// AgentInstanceUpstream is one backend in a weighted routing set
type AgentInstanceUpstream struct {
	ID         string                    `json:"id,omitempty"`
	Weight     int                       `json:"weight"`
	URI        string                    `json:"uri,omitempty"`
	Label      string                    `json:"label,omitempty"`
	TLSContext *AgentInstanceUpstreamTLS `json:"tlsContext,omitempty"`
}

// AgentInstanceUpstreamTLS holds TLS context for an upstream backend
type AgentInstanceUpstreamTLS struct {
	SecretGroupID string `json:"secretGroupId"`
	TLSContextID  string `json:"tlsContextId"`
}

// AgentInstanceRules are match conditions for a route
type AgentInstanceRules struct {
	Methods string            `json:"methods,omitempty"`
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// AgentInstanceDeployment describes where this agent is deployed
type AgentInstanceDeployment struct {
	EnvironmentID  string `json:"environmentId,omitempty"`
	Type           string `json:"type,omitempty"`
	ExpectedStatus string `json:"expectedStatus,omitempty"`
	Overwrite      bool   `json:"overwrite,omitempty"`
	TargetID       string `json:"targetId,omitempty"`
	TargetName     string `json:"targetName,omitempty"`
	GatewayVersion string `json:"gatewayVersion,omitempty"`
}

// GetGatewayInfo fetches gateway details from the Gateway Manager API
func (c *AgentInstanceClient) GetGatewayInfo(ctx context.Context, orgID, envID, gatewayID string) (*GatewayInfo, error) {
	return GetGatewayInfo(ctx, c.HTTPClient, c.Token, c.BaseURL, orgID, envID, gatewayID)
}

// --- Request Models ---

// CreateAgentInstanceRequest is the payload sent to create an agent instance
type CreateAgentInstanceRequest struct {
	Technology     string                   `json:"technology"`
	ApprovalMethod *string                  `json:"approvalMethod"`
	ProviderID     *string                  `json:"providerId"`
	EndpointURI    *string                  `json:"endpointUri"`
	InstanceLabel  string                   `json:"instanceLabel,omitempty"`
	Endpoint       *AgentInstanceEndpoint   `json:"endpoint,omitempty"`
	Spec           *AgentInstanceSpec       `json:"spec"`
	Routing        []AgentInstanceRoute     `json:"routing,omitempty"`
	Deployment     *AgentInstanceDeployment `json:"deployment,omitempty"`
}

// UpdateAgentInstanceRequest uses pointer fields to distinguish "not provided" from zero value
type UpdateAgentInstanceRequest struct {
	Technology    *string                  `json:"technology,omitempty"`
	EndpointURI   *string                  `json:"endpointUri,omitempty"`
	InstanceLabel *string                  `json:"instanceLabel,omitempty"`
	Endpoint      *AgentInstanceEndpoint   `json:"endpoint,omitempty"`
	Spec          *AgentInstanceSpec       `json:"spec,omitempty"`
	Routing       []AgentInstanceRoute     `json:"routing,omitempty"`
	Deployment    *AgentInstanceDeployment `json:"deployment,omitempty"`
}

// --- CRUD Operations ---

// CreateAgentInstance creates a new agent instance in API Manager.
// Uses the /apimanager/xapi/v1 endpoint (same as UI).
// Retries up to 5 times with 20s backoff on GatewayNotReadyError.
func (c *AgentInstanceClient) CreateAgentInstance(ctx context.Context, orgID, envID string, request *CreateAgentInstanceRequest) (*AgentInstance, error) {
	const maxRetries = 5
	const retryDelay = 20 * time.Second

	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent instance request: %w", err)
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
			var instance AgentInstance
			if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			resp.Body.Close()
			return &instance, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("failed to create agent instance with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "GatewayNotReadyError") {
			continue
		}

		return nil, lastErr
	}

	return nil, fmt.Errorf("gateway not ready after %d retries: %w", maxRetries, lastErr)
}

// GetAgentInstance retrieves an agent instance by its numeric ID
func (c *AgentInstanceClient) GetAgentInstance(ctx context.Context, orgID, envID string, agentID int) (*AgentInstance, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d?includeProxyConfiguration=true&includeTlsContexts=true",
		c.BaseURL, orgID, envID, agentID)

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
		return nil, client.NewNotFoundError("Agent instance")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get agent instance with status %d: %s", resp.StatusCode, string(body))
	}

	var instance AgentInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instance, nil
}

// UpdateAgentInstance patches an existing agent instance
func (c *AgentInstanceClient) UpdateAgentInstance(ctx context.Context, orgID, envID string, agentID int, request *UpdateAgentInstanceRequest) (*AgentInstance, error) {
	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, agentID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent instance update request: %w", err)
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
		return nil, client.NewNotFoundError("Agent instance")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update agent instance with status %d: %s", resp.StatusCode, string(body))
	}

	var instance AgentInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &instance, nil
}

// DeleteAgentInstance deletes an agent instance by its numeric ID
func (c *AgentInstanceClient) DeleteAgentInstance(ctx context.Context, orgID, envID string, agentID int) error {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, agentID)

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
		return fmt.Errorf("failed to delete agent instance with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AgentInstanceListResponse wraps the response from the list endpoint
type AgentInstanceListResponse struct {
	Instances []AgentInstance `json:"instances"`
	Total     int             `json:"total"`
}

// ListAgentInstances returns all agent instances for the given org/environment
func (c *AgentInstanceClient) ListAgentInstances(ctx context.Context, orgID, envID string) ([]AgentInstance, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

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
		return nil, fmt.Errorf("failed to list agent instances with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp AgentInstanceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Instances, nil
}
