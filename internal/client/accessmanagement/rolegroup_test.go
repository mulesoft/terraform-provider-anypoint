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

func TestNewRoleGroupClient(t *testing.T) {
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

			client, err := NewRoleGroupClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRoleGroupClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewRoleGroupClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewRoleGroupClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewRoleGroupClient() returned nil client")
				}
			}
		})
	}
}


func TestRoleGroupClient_CreateRoleGroup(t *testing.T) {
	mockRoleGroup := &RoleGroup{
		RoleGroupID: "test-RoleGroup-id",
		Name:        "Test Role Group",
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
				Name: "Test Role Group",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/rolegroups")
				testutil.JSONResponse(w, http.StatusCreated, mockRoleGroup)
			},
			wantErr: false,
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			request: &CreateRoleGroupRequest{
				Name: "Test Role Group",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups", tt.orgID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &RoleGroupClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreateRoleGroup(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateRoleGroup() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreateRoleGroup() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreateRoleGroup() returned nil")
				}
			}
		})
	}
}



func TestRoleGroupClient_GetRoleGroup(t *testing.T) {
	mockRoleGroup := &RoleGroup{
		RoleGroupID: "test-RoleGroup-id",
		Name:        "Test Role Group",
	}

	tests := []struct {
		name        string
		orgID       string
		id          string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful get",
			orgID: "test-org-id",
			id:    "test-RoleGroup-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockRoleGroup)
			},
			wantErr: false,
		},
		{
			name:  "not found",
			orgID: "test-org-id",
			id:    "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Not found")
			},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s", tt.orgID, tt.id): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &RoleGroupClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetRoleGroup(context.Background(), tt.orgID, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetRoleGroup() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetRoleGroup() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetRoleGroup() returned nil")
				}
			}
		})
	}
}



func TestRoleGroupClient_UpdateRoleGroup(t *testing.T) {
	mockRoleGroup := &RoleGroup{
		RoleGroupID: "test-RoleGroup-id",
		Name:        "Updated Role Group",
	}

	tests := []struct {
		name        string
		orgID       string
		id          string
		request     *UpdateRoleGroupRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:  "successful update",
			orgID: "test-org-id",
			id:    "test-RoleGroup-id",
			request: &UpdateRoleGroupRequest{
				Name: "Updated Role Group",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				expectedMethods := []string{"PUT", "PATCH"}
				found := false
				for _, method := range expectedMethods {
					if r.Method == method {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected PUT or PATCH request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockRoleGroup)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s", tt.orgID, tt.id): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &RoleGroupClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdateRoleGroup(context.Background(), tt.orgID, tt.id, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateRoleGroup() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateRoleGroup() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdateRoleGroup() returned nil")
				}
			}
		})
	}
}



func TestRoleGroupClient_DeleteRoleGroup(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		id          string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:  "successful deletion",
			orgID: "test-org-id",
			id:    "test-RoleGroup-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:  "not found",
			orgID: "test-org-id",
			id:    "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Not found")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/rolegroups/%s", tt.orgID, tt.id): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &RoleGroupClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteRoleGroup(context.Background(), tt.orgID, tt.id)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteRoleGroup() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeleteRoleGroup() unexpected error = %v", err)
				}
			}
		})
	}
}


// JSON serialization test
func TestRoleGroup_JSONSerialization(t *testing.T) {
	item := &RoleGroup{
		RoleGroupID: "test-id",
		Name:        "Test Role Group",
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal RoleGroup: %v", err)
	}

	// Test unmarshaling
	var decoded RoleGroup
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RoleGroup: %v", err)
	}

	// Validate key fields
	if decoded.RoleGroupID != item.RoleGroupID {
		t.Errorf("Unmarshaled RoleGroupID = %v, want %v", decoded.RoleGroupID, item.RoleGroupID)
	}
}
