package accessmanagement

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// paginatedRolesHandler returns an http handler that serves `total` synthetic
// assignments across pages, honoring the limit/offset query params. It mimics
// the real accounts API behavior of defaulting to a SMALL page size (25) when
// the client omits limit — so a non-paginating client is truncated to 25 and
// the test fails. `hits` is incremented once per request so tests can assert
// that pagination actually issued multiple round-trips.
func paginatedTeamRolesHandler(total int, hits *int) func(w http.ResponseWriter, r *http.Request) {
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

		page := make([]TeamRoleAssignment, 0, limit)
		for i := offset; i < offset+limit && i < total; i++ {
			page = append(page, TeamRoleAssignment{
				RoleID:        fmt.Sprintf("role-%d", i),
				Name:          fmt.Sprintf("Role %d", i),
				ContextParams: map[string]string{"org": "test-org-id"},
			})
		}
		testutil.JSONResponse(w, http.StatusOK, ListTeamRolesResponse{Data: page, Total: total})
	}
}

// TestTeamRolesClient_ListTeamRoles_Paginates proves that a team with more than
// one page of role assignments is fully returned. total (250) is deliberately
// larger than the client's page size (limit=100) so the offset loop MUST fire
// several times. Regression test for the truncation bug: before ListTeamRoles
// paginated it issued a single GET with no limit, so the server defaulted to a
// 25-item page and silently dropped the rest — corrupting the authoritative
// reconcile. The handler below reproduces that default, so any regression that
// stops passing limit (→ 25 returned) or stops looping (→ 100 returned) fails
// the count assertion.
func TestTeamRolesClient_ListTeamRoles_Paginates(t *testing.T) {
	const total = 250
	hits := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/teams/test-team-id/roles": paginatedTeamRolesHandler(total, &hits),
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &TeamRolesClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	roles, err := c.ListTeamRoles(context.Background(), "test-org-id", "test-team-id")
	if err != nil {
		t.Fatalf("ListTeamRoles() unexpected error = %v", err)
	}
	if len(roles) != total {
		t.Fatalf("ListTeamRoles() returned %d roles, want %d (pagination truncated the result)", len(roles), total)
	}
	if hits < 2 {
		t.Errorf("expected pagination to issue >=2 requests, got %d", hits)
	}
	// Verify no duplicates and the last-page item is present.
	seen := map[string]bool{}
	for _, a := range roles {
		if seen[a.RoleID] {
			t.Errorf("duplicate role_id across pages: %s", a.RoleID)
		}
		seen[a.RoleID] = true
	}
	if !seen[fmt.Sprintf("role-%d", total-1)] {
		t.Errorf("last-page assignment role-%d missing from result", total-1)
	}
}

func TestNewTeamRolesClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *client.UserClientConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &client.UserClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.UserClientConfig{
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

			if tt.config != nil {
				tt.config.BaseURL = server.URL
			}

			client, err := NewTeamRolesClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTeamRolesClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewTeamRolesClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewTeamRolesClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewTeamRolesClient() returned nil client")
				}
			}
		})
	}
}

func TestTeamRolesClient_AssignTeamRole(t *testing.T) {
	mockResponse := &AssignTeamRoleResponse{
		RoleGroupID:           "test-role-group-id",
		RoleID:                "test-role-id",
		RoleGroupAssignmentID: "test-assignment-id",
		ContextParams:         map[string]string{"org": "test-org-id"},
	}

	tests := []struct {
		name        string
		orgID       string
		teamID      string
		request     *AssignTeamRoleRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful assign",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &AssignTeamRoleRequest{
				RoleID:        "test-role-id",
				ContextParams: map[string]string{"org": "test-org-id"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/teams/test-team-id/roles")
				testutil.JSONResponse(w, http.StatusCreated, []AssignTeamRoleResponse{*mockResponse})
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &AssignTeamRoleRequest{
				RoleID:        "test-role-id",
				ContextParams: map[string]string{"org": "test-org-id"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to assign role to team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/roles", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamRolesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			resp, err := client.AssignTeamRole(context.Background(), tt.orgID, tt.teamID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AssignTeamRole() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AssignTeamRole() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AssignTeamRole() unexpected error = %v", err)
				}
				if resp == nil {
					t.Errorf("AssignTeamRole() returned nil response")
				}
				if resp != nil && resp.RoleID != mockResponse.RoleID {
					t.Errorf("AssignTeamRole() RoleID = %v, want %v", resp.RoleID, mockResponse.RoleID)
				}
			}
		})
	}
}

func TestTeamRolesClient_UnassignTeamRole(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		teamID      string
		request     *AssignTeamRoleRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful unassign",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &AssignTeamRoleRequest{
				RoleID:        "test-role-id",
				ContextParams: map[string]string{"org": "test-org-id"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/teams/test-team-id/roles")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:   "already unassigned - not found",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &AssignTeamRoleRequest{
				RoleID:        "nonexistent-role-id",
				ContextParams: map[string]string{"org": "test-org-id"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Role not found")
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &AssignTeamRoleRequest{
				RoleID:        "test-role-id",
				ContextParams: map[string]string{"org": "test-org-id"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to unassign role from team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/roles", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamRolesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.UnassignTeamRole(context.Background(), tt.orgID, tt.teamID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UnassignTeamRole() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UnassignTeamRole() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UnassignTeamRole() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTeamRolesClient_ListTeamRoles(t *testing.T) {
	mockRoles := []TeamRoleAssignment{
		{
			RoleGroupAssignmentID: "assignment-1",
			RoleGroupID:           "group-1",
			RoleID:                "role-1",
			Name:                  "Organization Administrator",
			Description:           "Full access to the organization",
			Internal:              false,
			OrgID:                 "test-org-id",
			ContextParams:         map[string]string{"org": "test-org-id"},
			CreatedAt:             "2023-01-01T00:00:00Z",
		},
		{
			RoleGroupAssignmentID: "assignment-2",
			RoleGroupID:           "group-2",
			RoleID:                "role-2",
			Name:                  "CloudHub Admin",
			Description:           "CloudHub environment administration",
			Internal:              false,
			OrgID:                 "test-org-id",
			ContextParams:         map[string]string{"org": "test-org-id", "envId": "test-env-id"},
			CreatedAt:             "2023-01-02T00:00:00Z",
		},
	}

	tests := []struct {
		name          string
		orgID         string
		teamID        string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
		expectedCount int
	}{
		{
			name:   "successful list",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/teams/test-team-id/roles")
				testutil.JSONResponse(w, http.StatusOK, ListTeamRolesResponse{
					Data:  mockRoles,
					Total: 2,
				})
			},
			wantErr:       false,
			expectedCount: 2,
		},
		{
			name:   "team not found",
			orgID:  "test-org-id",
			teamID: "nonexistent-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "team",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to list team roles with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/roles", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamRolesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			roles, err := client.ListTeamRoles(context.Background(), tt.orgID, tt.teamID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListTeamRoles() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ListTeamRoles() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListTeamRoles() unexpected error = %v", err)
				}
				if len(roles) != tt.expectedCount {
					t.Errorf("ListTeamRoles() returned %d roles, want %d", len(roles), tt.expectedCount)
				}
			}
		})
	}
}
