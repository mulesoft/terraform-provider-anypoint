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

func TestNewConnectedAppClient(t *testing.T) {
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
		{
			name: "missing client secret",
			config: &client.UserClientConfig{
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

			client, err := NewConnectedAppClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewConnectedAppClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewConnectedAppClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewConnectedAppClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewConnectedAppClient() returned nil client")
				}
			}
		})
	}
}

func TestConnectedAppClient_CreateConnectedApp(t *testing.T) {
	mockApp := &ConnectedApp{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ClientName:   "Test App",
		OrgID:        "test-org-id",
		OwnerOrgID:   "test-org-id",
		OwnerUserID:  "test-user-id",
		GrantTypes:   []string{"client_credentials"},
		RedirectURIs: []string{},
		PublicKeys:   []string{},
		Audience:     "internal",
		ClientURI:    "https://example.com",
		Enabled:      true,
		CreatedAt:    "2023-01-01T00:00:00Z",
		UpdatedAt:    "2023-01-01T00:00:00Z",
	}

	tests := []struct {
		name        string
		orgID       string
		request     *CreateConnectedAppRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful creation",
			orgID: "test-org-id",
			request: &CreateConnectedAppRequest{
				ClientName:   "Test App",
				GrantTypes:   []string{"client_credentials"},
				RedirectURIs: []string{},
				PublicKeys:   []string{},
				Audience:     "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/connectedApplications")
				body := testutil.AssertJSONBody(t, r, "client_name", "grant_types", "redirect_uris")
				if body["client_name"] != "Test App" {
					t.Errorf("Expected client_name 'Test App', got %v", body["client_name"])
				}
				testutil.JSONResponse(w, http.StatusCreated, mockApp)
			},
			wantErr: false,
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			request: &CreateConnectedAppRequest{
				ClientName:   "Test App",
				GrantTypes:   []string{"client_credentials"},
				RedirectURIs: []string{},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create connected app with status 500",
		},
		{
			name:  "malformed response",
			orgID: "test-org-id",
			request: &CreateConnectedAppRequest{
				ClientName:   "Test App",
				GrantTypes:   []string{"client_credentials"},
				RedirectURIs: []string{},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/connectedApplications", tt.orgID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			app, err := client.CreateConnectedApp(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateConnectedApp() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateConnectedApp() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateConnectedApp() unexpected error = %v", err)
				}
				if app == nil {
					t.Errorf("CreateConnectedApp() returned nil app")
				}
				if app != nil && app.ClientName != mockApp.ClientName {
					t.Errorf("CreateConnectedApp() ClientName = %v, want %v", app.ClientName, mockApp.ClientName)
				}
			}
		})
	}
}

func TestConnectedAppClient_GetConnectedApp(t *testing.T) {
	mockApp := &ConnectedApp{
		ClientID:    "test-client-id",
		ClientName:  "Test App",
		OrgID:       "test-org-id",
		OwnerUserID: "test-user-id",
		GrantTypes:  []string{"client_credentials"},
		Audience:    "internal",
		Enabled:     true,
		CreatedAt:   "2023-01-01T00:00:00Z",
		UpdatedAt:   "2023-01-01T00:00:00Z",
	}

	tests := []struct {
		name        string
		orgID       string
		clientID    string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful get",
			orgID:    "test-org-id",
			clientID: "test-client-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/connectedApplications/test-client-id")
				testutil.JSONResponse(w, http.StatusOK, mockApp)
			},
			wantErr: false,
		},
		{
			name:     "not found",
			orgID:    "test-org-id",
			clientID: "nonexistent-client-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Connected app not found")
			},
			wantErr:     true,
			errContains: "connected app",
		},
		{
			name:     "server error",
			orgID:    "test-org-id",
			clientID: "test-client-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get connected app with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/connectedApplications/%s", tt.orgID, tt.clientID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			app, err := client.GetConnectedApp(context.Background(), tt.orgID, tt.clientID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetConnectedApp() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetConnectedApp() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetConnectedApp() unexpected error = %v", err)
				}
				if app == nil {
					t.Errorf("GetConnectedApp() returned nil app")
				}
			}
		})
	}
}

func TestConnectedAppClient_UpdateConnectedApp(t *testing.T) {
	mockApp := &ConnectedApp{
		ClientID:    "test-client-id",
		ClientName:  "Updated App",
		OrgID:       "test-org-id",
		OwnerUserID: "test-user-id",
		GrantTypes:  []string{"client_credentials"},
		Enabled:     false,
		UpdatedAt:   "2023-01-02T00:00:00Z",
	}

	tests := []struct {
		name        string
		orgID       string
		clientID    string
		request     *UpdateConnectedAppRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful update",
			orgID:    "test-org-id",
			clientID: "test-client-id",
			request: &UpdateConnectedAppRequest{
				ClientName: testutil.StringPtr("Updated App"),
				Enabled:    testutil.BoolPtr(false),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PATCH", "/accounts/api/organizations/test-org-id/connectedApplications/test-client-id")
				body := testutil.AssertJSONBody(t, r, "client_name", "enabled")
				if body["client_name"] != "Updated App" {
					t.Errorf("Expected client_name 'Updated App', got %v", body["client_name"])
				}
				testutil.JSONResponse(w, http.StatusOK, mockApp)
			},
			wantErr: false,
		},
		{
			name:     "not found",
			orgID:    "test-org-id",
			clientID: "nonexistent-client-id",
			request: &UpdateConnectedAppRequest{
				ClientName: testutil.StringPtr("Updated App"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Connected app not found")
			},
			wantErr:     true,
			errContains: "connected app",
		},
		{
			name:     "server error",
			orgID:    "test-org-id",
			clientID: "test-client-id",
			request: &UpdateConnectedAppRequest{
				ClientName: testutil.StringPtr("Updated App"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to update connected app with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/connectedApplications/%s", tt.orgID, tt.clientID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			app, err := client.UpdateConnectedApp(context.Background(), tt.orgID, tt.clientID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateConnectedApp() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateConnectedApp() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateConnectedApp() unexpected error = %v", err)
				}
				if app == nil {
					t.Errorf("UpdateConnectedApp() returned nil app")
				}
			}
		})
	}
}

func TestConnectedAppClient_DeleteConnectedApp(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		clientID    string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful deletion",
			orgID:    "test-org-id",
			clientID: "test-client-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/connectedApplications/test-client-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:     "already deleted - not found",
			orgID:    "test-org-id",
			clientID: "nonexistent-client-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Connected app not found")
			},
			wantErr: false,
		},
		{
			name:     "server error",
			orgID:    "test-org-id",
			clientID: "test-client-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete connected app with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/connectedApplications/%s", tt.orgID, tt.clientID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteConnectedApp(context.Background(), tt.orgID, tt.clientID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteConnectedApp() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteConnectedApp() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteConnectedApp() unexpected error = %v", err)
				}
			}
		})
	}
}
