package agentstools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

// MCPBridgeClient orchestrates the durable public APIs that back an MCP bridge.
//
// An MCP bridge is NOT the same as an anypoint_mcp_server: the mcp_server takes a
// user-supplied Exchange asset (spec is Required input), whereas a bridge GENERATES
// its Exchange asset (spec is platform-computed). The /api/mcp/* one-click
// orchestration API is UI-only (returns HTML), so this client replicates the
// provision sequence against the same durable APIs used by anypoint_exchange_asset,
// anypoint_mcp_server, and anypoint_api_policy:
//
//	CREATE: publish generated Exchange asset (mcp-metadata.json)  →
//	        create APIM xapi instance (technology=flexGateway, endpoint.type=mcp,
//	        routing[] with connection, metadata.generatedBy=mcp_bridge)  →
//	        attach 4 default policies  →  get-after-create readback.
//	UPDATE: republish asset at bumped patch version  →  PATCH xapi instance assetVersion.
//	DELETE: delete policies  →  DELETE instance  →  hard-delete Exchange asset version.
//
// The embedded AnypointClient provides auth/token/HTTPClient/BaseURL. The Assets and
// Policies sub-clients are the SAME tested clients used elsewhere — composed here (no
// import cycle: neither exchange nor apimanagement import agentstools) so the bridge
// reuses their multipart/publish/delete and inbound/outbound policy transport verbatim.
type MCPBridgeClient struct {
	*client.AnypointClient
	Assets   *exchange.AssetClient
	Policies *apimanagement.APIPolicyClient
}

// NewMCPBridgeClient builds an MCP bridge client and its composed sub-clients from a
// single Config. NewAnypointClient caches the token on the Config after the first call,
// so the three constructors share one authentication round-trip.
func NewMCPBridgeClient(config *client.Config) (*MCPBridgeClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	assets, err := exchange.NewAssetClient(config)
	if err != nil {
		return nil, err
	}
	policies, err := apimanagement.NewAPIPolicyClient(config)
	if err != nil {
		return nil, err
	}
	return &MCPBridgeClient{
		AnypointClient: anypointClient,
		Assets:         assets,
		Policies:       policies,
	}, nil
}

// --- Domain Models (mirror the live GET contract of the bridge APIM instance) ---

// MCPBridge is the API Manager instance that backs an MCP bridge. It is read via the
// same api/v1 endpoint as an MCP server, but carries N routes (one per source API),
// each matched by an X-UPSTREAM-NAME header, and a metadata.generatedBy marker.
type MCPBridge struct {
	ID             int                  `json:"id"`
	AssetID        string               `json:"assetId,omitempty"`
	AssetVersion   string               `json:"assetVersion,omitempty"`
	ProductVersion string               `json:"productVersion,omitempty"`
	GroupID        string               `json:"groupId,omitempty"`
	Technology     string               `json:"technology,omitempty"` // "flexGateway"
	EndpointURI    string               `json:"endpointUri,omitempty"`
	InstanceLabel  string               `json:"instanceLabel,omitempty"`
	Status         string               `json:"status,omitempty"`
	EnvironmentID  string               `json:"environmentId,omitempty"`
	Endpoint       *MCPBridgeEndpoint   `json:"endpoint,omitempty"`
	Spec           *MCPBridgeSpec       `json:"spec,omitempty"`
	Routing        []MCPBridgeRoute     `json:"routing,omitempty"`
	Deployment     *MCPBridgeDeployment `json:"deployment,omitempty"`
	Metadata       map[string]string    `json:"metadata,omitempty"` // {"generatedBy":"mcp_bridge"}
}

// MCPBridgeEndpoint is the MCP-specific endpoint block. On the bridge, type is "mcp",
// deploymentType is "HY", and proxyUri is http://0.0.0.0:8081/<basePath>.
type MCPBridgeEndpoint struct {
	Type              string  `json:"type,omitempty"`
	ProxyURI          *string `json:"proxyUri"`
	DeploymentType    string  `json:"deploymentType,omitempty"`
	APIGatewayVersion string  `json:"apiGatewayVersion,omitempty"`
	IsCloudHub        *bool   `json:"isCloudHub"`
	ResponseTimeout   *int    `json:"responseTimeout"`
}

// MCPBridgeSpec identifies the GENERATED Exchange asset backing the bridge. Unlike
// MCPServerSpec (Required user input), this is platform-computed / read back.
type MCPBridgeSpec struct {
	AssetID string `json:"assetId"`
	GroupID string `json:"groupId"`
	Version string `json:"version"`
}

// MCPBridgeRoute is one route (one per source API). On CREATE we send
// upstreams[].uri + connection; the platform materializes upstream records and the
// read-back route carries upstreams[].id + weight only (URI/connection live in the
// upstreams sub-resource — see GetBridgeUpstreams).
type MCPBridgeRoute struct {
	Label     string                   `json:"label,omitempty"`
	Rules     *MCPBridgeRules          `json:"rules,omitempty"`
	Upstreams []MCPBridgeRouteUpstream `json:"upstreams"`
}

// MCPBridgeRouteUpstream is an upstream reference inside a route.
//
//   - On CREATE: {uri, weight, connection} — connection MUST be sent here; the
//     standalone /upstreams POST strips it (LIVE + BFF source).
//   - On READ:   {id, weight} — the platform rewrites routing to reference the
//     materialized upstream by id.
type MCPBridgeRouteUpstream struct {
	ID         string               `json:"id,omitempty"`
	Weight     int                  `json:"weight"`
	URI        string               `json:"uri,omitempty"`
	Connection *MCPBridgeConnection `json:"connection,omitempty"`
}

// MCPBridgeRules are route match conditions. For a bridge, routing is by the
// X-UPSTREAM-NAME header whose value is the source API's label.
type MCPBridgeRules struct {
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPBridgeConnection links an upstream back to the SOURCE REST API's Exchange asset.
// On CREATE only {assetId, groupId, version, label} are needed; the rest are read back.
type MCPBridgeConnection struct {
	Label             string `json:"label,omitempty"`
	AssetID           string `json:"assetId,omitempty"`
	GroupID           string `json:"groupId,omitempty"`
	Version           string `json:"version,omitempty"`
	AssetName         string `json:"assetName,omitempty"`
	AssetType         string `json:"assetType,omitempty"`
	APIVersionID      string `json:"apiVersionId,omitempty"`
	ProductAPIVersion string `json:"productAPIVersion,omitempty"`
}

// MCPBridgeDeployment describes where the bridge is deployed (auto-filled from the
// target gateway; mirror mcp_server's deployment block).
type MCPBridgeDeployment struct {
	EnvironmentID  string `json:"environmentId,omitempty"`
	Type           string `json:"type,omitempty"`
	ExpectedStatus string `json:"expectedStatus,omitempty"`
	TargetID       string `json:"targetId,omitempty"`
	TargetName     string `json:"targetName,omitempty"`
	GatewayVersion string `json:"gatewayVersion,omitempty"`
}

// MCPBridgeUpstreamDetail is one record from the upstreams sub-resource. It carries the
// real backend URI and the connection link to the source REST API.
type MCPBridgeUpstreamDetail struct {
	ID                 string               `json:"id"`
	InstanceUpstreamID string               `json:"instanceUpstreamId,omitempty"`
	URI                string               `json:"uri,omitempty"`
	Label              *string              `json:"label"`
	Connection         *MCPBridgeConnection `json:"connection,omitempty"`
}

// MCPBridgeUpstreamsResponse wraps the upstreams sub-resource listing.
type MCPBridgeUpstreamsResponse struct {
	Total     int                       `json:"total"`
	Upstreams []MCPBridgeUpstreamDetail `json:"upstreams"`
}

// --- Request Models ---

// CreateBridgeInstanceRequest is the body sent to create the bridge's APIM instance.
// technology is fixed "flexGateway", endpoint.type "mcp", and routing[] carries
// connection at create-time (the only durable way to persist it).
type CreateBridgeInstanceRequest struct {
	Spec          *MCPBridgeSpec       `json:"spec"`
	Endpoint      *MCPBridgeEndpoint   `json:"endpoint,omitempty"`
	Technology    string               `json:"technology"`
	InstanceLabel string               `json:"instanceLabel,omitempty"`
	Routing       []MCPBridgeRoute     `json:"routing,omitempty"`
	Deployment    *MCPBridgeDeployment `json:"deployment,omitempty"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
}

// --- MCP metadata (mcp-metadata.json) ---

// MCPBridgeMetadata is the mcp-metadata.json published to the generated Exchange asset.
// The envelope (protocolVersion/transport/capabilities) is fixed; only Tools varies.
// Tools is left as []interface{} because the tool shape (name/description/inputSchema/
// _meta.flexTranscoding) is assembled by the resource layer from its input model — the
// client only serializes the envelope, keeping transport independent of that model.
type MCPBridgeMetadata struct {
	ProtocolVersion string                `json:"protocolVersion"`
	Transport       MCPBridgeTransport    `json:"transport"`
	Capabilities    MCPBridgeCapabilities `json:"capabilities"`
	Tools           []interface{}         `json:"tools"`
}

// MCPBridgeTransport is the fixed transport descriptor.
type MCPBridgeTransport struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// MCPBridgeCapabilities is the fixed capabilities descriptor.
type MCPBridgeCapabilities struct {
	Tools        MCPBridgeToolsCapability `json:"tools"`
	Experimental map[string]interface{}   `json:"experimental"`
}

// MCPBridgeToolsCapability mirrors capabilities.tools.
type MCPBridgeToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// NewMCPBridgeMetadata builds the mcp-metadata.json envelope around the given tools.
// When managed is true, capabilities.experimental.trusted is added (with mcpUrl when
// non-empty), matching the BFF's _build_merged_metadata.
func NewMCPBridgeMetadata(tools []interface{}, managed bool, mcpURL string) *MCPBridgeMetadata {
	if tools == nil {
		tools = []interface{}{}
	}
	experimental := map[string]interface{}{
		"flexTranscoding": map[string]interface{}{"version": "1.0"},
	}
	if managed {
		trusted := map[string]interface{}{"managed": true}
		if mcpURL != "" {
			trusted["mcpUrl"] = mcpURL
		}
		experimental["trusted"] = trusted
	}
	return &MCPBridgeMetadata{
		ProtocolVersion: "2025-03-26",
		Transport:       MCPBridgeTransport{Kind: "streamableHttp", Path: "/mcp"},
		Capabilities: MCPBridgeCapabilities{
			Tools:        MCPBridgeToolsCapability{ListChanged: false},
			Experimental: experimental,
		},
		Tools: tools,
	}
}

// JSON serializes the metadata to bytes for the multipart mcp-metadata.json part.
func (m *MCPBridgeMetadata) JSON() ([]byte, error) {
	return json.Marshal(m)
}

// --- Version helper ---

// BumpPatchVersion increments the patch component of a semver-ish string
// (1.0.0 -> 1.0.1). If the string is not "x.y.z" with a numeric patch, it appends
// ".1" — mirroring the BFF's _bump_patch used by the bridge update path.
func BumpPatchVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) == 3 {
		if patch, err := strconv.Atoi(parts[2]); err == nil {
			return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
		}
	}
	return version + ".1"
}

// --- Gateway ---

// GetGatewayInfo fetches gateway details (id/name/runtimeVersion) to auto-fill the
// bridge's deployment block. Delegates to the shared package-level helper.
func (c *MCPBridgeClient) GetGatewayInfo(ctx context.Context, orgID, envID, gatewayID string) (*GatewayInfo, error) {
	return GetGatewayInfo(ctx, c.HTTPClient, c.Token, c.BaseURL, orgID, envID, gatewayID)
}

// --- Exchange asset (publish / delete) ---

// PublishBridgeAssetInput is the normalized input for publishing the generated MCP
// Exchange asset. The metadata bytes are built in-memory (no temp file on disk).
type PublishBridgeAssetInput struct {
	OrganizationID string
	GroupID        string
	AssetID        string
	Version        string
	Name           string
	Description    string
	MetadataJSON   []byte // the mcp-metadata.json content
	Status         string // "development" or "published"; defaults to "published"
}

// PublishBridgeAsset publishes (or republishes) the generated MCP Exchange asset. It
// reuses the tested exchange multipart publish path with an in-memory mcp-metadata.json
// file part and the type=mcp / properties.platform=mulesoft form fields the platform
// expects for a generated bridge asset.
func (c *MCPBridgeClient) PublishBridgeAsset(ctx context.Context, in *PublishBridgeAssetInput) (*exchange.Asset, error) {
	status := in.Status
	if status == "" {
		status = "published"
	}
	req := &exchange.CreateAssetRequest{
		OrganizationID: in.OrganizationID,
		GroupID:        in.GroupID,
		AssetID:        in.AssetID,
		Version:        in.Version,
		Name:           in.Name,
		Description:    in.Description,
		Type:           "mcp",
		Status:         status,
		Properties:     map[string]string{"platform": "mulesoft"},
		InMemoryFiles: []exchange.AssetInMemoryFile{
			{Classifier: "mcp-metadata", FileName: "mcp-metadata.json", Content: in.MetadataJSON},
		},
	}
	return c.Assets.CreateAsset(ctx, req)
}

// DeleteBridgeAssetVersion hard-deletes the generated Exchange asset version, freeing
// the GAV for reuse. Reuses the tested exchange delete path (x-delete-type: hard-delete).
func (c *MCPBridgeClient) DeleteBridgeAssetVersion(ctx context.Context, groupID, assetID, version string) error {
	return c.Assets.DeleteAssetVersion(ctx, groupID, assetID, version, true)
}

// --- APIM instance CRUD (bridge-specific transport, mirrors mcp_server_client) ---

// bridgeGatewayRetryDelay is the backoff between CreateBridgeInstance retries when the
// gateway is not yet ready. It is a package var (not a const) SOLELY so hermetic tests
// can shrink it; production always uses 20s, matching CreateMCPServer.
var bridgeGatewayRetryDelay = 20 * time.Second

// CreateBridgeInstance creates the bridge's API Manager instance. Uses the
// /apimanager/xapi/v1 endpoint and retries up to 5 times with 20s backoff on
// GatewayNotReadyError (the gateway may still be spinning up), identical to
// CreateMCPServer.
func (c *MCPBridgeClient) CreateBridgeInstance(ctx context.Context, orgID, envID string, request *CreateBridgeInstanceRequest) (*MCPBridge, error) {
	const maxRetries = 5
	retryDelay := bridgeGatewayRetryDelay

	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis", c.BaseURL, orgID, envID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP bridge request: %w", err)
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
			var bridge MCPBridge
			if err := json.NewDecoder(resp.Body).Decode(&bridge); err != nil {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			_ = resp.Body.Close()
			return &bridge, nil
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("failed to create MCP bridge with status %d: %s", resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "GatewayNotReadyError") {
			continue
		}

		return nil, lastErr
	}

	return nil, fmt.Errorf("gateway not ready after %d retries: %w", maxRetries, lastErr)
}

// GetBridge retrieves a bridge APIM instance by numeric ID, including proxy config and
// TLS contexts (same endpoint/params as GetMCPServer).
func (c *MCPBridgeClient) GetBridge(ctx context.Context, orgID, envID string, bridgeID int) (*MCPBridge, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d?includeProxyConfiguration=true&includeTlsContexts=true",
		c.BaseURL, orgID, envID, bridgeID)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("MCP bridge")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get MCP bridge with status %d: %s", resp.StatusCode, string(body))
	}

	var bridge MCPBridge
	if err := json.NewDecoder(resp.Body).Decode(&bridge); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &bridge, nil
}

// GetBridgeUpstreams retrieves the upstreams sub-resource, which holds each backend URI
// and the connection link to the source REST API (the routing block only has ids).
func (c *MCPBridgeClient) GetBridgeUpstreams(ctx context.Context, orgID, envID string, bridgeID int) ([]MCPBridgeUpstreamDetail, error) {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/upstreams",
		c.BaseURL, orgID, envID, bridgeID)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("MCP bridge upstreams")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get MCP bridge upstreams with status %d: %s", resp.StatusCode, string(body))
	}

	var out MCPBridgeUpstreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return out.Upstreams, nil
}

// UpdateBridgeAssetVersion moves the bridge instance to a new (bumped) Exchange asset
// version. The legacy /api/v1 PATCH does NOT move assetVersion — only the xapi/v1 PATCH
// with {assetVersion} and ?checkAutomatedPolicies=true does (LIVE + BFF source).
func (c *MCPBridgeClient) UpdateBridgeAssetVersion(ctx context.Context, orgID, envID string, bridgeID int, assetVersion string) (*MCPBridge, error) {
	url := fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis/%d?checkAutomatedPolicies=true",
		c.BaseURL, orgID, envID, bridgeID)

	payload := map[string]string{"assetVersion": assetVersion}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("MCP bridge")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update MCP bridge with status %d: %s", resp.StatusCode, string(body))
	}

	var bridge MCPBridge
	if err := json.NewDecoder(resp.Body).Decode(&bridge); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &bridge, nil
}

// DeleteBridgeInstance deletes the bridge's APIM instance (cascades its upstreams). Uses
// the api/v1 endpoint (same as DeleteMCPServer). The generated Exchange asset is removed
// separately via DeleteBridgeAssetVersion.
func (c *MCPBridgeClient) DeleteBridgeInstance(ctx context.Context, orgID, envID string, bridgeID int) error {
	url := fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d", c.BaseURL, orgID, envID, bridgeID)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // already deleted
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete MCP bridge with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// MCPBridgeListResponse wraps the list endpoint response.
type MCPBridgeListResponse struct {
	Instances []MCPBridge `json:"instances"`
	Total     int         `json:"total"`
}

// ListBridges returns all API Manager instances marked as MCP bridges
// (metadata.generatedBy == "mcp_bridge") in the given org/environment. Used by the
// plural data source and by import to distinguish bridges from plain MCP servers.
func (c *MCPBridgeClient) ListBridges(ctx context.Context, orgID, envID string) ([]MCPBridge, error) {
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list MCP bridges with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp MCPBridgeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	bridges := make([]MCPBridge, 0, len(listResp.Instances))
	for _, b := range listResp.Instances {
		if b.Metadata != nil && b.Metadata["generatedBy"] == "mcp_bridge" {
			bridges = append(bridges, b)
		}
	}
	return bridges, nil
}
