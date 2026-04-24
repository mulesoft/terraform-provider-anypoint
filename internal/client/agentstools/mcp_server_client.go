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

// MCPServerClient wraps the AnypointClient for MCP Server operations
type MCPServerClient struct {
	*client.AnypointClient
}

// NewMCPServerClient creates a new MCPServerClient
func NewMCPServerClient(config *client.ClientConfig) (*MCPServerClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &MCPServerClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// MCPServer represents an MCP Server instance in API Manager
type MCPServer struct {
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
	Endpoint                  *MCPServerEndpoint     `json:"endpoint,omitempty"`
	Spec                      *MCPServerSpec         `json:"spec,omitempty"`
	Routing                   []MCPServerRoute       `json:"routing,omitempty"`
	Deployment                *MCPServerDeployment   `json:"deployment,omitempty"`
}

// MCPServerEndpoint represents the MCP-specific endpoint configuration
type MCPServerEndpoint struct {
	DeploymentType       string                  `json:"deploymentType,omitempty"`
	MuleVersion4OrAbove  *bool                   `json:"muleVersion4OrAbove"`
	Type                 string                  `json:"type,omitempty"` // "mcp" for MCP servers
	IsCloudHub           *bool                   `json:"isCloudHub"`
	ProxyURI             *string                 `json:"proxyUri"` // MCP proxy URI (e.g., http://0.0.0.0:8081/mcp1)
	ReferencesUserDomain *bool                   `json:"referencesUserDomain"`
	ResponseTimeout      *int                    `json:"responseTimeout"`
	TLSContexts          *MCPServerTLSContexts   `json:"tlsContexts"`
}

// MCPServerTLSContexts holds TLS context references
type MCPServerTLSContexts struct {
	Inbound *MCPServerTLSContext `json:"inbound"`
}

// MCPServerTLSContext represents a single TLS context reference
type MCPServerTLSContext struct {
	SecretGroupID string `json:"secretGroupId,omitempty"`
	TLSID         string `json:"tlsContextId,omitempty"`
}

// MCPServerSpec identifies the Exchange asset backing this MCP server
type MCPServerSpec struct {
	AssetID string `json:"assetId"`
	GroupID string `json:"groupId"`
	Version string `json:"version"`
}

// MCPServerRoute represents routing rules with weighted upstreams
type MCPServerRoute struct {
	Label     string              `json:"label,omitempty"`
	Upstreams []MCPServerUpstream `json:"upstreams"`
	Rules     *MCPServerRules     `json:"rules,omitempty"`
}

// MCPServerUpstream is one backend in a weighted routing set
type MCPServerUpstream struct {
	ID         string                 `json:"id,omitempty"`
	Weight     int                    `json:"weight"`
	URI        string                 `json:"uri,omitempty"`
	Label      string                 `json:"label,omitempty"`
	TLSContext *MCPServerUpstreamTLS  `json:"tlsContext,omitempty"`
}

// MCPServerUpstreamTLS holds TLS context for an upstream backend
type MCPServerUpstreamTLS struct {
	SecretGroupID string `json:"secretGroupId"`
	TLSContextID  string `json:"tlsContextId"`
}

// MCPServerRules are match conditions for a route
type MCPServerRules struct {
	Methods string            `json:"methods,omitempty"`
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPServerDeployment describes where this MCP server is deployed
type MCPServerDeployment struct {
	EnvironmentID  string `json:"environmentId,omitempty"`
	Type           string `json:"type,omitempty"`
	ExpectedStatus string `json:"expectedStatus,omitempty"`
	Overwrite      bool   `json:"overwrite,omitempty"`
	TargetID       string `json:"targetId,omitempty"`
	TargetName     string `json:"targetName,omitempty"`
	GatewayVersion string `json:"gatewayVersion,omitempty"`
}

// GetGatewayInfo fetches gateway details from the Gateway Manager API
func (c *MCPServerClient) GetGatewayInfo(ctx context.Context, orgID, envID, gatewayID string) (*GatewayInfo, error) {
	return GetGatewayInfo(ctx, c.HTTPClient, c.Token, c.BaseURL, orgID, envID, gatewayID)
}

// --- Request Models ---

// CreateMCPServerRequest is the payload sent to create an MCP server
type CreateMCPServerRequest struct {
	Technology     string              `json:"technology"` // Should be "flexGateway"
	ApprovalMethod *string             `json:"approvalMethod"`
	ProviderID     *string             `json:"providerId"`
	EndpointURI    *string             `json:"endpointUri"`
	InstanceLabel  string              `json:"instanceLabel,omitempty"`
	Endpoint       *MCPServerEndpoint  `json:"endpoint,omitempty"`
	Spec           *MCPServerSpec      `json:"spec"`
	Routing        []MCPServerRoute    `json:"routing,omitempty"`
	Deployment     *MCPServerDeployment `json:"deployment,omitempty"`
}

// UpdateMCPServerRequest uses pointer fields to distinguish "not provided" from zero value
type UpdateMCPServerRequest struct {
	Technology    *string              `json:"technology,omitempty"`
	EndpointURI   *string              `json:"endpointUri,omitempty"`
	InstanceLabel *string              `json:"instanceLabel,omitempty"`
	Endpoint      *MCPServerEndpoint   `json:"endpoint,omitempty"`
	Spec          *MCPServerSpec       `json:"spec,omitempty"`
	Routing       []MCPServerRoute     `json:"routing,omitempty"`
	Deployment    *MCPServerDeployment `json:"deployment,omitempty"`
}

// --- CRUD Operations ---

// CreateMCPServer creates a new MCP server instance.
// Uses the /apimanager/xapi/v1 endpoint with MCP-specific configuration.
// Retries up to 5 times with 20s backoff on GatewayNotReadyError.
func (c *MCPServerClient) CreateMCPServer(ctx context.Context, orgID, envID string, request *CreateMCPServerRequest) (*MCPServer, error) {
	const maxRetries = 5
	const retryDelay = 20 * time.Second

	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP server request: %w", err)
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
		req.Header.Set("X-Envoy-Upstream-Rq-Timeout-Ms", "30000")
		req.Header.Set("X-Web-App", "api-manager-ui-lib")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			var server MCPServer
			if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			resp.Body.Close()
			return &server, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("failed to create MCP server with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "GatewayNotReadyError") {
			continue
		}

		return nil, lastErr
	}

	return nil, fmt.Errorf("gateway not ready after %d retries: %w", maxRetries, lastErr)
}

// GetMCPServer retrieves an MCP server by its numeric ID
func (c *MCPServerClient) GetMCPServer(ctx context.Context, orgID, envID string, serverID int) (*MCPServer, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d?includeProxyConfiguration=true&includeTlsContexts=true",
		c.BaseURL, orgID, envID, serverID)

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
		return nil, client.NewNotFoundError("MCP server")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get MCP server with status %d: %s", resp.StatusCode, string(body))
	}

	var server MCPServer
	if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &server, nil
}

// UpdateMCPServer patches an existing MCP server
func (c *MCPServerClient) UpdateMCPServer(ctx context.Context, orgID, envID string, serverID int, request *UpdateMCPServerRequest) (*MCPServer, error) {
	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, serverID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP server update request: %w", err)
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
		return nil, client.NewNotFoundError("MCP server")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update MCP server with status %d: %s", resp.StatusCode, string(body))
	}

	var server MCPServer
	if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &server, nil
}

// DeleteMCPServer deletes an MCP server by its numeric ID
func (c *MCPServerClient) DeleteMCPServer(ctx context.Context, orgID, envID string, serverID int) error {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, serverID)

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
		return fmt.Errorf("failed to delete MCP server with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// MCPServerListResponse wraps the response from the list endpoint
type MCPServerListResponse struct {
	Instances []MCPServer `json:"instances"`
	Total     int         `json:"total"`
}

// ListMCPServers returns all MCP servers for the given org/environment
func (c *MCPServerClient) ListMCPServers(ctx context.Context, orgID, envID string) ([]MCPServer, error) {
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
		return nil, fmt.Errorf("failed to list MCP servers with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp MCPServerListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Instances, nil
}
