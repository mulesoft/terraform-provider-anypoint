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

// SelfManagedGatewayClient wraps the AnypointClient for self-managed (connected-mode)
// Flex/Omni Gateway operations.
//
// Self-managed gateways are fundamentally different from the *managed* Omni Gateway
// (see managedomnigateway.go): the platform does NOT provision a runtime. The customer
// runs the Flex runtime on their own infrastructure (Docker/K8s/Linux via flexctl), and
// the gateway materializes on the platform as a side effect of the runtime self-registering
// with a short-lived registration token. Consequently there is NO "create gateway" endpoint:
// the only gateway-object lifecycle operations are LIST, GET-by-id, and DELETE. The genuine
// "create" primitive is minting a registration token.
//
// Contract triangulated + live-verified (READ-only) against production; see
// .agents/artifacts/self-managed-gateway-contract.md for the full evidence trail.
type SelfManagedGatewayClient struct {
	*client.AnypointClient
}

// NewSelfManagedGatewayClient creates a new SelfManagedGatewayClient
func NewSelfManagedGatewayClient(config *client.Config) (*SelfManagedGatewayClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &SelfManagedGatewayClient{AnypointClient: anypointClient}, nil
}

// --- Base paths (VERIFIED against production, 2026-07-21) ------------------------------
//
// Confirmed by live READ-only method-discovery probes on prod plus a UI network capture
// (Runtime Manager -> Omni Gateways -> Self-Managed Omni Gateway). Evidence:
//   - POST mint:   GET     /standalone/api/v1/.../gatewaytokens      -> 405 Allow: POST
//   - GET  list:   GET     /standalone/api/v1/.../gateways           -> 200 {content,pageNumber,pageSize,totalElements}
//   - GET  item:   GET     /standalone/api/v1/.../gateways/{uuid}    -> 404 (clean not-found for a valid-but-absent id)
//   - DELETE item: OPTIONS /standalone/api/v1/.../gateways/{uuid}    -> 200 Allow: GET,HEAD,DELETE,OPTIONS
//
// The permissions endpoint reports selfManagedGateways:{view,create,delete} (NO update),
// which matches this resource's no-update design.
//
// NOTE on the UI facade: the Anypoint UI's unified Omni Gateways list is served by
// /gatewaymanager/xapi/v1/.../gateways?kind=selfManaged (one endpoint feeding both the
// Managed and Self-Managed tabs; the /gatewaymanager/api/v1 variant rejects ?kind with
// 400). The provider instead rides the self-managed-native arm-standalone-manager-service
// base for ALL four operations, because (a) mint and the item-level GET/DELETE exist ONLY
// on /standalone (the xapi item endpoint returns 405 and xapi has no gatewaytokens), and
// (b) staying on one service guarantees the id returned by list is the same id GET/DELETE
// key on. Keep the leading slash; no trailing slash.
const (
	// selfManagedBasePath is the arm-standalone-manager-service base. It hosts the
	// registration-token mint and the gateway-object lifecycle (list/get/delete).
	selfManagedBasePath = "/standalone/api/v1"

	// registrationFacadeBasePath is the amc-agent-reg-facade base. It hosts CSR signing
	// (agent registration) and certificate renewal. Not used by the provider today
	// (see design note in the resource), reserved for when/if the provider performs the
	// full enrollment. Kept here so the whole contract lives in one file.
	//nolint:unused // reserved: documents the enrollment endpoint for future use.
	registrationFacadeBasePath = "/amc/registration-facade/api/v1"
)

// SelfManagedGatewayStatusDeleted is the status a self-managed gateway object carries after
// it has been (soft-)deleted. LIVE-VERIFIED (2026-07-21): DELETE is an async soft-delete that
// returns HTTP 202 and flips the object's status to this value, but the object is NEVER
// hard-removed — it lingers in list/GET forever as a tombstone. Callers that match gateways by
// name (resource resolve, data source list) must skip objects with this status so a destroyed
// gateway does not re-bind and a re-registered gateway that reuses the name resolves to the
// live object.
const SelfManagedGatewayStatusDeleted = "DELETED"

// --- Domain Models ---

// SelfManagedGatewayReplica represents one replica (running Flex runtime instance)
// of a self-managed gateway, as reported inside the gateway object's "replicas" array.
//
// LIVE-VERIFIED shape (2026-07-21, a real registered runtime): a gateway reports one
// replica entry PER connectivity status bucket, e.g.
//
//	"replicas":[{"status":"CONNECTED","count":0,"certificateExpirationDates":[]},
//	            {"status":"DISCONNECTED","count":0,"certificateExpirationDates":[]}]
type SelfManagedGatewayReplica struct {
	Status                     string   `json:"status,omitempty"`
	Count                      int64    `json:"count"`
	CertificateExpirationDates []string `json:"certificateExpirationDates,omitempty"`
}

// SelfManagedGatewayReplicaConfigStatus is the per-replica configuration-sync status
// nested inside a replica-detail entry (the "configurationStatus" object).
//
// LIVE-VERIFIED shape (2026-07-26): { "status":"UP_TO_DATE", "message":null }. The
// message is null while the replica's applied configuration matches the desired state;
// it carries a human-readable reason when the replica is drifting or failed to apply.
type SelfManagedGatewayReplicaConfigStatus struct {
	Status  string  `json:"status,omitempty"`
	Message *string `json:"message"`
}

// SelfManagedGatewayReplicaDetail is ONE concrete replica (a single running/registered
// Flex runtime node) as reported by the dedicated per-gateway /replicas endpoint that
// backs the Runtime Manager "Replicas" tab.
//
// This is deliberately DISTINCT from SelfManagedGatewayReplica: that coarse struct is the
// status-BUCKET summary embedded in the gateway object (one entry per connectivity status
// with a running count), whereas this struct is the RICH per-node detail — one entry per
// actual replica, with identity (id/nodeId/name/cid), version, connect/disconnect
// timestamps, per-replica certificate expiry, and configuration-sync status.
//
// LIVE-VERIFIED field set (2026-07-26) against a real registered gateway with two nodes:
//
//	{ "id":"…", "targetId":"<gatewayId>", "gatewayVersion":"1.13.3",
//	  "status":"CONNECTED", "connectedAt":"…", "disconnectedAt":null,
//	  "configurationStatus":{"status":"UP_TO_DATE","message":null},
//	  "name":"d6c016e2693e.default", "cid":"arm-mcm2-service-…",
//	  "certificateExpirationDate":"2027-10-31T15:31:46.00Z", "nodeId":"…", "provider":"RR" }
//
// disconnectedAt is null for a currently-connected replica. Unknown fields are ignored on
// decode, so this remains forward-compatible if new keys appear.
type SelfManagedGatewayReplicaDetail struct {
	ID                        string                                `json:"id"`
	TargetID                  string                                `json:"targetId,omitempty"`
	GatewayVersion            string                                `json:"gatewayVersion,omitempty"`
	Status                    string                                `json:"status,omitempty"`
	ConnectedAt               *string                               `json:"connectedAt"`
	DisconnectedAt            *string                               `json:"disconnectedAt"`
	ConfigurationStatus       SelfManagedGatewayReplicaConfigStatus `json:"configurationStatus"`
	Name                      string                                `json:"name,omitempty"`
	Cid                       string                                `json:"cid,omitempty"`
	CertificateExpirationDate *string                               `json:"certificateExpirationDate"`
	NodeID                    string                                `json:"nodeId,omitempty"`
	Provider                  string                                `json:"provider,omitempty"`
}

// SelfManagedGateway represents a self-managed (connected-mode) Flex gateway object
// as returned by the standalone gateways endpoint.
//
// LIVE-VERIFIED field set (2026-07-21) — captured from a real Flex runtime that
// self-registered against production. The authoritative LIST/GET payload is:
//
//	{ "organizationId":"…", "id":"…", "name":"…", "status":"DISCONNECTED",
//	  "tags":[], "lastUpdate":"2026-07-21T14:29:07.69Z",
//	  "replicas":[{"status":"CONNECTED","count":0,"certificateExpirationDates":[]}, …] }
//
// GET-by-id additionally returns "versions":[] (an array; empty until replicas report
// a runtime version). NOTE the key is "lastUpdate" (no trailing 'd'). Earlier drafts of
// this struct assumed version/region/connected/type/dateCreated/environmentId — NONE of
// those keys exist in the real payload, so they were removed. Unknown fields are ignored
// on decode, so this remains forward-compatible if new keys appear.
type SelfManagedGateway struct {
	ID             string                      `json:"id"`
	Name           string                      `json:"name"`
	Status         string                      `json:"status,omitempty"`
	OrganizationID string                      `json:"organizationId,omitempty"`
	LastUpdate     string                      `json:"lastUpdate,omitempty"`
	Tags           []string                    `json:"tags,omitempty"`
	Replicas       []SelfManagedGatewayReplica `json:"replicas,omitempty"`
	Versions       []string                    `json:"versions,omitempty"`
}

// SelfManagedGatewayListResponse wraps the paginated list response from the
// standalone gateways endpoint (envelope shape live-verified: content/totalElements/
// pageNumber/pageSize).
type SelfManagedGatewayListResponse struct {
	Content       []SelfManagedGateway `json:"content"`
	PageSize      int                  `json:"pageSize"`
	PageNumber    int                  `json:"pageNumber"`
	TotalElements int                  `json:"totalElements"`
}

// SelfManagedGatewayReplicaListResponse wraps the paginated list response from the
// dedicated per-gateway /replicas endpoint.
//
// LIVE-VERIFIED envelope (2026-07-26): { "content":[...], "totalElements":N,
// "pageNumber":P, "pageSize":S, "gateway":{...} }. The envelope also echoes the parent
// "gateway" object (the same coarse shape returned by GetSelfManagedGateway); we ignore
// it here because the caller already has the gateway and only wants the rich per-replica
// content.
type SelfManagedGatewayReplicaListResponse struct {
	Content       []SelfManagedGatewayReplicaDetail `json:"content"`
	PageSize      int                               `json:"pageSize"`
	PageNumber    int                               `json:"pageNumber"`
	TotalElements int                               `json:"totalElements"`
}

// RegistrationTokenResponse is the response from the token-mint endpoint.
// Body live-verified as { "registrationToken": "<opaque string>" }.
type RegistrationTokenResponse struct {
	RegistrationToken string `json:"registrationToken"`
}

// --- Registration token (the "create" primitive) ---

// MintRegistrationToken mints a short-lived registration token scoped to the given
// org/env. This token is what the customer feeds to the Flex runtime (via flexctl or a
// container env var) so the runtime can self-register as a connected-mode gateway.
//
// Contract (live-verified route: OPTIONS -> 200 Allow: POST,OPTIONS):
//
//	POST {BASE}/standalone/api/v1/organizations/{org}/environments/{env}/gatewaytokens
//	Body: EMPTY (token is minted from the auth context + org/env headers)
//	200 -> { "registrationToken": "<opaque string>" }
//
// The token is a WRITE-side secret: callers must treat it as sensitive and must not
// log it. Minting is a mutating call, so it is only invoked from resource Create.
func (c *SelfManagedGatewayClient) MintRegistrationToken(ctx context.Context, orgID, envID string) (*RegistrationTokenResponse, error) {
	url := fmt.Sprintf("%s%s/organizations/%s/environments/%s/gatewaytokens",
		c.BaseURL, selfManagedBasePath, orgID, envID)

	// Body is intentionally empty; the JMeter capture had postBodyRaw=false.
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		// The standalone gateway API needs the connected app's Runtime Manager server
		// scopes (Manage/Read Servers, View Organization); a token missing them is
		// rejected with 401/403. Surface actionable scope guidance.
		if authErr := client.AuthContextErrorIfUnauthorized(resp.StatusCode, url, string(body)); authErr != nil {
			return nil, authErr
		}
		return nil, fmt.Errorf("failed to mint self-managed gateway registration token with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp RegistrationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode registration token response: %w", err)
	}

	return &tokenResp, nil
}

// --- Gateway object lifecycle (LIST / GET / DELETE only) ---

// selfManagedGatewayPageSize is the page size used when listing self-managed gateways.
// LIVE-VERIFIED (2026-07-21): the /standalone list endpoint pages via pageSize+pageNumber
// (0-indexed) and enforces a MAXIMUM pageSize of 100 — pageSize>=200 returns HTTP 400 with
// body "Wrong page size". We therefore request the max (100) to minimize round-trips.
const selfManagedGatewayPageSize = 100

// ListSelfManagedGateways returns ALL self-managed gateways registered in the given
// org/environment, following pagination to completion.
//
// Contract (live-verified 200):
//
//	GET {BASE}/standalone/api/v1/organizations/{org}/environments/{env}/gateways?pageSize=100&pageNumber=N
//	-> { "content":[...], "totalElements":M, "pageNumber":N, "pageSize":100 }
//
// This MUST paginate. The endpoint defaults to pageSize=30 when the param is omitted, so a
// naive single-request implementation silently truncates any org/env with more than 30
// self-managed gateways. That truncation is NOT benign: the resource's resolveGateway scans
// this list by name, so a gateway on page 2+ would be reported "not currently registered" —
// clearGatewayFields would then wipe its gateway_id/status/last_update (phantom drift) AND,
// because Delete only fires when gateway_id != "", the gateway would become impossible to
// destroy via Terraform (a resource leak). The plural data source would likewise omit those
// gateways. Same bug class as the roles/team pagination fixes.
//
// Stop condition: a SHORT page (fewer than pageSize items) is the authoritative "last page"
// signal. We deliberately do NOT stop on totalElements — LIVE-VERIFIED, requesting a
// pageNumber beyond the last page returns HTTP 200 with an empty content array AND
// totalElements reset to 0, so totalElements is not a reliable running bound. An empty page
// therefore also breaks the loop.
func (c *SelfManagedGatewayClient) ListSelfManagedGateways(ctx context.Context, orgID, envID string) ([]SelfManagedGateway, error) {
	var all []SelfManagedGateway
	pageNumber := 0

	for {
		url := fmt.Sprintf("%s%s/organizations/%s/environments/%s/gateways?pageSize=%d&pageNumber=%d",
			c.BaseURL, selfManagedBasePath, orgID, envID, selfManagedGatewayPageSize, pageNumber)

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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			// The standalone gateway API needs the connected app's Runtime Manager server
			// scopes (Manage/Read Servers, View Organization); a token missing them is
			// rejected with 401/403. Surface actionable scope guidance.
			if authErr := client.AuthContextErrorIfUnauthorized(resp.StatusCode, url, string(body)); authErr != nil {
				return nil, authErr
			}
			return nil, fmt.Errorf("failed to list self-managed gateways with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp SelfManagedGatewayListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		all = append(all, listResp.Content...)

		// A short (or empty) page is the last page. See stop-condition note above; we cannot
		// trust totalElements because it resets to 0 on an out-of-range page request.
		if len(listResp.Content) < selfManagedGatewayPageSize {
			break
		}
		pageNumber++
	}

	return all, nil
}

// GetSelfManagedGateway retrieves a single self-managed gateway by ID.
//
// Contract (live-verified: 404 body "Gateway not found by id" for a missing id):
//
//	GET {BASE}/standalone/api/v1/organizations/{org}/environments/{env}/gateways/{id}
func (c *SelfManagedGatewayClient) GetSelfManagedGateway(ctx context.Context, orgID, envID, gatewayID string) (*SelfManagedGateway, error) {
	url := fmt.Sprintf("%s%s/organizations/%s/environments/%s/gateways/%s",
		c.BaseURL, selfManagedBasePath, orgID, envID, gatewayID)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("self-managed gateway")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get self-managed gateway with status %d: %s", resp.StatusCode, string(body))
	}

	var gw SelfManagedGateway
	if err := json.NewDecoder(resp.Body).Decode(&gw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &gw, nil
}

// GetSelfManagedGatewayReplicas returns the RICH per-replica detail for a gateway, following
// pagination to completion. This backs the Runtime Manager "Replicas" tab and is distinct
// from the coarse status-bucket replicas embedded in the gateway object (GetSelfManagedGateway):
// it returns one entry per concrete runtime node with identity, version, connect/disconnect
// timestamps, per-replica certificate expiry, and configuration-sync status.
//
// Contract (live-verified 200, 2026-07-26):
//
//	GET {BASE}/standalone/api/v1/organizations/{org}/environments/{env}/gateways/{id}/replicas?pageSize=100&pageNumber=N
//	-> { "content":[{id,targetId,gatewayVersion,status,connectedAt,disconnectedAt,
//	                 configurationStatus:{status,message},name,cid,
//	                 certificateExpirationDate,nodeId,provider}, ...],
//	     "totalElements":M, "pageNumber":N, "pageSize":S, "gateway":{...} }
//
// Pagination matches the /gateways endpoint EXACTLY (live-verified 2026-07-26): default
// pageSize=30, MAXIMUM pageSize=100 (pageSize>=200 → HTTP 400 "Wrong page size"), and an
// out-of-range pageNumber returns HTTP 200 with an empty content array. We therefore request
// the max page size (reusing selfManagedGatewayPageSize) and stop on a SHORT (or empty) page.
// We deliberately do NOT bound the loop on totalElements: like /gateways, an out-of-range page
// is a clean empty 200, so the short-page signal is the authoritative last-page condition.
//
// A 404 (gateway id absent) is surfaced as a NotFoundError so the data source can distinguish
// "gateway gone" from a transport error, mirroring GetSelfManagedGateway.
func (c *SelfManagedGatewayClient) GetSelfManagedGatewayReplicas(ctx context.Context, orgID, envID, gatewayID string) ([]SelfManagedGatewayReplicaDetail, error) {
	var all []SelfManagedGatewayReplicaDetail
	pageNumber := 0

	for {
		url := fmt.Sprintf("%s%s/organizations/%s/environments/%s/gateways/%s/replicas?pageSize=%d&pageNumber=%d",
			c.BaseURL, selfManagedBasePath, orgID, envID, gatewayID, selfManagedGatewayPageSize, pageNumber)

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

		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()
			return nil, client.NewNotFoundError("self-managed gateway")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to get self-managed gateway replicas with status %d: %s", resp.StatusCode, string(body))
		}

		var listResp SelfManagedGatewayReplicaListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		_ = resp.Body.Close()

		all = append(all, listResp.Content...)

		// A short (or empty) page is the last page. Out-of-range pages return an empty 200,
		// so this is the authoritative stop condition (totalElements is not a reliable bound).
		if len(listResp.Content) < selfManagedGatewayPageSize {
			break
		}
		pageNumber++
	}

	return all, nil
}

// DeleteSelfManagedGateway deletes a registered self-managed gateway by ID.
//
// Contract (live-verified 2026-07-21 with a real registered gateway):
//
//	DELETE {BASE}/standalone/api/v1/organizations/{org}/environments/{env}/gateways/{id}
//
// The delete is an ASYNC SOFT-DELETE: it returns HTTP 202 Accepted and flips the object's
// status to "DELETED", but the object LINGERS in list/GET permanently as a tombstone (it is
// never hard-removed; ?force=true does not change this). Consequences this method must
// tolerate for Terraform idempotency (destroy, or destroy after an out-of-band UI delete):
//
//   - 202/200/204 → success (the normal soft-delete acknowledgement).
//   - 404 → treated as success. The item route returns this for an id that never existed;
//     harmless for destroy.
//   - 400 with body "This target was already deleted" → treated as success. This is what the
//     real API returns when deleting an already-DELETED tombstone; from Terraform's
//     perspective the desired end-state (gateway gone) already holds, so destroy must not
//     fail. Any OTHER 400 is a genuine error and is surfaced.
//
// Deleting the platform-side object does NOT stop the customer's runtime; the operator
// must also tear down the Flex runtime on their own infrastructure.
func (c *SelfManagedGatewayClient) DeleteSelfManagedGateway(ctx context.Context, orgID, envID, gatewayID string) error {
	url := fmt.Sprintf("%s%s/organizations/%s/environments/%s/gateways/%s",
		c.BaseURL, selfManagedBasePath, orgID, envID, gatewayID)

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

	// Success statuses: 202 (async soft-delete ack), plus 200/204 for forward-compatibility.
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}

	// A missing gateway is a successful delete (idempotent) from Terraform's perspective.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	// The soft-delete tombstone lingers forever, so a second delete (e.g. destroy after an
	// out-of-band UI delete, or a retried destroy) hits an already-DELETED object and the API
	// returns 400 "This target was already deleted". The end-state already matches intent, so
	// this is idempotent success — but only for THIS specific message; other 400s are real.
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "already deleted") {
		return nil
	}

	return fmt.Errorf("failed to delete self-managed gateway with status %d: %s", resp.StatusCode, string(body))
}
