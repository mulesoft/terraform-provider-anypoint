package accessmanagement

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

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
