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

// NewAssetClientFromUserConfig creates a new Exchange AssetClient using password grant.
// Deprecated: use NewAssetClient with client_credentials instead.
func NewAssetClientFromUserConfig(config *client.UserClientConfig) (*AssetClient, error) {
	userClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &AssetClient{
		BaseURL:    userClient.BaseURL,
		Token:      userClient.Token,
		HTTPClient: userClient.HTTPClient,
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

	// Optional metadata
	APIVersion string // properties.apiVersion
	MainFile   string // properties.mainFile

	// Optional keywords
	Keywords string // comma-separated
}

// UpdateAssetRequest represents the request to update asset metadata.
// These are the only mutable fields after publish.
type UpdateAssetRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	ContactName  *string `json:"contactName,omitempty"`
	ContactEmail *string `json:"contactEmail,omitempty"`
	Manager      *string `json:"manager,omitempty"`
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
	if hardDelete {
		req.Header.Set("DeleteType", "hard-delete")
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
	if hardDelete {
		req.Header.Set("DeleteType", "hard-delete")
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
		url += "&type=" + req.Type
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
		// Page already exists — not an error for our use case
		return nil, fmt.Errorf("page '%s' already exists", pageName)
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

	// File upload (optional — some asset types like "custom" without files are valid)
	if req.FilePath != "" {
		fileContent, err := os.ReadFile(req.FilePath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read file %s: %w", req.FilePath, err)
		}

		fileName := filepath.Base(req.FilePath)
		classifier := req.Classifier
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
			return nil, "", fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := part.Write(fileContent); err != nil {
			return nil, "", fmt.Errorf("failed to write file content: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}
