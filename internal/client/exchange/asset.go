package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// AssetClient provides Exchange asset operations.
// It works with either client_credentials or user-based (password grant) authentication.
type AssetClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewAssetClient creates a new Exchange AssetClient from a provider Config.
// It uses client_credentials authentication (the regular AnypointClient).
// This is the preferred path — no username/password required.
func NewAssetClient(config *client.Config) (*AssetClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &AssetClient{
		BaseURL:    anypointClient.BaseURL,
		Token:      anypointClient.Token,
		HTTPClient: anypointClient.HTTPClient,
	}, nil
}

// --- Domain Models ---

// Asset represents an Exchange asset version (response from GET).
type Asset struct {
	GroupID      string `json:"groupId"`
	AssetID      string `json:"assetId"`
	Version      string `json:"version"`
	MinorVersion string `json:"minorVersion"`
	VersionGroup string `json:"versionGroup"`

	Name         string  `json:"name"`
	Description  string  `json:"description"`
	ContactName  *string `json:"contactName"`
	ContactEmail *string `json:"contactEmail"`
	Manager      *string `json:"manager"`

	Type       string `json:"type"`
	Status     string `json:"status"`
	IsPublic   bool   `json:"isPublic"`
	IsSnapshot bool   `json:"isSnapshot"`

	CreatedDate string `json:"createdDate"`
	UpdatedDate string `json:"updatedDate"`

	Labels       []string      `json:"labels"`
	Categories   []Category    `json:"categories"`
	CustomFields []CustomField `json:"customFields"`
	Files        []AssetFile   `json:"files"`
	Dependencies []Dependency  `json:"dependencies"`
	Instances    []interface{} `json:"instances"`
	Attributes   []interface{} `json:"attributes"`

	Rating        int `json:"rating"`
	NumberOfRates int `json:"numberOfRates"`

	CreatedBy    *CreatedBy    `json:"createdBy"`
	Organization *Organization `json:"organization"`

	ID string `json:"id"`
}

// Label represents a tag label on an asset.
type Label struct {
	Value string `json:"value"`
	Key   string `json:"key,omitempty"`
}

// Category represents a category assignment on an asset.
type Category struct {
	Value []string `json:"value"`
	Key   string   `json:"key"`
}

// CustomField represents a custom field value on an asset.
type CustomField struct {
	Value interface{} `json:"value"`
	Key   string      `json:"key"`
}

// AssetFile represents a file attached to an asset.
type AssetFile struct {
	Classifier  string `json:"classifier"`
	Packaging   string `json:"packaging"`
	DownloadURL string `json:"downloadURL"`
	MD5         string `json:"md5"`
	SHA1        string `json:"sha1"`
	SHA256      string `json:"sha256"`
	MainFile    string `json:"mainFile"`
	IsGenerated bool   `json:"isGenerated"`
}

// Dependency represents an asset dependency.
type Dependency struct {
	GroupID string `json:"groupId"`
	AssetID string `json:"assetId"`
	Version string `json:"version"`
}

// CreatedBy represents the user who created the asset.
type CreatedBy struct {
	ID        string `json:"id"`
	UserName  string `json:"userName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// Organization represents the org that owns the asset.
type Organization struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// --- Request Models ---

// CreateAssetRequest represents the request to publish an Exchange asset.
type CreateAssetRequest struct {
	OrganizationID string
	GroupID        string
	AssetID        string
	Version        string

	Name        string
	Description string
	Type        string // custom, rest-api, graphql-api, etc.
	Status      string // development, published (default: published)

	// File upload (optional for some types)
	FilePath   string // local file path to upload
	Classifier string // raml, oas, custom, wsdl, etc.

	// ExtraFiles holds ADDITIONAL files beyond the primary FilePath, uploaded in
	// the same multipart request. Required by multi-file types such as "policy",
	// which needs (schema.json + metadata.yaml) or (mule-policy.jar +
	// policy-definition.yaml). Empty for all single-file / fileless types.
	ExtraFiles []AssetFileUpload

	// Optional metadata
	APIVersion string // properties.apiVersion
	MainFile   string // properties.mainFile

	// Properties holds arbitrary extra `properties.{key}` form fields (e.g.
	// {"platform": "mulesoft"} for generated MCP assets). APIVersion/MainFile
	// remain dedicated fields for backward compatibility; anything here is
	// written in addition to them.
	Properties map[string]string

	// Optional keywords
	Keywords string // comma-separated

	// InMemoryFiles holds files whose bytes are built at runtime rather than read
	// from a local path (e.g. an MCP bridge's generated mcp-metadata.json). Each is
	// written as its own `files.{classifier}.{ext}` part, exactly like FilePath /
	// ExtraFiles, so the multipart field-name contract is identical for all files.
	InMemoryFiles []AssetInMemoryFile
}

// AssetFileUpload is one additional local file to attach to a publish request.
// Each becomes its own `files.{classifier}.{ext}` multipart part.
type AssetFileUpload struct {
	FilePath   string // local file path to upload
	Classifier string // e.g. metadata, policy-definition
}

// AssetInMemoryFile is a file whose content is already in memory (not on disk).
// Used by generated-asset publishers (e.g. MCP bridge) that construct the file
// content at runtime. The field name derives from FileName's extension exactly
// like the disk-based path: `files.{classifier}.{ext}`.
type AssetInMemoryFile struct {
	Classifier string // e.g. mcp-metadata
	FileName   string // e.g. mcp-metadata.json (extension drives the packaging suffix)
	Content    []byte
}

// UpdateAssetRequest represents the request to update asset metadata.
// These are the only mutable fields after publish.
//
// NOTE: `manager` is deliberately absent. Although the Exchange asset object HAS a
// manager field (see Asset.Manager, which we READ), the metadata PATCH endpoint does
// not accept a manager value: LIVE-VERIFIED (2026-07-22) that PATCH
// {"manager":"<username>"} returns HTTP 403 and {"manager":"<uuid>"} returns HTTP 400
// ("must be equal to one of the allowed values: ,  ... must match exactly one schema
// in oneOf"). There is no supported way to SET it from automation, so it is not a
// writable field here — exposing it only produced apply-time failures.
type UpdateAssetRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	ContactName  *string `json:"contactName,omitempty"`
	ContactEmail *string `json:"contactEmail,omitempty"`
}

// --- CRUD Operations ---

// CreateAsset publishes a new asset version to Exchange.
// API: POST /exchange/api/v2/organizations/{orgId}/assets/{groupId}/{assetId}/{version}
// Content-Type: multipart/form-data
// Uses x-sync-publication: true for synchronous response.
func (c *AssetClient) CreateAsset(ctx context.Context, req *CreateAssetRequest) (*Asset, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/organizations/%s/assets/%s/%s/%s",
		c.BaseURL, req.OrganizationID, req.GroupID, req.AssetID, req.Version)

	body, contentType, err := buildAssetMultipart(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("x-sync-publication", "true")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create asset with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Sync publish returns the asset. Read it back for full details.
	return c.GetAsset(ctx, req.GroupID, req.AssetID, req.Version)
}

// GetAsset retrieves a specific asset version.
// API: GET /exchange/api/v2/assets/{groupId}/{assetId}/{version}
func (c *AssetClient) GetAsset(ctx context.Context, groupID, assetID, version string) (*Asset, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s", c.BaseURL, groupID, assetID, version)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("exchange asset")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get asset with status %d: %s", resp.StatusCode, string(body))
	}

	var asset Asset
	if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &asset, nil
}

// UpdateAsset updates the mutable metadata of an asset (name, description, contact info).
// API: PATCH /exchange/api/v2/assets/{groupId}/{assetId}
func (c *AssetClient) UpdateAsset(ctx context.Context, groupID, assetID string, req *UpdateAssetRequest) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s", c.BaseURL, groupID, assetID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("exchange asset")
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update asset with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteAsset deletes an asset (all versions).
// API: DELETE /exchange/api/v2/assets/{groupId}/{assetId}
func (c *AssetClient) DeleteAsset(ctx context.Context, groupID, assetID string, hardDelete bool) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s", c.BaseURL, groupID, assetID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	// MuleSoft Exchange expects the "x-delete-type" header (NOT "DeleteType").
	// hard-delete permanently removes the asset and frees the groupId/assetId/version
	// for reuse; soft-delete leaves a tombstone that blocks recreation at the same GAV.
	// See: Exchange Experience API "Hard vs Soft Delete" docs.
	if hardDelete {
		req.Header.Set("x-delete-type", "hard-delete")
	} else {
		req.Header.Set("x-delete-type", "soft-delete")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete asset with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteAssetVersion deletes a specific version of an asset.
// API: DELETE /exchange/api/v2/assets/{groupId}/{assetId}/{version}
func (c *AssetClient) DeleteAssetVersion(ctx context.Context, groupID, assetID, version string, hardDelete bool) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s", c.BaseURL, groupID, assetID, version)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	// MuleSoft Exchange expects the "x-delete-type" header (NOT "DeleteType").
	// hard-delete permanently removes the version and frees the groupId/assetId/version
	// for reuse; soft-delete leaves a tombstone that blocks recreation at the same GAV.
	// See: Exchange Experience API "Hard vs Soft Delete" docs.
	if hardDelete {
		req.Header.Set("x-delete-type", "hard-delete")
	} else {
		req.Header.Set("x-delete-type", "soft-delete")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete asset version with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListAssetsRequest contains optional filters for listing assets.
type ListAssetsRequest struct {
	OrganizationID string
	Limit          int
	Offset         int
	Search         string // free-text search query
	Type           string // filter by asset type (rest-api, http-api, etc.)
}

// ListAssets lists assets in the organization with optional filters.
// API: GET /exchange/api/v2/assets?organizationId={orgId}
func (c *AssetClient) ListAssets(ctx context.Context, req *ListAssetsRequest) ([]Asset, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets?organizationId=%s&limit=%d&offset=%d",
		c.BaseURL, req.OrganizationID, req.Limit, req.Offset)

	if req.Search != "" {
		url += "&search=" + req.Search
	}
	if req.Type != "" {
		// Exchange Experience API expects the PLURAL param name `types` for
		// type filtering (verified against dev-portal.mulesoft.com Exchange
		// Experience API spec). The singular `type` is silently ignored by the
		// server, so the filter was a no-op (returned all types). Single value
		// is accepted; the server treats it as a one-element type filter.
		url += "&types=" + req.Type
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list assets with status %d: %s", resp.StatusCode, string(body))
	}

	var assets []Asset
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return assets, nil
}

// exchangeAssetsMaxPageSize is the maximum number of assets the Exchange assets
// endpoint returns in a single request. LIVE-VERIFIED (2026-07-21): limit>250
// returns HTTP 400 with body {"name":"Bad Request","status":400,"message":
// "request/query/limit must be <= 250"}. We request the max per page to minimize
// round-trips.
const exchangeAssetsMaxPageSize = 250

// ListAllAssets lists assets in the organization, following offset pagination to
// completion so callers are never silently truncated at a single page.
//
// The Exchange assets endpoint (GET /exchange/api/v2/assets) pages via limit+offset
// and returns a BARE JSON array — there is NO envelope and NO total-count header
// (LIVE-VERIFIED 2026-07-21). The ONLY reliable end-of-list signal is therefore a
// SHORT page: a response with fewer items than the page size we requested. Offset
// advances correctly with no page overlap, and an out-of-range offset returns HTTP
// 200 with an empty array (which is also "short", so the boundary is covered).
//
// req.Limit is interpreted as an OPTIONAL caller cap on the TOTAL number of assets
// returned (NOT a per-request page size):
//   - req.Limit <= 0 : return ALL matching assets (walk every page to completion).
//   - req.Limit  > 0 : return at most req.Limit assets, paginating as needed and
//     never returning more than the cap.
//
// This MUST paginate. A naive single-request implementation silently truncates any
// org with more matching assets than one page — the same bug class as the roles /
// team-roles / self-managed-gateway pagination fixes. For a data source that a user
// scans to find an asset, a match beyond the first page would simply be invisible.
func (c *AssetClient) ListAllAssets(ctx context.Context, req *ListAssetsRequest) ([]Asset, error) {
	var all []Asset
	offset := 0

	for {
		// Per-request page size: the server max (250), unless a caller cap is set and
		// the remaining budget is smaller, in which case only fetch what's left.
		pageSize := exchangeAssetsMaxPageSize
		if req.Limit > 0 {
			remaining := req.Limit - len(all)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		// Build a fresh per-page request so the caller's struct is never mutated and
		// the search/type filters are forwarded identically on every page.
		pageReq := &ListAssetsRequest{
			OrganizationID: req.OrganizationID,
			Limit:          pageSize,
			Offset:         offset,
			Search:         req.Search,
			Type:           req.Type,
		}

		page, err := c.ListAssets(ctx, pageReq)
		if err != nil {
			return nil, err
		}

		all = append(all, page...)

		// A short page (fewer than we asked for) is the authoritative last page — the
		// endpoint has no envelope/total to consult. An empty page (out-of-range
		// offset) is a special case of "short", so this also terminates the loop.
		if len(page) < pageSize {
			break
		}
		offset += pageSize
	}

	// Defensive trim: if a cap was requested and the server ever ignored our per-page
	// limit (returning more than asked), never hand back more than the cap.
	if req.Limit > 0 && len(all) > req.Limit {
		all = all[:req.Limit]
	}

	return all, nil
}

// --- Status & Tags Operations ---

// UpdateStatusRequest is the body for the status change endpoint.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus changes the lifecycle status of an asset version.
// API: PUT /exchange/api/v2/assets/{groupId}/{assetId}/{version}/status
// Valid transitions: development → published, published → deprecated
func (c *AssetClient) UpdateStatus(ctx context.Context, groupID, assetID, version, status string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/status",
		c.BaseURL, groupID, assetID, version)

	body := UpdateStatusRequest{Status: status}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal status request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("exchange asset")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update asset status with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// TagRequest represents a tag to set on an asset version.
type TagRequest struct {
	Value string `json:"value"`
	Key   string `json:"key,omitempty"`
}

// UpdateTags replaces all tags on an asset version.
// API: PUT /exchange/api/v2/assets/{groupId}/{assetId}/{version}/tags
func (c *AssetClient) UpdateTags(ctx context.Context, groupID, assetID, version string, tags []TagRequest) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/tags",
		c.BaseURL, groupID, assetID, version)

	jsonData, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("exchange asset")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update asset tags with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// --- Portal Pages Operations ---

// PortalPage represents a documentation page in the asset portal.
type PortalPage struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Synthetic bool   `json:"synthetic,omitempty"`
}

// ListPortalPages lists published documentation pages for an asset version.
// API: GET /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/pages
func (c *AssetClient) ListPortalPages(ctx context.Context, groupID, assetID, version string) ([]PortalPage, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/pages",
		c.BaseURL, groupID, assetID, version)

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

	if resp.StatusCode == http.StatusNotFound {
		return []PortalPage{}, nil // No portal yet
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list portal pages with status %d: %s", resp.StatusCode, string(body))
	}

	var pages []PortalPage
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return nil, fmt.Errorf("failed to decode portal pages response: %w", err)
	}

	return pages, nil
}

// GetPortalPageContent retrieves the markdown content of a published portal page.
// API: GET /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/pages/{pagePath}
func (c *AssetClient) GetPortalPageContent(ctx context.Context, groupID, assetID, version, pagePath string) (string, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/pages/%s",
		c.BaseURL, groupID, assetID, version, pagePath)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "text/markdown")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", client.NewNotFoundError("portal page")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get portal page content with status %d: %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read portal page content: %w", err)
	}

	return string(content), nil
}

// CreateDraftPage creates a new empty page in the asset portal draft.
// API: POST /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/draft/pages
// Returns the created page (with the full path including random prefix).
func (c *AssetClient) CreateDraftPage(ctx context.Context, groupID, assetID, version, pageName string) (*PortalPage, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/draft/pages",
		c.BaseURL, groupID, assetID, version)

	body := map[string]string{"pagePath": pageName}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create page request: %w", err)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusConflict {
		// Page already exists (e.g. Exchange auto-provisions a "home" page on every
		// asset version). Wrap the shared ErrConflict sentinel so callers can detect
		// this with client.IsConflict and adopt-and-upsert the existing page instead
		// of failing the whole apply. Message stays "page '<name>' already exists".
		return nil, client.NewConflictError(fmt.Sprintf("page '%s'", pageName))
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create draft page with status %d: %s", resp.StatusCode, string(respBody))
	}

	var page PortalPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode create page response: %w", err)
	}

	return &page, nil
}

// UpdateDraftPageContent sets the markdown content of a draft portal page.
// API: PUT /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/draft/pages/{pagePath}
// This is an upsert — creates the page if it doesn't exist.
func (c *AssetClient) UpdateDraftPageContent(ctx context.Context, groupID, assetID, version, pagePath, content string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/draft/pages/%s",
		c.BaseURL, groupID, assetID, version, pagePath)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBufferString(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/markdown")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("draft portal page")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update draft page content with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteDraftPage deletes a page from the asset portal draft.
// API: DELETE /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/draft/pages/{pagePath}
func (c *AssetClient) DeleteDraftPage(ctx context.Context, groupID, assetID, version, pagePath string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/draft/pages/%s",
		c.BaseURL, groupID, assetID, version, pagePath)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete draft page with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PublishDraft publishes the portal draft to make changes visible.
// API: PUT /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/draft
func (c *AssetClient) PublishDraft(ctx context.Context, groupID, assetID, version string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/draft",
		c.BaseURL, groupID, assetID, version)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to publish portal draft with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ListDraftPages lists pages in the portal draft.
// API: GET /exchange/api/v2/assets/{groupId}/{assetId}/{version}/portal/draft/pages
func (c *AssetClient) ListDraftPages(ctx context.Context, groupID, assetID, version string) ([]PortalPage, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/portal/draft/pages",
		c.BaseURL, groupID, assetID, version)

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

	if resp.StatusCode == http.StatusNotFound {
		return []PortalPage{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list draft pages with status %d: %s", resp.StatusCode, string(body))
	}

	var pages []PortalPage
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return nil, fmt.Errorf("failed to decode draft pages response: %w", err)
	}

	return pages, nil
}

// --- Non-Managed Instance Operations ---

// ExternalInstance represents a non-managed (external) API instance.
type ExternalInstance struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	EndpointURI       string `json:"endpointUri"`
	IsPublic          bool   `json:"isPublic"`
	Type              string `json:"type"`
	OrganizationID    string `json:"organizationId"`
	GroupID           string `json:"groupId"`
	AssetID           string `json:"assetId"`
	ProductAPIVersion string `json:"productApiVersion"`
}

// CreateExternalInstanceRequest represents the request to create an external instance.
type CreateExternalInstanceRequest struct {
	Name        string `json:"name"`
	EndpointURI string `json:"endpointUri"`
	IsPublic    bool   `json:"isPublic"`
}

// UpdateExternalInstanceRequest represents the request to update an external instance.
type UpdateExternalInstanceRequest struct {
	Name        string `json:"name,omitempty"`
	EndpointURI string `json:"endpointUri,omitempty"`
	IsPublic    *bool  `json:"isPublic,omitempty"`
}

// ListExternalInstances returns the non-managed (external) instances attached to an
// asset's version group.
// API: GET /exchange/api/v2/assets/{groupId}/{assetId}/versionGroups/{versionGroup}/instances
//
// This is the AUTHORITATIVE source for external instances. The main asset GET carries an
// `instances` array too, but it is not populated for every asset type — a `custom` asset
// with live external instances reports `instances: []` on the asset GET while this
// endpoint returns them correctly. Reading the asset's inline array made a freshly
// created instance look like it had vanished, failing the apply with
// "Provider produced inconsistent result after apply: .instances: element 0 has vanished".
func (c *AssetClient) ListExternalInstances(ctx context.Context, groupID, assetID, versionGroup string) ([]ExternalInstance, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/versionGroups/%s/instances",
		c.BaseURL, groupID, assetID, versionGroup)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no version group / no instances yet
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list external instances with status %d: %s", resp.StatusCode, string(body))
	}

	var all []ExternalInstance
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("failed to decode external instances response: %w", err)
	}

	// The endpoint returns managed instances too; only "external" ones are ours.
	out := make([]ExternalInstance, 0, len(all))
	for _, inst := range all {
		if inst.Type == "" || inst.Type == "external" {
			out = append(out, inst)
		}
	}
	return out, nil
}

// CreateExternalInstance creates a non-managed external instance for an asset version group.
// API: POST /exchange/api/v2/assets/{groupId}/{assetId}/versionGroups/{versionGroup}/instances/external
func (c *AssetClient) CreateExternalInstance(ctx context.Context, groupID, assetID, versionGroup string, req *CreateExternalInstanceRequest) (*ExternalInstance, error) {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/versionGroups/%s/instances/external",
		c.BaseURL, groupID, assetID, versionGroup)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create instance request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusConflict {
		// An instance with this name already exists on the version group. This happens
		// when a prior asset delete orphaned the instance (external instances are not
		// cascade-deleted with the asset version). Surface a typed error so callers can
		// treat create as idempotent (adopt the existing instance) instead of failing.
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &client.ConflictError{
			StatusCode: http.StatusConflict,
			Message:    fmt.Sprintf("external instance %q already exists: %s", req.Name, string(respBody)),
		}
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create external instance with status %d: %s", resp.StatusCode, string(respBody))
	}

	var instance ExternalInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("failed to decode create instance response: %w", err)
	}

	return &instance, nil
}

// UpdateExternalInstance updates a non-managed external instance.
// API: PATCH /exchange/api/v2/assets/{groupId}/{assetId}/versionGroups/{versionGroup}/instances/external/{instanceId}
func (c *AssetClient) UpdateExternalInstance(ctx context.Context, groupID, assetID, versionGroup, instanceID string, req *UpdateExternalInstanceRequest) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/versionGroups/%s/instances/external/%s",
		c.BaseURL, groupID, assetID, versionGroup, instanceID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal update instance request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("external instance")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update external instance with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteExternalInstance deletes a non-managed external instance.
// API: DELETE /exchange/api/v2/assets/{groupId}/{assetId}/versionGroups/{versionGroup}/instances/external/{instanceId}
func (c *AssetClient) DeleteExternalInstance(ctx context.Context, groupID, assetID, versionGroup, instanceID string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/versionGroups/%s/instances/external/%s",
		c.BaseURL, groupID, assetID, versionGroup, instanceID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete external instance with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// --- Category & Custom Field Operations ---

// SetCategory assigns category values to an asset version.
// API: PUT /exchange/api/v2/assets/{groupId}/{assetId}/{version}/tags/categories/{tagKey}
// Body: array of string values (e.g. ["Finance", "HR"])
func (c *AssetClient) SetCategory(ctx context.Context, groupID, assetID, version, categoryKey string, values []string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/tags/categories/%s",
		c.BaseURL, groupID, assetID, version, categoryKey)

	jsonData, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal category values: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("exchange asset or category")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set category '%s' with status %d: %s", categoryKey, resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteCategory removes a category assignment from an asset version.
// API: DELETE /exchange/api/v2/assets/{groupId}/{assetId}/{version}/tags/categories/{tagKey}
func (c *AssetClient) DeleteCategory(ctx context.Context, groupID, assetID, version, categoryKey string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/tags/categories/%s",
		c.BaseURL, groupID, assetID, version, categoryKey)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already removed
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete category '%s' with status %d: %s", categoryKey, resp.StatusCode, string(respBody))
	}

	return nil
}

// SetCustomField assigns a custom field value to an asset version.
// API: PUT /exchange/api/v2/assets/{groupId}/{assetId}/{version}/tags/fields/{tagKey}
// Body: array of string values (e.g. ["value1", "value2"]) — even single-valued fields use an array.
func (c *AssetClient) SetCustomField(ctx context.Context, groupID, assetID, version, fieldKey string, values []string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/tags/fields/%s",
		c.BaseURL, groupID, assetID, version, fieldKey)

	jsonData, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal custom field values: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return client.NewNotFoundError("exchange asset or custom field")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set custom field '%s' with status %d: %s", fieldKey, resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteCustomField removes a custom field assignment from an asset version.
// API: DELETE /exchange/api/v2/assets/{groupId}/{assetId}/{version}/tags/fields/{tagKey}
func (c *AssetClient) DeleteCustomField(ctx context.Context, groupID, assetID, version, fieldKey string) error {
	url := fmt.Sprintf("%s/exchange/api/v2/assets/%s/%s/%s/tags/fields/%s",
		c.BaseURL, groupID, assetID, version, fieldKey)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already removed
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete custom field '%s' with status %d: %s", fieldKey, resp.StatusCode, string(respBody))
	}

	return nil
}

// --- Helpers ---

// buildAssetMultipart constructs the multipart/form-data body for asset creation.
func buildAssetMultipart(req *CreateAssetRequest) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Required fields
	if err := writer.WriteField("name", req.Name); err != nil {
		return nil, "", fmt.Errorf("failed to write name field: %w", err)
	}

	if req.Description != "" {
		if err := writer.WriteField("description", req.Description); err != nil {
			return nil, "", fmt.Errorf("failed to write description field: %w", err)
		}
	}

	if req.Type != "" {
		if err := writer.WriteField("type", req.Type); err != nil {
			return nil, "", fmt.Errorf("failed to write type field: %w", err)
		}
	}

	if req.Status != "" {
		if err := writer.WriteField("status", req.Status); err != nil {
			return nil, "", fmt.Errorf("failed to write status field: %w", err)
		}
	}

	if req.Keywords != "" {
		if err := writer.WriteField("keywords", req.Keywords); err != nil {
			return nil, "", fmt.Errorf("failed to write keywords field: %w", err)
		}
	}

	// Properties (for API types)
	if req.APIVersion != "" {
		if err := writer.WriteField("properties.apiVersion", req.APIVersion); err != nil {
			return nil, "", fmt.Errorf("failed to write properties.apiVersion field: %w", err)
		}
	}

	if req.MainFile != "" {
		if err := writer.WriteField("properties.mainFile", req.MainFile); err != nil {
			return nil, "", fmt.Errorf("failed to write properties.mainFile field: %w", err)
		}
	}

	// Arbitrary extra properties (e.g. properties.platform for generated MCP assets).
	for k, v := range req.Properties {
		if err := writer.WriteField("properties."+k, v); err != nil {
			return nil, "", fmt.Errorf("failed to write properties.%s field: %w", k, err)
		}
	}

	// File upload (optional — some asset types like "custom" without files are valid).
	// The primary file comes from FilePath/Classifier; multi-file types (e.g. policy)
	// attach further files via ExtraFiles. Each file is written as its own
	// `files.{classifier}.{ext}` part in the SAME multipart request.
	if req.FilePath != "" {
		if err := writeAssetFilePart(writer, req.FilePath, req.Classifier); err != nil {
			return nil, "", err
		}
	}

	for _, f := range req.ExtraFiles {
		if f.FilePath == "" {
			continue
		}
		if err := writeAssetFilePart(writer, f.FilePath, f.Classifier); err != nil {
			return nil, "", err
		}
	}

	// In-memory files (content built at runtime, e.g. generated mcp-metadata.json).
	for _, f := range req.InMemoryFiles {
		if len(f.Content) == 0 {
			continue
		}
		if err := writeAssetFilePartBytes(writer, f.Classifier, f.FileName, f.Content); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}

// writeAssetFilePart writes one local file to the multipart writer as a
// `files.{classifier}.{ext}` part, matching the Exchange publish contract.
// Shared by the primary file and every ExtraFiles entry so field-name logic is
// identical for all files in a multi-file publish.
func writeAssetFilePart(writer *multipart.Writer, filePath, classifier string) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return writeAssetFilePartBytes(writer, classifier, filepath.Base(filePath), fileContent)
}

// writeAssetFilePartBytes writes already-in-memory file content to the multipart
// writer using the identical `files.{classifier}.{ext}` field-name contract as
// the disk-based path. This lets generated assets (e.g. an MCP bridge's runtime
// mcp-metadata.json) publish without staging a temp file. `ext` is derived from
// fileName's extension, falling back to the classifier when there is none.
func writeAssetFilePartBytes(writer *multipart.Writer, classifier, fileName string, content []byte) error {
	if classifier == "" {
		classifier = "custom"
	}

	// The form field name follows the pattern: files.{classifier}.{packaging}
	// where packaging is the file extension (e.g. raml, jar, json, zip, yaml)
	ext := filepath.Ext(fileName)
	if ext != "" {
		ext = ext[1:] // remove leading dot
	} else {
		ext = classifier // fallback to classifier if no extension
	}
	fieldName := fmt.Sprintf("files.%s.%s", classifier, ext)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}
	return nil
}
