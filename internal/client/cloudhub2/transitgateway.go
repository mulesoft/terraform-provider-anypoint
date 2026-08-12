package cloudhub2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// ErrTransitGatewayDetached signals that a PS-scoped GET-by-id returned HTTP 400
// because the attachment has been DETACHED from the private space but is NOT yet
// de-registered (the object still exists org-wide). This is a recoverable,
// terminal-ish state — distinct from a 404 (fully gone) — so callers should
// surface a "Detached" status and keep the resource in state rather than
// hard-erroring, otherwise refresh 400s and the resource is stranded with no
// clean IaC teardown (W-23819332). Wrapped with %w so callers use errors.Is via
// IsTransitGatewayDetached.
var ErrTransitGatewayDetached = errors.New("transit gateway attachment detached from private space")

// IsTransitGatewayDetached reports whether err is (or wraps) the detached-400
// signal from GetTransitGateway.
func IsTransitGatewayDetached(err error) bool {
	return errors.Is(err, ErrTransitGatewayDetached)
}

// isDetachedBody matches the platform's PS-scoped GET-by-id 400 body for a
// detached-but-registered attachment. The stable phrase is "not attached to the
// private space"; matched case-insensitively so minor wording/casing drift does
// not defeat detection. It is deliberately specific so an unrelated 400 (bad
// request, validation, auth) is NOT misclassified as detached.
func isDetachedBody(body string) bool {
	return strings.Contains(strings.ToLower(body), "not attached to the private space")
}

// TransitGatewayClient wraps the AnypointClient for transit gateway operations.
type TransitGatewayClient struct {
	*client.AnypointClient
}

// NewTransitGatewayClient creates a new TransitGatewayClient.
func NewTransitGatewayClient(config *client.Config) (*TransitGatewayClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TransitGatewayClient{AnypointClient: anypointClient}, nil
}

// TransitGateway represents a transit gateway attachment in a Private Space.
type TransitGateway struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Spec   TransitGatewaySpec   `json:"spec"`
	Status TransitGatewayStatus `json:"status"`
}

// TransitGatewaySpec holds the configuration details of a transit gateway.
type TransitGatewaySpec struct {
	ResourceShare ResourceShare `json:"resourceShare"`
	Region        string        `json:"region"`
	SpaceName     string        `json:"spaceName,omitempty"`
}

// ResourceShare represents the AWS RAM resource share details.
type ResourceShare struct {
	ID      string `json:"id"`
	Account string `json:"account"`
}

// TransitGatewayStatus holds the runtime status of a transit gateway.
type TransitGatewayStatus struct {
	Gateway     string   `json:"gateway"`
	Attachment  string   `json:"attachment"`
	TgwResource string   `json:"tgwResource"`
	Routes      []string `json:"routes"`
}

// CreateTransitGatewayRequest represents the request to create a transit gateway attachment.
// The platform discovers the AWS Transit Gateway from the resource share automatically.
type CreateTransitGatewayRequest struct {
	Name                 string   `json:"name"`
	ResourceShareID      string   `json:"resourceShareId"`      // AWS RAM resource share ID (UUID format)
	ResourceShareAccount string   `json:"resourceShareAccount"` // AWS account ID that owns the TGW (12 digits)
	Routes               []string `json:"routes"`               // Initial CIDR routes (at least one required)
}

// UpdateTransitGatewayRequest represents the body of the transit gateway rename
// PATCH. The body is name-only: {"name":"..."}.
//
// IMPORTANT — endpoint/spec divergence, triangulated live 2026-07-17 against the
// Anypoint UI (browser DevTools), the RAML, and observed server behaviour.
// There are TWO transit-gateway PATCH endpoints in the CloudHub 2.0 API:
//
//   - PRIVATE-SPACE-scoped: .../organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
//     The RAML documents this one (body PrivateSpaceTransitGatewayPatch = {name}),
//     but the LIVE handler is broken for renames: a name-only body is rejected
//     with 400 "Routes cannot be null", and a name+routes body returns 200 yet
//     SILENTLY IGNORES the name (only the routes field takes effect). This dead
//     end is what the earlier routes-in-body workaround was chasing.
//
//   - ORG-scoped: .../organizations/{orgId}/transitgateways/{tgwId}
//     The RAML does NOT document a PATCH here, but this is the endpoint the
//     Anypoint UI actually uses to rename: it accepts a name-only body
//     {"name":"..."} and applies the rename (200 OK, verified in DevTools).
//
// UpdateTransitGateway therefore targets the ORG-scoped endpoint with a name-only
// body, mirroring the UI. Routes are NOT sent here; the dedicated private-space
// {tgwId}/routes sub-resource is the authoritative route setter (see
// UpdateTransitGatewayRoutes).
type UpdateTransitGatewayRequest struct {
	Name string `json:"name"`
}

// CreateTransitGateway creates a new transit gateway attachment in a Private Space.
// The API returns 201 with no body, so we list transit gateways afterward to find
// the created resource (matched by the AWS TGW ID).
// API: POST /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways
func (c *TransitGatewayClient) CreateTransitGateway(ctx context.Context, orgID, privateSpaceID string, req *CreateTransitGatewayRequest) (*TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transit gateway request: %w", err)
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
		body, _ := io.ReadAll(resp.Body)
		// CloudHub 2.0 private spaces need the connected app's 'Cloudhub Organization
		// Admin' scope; a token missing it is rejected with 401/403. Surface actionable
		// scope guidance instead of a bare status error.
		if authErr := client.AuthContextErrorIfUnauthorized(resp.StatusCode, url, string(body)); authErr != nil {
			return nil, authErr
		}
		return nil, fmt.Errorf("failed to create transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	// Create returns 201 with no body. List transit gateways to find the new one.
	tgws, err := c.ListTransitGateways(ctx, orgID, privateSpaceID)
	if err != nil {
		return nil, fmt.Errorf("transit gateway created but failed to list: %w", err)
	}

	// Find the TGW by name (ID is assigned by the platform)
	for i := range tgws {
		if tgws[i].Name == req.Name {
			return &tgws[i], nil
		}
	}

	return nil, fmt.Errorf("transit gateway created but could not be found in list")
}

// ensureTransitGatewayDecoded guards against a silent zero-value decode. A
// successful single-object GET/PATCH always carries the transit gateway's id
// (the request is by-id). An empty id after a 200 therefore means the response
// shape did not match our struct — e.g. an object envelope wrapping the gateway,
// exactly the divergence the private-spaces LIST endpoint turned out to have.
// Surface a loud, diagnosable error instead of persisting an all-empty struct.
func ensureTransitGatewayDecoded(tgw *TransitGateway, body []byte) error {
	if tgw.ID != "" {
		return nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope) > 0 {
		keys := make([]string, 0, len(envelope))
		for k := range envelope {
			keys = append(keys, k)
		}
		return fmt.Errorf("decoded transit gateway has empty id; unexpected response shape (object keys %v)", keys)
	}
	return fmt.Errorf("decoded transit gateway has empty id; empty or unexpected response body")
}

// GetTransitGateway retrieves a specific transit gateway by its ID.
// API: GET /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) GetTransitGateway(ctx context.Context, orgID, privateSpaceID, transitGatewayID string) (*TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

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
		return nil, client.NewNotFoundError("transit gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Detached-but-registered attachment: the PS-scoped by-id GET returns 400
		// "attachment is not attached to the private space". Signal it as a typed,
		// recoverable error so Read/Import can surface a Detached status and keep
		// the resource (plan/destroy stay working) instead of hard-erroring on
		// refresh (W-23819332). Any OTHER 400 stays a generic error.
		if resp.StatusCode == http.StatusBadRequest && isDetachedBody(string(body)) {
			return nil, fmt.Errorf("%w: %s", ErrTransitGatewayDetached, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("failed to get transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tgw TransitGateway
	if err := json.Unmarshal(body, &tgw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if err := ensureTransitGatewayDecoded(&tgw, body); err != nil {
		return nil, err
	}

	return &tgw, nil
}

// UpdateTransitGateway renames a transit gateway connection. It PATCHes the
// ORG-scoped endpoint with a name-only body, mirroring the Anypoint UI — the
// private-space-scoped PATCH the RAML documents silently ignores the name (see
// UpdateTransitGatewayRequest for the full divergence). Routes are managed
// separately via UpdateTransitGatewayRoutes; this call does not touch them.
// Note: privateSpaceID is accepted for signature symmetry with the other
// methods but is intentionally unused — the org-scoped URL has no ps segment.
// API: PATCH /runtimefabric/api/organizations/{orgId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) UpdateTransitGateway(ctx context.Context, orgID, privateSpaceID, transitGatewayID string, req *UpdateTransitGatewayRequest) (*TransitGateway, error) {
	_ = privateSpaceID // org-scoped endpoint takes no private-space segment
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/transitgateways/%s", c.BaseURL, orgID, transitGatewayID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transit gateway update request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("transit gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update transit gateway with status %d: %s", resp.StatusCode, string(body))
	}

	// Response-shape divergence confirmed live 2026-07-17 (browser DevTools): the
	// org-scoped rename PATCH returns 200 with a JSON ARRAY (the org's TGW list),
	// not the single object one might expect. A plain json.Decode(&struct) errors
	// with "cannot unmarshal array into Go value of type ...TransitGateway" AFTER
	// a successful 200, which used to fail Update even though the server had
	// already applied the rename. Decode defensively: accept either shape. The
	// resource's Update discards this value and re-GETs for authoritative state,
	// so we deliberately do NOT apply the empty-id guard here — a minimal/empty/
	// array ack must not fail Update.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return decodeTransitGatewayObjectOrArray(body, transitGatewayID)
}

// decodeTransitGatewayObjectOrArray decodes a transit gateway from a response
// body that may be EITHER a single object (per the RAML spec) OR a JSON array
// (what the live connection PATCH actually returns). For the array shape it
// returns the element whose id matches wantID, falling back to the first
// element. An empty/whitespace body yields an empty gateway (some acks carry no
// body). It never errors on shape alone — callers needing authoritative state
// re-GET the gateway.
func decodeTransitGatewayObjectOrArray(body []byte, wantID string) (*TransitGateway, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return &TransitGateway{}, nil
	}
	if trimmed[0] == '[' {
		var tgws []TransitGateway
		if err := json.Unmarshal(trimmed, &tgws); err != nil {
			return nil, fmt.Errorf("failed to decode response array: %w", err)
		}
		for i := range tgws {
			if tgws[i].ID == wantID {
				return &tgws[i], nil
			}
		}
		if len(tgws) > 0 {
			return &tgws[0], nil
		}
		return &TransitGateway{}, nil
	}
	var tgw TransitGateway
	if err := json.Unmarshal(trimmed, &tgw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &tgw, nil
}

// DeleteTransitGateway deletes a transit gateway attachment.
// API: DELETE /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) DeleteTransitGateway(ctx context.Context, orgID, privateSpaceID, transitGatewayID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

	status, body, err := c.doDeleteTransitGateway(ctx, url)
	if err != nil {
		return err
	}

	if isTransitGatewayDeleteGone(status) {
		return nil
	}

	// Detached-but-registered attachment: the PS-scoped DELETE can't act because
	// the attachment is no longer on the private space, so it 400s "attachment is
	// not attached to the private space" (the same signal GetTransitGateway maps
	// to ErrTransitGatewayDetached). The object still exists ORG-wide, though, so
	// fall back to the org-scoped DELETE to fully de-register it. Without this,
	// destroy 400s and the resource is stranded in state with no clean teardown
	// (W-23819332, destroy-side). Any OTHER 400 stays a hard error.
	if status == http.StatusBadRequest && isDetachedBody(body) {
		return c.deleteTransitGatewayOrgScoped(ctx, orgID, transitGatewayID)
	}

	return fmt.Errorf("failed to delete transit gateway with status %d: %s", status, body)
}

// deleteTransitGatewayOrgScoped issues the ORG-scoped DELETE, which fully
// de-registers a transit gateway object org-wide. It is the teardown counterpart
// of the org-scoped rename endpoint (see UpdateTransitGateway): the RAML does not
// document it, but it is what the platform uses to remove a
// detached-but-still-registered object. A 404/204/200 all mean "gone".
// API: DELETE /runtimefabric/api/organizations/{orgId}/transitgateways/{tgwId}
func (c *TransitGatewayClient) deleteTransitGatewayOrgScoped(ctx context.Context, orgID, transitGatewayID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/transitgateways/%s", c.BaseURL, orgID, transitGatewayID)

	status, body, err := c.doDeleteTransitGateway(ctx, url)
	if err != nil {
		return err
	}

	if isTransitGatewayDeleteGone(status) {
		return nil
	}

	return fmt.Errorf("failed to delete transit gateway (org-scoped de-register) with status %d: %s", status, body)
}

// doDeleteTransitGateway issues a bearer-authenticated DELETE against url and
// returns (status, body, transportErr). It centralizes the request plumbing so
// both the PS-scoped and org-scoped delete paths share identical wire behaviour.
func (c *TransitGatewayClient) doDeleteTransitGateway(ctx context.Context, url string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

// isTransitGatewayDeleteGone reports whether a DELETE status means the object is
// gone (idempotent success): 204/200 (deleted now) or 404 (already deleted).
func isTransitGatewayDeleteGone(status int) bool {
	return status == http.StatusNoContent || status == http.StatusOK || status == http.StatusNotFound
}

// ListTransitGateways lists all transit gateways in a Private Space.
// API: GET /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways
func (c *TransitGatewayClient) ListTransitGateways(ctx context.Context, orgID, privateSpaceID string) ([]TransitGateway, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways", c.BaseURL, orgID, privateSpaceID)

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list transit gateways with status %d: %s", resp.StatusCode, string(body))
	}

	var tgws []TransitGateway
	if err := json.NewDecoder(resp.Body).Decode(&tgws); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tgws, nil
}

// tgwRoutesUpdateBody is the body of a routes update. Routes are a field on the
// private-space CONNECTION object, so the update PATCHes the connection itself
// with {"name":...,"routes":[...]}. The live handler requires a non-null routes
// array (400 "Routes cannot be null" otherwise) and treats name as required
// (per the RAML PrivateSpaceTransitGatewayPatch); we echo the current/desired
// name to satisfy that. This endpoint IGNORES name for updates — renames must go
// through UpdateTransitGateway (org-scoped) — so sending it here is harmless.
type tgwRoutesUpdateBody struct {
	Name   string   `json:"name"`
	Routes []string `json:"routes"`
}

// UpdateTransitGatewayRoutes replaces the routes on a transit gateway connection.
//
// IMPORTANT — endpoint divergence confirmed live 2026-07-17 (read-only probes):
// the RAML documents a dedicated {tgwId}/routes sub-resource, but that path
// returns 404 at BOTH the private-space and org scopes — it does not exist.
// Routes are instead a field on the private-space connection object and are
// updated by PATCHing the connection (.../{tgwId}) with a {name, routes} body.
// A nil slice is normalised to [] so the body can never contain "routes":null.
// API: PATCH /runtimefabric/api/organizations/{orgId}/privatespaces/{psId}/transitgateways/{tgwId}
// Body: {"name":"<name>","routes":["cidr1","cidr2",...]}
func (c *TransitGatewayClient) UpdateTransitGatewayRoutes(ctx context.Context, orgID, privateSpaceID, transitGatewayID, name string, routes []string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/transitgateways/%s", c.BaseURL, orgID, privateSpaceID, transitGatewayID)

	// A nil slice must marshal to "[]" (clear all routes), never "null" — the
	// live handler rejects "routes":null with 400 "Routes cannot be null".
	if routes == nil {
		routes = []string{}
	}
	jsonData, err := json.Marshal(tgwRoutesUpdateBody{Name: name, Routes: routes})
	if err != nil {
		return fmt.Errorf("failed to marshal routes request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update transit gateway routes with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
