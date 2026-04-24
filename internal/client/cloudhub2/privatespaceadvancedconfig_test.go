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

func TestNewPrivateSpaceAdvancedConfigClient(t *testing.T) {
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

			client, err := NewPrivateSpaceAdvancedConfigClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPrivateSpaceAdvancedConfigClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPrivateSpaceAdvancedConfigClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPrivateSpaceAdvancedConfigClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewPrivateSpaceAdvancedConfigClient() returned nil client")
				}
			}
		})
	}
}

func TestPrivateSpaceAdvancedConfigClient_GetPrivateSpace(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:   "test-space-id",
		Name: "Test Private Space",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful get",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr: false,
		},
		{
			name:           "not found",
			orgID:          "test-org-id",
			privateSpaceID: "nonexistent-id",
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceAdvancedConfigClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetPrivateSpace(context.Background(), tt.orgID, tt.privateSpaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrivateSpace() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetPrivateSpace() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetPrivateSpace() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceAdvancedConfigClient_UpdatePrivateSpaceAdvancedConfig(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:   "test-space-id",
		Name: "Test Private Space",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *UpdatePrivateSpaceAdvancedConfigRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful update",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &UpdatePrivateSpaceAdvancedConfigRequest{},
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
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceAdvancedConfigClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdatePrivateSpaceAdvancedConfig(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdatePrivateSpaceAdvancedConfig() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePrivateSpaceAdvancedConfig() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdatePrivateSpaceAdvancedConfig() returned nil")
				}
			}
		})
	}
}

// JSON serialization test
func TestPrivateSpaceAdvancedConfig_JSONSerialization(t *testing.T) {
	item := &PrivateSpaceAdvancedConfig{
		EnableIAMRole: true,
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal PrivateSpaceAdvancedConfig: %v", err)
	}

	// Test unmarshaling
	var decoded PrivateSpaceAdvancedConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PrivateSpaceAdvancedConfig: %v", err)
	}

	if decoded.EnableIAMRole != item.EnableIAMRole {
		t.Errorf("Unmarshaled EnableIAMRole = %v, want %v", decoded.EnableIAMRole, item.EnableIAMRole)
	}
}
