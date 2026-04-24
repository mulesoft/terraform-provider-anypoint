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

func TestNewRoleGroupRolesClient(t *testing.T) {
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

			c, err := NewRoleGroupRolesClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRoleGroupRolesClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewRoleGroupRolesClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewRoleGroupRolesClient() unexpected error = %v", err)
				}
				if c == nil {
					t.Errorf("NewRoleGroupRolesClient() returned nil client")
				}
			}
		})
	}
}

func TestRoleGroupRolesClient_AssignRolesToRoleGroup(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		roleGroupID string
		roles       []RoleAssignment
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful assignment",
			orgID:       "test-org-id",
			roleGroupID: "test-rolegroup-id",
			roles: []RoleAssignment{
				{RoleID: "role-1", ContextParams: map[string]interface{}{}},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:        "server error",
			orgID:       "test-org-id",
			roleGroupID: "test-rolegroup-id",
			roles: []RoleAssignment{
				{RoleID: "role-1", ContextParams: map[string]interface{}{}},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to assign roles to role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s/roles", tt.orgID, tt.roleGroupID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &RoleGroupRolesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.AssignRolesToRoleGroup(context.Background(), tt.orgID, tt.roleGroupID, tt.roles)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AssignRolesToRoleGroup() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AssignRolesToRoleGroup() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AssignRolesToRoleGroup() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRoleGroupRolesClient_GetRoleGroupRoles(t *testing.T) {
	mockRoles := []RoleAssignment{
		{RoleID: "role-1", ContextParams: map[string]interface{}{}},
	}

	tests := []struct {
		name        string
		orgID       string
		roleGroupID string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful get",
			orgID:       "test-org-id",
			roleGroupID: "test-rolegroup-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockRoles)
			},
			wantErr: false,
		},
		{
			name:        "not found returns empty",
			orgID:       "test-org-id",
			roleGroupID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: false,
		},
		{
			name:        "server error",
			orgID:       "test-org-id",
			roleGroupID: "test-rolegroup-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get roles for role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s/roles", tt.orgID, tt.roleGroupID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &RoleGroupRolesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := c.GetRoleGroupRoles(context.Background(), tt.orgID, tt.roleGroupID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetRoleGroupRoles() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetRoleGroupRoles() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetRoleGroupRoles() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetRoleGroupRoles() returned nil")
				}
			}
		})
	}
}

func TestRoleAssignment_JSONSerialization(t *testing.T) {
	item := &RoleAssignment{
		RoleID:        "test-role-id",
		ContextParams: map[string]interface{}{"org_id": "test-org"},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal RoleAssignment: %v", err)
	}

	var decoded RoleAssignment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RoleAssignment: %v", err)
	}

	if decoded.RoleID != item.RoleID {
		t.Errorf("Unmarshaled RoleID = %v, want %v", decoded.RoleID, item.RoleID)
	}
}
