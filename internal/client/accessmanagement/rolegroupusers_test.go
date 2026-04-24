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

func TestNewRoleGroupUsersClient(t *testing.T) {
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

			c, err := NewRoleGroupUsersClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRoleGroupUsersClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewRoleGroupUsersClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewRoleGroupUsersClient() unexpected error = %v", err)
				}
				if c == nil {
					t.Errorf("NewRoleGroupUsersClient() returned nil client")
				}
			}
		})
	}
}

func TestRoleGroupUsersClient_AssignUsersToRoleGroup(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		roleGroupID string
		userIDs     []string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful assignment",
			orgID:       "test-org-id",
			roleGroupID: "test-rolegroup-id",
			userIDs:     []string{"user-1", "user-2"},
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
			userIDs:     []string{"user-1"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to assign users to role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s/users", tt.orgID, tt.roleGroupID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &RoleGroupUsersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.AssignUsersToRoleGroup(context.Background(), tt.orgID, tt.roleGroupID, tt.userIDs)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AssignUsersToRoleGroup() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AssignUsersToRoleGroup() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AssignUsersToRoleGroup() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRoleGroupUsersClient_GetRoleGroupUsers(t *testing.T) {
	mockResponse := RoleGroupUsersResponse{
		Data: []UserAssignment{
			{ID: "user-1", Username: "testuser", Email: "test@example.com"},
		},
		Total: 1,
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
				testutil.JSONResponse(w, http.StatusOK, mockResponse)
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
			errContains: "failed to get users for role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s/users", tt.orgID, tt.roleGroupID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &RoleGroupUsersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := c.GetRoleGroupUsers(context.Background(), tt.orgID, tt.roleGroupID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetRoleGroupUsers() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetRoleGroupUsers() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetRoleGroupUsers() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetRoleGroupUsers() returned nil")
				}
			}
		})
	}
}

func TestUserAssignment_JSONSerialization(t *testing.T) {
	item := &UserAssignment{
		ID:       "test-user-id",
		Username: "testuser",
		Email:    "test@example.com",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal UserAssignment: %v", err)
	}

	var decoded UserAssignment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UserAssignment: %v", err)
	}

	if decoded.ID != item.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, item.ID)
	}
	if decoded.Username != item.Username {
		t.Errorf("Unmarshaled Username = %v, want %v", decoded.Username, item.Username)
	}
}
