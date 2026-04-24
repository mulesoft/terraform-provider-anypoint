package accessmanagement

import (
	"context"
	"encoding/json"
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
		config      *client.ClientConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &client.ClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.ClientConfig{
				ClientSecret: "test-client-secret",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.ClientConfig{
				ClientID: "test-client-id",
			},
			wantErr:     true,
			errContains: "client_secret is required",
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

func TestTeamRolesClient_AssignRolesToTeam(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		teamID      string
		roles       []TeamRoleAssignment
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful assignment",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			roles: []TeamRoleAssignment{
				{RoleID: "role-1", ContextParams: map[string]interface{}{}},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			roles: []TeamRoleAssignment{
				{RoleID: "role-1", ContextParams: map[string]interface{}{}},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to assign roles to team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/roles", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamRolesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.AssignRolesToTeam(context.Background(), tt.orgID, tt.teamID, tt.roles)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AssignRolesToTeam() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AssignRolesToTeam() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AssignRolesToTeam() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTeamRolesClient_GetTeamRoles(t *testing.T) {
	mockRoles := []TeamRoleAssignment{
		{RoleID: "role-1", ContextParams: map[string]interface{}{}},
	}

	tests := []struct {
		name        string
		orgID       string
		teamID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful get",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockRoles)
			},
			wantErr: false,
		},
		{
			name:   "not found returns empty",
			orgID:  "test-org-id",
			teamID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get roles for team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/roles", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamRolesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetTeamRoles(context.Background(), tt.orgID, tt.teamID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTeamRoles() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetTeamRoles() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetTeamRoles() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetTeamRoles() returned nil")
				}
			}
		})
	}
}

func TestTeamRoleAssignment_JSONSerialization(t *testing.T) {
	item := &TeamRoleAssignment{
		RoleID:        "test-role-id",
		ContextParams: map[string]interface{}{"org_id": "test-org"},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal TeamRoleAssignment: %v", err)
	}

	var decoded TeamRoleAssignment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TeamRoleAssignment: %v", err)
	}

	if decoded.RoleID != item.RoleID {
		t.Errorf("Unmarshaled RoleID = %v, want %v", decoded.RoleID, item.RoleID)
	}
}
