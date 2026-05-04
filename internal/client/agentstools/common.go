package agentstools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// GatewayInfo holds the subset of gateway details needed to auto-build deployment
type GatewayInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	RuntimeVersion string `json:"runtimeVersion"`
}

// GetGatewayInfo fetches gateway details from the Gateway Manager API
// This is a shared method used by both AgentInstanceClient and MCPServerClient
func GetGatewayInfo(ctx context.Context, httpClient *http.Client, token, baseURL, orgID, envID, gatewayID string) (*GatewayInfo, error) {
	url := fmt.Sprintf("%s/gatewaymanager/xapi/v1/organizations/%s/environments/%s/gateways/%s",
		baseURL, orgID, envID, gatewayID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

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
