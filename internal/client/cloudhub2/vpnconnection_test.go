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

func TestNewVPNConnectionClient(t *testing.T) {
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

			client, err := NewVPNConnectionClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewVPNConnectionClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewVPNConnectionClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewVPNConnectionClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewVPNConnectionClient() returned nil client")
				}
			}
		})
	}
}

func TestVPNConnectionClient_CreateVPNConnection(t *testing.T) {
	mockVPNConnection := &VPNConnection{
		ID: "test-vpn-id",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *CreateVPNConnectionRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful creation",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreateVPNConnectionRequest{},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusCreated, mockVPNConnection)
			},
			wantErr: false,
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreateVPNConnectionRequest{},
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

			client := &VPNConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreateVPNConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateVPNConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreateVPNConnection() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreateVPNConnection() returned nil")
				}
			}
		})
	}
}

func TestVPNConnectionClient_GetVPNConnection(t *testing.T) {
	mockVPNConnection := &VPNConnection{
		ID: "test-vpn-id",
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
			connectionID:   "test-vpn-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockVPNConnection)
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

			client := &VPNConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetVPNConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.connectionID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetVPNConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetVPNConnection() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetVPNConnection() returned nil")
				}
			}
		})
	}
}

func TestVPNConnectionClient_DeleteVPNConnection(t *testing.T) {
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
			connectionID:   "test-vpn-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:           "not found returns nil",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			connectionID:   "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
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

			client := &VPNConnectionClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteVPNConnection(context.Background(), tt.orgID, tt.privateSpaceID, tt.connectionID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteVPNConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeleteVPNConnection() unexpected error = %v", err)
				}
			}
		})
	}
}

// JSON serialization test
func TestVPNConnection_JSONSerialization(t *testing.T) {
	item := &VPNConnection{
		ID: "test-id",
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal VPNConnection: %v", err)
	}

	// Test unmarshaling
	var decoded VPNConnection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal VPNConnection: %v", err)
	}

	// Validate key fields
	if decoded.ID != item.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, item.ID)
	}
}
