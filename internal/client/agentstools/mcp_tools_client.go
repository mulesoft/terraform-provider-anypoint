package agentstools

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

// MCPToolsClient parses a source REST API's Exchange spec (OpenAPI/RAML) into a
// normalized MCP tool list — the read-only engine behind the anypoint_mcp_tools
// data source (Approach D / DS-hybrid). It reuses the tested exchange.AssetClient to
// fetch the asset metadata, downloads the spec file, and delegates to the pure
// ParseAPISpecFile parser. Nothing here mutates the platform.
type MCPToolsClient struct {
	*client.AnypointClient
	Assets *exchange.AssetClient
}

// NewMCPToolsClient builds the tools client and its composed asset sub-client from a
// single Config (sharing one auth round-trip via the token cache).
func NewMCPToolsClient(config *client.Config) (*MCPToolsClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	assets, err := exchange.NewAssetClient(config)
	if err != nil {
		return nil, err
	}
	return &MCPToolsClient{AnypointClient: anypointClient, Assets: assets}, nil
}

// specFilePreference is the classifier preference order for parsing. fat- bundles are
// self-contained (external $refs inlined) so they parse most reliably; plain oas/raml
// are the fallbacks.
var specFilePreference = []string{"fat-oas", "oas", "fat-raml", "raml"}

// GetAssetTools resolves the best API-spec file on the Exchange asset, downloads it,
// and parses it into a deterministic tool list. Returns the tools plus the detected
// spec type ("oas3"/"oas2"/"raml").
func (c *MCPToolsClient) GetAssetTools(ctx context.Context, groupID, assetID, version string) ([]ParsedTool, string, error) {
	asset, err := c.Assets.GetAsset(ctx, groupID, assetID, version)
	if err != nil {
		return nil, "", err
	}
	file := selectSpecFile(asset.Files)
	if file == nil {
		return nil, "", fmt.Errorf("asset %s/%s/%s has no OpenAPI or RAML spec file to parse; declare tools explicitly", groupID, assetID, version)
	}
	data, err := c.downloadFile(ctx, file.DownloadURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download spec (%s): %w", file.Classifier, err)
	}
	return ParseAPISpecFile(file.Classifier, data, file.MainFile)
}

// selectSpecFile picks the highest-preference API-spec file present on the asset.
func selectSpecFile(files []exchange.AssetFile) *exchange.AssetFile {
	byClassifier := map[string]*exchange.AssetFile{}
	classifiers := make([]string, 0, len(files))
	for i := range files {
		f := &files[i]
		if _, seen := byClassifier[f.Classifier]; !seen {
			byClassifier[f.Classifier] = f
			classifiers = append(classifiers, f.Classifier)
		}
	}
	pick := selectSpecFileClassifier(classifiers)
	if pick == "" {
		return nil
	}
	return byClassifier[pick]
}

// selectSpecFileClassifier returns the highest-preference spec classifier present in
// the given set, or "" if none is a parseable API spec.
func selectSpecFileClassifier(classifiers []string) string {
	present := map[string]bool{}
	for _, c := range classifiers {
		present[c] = true
	}
	for _, pref := range specFilePreference {
		if present[pref] {
			return pref
		}
	}
	return ""
}

// downloadFile GETs an Exchange file download URL. Exchange 302-redirects to a
// pre-signed storage URL that REJECTS the Anypoint bearer token; Go's http.Client
// strips the Authorization header automatically on a cross-host redirect (since
// Go 1.8), so a plain authenticated GET works for both the direct and redirected legs.
func (c *MCPToolsClient) downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("download returned status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}
