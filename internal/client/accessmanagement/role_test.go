package accessmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRoleClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *client.Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &client.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.Config{
				ClientID: "test-client-id",
				Username: "test-user",
				Password: "test-password",
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

			roleClient, err := NewRoleClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRoleClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewRoleClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewRoleClient() unexpected error = %v", err)
				}
				if roleClient == nil {
					t.Errorf("NewRoleClient() returned nil client")
				}
			}
		})
	}
}

func TestRoleClient_CreateRoleGroup(t *testing.T) {
	mockRoleGroup := &RoleGroup{
		ID:            "test-role-group-id",
		Name:          "Test Role Group",
		Description:   "A test role group",
		OrgID:         "test-org-id",
		Editable:      true,
		ExternalNames: []string{},
		CreatedAt:     "2023-01-01T00:00:00Z",
		UpdatedAt:     "2023-01-01T00:00:00Z",
	}

	tests := []struct {
		name        string
		orgID       string
		request     *CreateRoleGroupRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful creation",
			orgID: "test-org-id",
			request: &CreateRoleGroupRequest{
				Name:        "Test Role Group",
				Description: "A test role group",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/rolegroups")

				body := testutil.AssertJSONBody(t, r, "name", "description")

				if body["name"] != "Test Role Group" {
					t.Errorf("Expected name 'Test Role Group', got %v", body["name"])
				}
				if body["description"] != "A test role group" {
					t.Errorf("Expected description 'A test role group', got %v", body["description"])
				}

				testutil.JSONResponse(w, http.StatusCreated, mockRoleGroup)
			},
			wantErr: false,
		},
		{
			name:  "creation without description",
			orgID: "test-org-id",
			request: &CreateRoleGroupRequest{
				Name: "Minimal Role Group",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/rolegroups")
				testutil.JSONResponse(w, http.StatusCreated, &RoleGroup{
					ID:            "minimal-role-id",
					Name:          "Minimal Role Group",
					OrgID:         "test-org-id",
					Editable:      true,
					ExternalNames: []string{},
					CreatedAt:     "2023-01-01T00:00:00Z",
					UpdatedAt:     "2023-01-01T00:00:00Z",
				})
			},
			wantErr: false,
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			request: &CreateRoleGroupRequest{
				Name: "Error Role Group",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := testutil.StandardMockHandlers()
			handlers["/accounts/api/organizations/test-org-id/rolegroups"] = tt.mockHandler

			server := testutil.MockHTTPServer(t, handlers)

			roleClient, err := NewRoleClient(&client.Config{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			roleGroup, err := roleClient.CreateRoleGroup(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateRoleGroup() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateRoleGroup() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateRoleGroup() unexpected error = %v", err)
				}
				if roleGroup == nil {
					t.Errorf("CreateRoleGroup() returned nil role group")
				}
			}
		})
	}
}

func TestRoleClient_GetRoleGroup(t *testing.T) {
	mockRoleGroup := &RoleGroup{
		ID:            "test-role-group-id",
		Name:          "Test Role Group",
		Description:   "A test role group",
		OrgID:         "test-org-id",
		Editable:      true,
		ExternalNames: []string{"external-group-1"},
		CreatedAt:     "2023-01-01T00:00:00Z",
		UpdatedAt:     "2023-01-01T00:00:00Z",
	}

	tests := []struct {
		name        string
		orgID       string
		roleGroupID string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		isNotFound  bool
	}{
		{
			name:        "successful get",
			orgID:       "test-org-id",
			roleGroupID: "test-role-group-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id")
				testutil.JSONResponse(w, http.StatusOK, mockRoleGroup)
			},
			wantErr: false,
		},
		{
			name:        "not found",
			orgID:       "test-org-id",
			roleGroupID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Role group not found"))
			},
			wantErr:    true,
			isNotFound: true,
		},
		{
			name:        "server error",
			orgID:       "test-org-id",
			roleGroupID: "error-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := testutil.StandardMockHandlers()
			handlers["/accounts/api/organizations/test-org-id/rolegroups/"+tt.roleGroupID] = tt.mockHandler

			server := testutil.MockHTTPServer(t, handlers)

			roleClient, err := NewRoleClient(&client.Config{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			roleGroup, err := roleClient.GetRoleGroup(context.Background(), tt.orgID, tt.roleGroupID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetRoleGroup() expected error, got nil")
				}
				if tt.isNotFound && !client.IsNotFound(err) {
					t.Errorf("GetRoleGroup() expected NotFound error, got: %v", err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetRoleGroup() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetRoleGroup() unexpected error = %v", err)
				}
				if roleGroup == nil {
					t.Errorf("GetRoleGroup() returned nil role group")
				} else {
					if roleGroup.Name != mockRoleGroup.Name {
						t.Errorf("GetRoleGroup() name = %v, want %v", roleGroup.Name, mockRoleGroup.Name)
					}
					if len(roleGroup.ExternalNames) != 1 {
						t.Errorf("GetRoleGroup() external_names length = %d, want 1", len(roleGroup.ExternalNames))
					}
				}
			}
		})
	}
}

func TestRoleClient_UpdateRoleGroup(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		roleGroupID string
		request     *UpdateRoleGroupRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful update",
			orgID:       "test-org-id",
			roleGroupID: "test-role-group-id",
			request: &UpdateRoleGroupRequest{
				Name:        "Updated Role Group",
				Description: "Updated description",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id")

				body := testutil.AssertJSONBody(t, r, "name", "description")

				if body["name"] != "Updated Role Group" {
					t.Errorf("Expected name 'Updated Role Group', got %v", body["name"])
				}

				// Ensure external_names is NOT in the body (API rejects it)
				if _, exists := body["external_names"]; exists {
					t.Errorf("external_names should not be in update request body")
				}

				testutil.JSONResponse(w, http.StatusOK, &RoleGroup{
					ID:            "test-role-group-id",
					Name:          "Updated Role Group",
					Description:   "Updated description",
					OrgID:         "test-org-id",
					Editable:      true,
					ExternalNames: []string{},
					CreatedAt:     "2023-01-01T00:00:00Z",
					UpdatedAt:     "2023-01-02T00:00:00Z",
				})
			},
			wantErr: false,
		},
		{
			name:        "not found",
			orgID:       "test-org-id",
			roleGroupID: "nonexistent-id",
			request: &UpdateRoleGroupRequest{
				Name: "Updated",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Role group not found"))
			},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := testutil.StandardMockHandlers()
			handlers["/accounts/api/organizations/test-org-id/rolegroups/"+tt.roleGroupID] = tt.mockHandler

			server := testutil.MockHTTPServer(t, handlers)

			roleClient, err := NewRoleClient(&client.Config{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			roleGroup, err := roleClient.UpdateRoleGroup(context.Background(), tt.orgID, tt.roleGroupID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateRoleGroup() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateRoleGroup() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateRoleGroup() unexpected error = %v", err)
				}
				if roleGroup == nil {
					t.Errorf("UpdateRoleGroup() returned nil role group")
				} else if roleGroup.Name != "Updated Role Group" {
					t.Errorf("UpdateRoleGroup() name = %v, want 'Updated Role Group'", roleGroup.Name)
				}
			}
		})
	}
}

func TestRoleClient_DeleteRoleGroup(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		roleGroupID string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		isNotFound  bool
	}{
		{
			name:        "successful delete",
			orgID:       "test-org-id",
			roleGroupID: "test-role-group-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id")
				// The role group DELETE API returns 200 with an org ID array
				testutil.JSONResponse(w, http.StatusOK, []string{"test-org-id"})
			},
			wantErr: false,
		},
		{
			name:        "not found",
			orgID:       "test-org-id",
			roleGroupID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Role group not found"))
			},
			wantErr:    true,
			isNotFound: true,
		},
		{
			name:        "server error",
			orgID:       "test-org-id",
			roleGroupID: "error-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete role group with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := testutil.StandardMockHandlers()
			handlers["/accounts/api/organizations/test-org-id/rolegroups/"+tt.roleGroupID] = tt.mockHandler

			server := testutil.MockHTTPServer(t, handlers)

			roleClient, err := NewRoleClient(&client.Config{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = roleClient.DeleteRoleGroup(context.Background(), tt.orgID, tt.roleGroupID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteRoleGroup() expected error, got nil")
				}
				if tt.isNotFound && !client.IsNotFound(err) {
					t.Errorf("DeleteRoleGroup() expected NotFound error, got: %v", err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteRoleGroup() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteRoleGroup() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRoleClient_ListRoleGroups(t *testing.T) {
	mockRoleGroups := []RoleGroup{
		{
			ID:            "role-1",
			Name:          "Organization Administrators",
			Description:   "Org admin role",
			OrgID:         "test-org-id",
			Editable:      false,
			ExternalNames: []string{},
			CreatedAt:     "2023-01-01T00:00:00Z",
			UpdatedAt:     "2023-01-01T00:00:00Z",
		},
		{
			ID:            "role-2",
			Name:          "Custom Role",
			Description:   "A custom role",
			OrgID:         "test-org-id",
			Editable:      true,
			ExternalNames: []string{"saml-group-1"},
			CreatedAt:     "2023-02-01T00:00:00Z",
			UpdatedAt:     "2023-02-01T00:00:00Z",
		},
	}

	tests := []struct {
		name        string
		orgID       string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		wantCount   int
	}{
		{
			name:  "successful list",
			orgID: "test-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/rolegroups")
				testutil.JSONResponse(w, http.StatusOK, ListRoleGroupsResponse{Data: mockRoleGroups})
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:  "empty list",
			orgID: "test-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, ListRoleGroupsResponse{Data: []RoleGroup{}})
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to list role groups with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := testutil.StandardMockHandlers()
			handlers["/accounts/api/organizations/test-org-id/rolegroups"] = tt.mockHandler

			server := testutil.MockHTTPServer(t, handlers)

			roleClient, err := NewRoleClient(&client.Config{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			roleGroups, err := roleClient.ListRoleGroups(context.Background(), tt.orgID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListRoleGroups() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ListRoleGroups() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListRoleGroups() unexpected error = %v", err)
				}
				if len(roleGroups) != tt.wantCount {
					t.Errorf("ListRoleGroups() returned %d role groups, want %d", len(roleGroups), tt.wantCount)
				}
			}
		})
	}
}
