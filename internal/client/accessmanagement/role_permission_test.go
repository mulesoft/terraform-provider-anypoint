package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// paginatedRoleAssignmentsHandler returns an http handler that serves `total`
// synthetic role assignments across pages, honoring the limit/offset query
// params. Like the real accounts API, it defaults to a SMALL page size (25)
// when the client omits limit — so a non-paginating client is truncated to 25
// and the test fails. `hits` counts requests so tests can assert that
// pagination actually issued multiple round-trips.
func paginatedRoleAssignmentsHandler(total int, hits *int) func(w http.ResponseWriter, r *http.Request) {
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

		page := make([]RoleAssignment, 0, limit)
		for i := offset; i < offset+limit && i < total; i++ {
			page = append(page, RoleAssignment{
				RoleID:        fmt.Sprintf("role-%d", i),
				Name:          fmt.Sprintf("Role %d", i),
				ContextParams: map[string]string{"org": "test-org"},
			})
		}
		testutil.JSONResponse(w, http.StatusOK, ListRoleAssignmentsResponse{Data: page, Total: total})
	}
}

// TestRolePermissionClient_ListRoleAssignments_Paginates proves that a role
// group with more than one page of assignments is fully returned. total (250)
// is deliberately larger than the client's page size (limit=100) so the offset
// loop MUST fire several times. Regression test for the truncation bug: before
// ListRoleAssignments paginated it issued a single GET with no limit, so the
// server defaulted to a 25-item page and silently dropped the rest — corrupting
// the authoritative reconcile (perpetual diff for assignments past the first
// page, and inability to remove them since they were invisible). The handler
// below reproduces that default, so any regression that stops passing limit
// (→ 25 returned) or stops looping (→ 100 returned) fails the count assertion.
func TestRolePermissionClient_ListRoleAssignments_Paginates(t *testing.T) {
	const total = 250
	hits := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org/rolegroups/test-rg/roles": paginatedRoleAssignmentsHandler(total, &hits),
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	assignments, err := c.ListRoleAssignments(context.Background(), "test-org", "test-rg")
	if err != nil {
		t.Fatalf("ListRoleAssignments() unexpected error = %v", err)
	}
	if len(assignments) != total {
		t.Fatalf("ListRoleAssignments() returned %d assignments, want %d (pagination truncated the result)", len(assignments), total)
	}
	if hits < 2 {
		t.Errorf("expected pagination to issue >=2 requests, got %d", hits)
	}
	// Verify no duplicates and the last-page item is present.
	seen := map[string]bool{}
	for _, a := range assignments {
		if seen[a.RoleID] {
			t.Errorf("duplicate role_id across pages: %s", a.RoleID)
		}
		seen[a.RoleID] = true
	}
	if !seen[fmt.Sprintf("role-%d", total-1)] {
		t.Errorf("last-page assignment role-%d missing from result", total-1)
	}
}

func TestNewRolePermissionClient(t *testing.T) {
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

	config := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Username:     "test-user",
		Password:     "test-pass",
	}

	c, err := NewRolePermissionClient(config)
	if err != nil {
		t.Fatalf("NewRolePermissionClient() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewRolePermissionClient() returned nil")
	}
	if c.Token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestNewRolePermissionClient_MissingClientID(t *testing.T) {
	config := &client.Config{
		BaseURL:      "http://localhost",
		ClientID:     "",
		ClientSecret: "test-secret",
		Username:     "test-user",
		Password:     "test-pass",
	}

	_, err := NewRolePermissionClient(config)
	if err == nil {
		t.Fatal("NewRolePermissionClient() should error with missing client_id")
	}
}

func TestRolePermissionClient_AssignRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/api/organizations/test-org/rolegroups/test-rg/roles" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Verify body is an array
		body, _ := io.ReadAll(r.Body)
		var req []AssignRoleRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}
		if len(req) != 1 {
			t.Fatalf("Expected 1 item in request array, got %d", len(req))
		}
		if req[0].RoleID != "role-123" {
			t.Errorf("Expected role_id 'role-123', got %s", req[0].RoleID)
		}
		if req[0].ContextParams["org"] != "test-org" {
			t.Errorf("Expected context_params.org 'test-org', got %s", req[0].ContextParams["org"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]AssignRoleResponse{{
			RoleGroupID:           "test-rg",
			RoleID:                "role-123",
			RoleGroupAssignmentID: "assign-456",
			ContextParams:         map[string]string{"org": "test-org"},
		}})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	resp, err := c.AssignRole(context.Background(), "test-org", "test-rg", &AssignRoleRequest{
		RoleID:        "role-123",
		ContextParams: map[string]string{"org": "test-org"},
	})
	if err != nil {
		t.Fatalf("AssignRole() error = %v", err)
	}
	if resp.RoleGroupAssignmentID != "assign-456" {
		t.Errorf("Expected assignment ID 'assign-456', got %s", resp.RoleGroupAssignmentID)
	}
}

func TestRolePermissionClient_AssignRole_NullResponse(t *testing.T) {
	// Some roles return null body on HTTP 200 success.
	// The client should fall back to a read-back.
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "POST" {
			// Return null like the real API does for certain roles
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("null"))
			return
		}
		if r.Method == "GET" {
			// Read-back: return the assignment in the list
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ListRoleAssignmentsResponse{
				Data: []RoleAssignment{
					{
						RoleGroupAssignmentID: "assign-readback",
						RoleGroupID:           "test-rg",
						RoleID:                "role-null",
						OrgID:                 "test-org",
						Name:                  "Create Applications",
						ContextParams:         map[string]string{"org": "test-org", "envId": "env-1"},
						CreatedAt:             "2024-01-01T00:00:00Z",
					},
				},
				Total: 1,
			})
			return
		}
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	resp, err := c.AssignRole(context.Background(), "test-org", "test-rg", &AssignRoleRequest{
		RoleID:        "role-null",
		ContextParams: map[string]string{"org": "test-org", "envId": "env-1"},
	})
	if err != nil {
		t.Fatalf("AssignRole() with null response should not error, got: %v", err)
	}
	if resp.RoleGroupAssignmentID != "assign-readback" {
		t.Errorf("Expected assignment ID from read-back 'assign-readback', got %s", resp.RoleGroupAssignmentID)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 HTTP calls (POST + GET read-back), got %d", callCount)
	}
}

func TestRolePermissionClient_AssignRole_WithEnvID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req []AssignRoleRequest
		_ = json.Unmarshal(body, &req)

		if req[0].ContextParams["envId"] != "env-789" {
			t.Errorf("Expected context_params.envId 'env-789', got %s", req[0].ContextParams["envId"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]AssignRoleResponse{{
			RoleGroupID:           "test-rg",
			RoleID:                "role-123",
			RoleGroupAssignmentID: "assign-789",
			ContextParams:         map[string]string{"org": "test-org", "envId": "env-789"},
		}})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	resp, err := c.AssignRole(context.Background(), "test-org", "test-rg", &AssignRoleRequest{
		RoleID:        "role-123",
		ContextParams: map[string]string{"org": "test-org", "envId": "env-789"},
	})
	if err != nil {
		t.Fatalf("AssignRole() error = %v", err)
	}
	if resp.ContextParams["envId"] != "env-789" {
		t.Errorf("Expected envId in response context_params")
	}
}

func TestRolePermissionClient_AssignRole_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	_, err := c.AssignRole(context.Background(), "test-org", "test-rg", &AssignRoleRequest{
		RoleID:        "role-123",
		ContextParams: map[string]string{"org": "test-org"},
	})
	if err == nil {
		t.Fatal("AssignRole() should error on server error")
	}
}

func TestRolePermissionClient_UnassignRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var req []AssignRoleRequest
		_ = json.Unmarshal(body, &req)

		if req[0].RoleID != "role-123" {
			t.Errorf("Expected role_id 'role-123', got %s", req[0].RoleID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]int{1})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	err := c.UnassignRole(context.Background(), "test-org", "test-rg", &AssignRoleRequest{
		RoleID:        "role-123",
		ContextParams: map[string]string{"org": "test-org"},
	})
	if err != nil {
		t.Fatalf("UnassignRole() error = %v", err)
	}
}

func TestRolePermissionClient_UnassignRole_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	err := c.UnassignRole(context.Background(), "test-org", "test-rg", &AssignRoleRequest{
		RoleID:        "role-123",
		ContextParams: map[string]string{"org": "test-org"},
	})
	if err == nil {
		t.Fatal("UnassignRole() should error on 404")
	}
	if !client.IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got: %v", err)
	}
}

func TestRolePermissionClient_ListRoleAssignments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRoleAssignmentsResponse{
			Data: []RoleAssignment{
				{
					RoleGroupAssignmentID: "assign-1",
					RoleGroupID:           "test-rg",
					RoleID:                "role-aaa",
					OrgID:                 "test-org",
					Name:                  "Audit Log Viewer",
					Description:           "View audit logs",
					ContextParams:         map[string]string{"org": "test-org"},
					CreatedAt:             "2024-01-01T00:00:00Z",
				},
				{
					RoleGroupAssignmentID: "assign-2",
					RoleGroupID:           "test-rg",
					RoleID:                "role-bbb",
					OrgID:                 "test-org",
					Name:                  "Read Applications",
					Description:           "Read apps",
					ContextParams:         map[string]string{"org": "test-org", "envId": "env-1"},
					CreatedAt:             "2024-01-02T00:00:00Z",
				},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	assignments, err := c.ListRoleAssignments(context.Background(), "test-org", "test-rg")
	if err != nil {
		t.Fatalf("ListRoleAssignments() error = %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("Expected 2 assignments, got %d", len(assignments))
	}
	if assignments[0].Name != "Audit Log Viewer" {
		t.Errorf("Expected first assignment name 'Audit Log Viewer', got %s", assignments[0].Name)
	}
	if assignments[1].ContextParams["envId"] != "env-1" {
		t.Errorf("Expected second assignment envId 'env-1'")
	}
}

func TestRolePermissionClient_ListRoleAssignments_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Role group not found"))
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	_, err := c.ListRoleAssignments(context.Background(), "test-org", "nonexistent-rg")
	if err == nil {
		t.Fatal("ListRoleAssignments() should error on 404")
	}
	if !client.IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got: %v", err)
	}
}

func TestRolePermissionClient_GetRoleAssignment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRoleAssignmentsResponse{
			Data: []RoleAssignment{
				{
					RoleGroupAssignmentID: "assign-1",
					RoleGroupID:           "test-rg",
					RoleID:                "role-aaa",
					OrgID:                 "test-org",
					Name:                  "Audit Log Viewer",
					ContextParams:         map[string]string{"org": "test-org"},
					CreatedAt:             "2024-01-01T00:00:00Z",
				},
				{
					RoleGroupAssignmentID: "assign-2",
					RoleGroupID:           "test-rg",
					RoleID:                "role-bbb",
					OrgID:                 "test-org",
					Name:                  "Read Applications",
					ContextParams:         map[string]string{"org": "test-org", "envId": "env-1"},
					CreatedAt:             "2024-01-02T00:00:00Z",
				},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	// Find by role_id and context_params
	assignment, err := c.GetRoleAssignment(context.Background(), "test-org", "test-rg", "role-bbb", map[string]string{"org": "test-org", "envId": "env-1"})
	if err != nil {
		t.Fatalf("GetRoleAssignment() error = %v", err)
	}
	if assignment.RoleGroupAssignmentID != "assign-2" {
		t.Errorf("Expected assignment ID 'assign-2', got %s", assignment.RoleGroupAssignmentID)
	}
	if assignment.Name != "Read Applications" {
		t.Errorf("Expected name 'Read Applications', got %s", assignment.Name)
	}
}

func TestRolePermissionClient_GetRoleAssignment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRoleAssignmentsResponse{
			Data:  []RoleAssignment{},
			Total: 0,
		})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	_, err := c.GetRoleAssignment(context.Background(), "test-org", "test-rg", "nonexistent", map[string]string{"org": "test-org"})
	if err == nil {
		t.Fatal("GetRoleAssignment() should error when not found")
	}
	if !client.IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got: %v", err)
	}
}

func TestRolePermissionClient_ListAvailableRoles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/api/roles" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListAvailableRolesResponse{
			Data: []AvailableRole{
				{RoleID: "role-aaa", Name: "Audit Log Viewer", Description: "View audit logs", Internal: false},
				{RoleID: "role-bbb", Name: "Read Applications", Description: "Read applications", Internal: false},
				{RoleID: "role-ccc", Name: "Organization Admin", Description: "Admin access", Internal: true},
			},
			Total: 3,
		})
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	roles, err := c.ListAvailableRoles(context.Background())
	if err != nil {
		t.Fatalf("ListAvailableRoles() error = %v", err)
	}
	if len(roles) != 3 {
		t.Fatalf("Expected 3 roles, got %d", len(roles))
	}
	if roles[0].Name != "Audit Log Viewer" {
		t.Errorf("Expected first role name 'Audit Log Viewer', got %s", roles[0].Name)
	}
	if roles[2].Internal != true {
		t.Errorf("Expected third role to be internal")
	}
}

func TestRolePermissionClient_ListAvailableRoles_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := &RolePermissionClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	_, err := c.ListAvailableRoles(context.Background())
	if err == nil {
		t.Fatal("ListAvailableRoles() should error on server error")
	}
}

func TestContextParamsMatch(t *testing.T) {
	tests := []struct {
		name     string
		a        map[string]string
		b        map[string]string
		expected bool
	}{
		{
			name:     "identical org-only",
			a:        map[string]string{"org": "abc"},
			b:        map[string]string{"org": "abc"},
			expected: true,
		},
		{
			name:     "identical org+env",
			a:        map[string]string{"org": "abc", "envId": "def"},
			b:        map[string]string{"org": "abc", "envId": "def"},
			expected: true,
		},
		{
			name:     "different org",
			a:        map[string]string{"org": "abc"},
			b:        map[string]string{"org": "xyz"},
			expected: false,
		},
		{
			name:     "different length",
			a:        map[string]string{"org": "abc"},
			b:        map[string]string{"org": "abc", "envId": "def"},
			expected: false,
		},
		{
			name:     "both empty",
			a:        map[string]string{},
			b:        map[string]string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contextParamsMatch(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("contextParamsMatch() = %v, want %v", got, tt.expected)
			}
		})
	}
}
