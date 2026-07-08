package accessmanagement

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// paginatedScopesHandler serves `total` synthetic scopes across pages, honoring the limit/offset
// query params. It mimics the real accounts API behavior of defaulting to a SMALL page size (25)
// when the client omits limit — so a non-paginating client is truncated to 25 and the test fails.
// `hits` is incremented once per request so tests can assert pagination issued multiple round-trips.
func paginatedScopesHandler(total int, hits *int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		*hits++
		q := r.URL.Query()
		limit := 25 // server default when the client omits limit
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		offset := 0
		if v := q.Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				offset = n
			}
		}

		page := make([]Scope, 0, limit)
		for i := offset; i < offset+limit && i < total; i++ {
			page = append(page, Scope{
				Scope:         fmt.Sprintf("scope:%d", i),
				ContextParams: map[string]interface{}{"org": "test-org"},
			})
		}
		testutil.JSONResponse(w, http.StatusOK, ConnectedAppScopes{Scopes: page, Total: total})
	}
}

// TestConnectedAppScopesClient_GetConnectedAppScopes_Paginates proves that an app with more than
// one page of scopes is fully returned. total (250) is deliberately larger than the client page
// size (limit=100) so the offset loop MUST fire several times. Regression test for the truncation
// bug: before GetConnectedAppScopes paginated it issued a single GET with no limit, so the server
// defaulted to a 25-item page and silently dropped the rest — corrupting the authoritative
// reconcile. The handler reproduces that default, so a regression that stops passing limit (→ 25
// returned) or stops looping (→ 100 returned) fails the count assertion.
func TestConnectedAppScopesClient_GetConnectedAppScopes_Paginates(t *testing.T) {
	const total = 250
	hits := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/connectedApplications/test-app-id/scopes": paginatedScopesHandler(total, &hits),
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &ConnectedAppScopesClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	got, err := c.GetConnectedAppScopes(context.Background(), "test-app-id")
	if err != nil {
		t.Fatalf("GetConnectedAppScopes() unexpected error = %v", err)
	}
	if len(got.Scopes) != total {
		t.Fatalf("GetConnectedAppScopes() returned %d scopes, want %d (pagination truncated the result)", len(got.Scopes), total)
	}
	if hits < 2 {
		t.Errorf("expected pagination to issue >=2 requests, got %d", hits)
	}
	// Verify no duplicates and the last-page item is present.
	seen := map[string]bool{}
	for _, s := range got.Scopes {
		if seen[s.Scope] {
			t.Errorf("duplicate scope across pages: %s", s.Scope)
		}
		seen[s.Scope] = true
	}
	if !seen[fmt.Sprintf("scope:%d", total-1)] {
		t.Errorf("last-page scope scope:%d missing from result", total-1)
	}
}

// TestConnectedAppScopesClient_ReplaceConnectedAppScopes verifies the PUT-based authoritative
// replace: the client PUTs the whole list, the server returns 204, and the client re-GETs the
// resulting scopes.
func TestConnectedAppScopesClient_ReplaceConnectedAppScopes(t *testing.T) {
	replaced := []Scope{
		{Scope: "read:applications", ContextParams: map[string]interface{}{"org": "test-org"}},
		{Scope: "create:generations", ContextParams: map[string]interface{}{"org": "test-org"}},
	}

	tests := []struct {
		name        string
		putStatus   int
		wantErr     bool
		errContains string
	}{
		{name: "successful replace", putStatus: http.StatusNoContent, wantErr: false},
		{name: "server rejects put", putStatus: http.StatusBadRequest, wantErr: true, errContains: "API request failed with status 400"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawPut bool
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/connectedApplications/test-app-id/scopes": func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodPut:
						sawPut = true
						if tt.putStatus != http.StatusNoContent {
							testutil.ErrorResponse(w, tt.putStatus, "rejected")
							return
						}
						w.WriteHeader(http.StatusNoContent)
					case http.MethodGet:
						testutil.JSONResponse(w, http.StatusOK, ConnectedAppScopes{Scopes: replaced, Total: len(replaced)})
					default:
						t.Errorf("unexpected method %s", r.Method)
					}
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &ConnectedAppScopesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			got, err := c.ReplaceConnectedAppScopes(context.Background(), "test-app-id", replaced)
			if !sawPut {
				t.Errorf("expected a PUT request to be issued")
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ReplaceConnectedAppScopes() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReplaceConnectedAppScopes() unexpected error = %v", err)
			}
			if got == nil || len(got.Scopes) != len(replaced) {
				t.Fatalf("ReplaceConnectedAppScopes() returned %v, want %d scopes", got, len(replaced))
			}
		})
	}
}
