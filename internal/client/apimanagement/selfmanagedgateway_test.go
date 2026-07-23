package apimanagement

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func newTestSelfManagedClient(baseURL string) *SelfManagedGatewayClient {
	return &SelfManagedGatewayClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    baseURL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
}

func TestNewSelfManagedGatewayClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *client.Config
		wantErr bool
	}{
		{name: "valid", config: &client.Config{ClientID: "id", ClientSecret: "secret"}, wantErr: false},
		{name: "missing client id", config: &client.Config{ClientSecret: "secret"}, wantErr: true},
		{name: "missing client secret", config: &client.Config{ClientID: "id"}, wantErr: true},
		{name: "nil config", config: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
				tt.config.BaseURL = server.URL
			}
			_, err := NewSelfManagedGatewayClient(tt.config)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// --- MintRegistrationToken ---

func TestSelfManagedGatewayClient_MintRegistrationToken(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gatewaytokens"

	var gotMethod string
	var gotBody string
	var gotOrgHeader, gotEnvHeader string

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			gotOrgHeader = r.Header.Get("X-ANYPNT-ORG-ID")
			gotEnvHeader = r.Header.Get("X-ANYPNT-ENV-ID")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"registrationToken": "opaque-token-value",
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	resp, err := c.MintRegistrationToken(context.Background(), "test-org-id", "test-env-id")
	if err != nil {
		t.Fatalf("MintRegistrationToken() error: %v", err)
	}
	if resp.RegistrationToken != "opaque-token-value" {
		t.Errorf("RegistrationToken = %q, want opaque-token-value", resp.RegistrationToken)
	}
	// Contract: the mint is a POST with an EMPTY body — the token is derived from the
	// auth context + org/env headers, not from a request payload.
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if strings.TrimSpace(gotBody) != "" {
		t.Errorf("body = %q, want empty (token minted from headers/context)", gotBody)
	}
	if gotOrgHeader != "test-org-id" {
		t.Errorf("X-ANYPNT-ORG-ID = %q, want test-org-id", gotOrgHeader)
	}
	if gotEnvHeader != "test-env-id" {
		t.Errorf("X-ANYPNT-ENV-ID = %q, want test-env-id", gotEnvHeader)
	}
}

func TestSelfManagedGatewayClient_MintRegistrationToken_Error(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gatewaytokens"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	_, err := c.MintRegistrationToken(context.Background(), "test-org-id", "test-env-id")
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

// --- ListSelfManagedGateways ---

func TestSelfManagedGatewayClient_ListSelfManagedGateways(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			// Payload shape LIVE-VERIFIED (2026-07-21) against a real registered runtime:
			// each gateway has id/name/status/organizationId/lastUpdate/tags/replicas. The
			// replicas array reports one entry per connectivity status bucket. NOTE the key
			// is "lastUpdate" (no trailing 'd'); there is no version/region/dateCreated key.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"id":             "gw-1",
						"name":           "flex-a",
						"status":         "DISCONNECTED",
						"organizationId": "test-org-id",
						"lastUpdate":     "2026-07-21T14:29:07.69Z",
						"tags":           []string{"team-a", "prod"},
						"replicas": []map[string]interface{}{
							{"status": "CONNECTED", "count": 2, "certificateExpirationDates": []string{"2027-01-01T00:00:00Z"}},
							{"status": "DISCONNECTED", "count": 0, "certificateExpirationDates": []string{}},
						},
					},
					{
						"id":         "gw-2",
						"name":       "flex-b",
						"status":     "DELETED",
						"lastUpdate": "2026-07-20T10:00:00Z",
					},
				},
				"totalElements": 2,
				"pageNumber":    0,
				"pageSize":      30,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	gws, err := c.ListSelfManagedGateways(context.Background(), "test-org-id", "test-env-id")
	if err != nil {
		t.Fatalf("ListSelfManagedGateways() error: %v", err)
	}
	if len(gws) != 2 {
		t.Fatalf("len = %d, want 2", len(gws))
	}
	if gws[0].ID != "gw-1" || gws[0].Name != "flex-a" || gws[0].Status != "DISCONNECTED" {
		t.Errorf("gw[0] = %+v, unexpected", gws[0])
	}
	if gws[0].LastUpdate != "2026-07-21T14:29:07.69Z" {
		t.Errorf("gw[0].LastUpdate = %q, want 2026-07-21T14:29:07.69Z", gws[0].LastUpdate)
	}
	if len(gws[0].Tags) != 2 || gws[0].Tags[0] != "team-a" {
		t.Errorf("gw[0].Tags = %v, want [team-a prod]", gws[0].Tags)
	}
	if len(gws[0].Replicas) != 2 {
		t.Fatalf("gw[0].Replicas len = %d, want 2", len(gws[0].Replicas))
	}
	if gws[0].Replicas[0].Status != "CONNECTED" || gws[0].Replicas[0].Count != 2 {
		t.Errorf("gw[0].Replicas[0] = %+v, want {CONNECTED 2 ...}", gws[0].Replicas[0])
	}
	if len(gws[0].Replicas[0].CertificateExpirationDates) != 1 {
		t.Errorf("gw[0].Replicas[0].CertificateExpirationDates = %v, want 1 entry", gws[0].Replicas[0].CertificateExpirationDates)
	}
	// The list surfaces the DELETED tombstone verbatim; filtering is the caller's job
	// (resource resolve / data source), verified in those packages' tests.
	if gws[1].Status != "DELETED" {
		t.Errorf("gw[1].Status = %q, want DELETED (tombstone surfaced by list)", gws[1].Status)
	}
}

// TestSelfManagedGatewayClient_ListSelfManagedGateways_Paginates is the regression guard for
// the single-page truncation bug (same class as the ListTeamRoles / ListRoleAssignments
// pagination fixes). The /standalone list endpoint defaults to pageSize=30 and caps pageSize
// at 100, so an org/env with more than one page of self-managed gateways would be silently
// truncated by a naive single-request implementation. That truncation is severe: the
// resource's resolveGateway scans this list by name, so a gateway on page 2+ would be treated
// as "not registered" — its gateway_id would be wiped (phantom drift) and it could never be
// deleted via Terraform (Delete gates on gateway_id != ""). This test serves TWO full pages
// (100 items each) then a short final page, and asserts the client walks all three and
// concatenates every gateway. It also asserts the client requests the max pageSize (100).
func TestSelfManagedGatewayClient_ListSelfManagedGateways_Paginates(t *testing.T) {
	const pageSize = 100
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways"

	// Three pages: page 0 -> 100 items, page 1 -> 100 items, page 2 -> 5 items (short => last).
	pages := map[string]int{"0": pageSize, "1": pageSize, "2": 5}
	total := pageSize + pageSize + 5

	var sawPageSizes []string
	var sawPageNumbers []string

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			gotPageSize := q.Get("pageSize")
			pageNumber := q.Get("pageNumber")
			sawPageSizes = append(sawPageSizes, gotPageSize)
			sawPageNumbers = append(sawPageNumbers, pageNumber)

			pageNumberInt, _ := strconv.Atoi(pageNumber)

			n, ok := pages[pageNumber]
			if !ok {
				// Beyond the last page: the real API returns 200 + empty content (with
				// totalElements reset to 0). Model that so a buggy loop that ignored the
				// short-page break would still terminate rather than hang the test.
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"content": []map[string]interface{}{}, "totalElements": 0,
					"pageNumber": pageNumberInt, "pageSize": pageSize,
				})
				return
			}

			content := make([]map[string]interface{}, 0, n)
			for i := 0; i < n; i++ {
				content = append(content, map[string]interface{}{
					"id":     fmt.Sprintf("gw-%s-%d", pageNumber, i),
					"name":   fmt.Sprintf("flex-%s-%d", pageNumber, i),
					"status": "CONNECTED",
				})
			}
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": content, "totalElements": total,
				"pageNumber": pageNumberInt, "pageSize": pageSize,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	gws, err := c.ListSelfManagedGateways(context.Background(), "test-org-id", "test-env-id")
	if err != nil {
		t.Fatalf("ListSelfManagedGateways() error: %v", err)
	}

	// All three pages must be concatenated — proves we did not truncate at the first page.
	if len(gws) != total {
		t.Fatalf("len = %d, want %d (all pages concatenated)", len(gws), total)
	}
	// Spot-check that items from the LAST page made it in (the ones a truncating impl drops).
	if gws[total-1].ID != "gw-2-4" {
		t.Errorf("last gateway ID = %q, want gw-2-4 (item from final page)", gws[total-1].ID)
	}
	// The client must request exactly 3 pages: 0, 1, 2 (stopping on the short third page).
	if len(sawPageNumbers) != 3 {
		t.Fatalf("requested %d pages (%v), want 3", len(sawPageNumbers), sawPageNumbers)
	}
	for i, want := range []string{"0", "1", "2"} {
		if sawPageNumbers[i] != want {
			t.Errorf("request %d pageNumber = %q, want %q", i, sawPageNumbers[i], want)
		}
	}
	// The client must request the max supported page size (100) to minimize round-trips.
	for i, ps := range sawPageSizes {
		if ps != "100" {
			t.Errorf("request %d pageSize = %q, want 100", i, ps)
		}
	}
}

func TestSelfManagedGatewayClient_ListSelfManagedGateways_Error(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusForbidden, "nope")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	_, err := c.ListSelfManagedGateways(context.Background(), "test-org-id", "test-env-id")
	if err == nil {
		t.Fatal("expected error on 403, got nil")
	}
}

// --- GetSelfManagedGateway ---

func TestSelfManagedGatewayClient_GetSelfManagedGateway(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			// GET-by-id LIVE shape adds "versions":[] on top of the list item shape.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "gw-1",
				"name":           "flex-a",
				"status":         "CONNECTED",
				"organizationId": "test-org-id",
				"lastUpdate":     "2026-07-21T14:29:07.69Z",
				"tags":           []string{},
				"replicas": []map[string]interface{}{
					{"status": "CONNECTED", "count": 1, "certificateExpirationDates": []string{}},
				},
				"versions": []string{"1.9.0"},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	gw, err := c.GetSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1")
	if err != nil {
		t.Fatalf("GetSelfManagedGateway() error: %v", err)
	}
	if gw.ID != "gw-1" || gw.Name != "flex-a" || gw.Status != "CONNECTED" {
		t.Errorf("gw = %+v, unexpected", gw)
	}
	if gw.LastUpdate != "2026-07-21T14:29:07.69Z" {
		t.Errorf("gw.LastUpdate = %q, want 2026-07-21T14:29:07.69Z", gw.LastUpdate)
	}
	if len(gw.Replicas) != 1 || gw.Replicas[0].Count != 1 {
		t.Errorf("gw.Replicas = %+v, want one replica count=1", gw.Replicas)
	}
	if len(gw.Versions) != 1 || gw.Versions[0] != "1.9.0" {
		t.Errorf("gw.Versions = %v, want [1.9.0]", gw.Versions)
	}
}

func TestSelfManagedGatewayClient_GetSelfManagedGateway_NotFound(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/missing"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Gateway not found by id")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	_, err := c.GetSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "missing")
	if err == nil {
		t.Fatal("expected error for missing gateway")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

// --- DeleteSelfManagedGateway ---

// The real delete is an async soft-delete that returns HTTP 202 Accepted. (200/204 are also
// accepted for forward-compatibility, covered by the table test below.)
func TestSelfManagedGatewayClient_DeleteSelfManagedGateway(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
	var gotMethod string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.WriteHeader(http.StatusAccepted)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	if err := c.DeleteSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1"); err != nil {
		t.Fatalf("DeleteSelfManagedGateway() error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// All of 202/200/204 must be treated as delete success.
func TestSelfManagedGatewayClient_DeleteSelfManagedGateway_SuccessStatuses(t *testing.T) {
	for _, code := range []int{http.StatusAccepted, http.StatusOK, http.StatusNoContent} {
		path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			path: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c := newTestSelfManagedClient(server.URL)

		if err := c.DeleteSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1"); err != nil {
			t.Errorf("DeleteSelfManagedGateway() on %d should be nil, got %v", code, err)
		}
	}
}

// A 404 on delete is idempotent-success: the object is already gone.
func TestSelfManagedGatewayClient_DeleteSelfManagedGateway_NotFoundIsSuccess(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Gateway not found by id")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	if err := c.DeleteSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1"); err != nil {
		t.Errorf("DeleteSelfManagedGateway() on 404 should be nil (idempotent), got %v", err)
	}
}

// Deleting an already-soft-deleted (tombstone) gateway returns the LIVE-VERIFIED
// 400 "This target was already deleted". Because the desired end-state already holds,
// this must be idempotent-success so `terraform destroy` (or a retried destroy) does not
// fail. Regression guard for Bug C.
func TestSelfManagedGatewayClient_DeleteSelfManagedGateway_AlreadyDeletedIsSuccess(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			// Exact live body observed on a second delete of the tombstone.
			testutil.ErrorResponse(w, http.StatusBadRequest, "This target was already deleted")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	if err := c.DeleteSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1"); err != nil {
		t.Errorf("DeleteSelfManagedGateway() on 400 'already deleted' should be nil (idempotent), got %v", err)
	}
}

// A 400 that is NOT the "already deleted" tombstone message is a genuine error and must be
// surfaced — the idempotency shortcut is narrow, message-specific, and must not swallow
// other bad-request failures.
func TestSelfManagedGatewayClient_DeleteSelfManagedGateway_OtherBadRequestIsError(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid gateway id format")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	if err := c.DeleteSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1"); err == nil {
		t.Fatal("expected error on non-tombstone 400, got nil")
	}
}

func TestSelfManagedGatewayClient_DeleteSelfManagedGateway_Error(t *testing.T) {
	path := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	c := newTestSelfManagedClient(server.URL)

	if err := c.DeleteSelfManagedGateway(context.Background(), "test-org-id", "test-env-id", "gw-1"); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}
