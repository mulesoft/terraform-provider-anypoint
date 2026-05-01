package apimanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// APIPolicyClient wraps the AnypointClient for API Policy operations
type APIPolicyClient struct {
	*client.AnypointClient
}

// NewAPIPolicyClient creates a new APIPolicyClient
func NewAPIPolicyClient(config *client.Config) (*APIPolicyClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &APIPolicyClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

// APIPolicy represents an applied policy on an API instance
type APIPolicy struct {
	ID                int                    `json:"id"`
	PolicyTemplateID  string                 `json:"policyTemplateId,omitempty"`
	GroupID           string                 `json:"groupId,omitempty"`
	AssetID           string                 `json:"assetId,omitempty"`
	AssetVersion      string                 `json:"assetVersion,omitempty"`
	Label             string                 `json:"label,omitempty"`
	Order             int                    `json:"order"`
	Disabled          bool                   `json:"disabled"`
	PointcutData      interface{}            `json:"pointcutData"`
	ConfigurationData map[string]interface{} `json:"configurationData,omitempty"`
	APIID             int                    `json:"apiId,omitempty"`
	UpstreamIDs       []string               `json:"upstreamIds,omitempty"`
	Audit             *PolicyAudit           `json:"audit,omitempty"`
}

// PolicyAudit contains created/updated timestamps
type PolicyAudit struct {
	Created *PolicyAuditEntry `json:"created,omitempty"`
	Updated *PolicyAuditEntry `json:"updated,omitempty"`
}

// PolicyAuditEntry is a single audit timestamp
type PolicyAuditEntry struct {
	Date string `json:"date,omitempty"`
}

// --- Request Models ---

// CreateAPIPolicyRequest is the payload for creating a policy
type CreateAPIPolicyRequest struct {
	ConfigurationData map[string]interface{} `json:"configurationData"`
	APIVersionID      int                    `json:"apiVersionId,omitempty"`
	PolicyTemplateID  int                    `json:"policyTemplateId,omitempty"`
	GroupID           string                 `json:"groupId"`
	AssetID           string                 `json:"assetId"`
	AssetVersion      string                 `json:"assetVersion"`
	Label             string                 `json:"label,omitempty"`
	PointcutData      interface{}            `json:"pointcutData"`
	Order             *int                   `json:"order,omitempty"`
	Disabled          *bool                  `json:"disabled,omitempty"`
	UpstreamIDs       []string               `json:"upstreamIds,omitempty"`
}

// UpdateAPIPolicyRequest is the payload for updating an inbound policy
type UpdateAPIPolicyRequest struct {
	ConfigurationData map[string]interface{} `json:"configurationData,omitempty"`
	AssetVersion      string                 `json:"assetVersion,omitempty"`
	Label             string                 `json:"label,omitempty"`
	PointcutData      interface{}            `json:"pointcutData,omitempty"`
	Order             *int                   `json:"order,omitempty"`
	Disabled          *bool                  `json:"disabled,omitempty"`
	UpstreamIDs       []string               `json:"upstreamIds,omitempty"`
}

// CreateOutboundAPIPolicyRequest is the payload for creating an outbound policy.
// The xapi/v1 outbound endpoint does NOT accept pointcutData, order, or disabled.
type CreateOutboundAPIPolicyRequest struct {
	ConfigurationData map[string]interface{} `json:"configurationData"`
	GroupID           string                 `json:"groupId"`
	AssetID           string                 `json:"assetId"`
	AssetVersion      string                 `json:"assetVersion"`
	Label             string                 `json:"label,omitempty"`
	UpstreamIDs       []string               `json:"upstreamIds,omitempty"`
}

// UpdateOutboundAPIPolicyRequest is the payload for updating an outbound policy.
// The xapi/v1 outbound endpoint does NOT accept pointcutData, order, or disabled.
type UpdateOutboundAPIPolicyRequest struct {
	ConfigurationData map[string]interface{} `json:"configurationData,omitempty"`
	AssetVersion      string                 `json:"assetVersion,omitempty"`
	Label             string                 `json:"label,omitempty"`
	UpstreamIDs       []string               `json:"upstreamIds,omitempty"`
}

// --- CRUD Operations ---

func (c *APIPolicyClient) basePath(orgID, envID string, apiID int) string {
	return fmt.Sprintf("%s/apimanager/api/v1/organizations/%s/environments/%s/apis/%d/policies",
		c.BaseURL, orgID, envID, apiID)
}

func (c *APIPolicyClient) outboundBasePath(orgID, envID string, apiID int) string {
	return fmt.Sprintf("%s/apimanager/xapi/v1/organizations/%s/environments/%s/apis/%d/policies/outbound-policies",
		c.BaseURL, orgID, envID, apiID)
}

// --- internal helpers ---

func (c *APIPolicyClient) doCreatePolicy(ctx context.Context, createURL, orgID, envID string, request *CreateAPIPolicyRequest) (*APIPolicy, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", createURL, bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("failed to create policy with status %d: %s", resp.StatusCode, string(body))
	}
	var policy APIPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &policy, nil
}

func (c *APIPolicyClient) doGetPolicy(ctx context.Context, getURL, orgID, envID string) (*APIPolicy, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
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
		return nil, client.NewNotFoundError("policy")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get policy with status %d: %s", resp.StatusCode, string(body))
	}
	var policy APIPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &policy, nil
}

func (c *APIPolicyClient) doUpdatePolicy(ctx context.Context, updateURL, orgID, envID string, request *UpdateAPIPolicyRequest) (*APIPolicy, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy update request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "PATCH", updateURL, bytes.NewBuffer(jsonData))
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
		return nil, client.NewNotFoundError("policy")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update policy with status %d: %s", resp.StatusCode, string(body))
	}
	var policy APIPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &policy, nil
}

func (c *APIPolicyClient) doDeletePolicy(ctx context.Context, deleteURL, orgID, envID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
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
		return fmt.Errorf("failed to delete policy with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// --- Public CRUD: inbound policies (api/v1) ---

// CreateAPIPolicy applies a policy to an API instance
func (c *APIPolicyClient) CreateAPIPolicy(ctx context.Context, orgID, envID string, apiID int, request *CreateAPIPolicyRequest) (*APIPolicy, error) {
	return c.doCreatePolicy(ctx, c.basePath(orgID, envID, apiID)+"?allowDuplicated=true", orgID, envID, request)
}

// GetAPIPolicy retrieves a single policy by ID
func (c *APIPolicyClient) GetAPIPolicy(ctx context.Context, orgID, envID string, apiID, policyID int) (*APIPolicy, error) {
	return c.doGetPolicy(ctx, fmt.Sprintf("%s/%d", c.basePath(orgID, envID, apiID), policyID), orgID, envID)
}

// UpdateAPIPolicy patches an existing policy
func (c *APIPolicyClient) UpdateAPIPolicy(ctx context.Context, orgID, envID string, apiID, policyID int, request *UpdateAPIPolicyRequest) (*APIPolicy, error) {
	return c.doUpdatePolicy(ctx, fmt.Sprintf("%s/%d", c.basePath(orgID, envID, apiID), policyID), orgID, envID, request)
}

// DeleteAPIPolicy removes a policy from an API instance
func (c *APIPolicyClient) DeleteAPIPolicy(ctx context.Context, orgID, envID string, apiID, policyID int) error {
	return c.doDeletePolicy(ctx, fmt.Sprintf("%s/%d", c.basePath(orgID, envID, apiID), policyID), orgID, envID)
}

// --- internal helpers for outbound (separate structs, no pointcutData/order/disabled) ---

func (c *APIPolicyClient) doCreateOutboundPolicy(ctx context.Context, createURL, orgID, envID string, request *CreateOutboundAPIPolicyRequest) (*APIPolicy, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal outbound policy request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", createURL, bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("failed to create policy with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// The outbound-policies endpoint may return either a single object or an array.
	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '[' {
		var policies []APIPolicy
		if err := json.Unmarshal(body, &policies); err != nil {
			return nil, fmt.Errorf("failed to decode array response: %w", err)
		}
		if len(policies) == 0 {
			return nil, fmt.Errorf("outbound policy create returned empty array")
		}
		return &policies[0], nil
	}

	var policy APIPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &policy, nil
}

func (c *APIPolicyClient) doUpdateOutboundPolicy(ctx context.Context, updateURL, orgID, envID string, request *UpdateOutboundAPIPolicyRequest) (*APIPolicy, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal outbound policy update request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "PATCH", updateURL, bytes.NewBuffer(jsonData))
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
		return nil, client.NewNotFoundError("policy")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update policy with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '[' {
		var policies []APIPolicy
		if err := json.Unmarshal(body, &policies); err != nil {
			return nil, fmt.Errorf("failed to decode array response: %w", err)
		}
		if len(policies) == 0 {
			return nil, fmt.Errorf("outbound policy update returned empty array")
		}
		return &policies[0], nil
	}

	var policy APIPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &policy, nil
}

// --- Public CRUD: outbound policies (xapi/v1) ---

// CreateOutboundAPIPolicy applies an outbound policy to an API instance.
// Uses CreateOutboundAPIPolicyRequest which omits pointcutData, order, and disabled.
func (c *APIPolicyClient) CreateOutboundAPIPolicy(ctx context.Context, orgID, envID string, apiID int, request *CreateOutboundAPIPolicyRequest) (*APIPolicy, error) {
	return c.doCreateOutboundPolicy(ctx, c.outboundBasePath(orgID, envID, apiID)+"?allowDuplicated=true", orgID, envID, request)
}

// GetOutboundAPIPolicy retrieves a single outbound policy by ID
func (c *APIPolicyClient) GetOutboundAPIPolicy(ctx context.Context, orgID, envID string, apiID, policyID int) (*APIPolicy, error) {
	return c.doGetPolicy(ctx, fmt.Sprintf("%s/%d", c.outboundBasePath(orgID, envID, apiID), policyID), orgID, envID)
}

// UpdateOutboundAPIPolicy patches an existing outbound policy.
// Uses UpdateOutboundAPIPolicyRequest which omits pointcutData, order, and disabled.
func (c *APIPolicyClient) UpdateOutboundAPIPolicy(ctx context.Context, orgID, envID string, apiID, policyID int, request *UpdateOutboundAPIPolicyRequest) (*APIPolicy, error) {
	return c.doUpdateOutboundPolicy(ctx, fmt.Sprintf("%s/%d", c.outboundBasePath(orgID, envID, apiID), policyID), orgID, envID, request)
}

// DeleteOutboundAPIPolicy removes an outbound policy from an API instance
func (c *APIPolicyClient) DeleteOutboundAPIPolicy(ctx context.Context, orgID, envID string, apiID, policyID int) error {
	return c.doDeletePolicy(ctx, fmt.Sprintf("%s/%d", c.outboundBasePath(orgID, envID, apiID), policyID), orgID, envID)
}

// --- Known Policy Registry ---

// PolicyInfo holds the Exchange coordinates for a known policy type.
type PolicyInfo struct {
	GroupID        string
	AssetID        string
	DefaultVersion string
	// SupportedTechnologies lists the runtime technologies this policy supports
	// (e.g. []string{"flexGateway"}, []string{"mule4"}). Empty means all technologies.
	SupportedTechnologies []string
	// InboundPolicy marks standard inbound policies (api/v1 endpoint).
	InboundPolicy bool
	// OutboundPolicy marks policies that use the xapi/v1 outbound-policies endpoint
	// and require upstreamIds in the payload.
	OutboundPolicy bool
}

// KnownPolicies maps a human-readable policy type name to its Exchange coordinates.
// Users can specify policy_type instead of group_id + asset_id.
var KnownPolicies = map[string]PolicyInfo{
	"rate-limiting":                            {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "rate-limiting", DefaultVersion: "1.4.1", InboundPolicy: true},
	"rate-limiting-sla-based":                  {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "rate-limiting-sla-based", DefaultVersion: "1.3.1", InboundPolicy: true},
	"spike-control":                            {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "spike-control", DefaultVersion: "1.2.2", InboundPolicy: true},
	"ip-blocklist":                             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "ip-blocklist", DefaultVersion: "1.1.2", InboundPolicy: true},
	"ip-allowlist":                             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "ip-allowlist", DefaultVersion: "1.1.2", InboundPolicy: true},
	"jwt-validation":                           {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "jwt-validation", DefaultVersion: "0.12.0", InboundPolicy: true},
	"client-id-enforcement":                    {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "client-id-enforcement", DefaultVersion: "1.3.3", InboundPolicy: true},
	"cors":                                     {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "cors", DefaultVersion: "1.3.2", InboundPolicy: true},
	"json-threat-protection":                   {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "json-threat-protection", DefaultVersion: "1.2.1", InboundPolicy: true},
	"xml-threat-protection":                    {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "xml-threat-protection", DefaultVersion: "1.2.1", InboundPolicy: true, SupportedTechnologies: []string{"mule4"}},
	"message-logging":                          {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "message-logging", DefaultVersion: "2.0.2", InboundPolicy: true},
	"header-injection":                         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "header-injection", DefaultVersion: "1.3.2", InboundPolicy: true},
	"header-removal":                           {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "header-removal", DefaultVersion: "1.1.2", InboundPolicy: true},
	"native-ext-authz":                         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "native-ext-authz", DefaultVersion: "1.2.1", InboundPolicy: true},
	"native-ext-proc":                          {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "native-ext-proc", DefaultVersion: "1.1.1", InboundPolicy: true},
	"sse-logging":                              {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "sse-logging", DefaultVersion: "1.0.1", InboundPolicy: true},
	"response-timeout":                         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "response-timeout", DefaultVersion: "1.0.1", InboundPolicy: true},
	"stream-idle-timeout":                      {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "stream-idle-timeout", DefaultVersion: "1.0.1", InboundPolicy: true},
	"health-check":                             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "health-check", DefaultVersion: "1.0.1", InboundPolicy: true},
	"http-caching":                             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "http-caching", DefaultVersion: "1.1.1", InboundPolicy: true},
	"oauth2-token-introspection":               {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "oauth2-token-introspection", DefaultVersion: "1.0.1", InboundPolicy: true},
	"access-block":                             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "access-block", DefaultVersion: "1.0.0", InboundPolicy: true},
	"http-basic-authentication":                {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "http-basic-authentication", DefaultVersion: "1.3.2", InboundPolicy: true},
	"ldap-authentication":                      {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "ldap-authentication", DefaultVersion: "1.4.1", InboundPolicy: true},
	"agent-connection-telemetry":               {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "agent-connection-telemetry", DefaultVersion: "1.0.0", InboundPolicy: true},
	"tracing":                                  {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "tracing", DefaultVersion: "1.1.1", InboundPolicy: true},
	"injection-protection":                     {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "injection-protection", DefaultVersion: "1.0.0", InboundPolicy: true},
	"dataweave-request-filter":                 {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "dataweave-request-filter", DefaultVersion: "1.0.0", InboundPolicy: true},
	"body-transformation":                      {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "body-transformation", DefaultVersion: "1.0.0-20260127.133848", InboundPolicy: true},
	"header-transformation":                    {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "header-transformation", DefaultVersion: "1.0.0-20260127.134148", InboundPolicy: true},
	"dataweave-body-transformation":            {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "dataweave-body-transformation", DefaultVersion: "1.0.0", InboundPolicy: true},
	"dataweave-headers-transformation":         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "dataweave-headers-transformation", DefaultVersion: "1.0.0", InboundPolicy: true},
	"script-evaluation-transformation":         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "script-evaluation-transformation", DefaultVersion: "1.0.0-20260127.133315", InboundPolicy: true},
	"spec-validation":                          {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "spec-validation", DefaultVersion: "1.0.1", InboundPolicy: true},
	"external-oauth2-access-token-enforcement": {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "external-oauth2-access-token-enforcement", DefaultVersion: "1.6.0", InboundPolicy: true, SupportedTechnologies: []string{"mule4"}},
	"message-logging-outbound":                 {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "message-logging-outbound", DefaultVersion: "2.0.3", OutboundPolicy: true},
	"intask-authorization-code-policy":         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "intask-authorization-code-policy", DefaultVersion: "1.0.0", OutboundPolicy: true},
	"credential-injection-oauth2":              {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "credential-injection-oauth2", DefaultVersion: "1.0.1", OutboundPolicy: true},
	"credential-injection-basic-auth":          {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "credential-injection-basic-auth", DefaultVersion: "1.0.1", OutboundPolicy: true},
	"credential-injection-oauth2-obo":          {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "credential-injection-oauth2-obo", DefaultVersion: "1.1.0", OutboundPolicy: true},
	"idle-timeout":                             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "idle-timeout", DefaultVersion: "1.0.1", OutboundPolicy: true},
	"circuit-breaker":                          {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "circuit-breaker", DefaultVersion: "1.0.1", OutboundPolicy: true},
	"intask-authentication-policy":             {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "intask-authentication-policy", DefaultVersion: "1.0.0-20260113204639", OutboundPolicy: true},
	"native-aws-lambda":                        {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "native-aws-lambda", DefaultVersion: "1.0.1", OutboundPolicy: true},
	// MCP (Model Context Protocol) policies
	"mcp-pii-detector":         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-pii-detector", DefaultVersion: "1.0.0", InboundPolicy: true},
	"mcp-schema-validation":    {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-schema-validation", DefaultVersion: "1.1.0", InboundPolicy: true},
	"mcp-access-control":       {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-access-control", DefaultVersion: "1.0.1", InboundPolicy: true},
	"mcp-support":              {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-support", DefaultVersion: "1.0.1", InboundPolicy: true},
	"mcp-global-access-policy": {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-global-access-policy", DefaultVersion: "1.0.0", InboundPolicy: true},
	"mcp-tool-mapping":         {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-tool-mapping", DefaultVersion: "1.0.0", InboundPolicy: true},
	"mcp-transcoding-router":   {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "mcp-transcoding-router", DefaultVersion: "1.0.1-20260414150102", InboundPolicy: true},
	// LLM Gateway policies
	"semantic-routing-policy-huggingface": {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "semantic-routing-policy-huggingface", DefaultVersion: "1.0.0-20260130095514", InboundPolicy: true},
	"llm-proxy-core-policy":               {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "llm-proxy-core-policy", DefaultVersion: "1.0.0-20260108100848", InboundPolicy: true},
	"llm-gw-core-policy":                  {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "llm-gw-core-policy", DefaultVersion: "1.0.0-20251230075635", InboundPolicy: true},
	"llm-proxy-core":                      {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "llm-proxy-core", DefaultVersion: "1.0.0-20260127095720", InboundPolicy: true},
	"model-based-routing":                 {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "model-based-routing", DefaultVersion: "1.0.0-20260127100214", InboundPolicy: true},
	"semantic-prompt-guard-policy-openai": {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "semantic-prompt-guard-policy-openai", DefaultVersion: "1.0.0-20260130084752", InboundPolicy: true},
	// LLM Provider outbound policies
	"bedrock-llm-provider-policy": {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "bedrock-llm-provider-policy", DefaultVersion: "1.0.1", OutboundPolicy: true},
	"gemini-llm-provider-policy":  {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "gemini-llm-provider-policy", DefaultVersion: "1.0.0", OutboundPolicy: true},
	"openai-transcoding-policy":   {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "openai-transcoding-policy", DefaultVersion: "1.0.0", OutboundPolicy: true},
	"gemini-transcoding-policy":   {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "gemini-transcoding-policy", DefaultVersion: "1.0.0", OutboundPolicy: true},
	// A2A (Agent-to-Agent) policies
	"a2a-pii-detector":      {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "a-two-a-pii-detector", DefaultVersion: "1.0.1", InboundPolicy: true},
	"a2a-agent-card":        {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "a-two-a-agent-card", DefaultVersion: "2.0.0-20260327083212", InboundPolicy: true},
	"a2a-schema-validation": {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "a-two-a-schema-validation", DefaultVersion: "1.0.1", InboundPolicy: true},
	"a2a-token-rate-limit":  {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "a-two-a-token-rate-limit", DefaultVersion: "1.0.0", InboundPolicy: true},
	"a2a-prompt-decorator":  {GroupID: "68ef9520-24e9-4cf2-b2f5-620025690913", AssetID: "a-two-a-prompt-decorator", DefaultVersion: "1.0.1", InboundPolicy: true},
}

// LookupPolicy resolves a policy_type name to its Exchange coordinates.
// Returns the PolicyInfo and true if found, zero value and false otherwise.
func LookupPolicy(policyType string) (PolicyInfo, bool) {
	info, ok := KnownPolicies[policyType]
	return info, ok
}

// CamelToSnake converts a camelCase string to snake_case.
// Handles consecutive uppercase letters correctly:
// "rateLimits" → "rate_limits", "introspectionURL" → "introspection_url"
func CamelToSnake(s string) string {
	runes := []rune(s)
	var result []rune
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prev := runes[i-1]
				if prev >= 'a' && prev <= 'z' {
					result = append(result, '_')
				} else if prev >= 'A' && prev <= 'Z' && i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
					result = append(result, '_')
				}
			}
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// SnakeToCamel converts a snake_case string to camelCase.
// e.g. "rate_limits" → "rateLimits", "jwks_url" → "jwksUrl"
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// --- Policy Configuration Validation ---

// PolicySchemaField describes one field in a policy's configuration
type PolicySchemaField struct {
	Required bool
	Type     string   // "string", "int", "bool", "array", "object"
	Min      *float64 // optional inclusive lower bound (for "int" fields)
	Max      *float64 // optional inclusive upper bound (for "int" fields)
}

// KnownPolicySchemas maps assetId → field definitions for built-in validation.
// Only the most common policies are listed; unknown policies skip validation.
var KnownPolicySchemas = map[string]map[string]PolicySchemaField{
	"rate-limiting": {
		"rateLimits":    {Required: true, Type: "array"},
		"keySelector":   {Required: false, Type: "string"},
		"exposeHeaders": {Required: false, Type: "bool"},
		"clusterizable": {Required: false, Type: "bool"},
	},
	"rate-limiting-sla-based": {
		"clientIdExpression":     {Required: false, Type: "string"},
		"clientSecretExpression": {Required: false, Type: "string"},
		"exposeHeaders":          {Required: false, Type: "bool"},
		"clusterizable":          {Required: false, Type: "bool"},
	},
	"spike-control": {
		"maximumRequests":          {Required: true, Type: "int"},
		"timePeriodInMilliseconds": {Required: true, Type: "int", Max: float64Ptr(5000)},
		"delayTimeInMillis":        {Required: true, Type: "int"},
		"delayAttempts":            {Required: true, Type: "int"},
		"queuingLimit":             {Required: false, Type: "int"},
		"exposeHeaders":            {Required: false, Type: "bool"},
	},
	"http-basic-authentication": {
		"username": {Required: true, Type: "string"},
		"password": {Required: true, Type: "string"},
	},
	"client-id-enforcement": {
		"credentialsOriginHasHttpBasicAuthenticationHeader": {Required: false, Type: "string"},
		"clientIdExpression":     {Required: false, Type: "string"},
		"clientSecretExpression": {Required: false, Type: "string"},
	},
	"jwt-validation": {
		"jwtOrigin":                    {Required: true, Type: "string"},
		"jwtExpression":                {Required: false, Type: "string"},
		"signingMethod":                {Required: false, Type: "string"},
		"signingKeyLength":             {Required: false, Type: "int"},
		"jwtKeyOrigin":                 {Required: false, Type: "string"},
		"textKey":                      {Required: false, Type: "string"},
		"customKeyExpression":          {Required: false, Type: "string"},
		"jwksUrl":                      {Required: false, Type: "string"},
		"jwksServiceTimeToLive":        {Required: false, Type: "int"},
		"jwksServiceConnectionTimeout": {Required: false, Type: "int"},
		"skipClientIdValidation":       {Required: false, Type: "bool"},
		"clientIdExpression":           {Required: false, Type: "string"},
		"validateAudClaim":             {Required: false, Type: "bool"},
		"mandatoryAudClaim":            {Required: false, Type: "bool"},
		"supportedAudiences":           {Required: false, Type: "string"},
		"mandatoryExpClaim":            {Required: false, Type: "bool"},
		"mandatoryNbfClaim":            {Required: false, Type: "bool"},
		"validateCustomClaim":          {Required: false, Type: "bool"},
		"claimsToHeaders":              {Required: false, Type: "array"},
		"mandatoryCustomClaims":        {Required: false, Type: "array"},
		"nonMandatoryCustomClaims":     {Required: false, Type: "array"},
	},
	"oauth2-access-token-enforcement": {
		"oauthProvider":          {Required: false, Type: "string"},
		"scopes":                 {Required: false, Type: "string"},
		"tokenUrl":               {Required: false, Type: "string"},
		"exposeHeaders":          {Required: false, Type: "bool"},
		"skipClientIdValidation": {Required: false, Type: "bool"},
	},
	"ip-allowlist": {
		"ipExpression":  {Required: true, Type: "string"},
		"ips":           {Required: true, Type: "string_array"},
		"methodsString": {Required: false, Type: "string"},
	},
	"ip-blocklist": {
		"ipExpression":  {Required: true, Type: "string"},
		"ips":           {Required: true, Type: "string_array"},
		"methodsString": {Required: false, Type: "string"},
	},
	"json-threat-protection": {
		"maxContainerDepth":        {Required: false, Type: "int"},
		"maxStringValueLength":     {Required: false, Type: "int"},
		"maxObjectEntryNameLength": {Required: false, Type: "int"},
		"maxObjectEntryCount":      {Required: false, Type: "int"},
		"maxArrayElementCount":     {Required: false, Type: "int"},
	},
	"message-logging": {
		"loggingConfiguration": {Required: true, Type: "array"},
	},
	"cors": {
		"publicResource":     {Required: false, Type: "bool"},
		"supportCredentials": {Required: false, Type: "bool"},
		"originGroups":       {Required: true, Type: "array"},
	},
	"header-injection": {
		"inboundHeaders":  {Required: false, Type: "array"},
		"outboundHeaders": {Required: false, Type: "array"},
	},
	"header-removal": {
		"inboundHeaders":  {Required: false, Type: "array"},
		"outboundHeaders": {Required: false, Type: "array"},
	},
	"native-ext-authz": {
		"uri":                                    {Required: true, Type: "string"},
		"serverType":                             {Required: true, Type: "string"},
		"requestTimeout":                         {Required: false, Type: "int"},
		"serverApiVersion":                       {Required: false, Type: "string"},
		"includePeerCertificate":                 {Required: false, Type: "bool"},
		"allowedHeaders":                         {Required: false, Type: "array"},
		"serviceRequestHeadersToAdd":             {Required: false, Type: "array"},
		"serviceResponseUpstreamHeaders":         {Required: false, Type: "array"},
		"serviceResponseUpstreamHeadersToAppend": {Required: false, Type: "array"},
		"serviceResponseClientHeaders":           {Required: false, Type: "array"},
		"serviceResponseClientHeadersOnSuccess":  {Required: false, Type: "array"},
		"pathPrefix":                             {Required: false, Type: "string"},
	},
	"native-ext-proc": {
		"uri":                 {Required: true, Type: "string"},
		"messageTimeout":      {Required: false, Type: "int"},
		"maxMessageTimeout":   {Required: false, Type: "int"},
		"failureModeAllow":    {Required: false, Type: "bool"},
		"allowModeOverride":   {Required: false, Type: "bool"},
		"requestHeaderMode":   {Required: false, Type: "string"},
		"responseHeaderMode":  {Required: false, Type: "string"},
		"requestBodyMode":     {Required: false, Type: "string"},
		"responseBodyMode":    {Required: false, Type: "string"},
		"requestTrailerMode":  {Required: false, Type: "string"},
		"responseTrailerMode": {Required: false, Type: "string"},
	},
	"sse-logging": {
		"logs": {Required: true, Type: "array"},
	},
	"response-timeout": {
		"timeout": {Required: true, Type: "int"},
	},
	"stream-idle-timeout": {
		"timeout": {Required: true, Type: "int"},
	},
	"health-check": {
		"endpoint":   {Required: false, Type: "string"},
		"path":       {Required: false, Type: "string"},
		"statusCode": {Required: false, Type: "string"},
	},
	"http-caching": {
		"httpCachingKey":      {Required: false, Type: "string"},
		"maxCacheEntries":     {Required: false, Type: "int"},
		"ttl":                 {Required: false, Type: "int"},
		"distributed":         {Required: false, Type: "bool"},
		"persistCache":        {Required: false, Type: "bool"},
		"useHttpCacheHeaders": {Required: false, Type: "bool"},
		"invalidationHeader":  {Required: false, Type: "string"},
		"requestExpression":   {Required: false, Type: "string"},
		"responseExpression":  {Required: false, Type: "string"},
	},
	"access-block": {},
	"agent-connection-telemetry": {
		"sourceAgentId": {Required: false, Type: "string"},
	},
	"oauth2-token-introspection": {
		"introspectionURL":        {Required: true, Type: "string"},
		"authorizationValue":      {Required: true, Type: "string"},
		"validatedTokenTTL":       {Required: false, Type: "int"},
		"scopeValidationCriteria": {Required: false, Type: "string"},
		"skipClientIdValidation":  {Required: false, Type: "bool"},
		"consumerBy":              {Required: false, Type: "string"},
		"exposeHeaders":           {Required: false, Type: "bool"},
		"maxCacheEntries":         {Required: false, Type: "int"},
		"authenticationTimeout":   {Required: false, Type: "int"},
	},
	"ldap-authentication": {
		"ldapServerURL":          {Required: true, Type: "string"},
		"ldapServerUserDn":       {Required: true, Type: "string"},
		"ldapServerUserPassword": {Required: true, Type: "string"},
		"ldapSearchBase":         {Required: true, Type: "string"},
		"ldapSearchFilter":       {Required: true, Type: "string"},
		"ldapSearchInSubtree":    {Required: false, Type: "bool"},
	},
	"tracing": {
		"sampling": {Required: false, Type: "object"},
		"spanName": {Required: false, Type: "string"},
		"labels":   {Required: false, Type: "array"},
	},
	"xml-threat-protection": {
		"maxNodeDepth":                {Required: false, Type: "int"},
		"maxAttributeCountPerElement": {Required: false, Type: "int"},
		"maxChildCount":               {Required: false, Type: "int"},
		"maxTextLength":               {Required: false, Type: "int"},
		"maxAttributeLength":          {Required: false, Type: "int"},
		"maxCommentLength":            {Required: false, Type: "int"},
	},
	"injection-protection": {
		"builtInProtections":  {Required: false, Type: "array"},
		"customProtections":   {Required: false, Type: "array"},
		"protectPathAndQuery": {Required: false, Type: "bool"},
		"protectHeaders":      {Required: false, Type: "bool"},
		"protectBody":         {Required: false, Type: "bool"},
		"rejectRequests":      {Required: false, Type: "bool"},
	},
	"dataweave-request-filter": {
		"script":          {Required: true, Type: "string"},
		"requiresPayload": {Required: false, Type: "bool"},
	},
	"body-transformation": {
		"script":      {Required: true, Type: "string"},
		"requestFlow": {Required: false, Type: "string"},
	},
	"header-transformation": {
		"script":          {Required: true, Type: "string"},
		"requiresPayload": {Required: false, Type: "bool"},
		"requestFlow":     {Required: false, Type: "string"},
	},
	"dataweave-body-transformation": {
		"script":      {Required: true, Type: "string"},
		"requestFlow": {Required: false, Type: "string"},
	},
	"dataweave-headers-transformation": {
		"script":          {Required: true, Type: "string"},
		"requiresPayload": {Required: false, Type: "bool"},
		"requestFlow":     {Required: false, Type: "string"},
	},
	"script-evaluation-transformation": {
		"script":          {Required: true, Type: "string"},
		"requiresPayload": {Required: false, Type: "bool"},
		"requestFlow":     {Required: false, Type: "string"},
	},
	"spec-validation": {
		"blockOperation":         {Required: false, Type: "bool"},
		"strictParamsValidation": {Required: false, Type: "bool"},
	},
	"external-oauth2-access-token-enforcement": {
		"tokenUrl":                {Required: true, Type: "string"},
		"scopeValidationCriteria": {Required: false, Type: "string"},
		"secureTrustStore":        {Required: false, Type: "bool"},
		"exposeHeaders":           {Required: false, Type: "bool"},
		"skipClientIdValidation":  {Required: false, Type: "bool"},
		"authenticationTimeout":   {Required: false, Type: "int"},
		"scopes":                  {Required: false, Type: "string"},
		"maxCacheEntries":         {Required: false, Type: "int"},
	},
	// loggingConfiguration is an array of {itemName, itemData{message,conditional,level,firstSection,secondSection}}
	"message-logging-outbound": {
		"loggingConfiguration": {Required: true, Type: "array"},
	},
	"intask-authorization-code-policy": {
		"secondaryAuthProvider":       {Required: true, Type: "string"},
		"authorizationEndpoint":       {Required: true, Type: "string"},
		"tokenEndpoint":               {Required: true, Type: "string"},
		"scopes":                      {Required: false, Type: "string"},
		"redirectUri":                 {Required: true, Type: "string"},
		"responseType":                {Required: false, Type: "string"},
		"codeChallengeMethod":         {Required: false, Type: "string"},
		"bodyEncoding":                {Required: false, Type: "string"},
		"tokenTimeout":                {Required: false, Type: "int"},
		"challengeResponseStatusCode": {Required: false, Type: "int"},
	},
	"credential-injection-oauth2": {
		"oauthService":                  {Required: true, Type: "string"},
		"clientId":                      {Required: true, Type: "string"},
		"clientSecret":                  {Required: true, Type: "string"},
		"scope":                         {Required: false, Type: "array"},
		"overwrite":                     {Required: false, Type: "bool"},
		"tokenFetchTimeout":             {Required: false, Type: "int"},
		"allowRequestWithoutCredential": {Required: false, Type: "bool"},
	},
	"credential-injection-basic-auth": {
		"username":     {Required: true, Type: "string"},
		"password":     {Required: true, Type: "string"},
		"overwrite":    {Required: false, Type: "bool"},
		"customHeader": {Required: false, Type: "string"},
	},
	"credential-injection-oauth2-obo": {
		"flow":          {Required: true, Type: "string"},
		"clientId":      {Required: true, Type: "string"},
		"clientSecret":  {Required: true, Type: "string"},
		"tokenEndpoint": {Required: true, Type: "string"},
		"scope":         {Required: false, Type: "string"},
		"timeout":       {Required: false, Type: "int"},
		"cibaEnabled":   {Required: false, Type: "bool"},
		"targetValue":   {Required: false, Type: "string"},
		"targetType":    {Required: false, Type: "string"},
	},
	"idle-timeout": {
		"timeout": {Required: true, Type: "int"},
	},
	// thresholds is a nested object; uses DynamicAttribute to accept arbitrary HCL maps
	"circuit-breaker": {
		"thresholds": {Required: true, Type: "object"},
	},
	"intask-authentication-policy": {
		"secondaryAuthProvider":       {Required: true, Type: "string"},
		"authorizationEndpoint":       {Required: true, Type: "string"},
		"tokenEndpoint":               {Required: true, Type: "string"},
		"scopes":                      {Required: false, Type: "string"},
		"redirectUri":                 {Required: true, Type: "string"},
		"responseType":                {Required: false, Type: "string"},
		"codeChallengeMethod":         {Required: false, Type: "string"},
		"tokenAudience":               {Required: false, Type: "string"},
		"bodyEncoding":                {Required: false, Type: "string"},
		"tokenTimeout":                {Required: false, Type: "int"},
		"userIdHeader":                {Required: false, Type: "string"},
		"userEmailHeader":             {Required: false, Type: "string"},
		"challengeResponseStatusCode": {Required: false, Type: "int"},
	},
	// credentials is a nested object {accessKeyId, secretAccessKey, sessionToken}; uses DynamicAttribute
	"native-aws-lambda": {
		"arn":                {Required: true, Type: "string"},
		"payloadPassthrough": {Required: false, Type: "bool"},
		"invocationMode":     {Required: false, Type: "string"},
		"authenticationMode": {Required: false, Type: "string"},
		"credentials":        {Required: false, Type: "object"},
	},
	// MCP (Model Context Protocol) policies
	"mcp-pii-detector": {
		"entities": {Required: true, Type: "array"},
	},
	"mcp-schema-validation": {
		"validateToolSchema": {Required: false, Type: "bool"},
	},
	"mcp-access-control": {
		"rules":    {Required: true, Type: "array"},
		"authType": {Required: false, Type: "string"},
	},
	"mcp-support": {},
	"mcp-global-access-policy": {
		"rules": {Required: true, Type: "array"},
	},
	"mcp-tool-mapping": {
		"toolMappings": {Required: true, Type: "array"},
		"logMappings":  {Required: false, Type: "bool"},
	},
	"mcp-transcoding-router": {
		"transcodingPath": {Required: false, Type: "string"},
		"routes":          {Required: true, Type: "array"},
	},
	// LLM Gateway policies
	"semantic-routing-policy-huggingface": {
		"huggingfaceUrl":    {Required: true, Type: "string"},
		"huggingfaceApiKey": {Required: true, Type: "string"},
		"threshold":         {Required: false, Type: "object"},
		"timeout":           {Required: false, Type: "int"},
		"routes":            {Required: true, Type: "array"},
		"fallbackRoute":     {Required: false, Type: "object"},
	},
	"llm-proxy-core-policy": {
		"headerName":          {Required: true, Type: "string"},
		"vendorHeaderMapping": {Required: true, Type: "array"},
	},
	"llm-gw-core-policy": {
		"headerName":          {Required: true, Type: "string"},
		"vendorHeaderMapping": {Required: true, Type: "array"},
	},
	"llm-proxy-core": {},
	"model-based-routing": {
		"supportedVendors": {Required: true, Type: "array"},
		"fallback":         {Required: false, Type: "object"},
	},
	"semantic-prompt-guard-policy-openai": {
		"openaiUrl":            {Required: true, Type: "string"},
		"openaiApiKey":         {Required: true, Type: "string"},
		"openaiEmbeddingModel": {Required: false, Type: "string"},
		"threshold":            {Required: false, Type: "object"},
		"timeout":              {Required: false, Type: "int"},
		"denyTopics":           {Required: true, Type: "array"},
	},
	// LLM Provider outbound policies
	"bedrock-llm-provider-policy": {
		"awsAccessKeyId":     {Required: true, Type: "string"},
		"awsSecretAccessKey": {Required: true, Type: "string"},
		"awsSessionToken":    {Required: false, Type: "string"},
		"awsRegion":          {Required: true, Type: "string"},
		"serviceName":        {Required: false, Type: "string"},
		"timeout":            {Required: false, Type: "int"},
	},
	"gemini-llm-provider-policy": {
		"apiKey":      {Required: true, Type: "string"},
		"timeout":     {Required: false, Type: "int"},
		"modelMapper": {Required: false, Type: "array"},
	},
	"openai-transcoding-policy": {
		"apiKey":      {Required: true, Type: "string"},
		"timeout":     {Required: false, Type: "int"},
		"modelMapper": {Required: false, Type: "array"},
	},
	"gemini-transcoding-policy": {},
	// A2A (Agent-to-Agent) policies
	"a2a-pii-detector": {
		"entities": {Required: true, Type: "array"},
		"action":   {Required: false, Type: "string"},
	},
	"a2a-agent-card": {
		"consumerUrl":  {Required: false, Type: "string"},
		"cardPath":     {Required: false, Type: "string"},
		"fileName":     {Required: false, Type: "string"},
		"fileMimeType": {Required: false, Type: "string"},
		"fileSource":   {Required: false, Type: "string"},
		"content":      {Required: true, Type: "string"},
	},
	"a2a-schema-validation": {},
	"a2a-token-rate-limit": {
		"maximumTokens":            {Required: true, Type: "int"},
		"timePeriodInMilliseconds": {Required: true, Type: "int"},
		"keySelector":              {Required: false, Type: "string"},
	},
	"a2a-prompt-decorator": {
		"textDecorators": {Required: false, Type: "array"},
		"fileDecorators": {Required: false, Type: "array"},
	},
}

// ValidatePolicyConfiguration checks configurationData against known schemas.
// Returns a list of validation error strings, or empty if valid / unknown policy.
func ValidatePolicyConfiguration(assetID string, configData map[string]interface{}) []string {
	schema, ok := KnownPolicySchemas[assetID]
	if !ok {
		return nil
	}

	var errs []string

	for field, spec := range schema {
		val, exists := configData[field]
		if spec.Required && (!exists || val == nil) {
			errs = append(errs, fmt.Sprintf("missing required field %q for policy %q", field, assetID))
		}
	}

	for field := range configData {
		if _, known := schema[field]; !known {
			errs = append(errs, fmt.Sprintf("unknown field %q for policy %q", field, assetID))
		}
	}

	return errs
}

func float64Ptr(v float64) *float64 { return &v }
