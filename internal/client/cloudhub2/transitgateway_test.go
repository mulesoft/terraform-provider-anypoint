package cloudhub2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestTransitGatewayClient_UpdateTransitGatewayRoutes verifies the routes update
// contract confirmed live 2026-07-17 (read-only probes): routes are a field on
// the private-space CONNECTION object, so the update PATCHes the connection
// itself (.../transitgateways/{tgwId}) — NOT a {tgwId}/routes sub-resource, which
// 404s at every scope — with an OBJECT body {"name":...,"routes":[...]}. A nil
// slice must serialise to "routes":[] (never null): the live handler rejects
// "routes":null with 400 "Routes cannot be null".
func TestTransitGatewayClient_UpdateTransitGatewayRoutes(t *testing.T) {
	const connPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"
	// The dedicated /routes sub-resource must NEVER be hit — it does not exist.
	const routesSubPath = connPath + "/routes"

	tests := []struct {
		name        string
		tgwName     string
		routes      []string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:    "object body {name,routes} on the connection path",
			tgwName: "my-tgw",
			routes:  []string{"10.0.0.0/8", "192.168.0.0/16"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, http.MethodPatch, connPath)

				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}
				// The body MUST be an object carrying name + routes. A bare array
				// would fail to decode into this struct.
				var body struct {
					Name   string   `json:"name"`
					Routes []string `json:"routes"`
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatalf("body must be a {name,routes} object, got %q (err: %v)", string(raw), err)
				}
				if body.Name != "my-tgw" {
					t.Errorf("expected name echoed as 'my-tgw', got %q", body.Name)
				}
				if len(body.Routes) != 2 || body.Routes[0] != "10.0.0.0/8" || body.Routes[1] != "192.168.0.0/16" {
					t.Errorf("unexpected routes body: %v", body.Routes)
				}
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name:    "nil routes marshals to empty array, never null (clear all)",
			tgwName: "my-tgw",
			routes:  nil,
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				// The handler rejects "routes":null with 400; the body must carry
				// an explicit empty array instead.
				if strings.Contains(string(raw), "null") {
					t.Errorf("nil routes must not marshal to null, got %q", string(raw))
				}
				if !strings.Contains(string(raw), "\"routes\":[]") {
					t.Errorf("nil routes must serialise to \"routes\":[], got %q", string(raw))
				}
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name:    "server error is surfaced",
			tgwName: "my-tgw",
			routes:  []string{"10.0.0.0/8"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
			},
			wantErr:     true,
			errContains: "failed to update transit gateway routes with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				connPath: tt.mockHandler,
				routesSubPath: func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("routes update must NOT hit the non-existent /routes sub-resource %s", routesSubPath)
					testutil.ErrorResponse(w, http.StatusNotFound, "no such endpoint")
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TransitGatewayClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.UpdateTransitGatewayRoutes(context.Background(), "test-org-id", "test-ps-id", "tgw-123", tt.tgwName, tt.routes)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestTransitGatewayClient_GetTransitGateway exercises the single-object GET
// path used by the resource's Read/Update/Import. The most important case is
// "enveloped 200 body": the private-spaces LIST endpoint proved the platform can
// wrap a payload the spec says is bare, and a plain json.Decode(&struct) would
// swallow that into an all-zero struct with NO error. The decode guard must turn
// that into a loud, diagnosable failure instead of silently corrupting state.
func TestTransitGatewayClient_GetTransitGateway(t *testing.T) {
	const tgwPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"

	tests := []struct {
		name        string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		wantID      string
		wantStatus  string
	}{
		{
			name: "successful bare-object get",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, http.MethodGet, tgwPath)
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"id":   "tgw-123",
					"name": "prod-tgw",
					"spec": map[string]interface{}{
						"resourceShare": map[string]interface{}{"id": "share-uuid", "account": "123456789012"},
						"region":        "us-east-1",
					},
					"status": map[string]interface{}{
						"gateway": "available", "attachment": "available", "tgwResource": "tgw-0abc",
						"routes": []string{"10.0.0.0/8"},
					},
				})
			},
			wantID:     "tgw-123",
			wantStatus: "available",
		},
		{
			name: "not found maps to typed NotFound error",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
			wantErr:     true,
			errContains: "transit gateway",
		},
		{
			name: "server error is surfaced",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
			},
			wantErr:     true,
			errContains: "failed to get transit gateway with status 500",
		},
		{
			name: "malformed json errors",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
		{
			name: "enveloped 200 body is rejected loudly, not silently zero-valued",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				// The gateway is wrapped under a "data" key. A plain decode would
				// yield an empty struct with no error; the guard must catch it.
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"data": map[string]interface{}{
						"id":   "tgw-123",
						"name": "prod-tgw",
					},
				})
			},
			wantErr:     true,
			errContains: "empty id",
		},
		{
			name: "empty object 200 body is rejected loudly",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{})
			},
			wantErr:     true,
			errContains: "empty id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				tgwPath: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TransitGatewayClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			tgw, err := c.GetTransitGateway(context.Background(), "test-org-id", "test-ps-id", "tgw-123")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (tgw=%+v)", tgw)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tgw.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tgw.ID, tt.wantID)
			}
			if tgw.Status.Gateway != tt.wantStatus {
				t.Errorf("Status.Gateway = %q, want %q", tgw.Status.Gateway, tt.wantStatus)
			}
		})
	}
}

// TestTransitGatewayClient_GetTransitGateway_NotFoundType asserts the 404 path
// returns the typed NotFound error the resource relies on to detect deletion
// out-of-band (so Read can remove the resource from state instead of erroring).
func TestTransitGatewayClient_GetTransitGateway_NotFoundType(t *testing.T) {
	const tgwPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &TransitGatewayClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	_, err := c.GetTransitGateway(context.Background(), "test-org-id", "test-ps-id", "tgw-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected a NotFound-typed error, got %T: %v", err, err)
	}
}

// TestTransitGatewayClient_GetTransitGateway_DetachedType pins W-23819332: a
// detached-but-registered attachment makes the PS-scoped by-id GET return 400
// "attachment is not attached to the private space". That specific 400 must map
// to the typed ErrTransitGatewayDetached (so Read/Import can keep the resource
// and surface a Detached status), while an UNRELATED 400 must stay a plain error
// (never misclassified as detached) and a 404 must remain NotFound, not detached.
func TestTransitGatewayClient_GetTransitGateway_DetachedType(t *testing.T) {
	const tgwPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"

	tests := []struct {
		name       string
		status     int
		body       string
		wantDetach bool
		wantErrSub string
		wantNotFnd bool
	}{
		{
			name:       "detached 400 maps to typed detached error",
			status:     http.StatusBadRequest,
			body:       "attachment is not attached to the private space",
			wantDetach: true,
			wantErrSub: "detached",
		},
		{
			name:       "detached 400 with different casing still matches",
			status:     http.StatusBadRequest,
			body:       `{"message":"Attachment is NOT Attached To The Private Space"}`,
			wantDetach: true,
		},
		{
			name:       "detached 400 DELETE-verb phrasing (no 'the') still matches",
			status:     http.StatusBadRequest,
			body:       `{"message":"Transit gateway: tgw-123 is not attached to private space rf-p2-private-space"}`,
			wantDetach: true,
		},
		{
			name:       "unrelated 400 stays a generic error",
			status:     http.StatusBadRequest,
			body:       "some other validation failure",
			wantDetach: false,
			wantErrSub: "failed to get transit gateway with status 400",
		},
		{
			name:       "404 stays NotFound, not detached",
			status:     http.StatusNotFound,
			body:       "not found",
			wantDetach: false,
			wantNotFnd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				tgwPath: func(w http.ResponseWriter, r *http.Request) {
					testutil.ErrorResponse(w, tt.status, tt.body)
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TransitGatewayClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			_, err := c.GetTransitGateway(context.Background(), "test-org-id", "test-ps-id", "tgw-123")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := IsTransitGatewayDetached(err); got != tt.wantDetach {
				t.Errorf("IsTransitGatewayDetached = %v, want %v (err=%v)", got, tt.wantDetach, err)
			}
			if got := client.IsNotFound(err); got != tt.wantNotFnd {
				t.Errorf("IsNotFound = %v, want %v (err=%v)", got, tt.wantNotFnd, err)
			}
			if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error = %v, want containing %q", err, tt.wantErrSub)
			}
		})
	}
}

// TestTransitGatewayClient_UpdateTransitGateway_OrgScopedNameOnly pins the rename
// contract that the Anypoint UI uses (confirmed live 2026-07-17 via browser
// DevTools):
//
//   - the PATCH targets the ORG-scoped endpoint
//     /organizations/{orgId}/transitgateways/{tgwId} — there is NO
//     /privatespaces/{psId}/ segment; and
//   - the body is name-only ({"name":...}) with NO routes field.
//
// This is a regression guard against two earlier wrong turns: (1) using the
// private-space-scoped endpoint the RAML documents, which silently ignores the
// name, and (2) adding a routes array to the body to dodge that endpoint's
// "Routes cannot be null" 400. The org-scoped endpoint needs neither.
func TestTransitGatewayClient_UpdateTransitGateway_OrgScopedNameOnly(t *testing.T) {
	const orgScopedPath = "/runtimefabric/api/organizations/test-org-id/transitgateways/tgw-123"
	// The private-space-scoped path must NEVER be hit for a rename.
	const psScopedPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		psScopedPath: func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("rename must NOT hit the private-space-scoped endpoint %s (it silently ignores name)", psScopedPath)
			testutil.ErrorResponse(w, http.StatusBadRequest, "wrong endpoint")
		},
		orgScopedPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, http.MethodPatch, orgScopedPath)
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			// Body must be name-only. Routes must NOT appear in any form.
			if strings.Contains(string(raw), "routes") {
				t.Errorf("org-scoped rename body must be name-only, must not mention routes, got %q", string(raw))
			}
			var body map[string]interface{}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			if body["name"] != "new-name" {
				t.Errorf("expected name 'new-name', got %v", body["name"])
			}
			if _, hasRoutes := body["routes"]; hasRoutes {
				t.Errorf("org-scoped rename body must not include a routes key, got %q", string(raw))
			}
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "tgw-123",
				"name": "new-name",
				"status": map[string]interface{}{
					"gateway": "available", "attachment": "available", "tgwResource": "tgw-0abc",
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &TransitGatewayClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	tgw, err := c.UpdateTransitGateway(context.Background(), "test-org-id", "test-ps-id", "tgw-123",
		&UpdateTransitGatewayRequest{Name: "new-name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tgw.Name != "new-name" {
		t.Errorf("expected name 'new-name', got %s", tgw.Name)
	}
}

// TestTransitGatewayClient_UpdateTransitGateway_ArrayResponse pins the response-shape
// divergence confirmed live 2026-07-17 (browser DevTools): the org-scoped rename
// PATCH returns 200 with a JSON ARRAY (the org's TGW list), not a single object.
// A plain decode into a struct errors with "cannot unmarshal array into Go value";
// UpdateTransitGateway must tolerate the array and pick the element matching the id.
func TestTransitGatewayClient_UpdateTransitGateway_ArrayResponse(t *testing.T) {
	const tgwPath = "/runtimefabric/api/organizations/test-org-id/transitgateways/tgw-123"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, http.MethodPatch, tgwPath)
			// Live shape: a JSON array of gateways, including one that is NOT ours
			// first, to prove we select by id rather than blindly taking [0].
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{
					"id":   "tgw-other",
					"name": "someone-else",
					"status": map[string]interface{}{
						"gateway": "available", "attachment": "available", "tgwResource": "tgw-0zzz",
					},
				},
				{
					"id":   "tgw-123",
					"name": "new-name",
					"status": map[string]interface{}{
						"gateway": "available", "attachment": "available", "tgwResource": "tgw-0abc",
						"routes": []string{"10.0.0.0/8"},
					},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &TransitGatewayClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	tgw, err := c.UpdateTransitGateway(context.Background(), "test-org-id", "test-ps-id", "tgw-123",
		&UpdateTransitGatewayRequest{Name: "new-name"})
	if err != nil {
		t.Fatalf("unexpected error decoding array response: %v", err)
	}
	if tgw.ID != "tgw-123" {
		t.Errorf("expected the element matching tgw-123, got id %q (name %q)", tgw.ID, tgw.Name)
	}
	if tgw.Name != "new-name" {
		t.Errorf("expected name 'new-name', got %q", tgw.Name)
	}
}

// TestTransitGatewayClient_DeleteTransitGateway_AlwaysDeregistersOrgScoped pins
// the full destroy contract for W-23819332. The platform models "attached to a
// private space" and "registered org-wide" as two separate bindings, so a
// complete teardown needs BOTH a PS-scoped detach AND an org-scoped de-register:
//
//   - A PS-scoped DELETE alone only detaches; the object lingers org-wide holding
//     the AWS RAM share, and a later create against the same TGW/share 400s
//     "resource share id already exists". So EVERY successful delete — even a
//     healthy 204 — must go on to hit the org-scoped endpoint (happy-path ghost).
//   - The PS-scoped 400 for an already-detached attachment is phrased WITHOUT the
//     article on the DELETE verb: "...is not attached to private space <name>"
//     (the GET verb says "...to THE private space"). Both must be tolerated as
//     "already detached, proceed to de-register", not treated as a hard error —
//     the article-only mismatch is exactly what made the earlier build's fallback
//     never fire (retest 2026-08-12).
//
// A genuine PS-scoped failure (unrelated 400, 5xx) must stay a hard error and
// must NOT reach the org-scoped call.
func TestTransitGatewayClient_DeleteTransitGateway_AlwaysDeregistersOrgScoped(t *testing.T) {
	const psPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"
	const orgPath = "/runtimefabric/api/organizations/test-org-id/transitgateways/tgw-123"
	// GET-verb phrasing (with "the").
	const detachedGetMsg = "attachment is not attached to the private space"
	// DELETE-verb phrasing (no "the", trailing space name) — the exact string
	// Lisandro's STGX destroy returned; the earlier matcher missed this.
	const detachedDeleteMsg = "Transit gateway: tgw-123 is not attached to private space rf-p2-private-space"

	tests := []struct {
		name        string
		psStatus    int
		psBody      string
		orgStatus   int
		wantErr     bool
		errContains string
		wantOrgHit  bool
	}{
		{
			name:       "healthy PS 204 STILL de-registers org-wide (no ghost left)",
			psStatus:   http.StatusNoContent,
			orgStatus:  http.StatusNoContent,
			wantOrgHit: true,
		},
		{
			name:       "PS 404 (already off space) still de-registers org-wide",
			psStatus:   http.StatusNotFound,
			psBody:     "not found",
			orgStatus:  http.StatusNoContent,
			wantOrgHit: true,
		},
		{
			name:       "detached 400 GET-phrasing (the private space) proceeds to org de-register",
			psStatus:   http.StatusBadRequest,
			psBody:     detachedGetMsg,
			orgStatus:  http.StatusNoContent,
			wantOrgHit: true,
		},
		{
			name:       "detached 400 DELETE-phrasing (no 'the') proceeds to org de-register",
			psStatus:   http.StatusBadRequest,
			psBody:     detachedDeleteMsg,
			orgStatus:  http.StatusNoContent,
			wantOrgHit: true,
		},
		{
			name:       "org-scoped 404 (already de-registered) is idempotent success",
			psStatus:   http.StatusNoContent,
			orgStatus:  http.StatusNotFound,
			wantOrgHit: true,
		},
		{
			name:       "202 Accepted (async delete) on both steps is success",
			psStatus:   http.StatusAccepted,
			orgStatus:  http.StatusAccepted,
			wantOrgHit: true,
		},
		{
			name:        "org-scoped error surfaces as a de-register failure",
			psStatus:    http.StatusNoContent,
			orgStatus:   http.StatusInternalServerError,
			wantErr:     true,
			errContains: "org-scoped de-register",
			wantOrgHit:  true,
		},
		{
			name:        "unrelated PS 400 stays a hard error, never reaches org-scoped",
			psStatus:    http.StatusBadRequest,
			psBody:      "some other validation failure",
			wantErr:     true,
			errContains: "failed to detach transit gateway from private space with status 400",
			wantOrgHit:  false,
		},
		{
			name:        "PS 5xx stays a hard error, never reaches org-scoped",
			psStatus:    http.StatusInternalServerError,
			psBody:      "boom",
			wantErr:     true,
			errContains: "failed to detach transit gateway from private space with status 500",
			wantOrgHit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgHit := false
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				psPath: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertHTTPRequest(t, r, http.MethodDelete, psPath)
					testutil.ErrorResponse(w, tt.psStatus, tt.psBody)
				},
				orgPath: func(w http.ResponseWriter, r *http.Request) {
					orgHit = true
					testutil.AssertHTTPRequest(t, r, http.MethodDelete, orgPath)
					testutil.ErrorResponse(w, tt.orgStatus, "org-scoped")
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TransitGatewayClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.DeleteTransitGateway(context.Background(), "test-org-id", "test-ps-id", "tgw-123")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if orgHit != tt.wantOrgHit {
				t.Errorf("org-scoped delete hit = %v, want %v", orgHit, tt.wantOrgHit)
			}
		})
	}
}

// TestTransitGatewayClient_CreateTransitGateway_ShareConflict pins the create-side
// half of W-23819332. When the AWS RAM resource share is still bound to an existing
// (often detached-but-org-registered) attachment, create fails with
// "...resource share id already exists". A bare status error is a dead end for the
// user; the client must instead surface actionable de-register guidance (mention
// the org-scoped delete) so a stranded ghost from an earlier build can be cleared.
// This runs whether the platform reports the conflict as 400 or 409. An UNRELATED
// error must keep the plain message (not the share-conflict guidance).
func TestTransitGatewayClient_CreateTransitGateway_ShareConflict(t *testing.T) {
	const createPath = "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways"

	tests := []struct {
		name        string
		status      int
		body        string
		wantGuide   bool
		errContains string
	}{
		{
			name:        "409 share-already-exists yields de-register guidance",
			status:      http.StatusConflict,
			body:        "Transit gateway attachment with the same resource share id already exists",
			wantGuide:   true,
			errContains: "De-register that attachment",
		},
		{
			name:        "400 share-already-exists also yields guidance",
			status:      http.StatusBadRequest,
			body:        `{"message":"Transit gateway attachment with the same Resource Share id already exists"}`,
			wantGuide:   true,
			errContains: "org-scoped delete",
		},
		{
			name:        "unrelated 400 keeps the plain error, no guidance",
			status:      http.StatusBadRequest,
			body:        "some other validation failure",
			wantGuide:   false,
			errContains: "failed to create transit gateway with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				createPath: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertHTTPRequest(t, r, http.MethodPost, createPath)
					testutil.ErrorResponse(w, tt.status, tt.body)
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TransitGatewayClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			_, err := c.CreateTransitGateway(context.Background(), "test-org-id", "test-ps-id",
				&CreateTransitGatewayRequest{Name: "my-tgw", ResourceShareID: "share-uuid", ResourceShareAccount: "123456789012", Routes: []string{"10.0.0.0/8"}})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, want containing %q", err, tt.errContains)
			}
			gotGuide := strings.Contains(err.Error(), "resource share is still bound")
			if gotGuide != tt.wantGuide {
				t.Errorf("share-conflict guidance present = %v, want %v (err=%v)", gotGuide, tt.wantGuide, err)
			}
		})
	}
}
