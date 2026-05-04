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

func TestNewConnectedAppClient(t *testing.T) {
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
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.Config{
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
	mockConnectedApp := &ConnectedApp{
		ClientID:     "test-connected-app-id",
		ClientName:   "Test Connected App",
		OwnerOrgID:   "test-org-id",
		ClientSecret: "test-secret",
		GrantTypes:   []string{"client_credentials"},
		Enabled:      true,
	}

	tests := []struct {
		name        string
		request     *CreateConnectedAppRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation",
			request: &CreateConnectedAppRequest{
				ClientID:     "test-connected-app-id",
				ClientName:   "Test Connected App",
				OwnerOrgID:   "test-org-id",
				ClientSecret: "test-secret",
				GrantTypes:   []string{"client_credentials"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/connectedApplications")

				body := testutil.AssertJSONBody(t, r, "client_name")
				if body["client_name"] != "Test Connected App" {
					t.Error("Expected client_name in request body")
				}

				testutil.JSONResponse(w, http.StatusCreated, mockConnectedApp)
			},
			wantErr: false,
		},
		{
			name: "server error",
			request: &CreateConnectedAppRequest{
				ClientID:     "test-connected-app-id",
				ClientName:   "Test Connected App",
				OwnerOrgID:   "test-org-id",
				ClientSecret: "test-secret",
				GrantTypes:   []string{"client_credentials"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/connectedApplications": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreateConnectedApp(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateConnectedApp() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateConnectedApp() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateConnectedApp() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreateConnectedApp() returned nil connected app")
				}

				// Validate returned connected app
				if result != nil {
					if result.ClientID != mockConnectedApp.ClientID {
						t.Errorf("CreateConnectedApp() ClientID = %v, want %v", result.ClientID, mockConnectedApp.ClientID)
					}
					if result.ClientName != mockConnectedApp.ClientName {
						t.Errorf("CreateConnectedApp() ClientName = %v, want %v", result.ClientName, mockConnectedApp.ClientName)
					}
				}
			}
		})
	}
}

func TestConnectedAppClient_GetConnectedApp(t *testing.T) {
	mockConnectedApp := &ConnectedApp{
		ClientID:     "test-connected-app-id",
		ClientName:   "Test Connected App",
		OwnerOrgID:   "test-org-id",
		ClientSecret: "test-secret",
		GrantTypes:   []string{"client_credentials"},
		Enabled:      true,
	}

	tests := []struct {
		name        string
		clientID    string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful get",
			clientID: "test-connected-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/connectedApplications/test-connected-app-id")
				testutil.JSONResponse(w, http.StatusOK, mockConnectedApp)
			},
			wantErr: false,
		},
		{
			name:     "server error",
			clientID: "test-connected-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/connectedApplications/%s", tt.clientID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetConnectedApp(context.Background(), tt.clientID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetConnectedApp() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetConnectedApp() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetConnectedApp() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetConnectedApp() returned nil connected app")
				}
			}
		})
	}
}

func TestConnectedAppClient_DeleteConnectedApp(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful deletion",
			clientID: "test-connected-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/connectedApplications/test-connected-app-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:     "server error",
			clientID: "test-connected-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/connectedApplications/%s", tt.clientID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteConnectedApp(context.Background(), tt.clientID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteConnectedApp() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
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

// JSON serialization tests
func TestCreateConnectedAppRequest_JSONSerialization(t *testing.T) {
	req := &CreateConnectedAppRequest{
		ClientID:     "test-connected-app-id",
		ClientName:   "Test Connected App",
		OwnerOrgID:   "test-org-id",
		ClientSecret: "test-secret",
		GrantTypes:   []string{"client_credentials"},
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CreateConnectedAppRequest: %v", err)
	}

	// Test unmarshaling
	var decoded CreateConnectedAppRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CreateConnectedAppRequest: %v", err)
	}

	// Validate key fields
	if decoded.ClientID != req.ClientID {
		t.Errorf("Unmarshaled ClientID = %v, want %v", decoded.ClientID, req.ClientID)
	}
	if decoded.ClientName != req.ClientName {
		t.Errorf("Unmarshaled ClientName = %v, want %v", decoded.ClientName, req.ClientName)
	}
	if decoded.OwnerOrgID != req.OwnerOrgID {
		t.Errorf("Unmarshaled OwnerOrgID = %v, want %v", decoded.OwnerOrgID, req.OwnerOrgID)
	}
}
