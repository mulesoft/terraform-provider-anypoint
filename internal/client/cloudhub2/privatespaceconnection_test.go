package cloudhub2

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

func TestNewPrivateSpaceConnectionClient(t *testing.T) {
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

			client, err := NewPrivateSpaceConnectionClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPrivateSpaceConnectionClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPrivateSpaceConnectionClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPrivateSpaceConnectionClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewPrivateSpaceConnectionClient() returned nil client")
				}
			}
		})
	}
}

func TestPrivateSpaceConnectionClient_CreatePrivateSpaceConnection(t *testing.T) {
	mockConnection := &PrivateSpaceConnection{
		ID: "test-connection-id",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *CreatePrivateSpaceConnectionRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful creation",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreatePrivateSpaceConnectionRequest{},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusCreated, mockConnection)
			},
			wantErr: false,
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreatePrivateSpaceConnectionRequest{},
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/connections", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreatePrivateSpaceConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreatePrivateSpaceConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreatePrivateSpaceConnection() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreatePrivateSpaceConnection() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceConnectionClient_GetPrivateSpaceConnection(t *testing.T) {
	mockConnection := &PrivateSpaceConnection{
		ID: "test-connection-id",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		connectionID   string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful get",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			connectionID:   "test-connection-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockConnection)
			},
			wantErr: false,
		},
		{
			name:           "not found",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			connectionID:   "nonexistent-id",
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", tt.orgID, tt.privateSpaceID, tt.connectionID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetPrivateSpaceConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.connectionID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrivateSpaceConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetPrivateSpaceConnection() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetPrivateSpaceConnection() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceConnectionClient_UpdatePrivateSpaceConnection(t *testing.T) {
	mockConnection := &PrivateSpaceConnection{
		ID: "test-connection-id",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		connectionID   string
		request        *UpdatePrivateSpaceConnectionRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful update",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			connectionID:   "test-connection-id",
			request:        &UpdatePrivateSpaceConnectionRequest{},
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
				testutil.JSONResponse(w, http.StatusOK, mockConnection)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", tt.orgID, tt.privateSpaceID, tt.connectionID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdatePrivateSpaceConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.connectionID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdatePrivateSpaceConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePrivateSpaceConnection() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdatePrivateSpaceConnection() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceConnectionClient_DeletePrivateSpaceConnection(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		connectionID   string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful deletion",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			connectionID:   "test-connection-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:           "not found",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			connectionID:   "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/connections/%s", tt.orgID, tt.privateSpaceID, tt.connectionID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeletePrivateSpaceConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.connectionID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeletePrivateSpaceConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeletePrivateSpaceConnection() unexpected error = %v", err)
				}
			}
		})
	}
}

// JSON serialization test
func TestPrivateSpaceConnection_JSONSerialization(t *testing.T) {
	item := &PrivateSpaceConnection{
		ID: "test-id",
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal PrivateSpaceConnection: %v", err)
	}

	// Test unmarshaling
	var decoded PrivateSpaceConnection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PrivateSpaceConnection: %v", err)
	}

	// Validate key fields
	if decoded.ID != item.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, item.ID)
	}
}
