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

func TestNewTLSContextClient(t *testing.T) {
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

			client, err := NewTLSContextClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTLSContextClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewTLSContextClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewTLSContextClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewTLSContextClient() returned nil client")
				}
			}
		})
	}
}

func TestTLSContextClient_CreateTLSContext(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *CreateTLSContextRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful creation",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreateTLSContextRequest{},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusCreated)
			},
			wantErr: false,
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreateTLSContextRequest{},
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TLSContextClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.CreateTLSContext(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateTLSContext() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreateTLSContext() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTLSContextClient_GetTLSContext(t *testing.T) {
	mockTLSContext := &TLSContext{
		ID: "test-tls-id",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		tlsContextID   string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful get",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			tlsContextID:   "test-tls-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockTLSContext)
			},
			wantErr: false,
		},
		{
			name:           "not found",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			tlsContextID:   "nonexistent-id",
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts/%s", tt.orgID, tt.privateSpaceID, tt.tlsContextID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TLSContextClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetTLSContext(context.Background(), tt.orgID, tt.privateSpaceID, tt.tlsContextID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTLSContext() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetTLSContext() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetTLSContext() returned nil")
				}
			}
		})
	}
}

func TestTLSContextClient_UpdateTLSContext(t *testing.T) {
	mockTLSContext := &TLSContext{
		ID: "test-tls-id",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		tlsContextID   string
		request        *UpdateTLSContextRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful update",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			tlsContextID:   "test-tls-id",
			request:        &UpdateTLSContextRequest{},
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
				testutil.JSONResponse(w, http.StatusOK, mockTLSContext)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts/%s", tt.orgID, tt.privateSpaceID, tt.tlsContextID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TLSContextClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdateTLSContext(context.Background(), tt.orgID, tt.privateSpaceID, tt.tlsContextID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateTLSContext() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateTLSContext() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdateTLSContext() returned nil")
				}
			}
		})
	}
}

func TestTLSContextClient_DeleteTLSContext(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		tlsContextID   string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful deletion",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			tlsContextID:   "test-tls-id",
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
			tlsContextID:   "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts/%s", tt.orgID, tt.privateSpaceID, tt.tlsContextID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TLSContextClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteTLSContext(context.Background(), tt.orgID, tt.privateSpaceID, tt.tlsContextID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteTLSContext() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeleteTLSContext() unexpected error = %v", err)
				}
			}
		})
	}
}

// JSON serialization test
func TestTLSContext_JSONSerialization(t *testing.T) {
	item := &TLSContext{
		ID: "test-id",
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal TLSContext: %v", err)
	}

	// Test unmarshaling
	var decoded TLSContext
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TLSContext: %v", err)
	}

	// Validate key fields
	if decoded.ID != item.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, item.ID)
	}
}
