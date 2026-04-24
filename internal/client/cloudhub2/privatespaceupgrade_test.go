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

func TestNewPrivateSpaceUpgradeClient(t *testing.T) {
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

			client, err := NewPrivateSpaceUpgradeClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPrivateSpaceUpgradeClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPrivateSpaceUpgradeClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPrivateSpaceUpgradeClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewPrivateSpaceUpgradeClient() returned nil client")
				}
			}
		})
	}
}

func TestPrivateSpaceUpgradeClient_UpgradePrivateSpace(t *testing.T) {
	mockResponse := &PrivateSpaceUpgradeResponse{
		ScheduledUpdateTime: "2026-05-01T00:00:00Z",
		Status:              "SCHEDULED",
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *UpgradePrivateSpaceRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful upgrade",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &UpgradePrivateSpaceRequest{
				Date:  "2026-05-01",
				OptIn: true,
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" {
					t.Errorf("Expected POST/PUT/PATCH request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockResponse)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/upgrade", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceUpgradeClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpgradePrivateSpace(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpgradePrivateSpace() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpgradePrivateSpace() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpgradePrivateSpace() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceUpgradeClient_GetPrivateSpaceUpgradeStatus(t *testing.T) {
	mockResponse := &PrivateSpaceUpgradeResponse{
		ScheduledUpdateTime: "2026-05-01T00:00:00Z",
		Status:              "SCHEDULED",
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
				testutil.JSONResponse(w, http.StatusOK, mockResponse)
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/upgradestatus", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceUpgradeClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetPrivateSpaceUpgradeStatus(context.Background(), tt.orgID, tt.privateSpaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrivateSpaceUpgradeStatus() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetPrivateSpaceUpgradeStatus() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetPrivateSpaceUpgradeStatus() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceUpgradeClient_DeletePrivateSpaceUpgrade(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful deletion",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
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
			privateSpaceID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/upgrade", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceUpgradeClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeletePrivateSpaceUpgrade(context.Background(), tt.orgID, tt.privateSpaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeletePrivateSpaceUpgrade() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeletePrivateSpaceUpgrade() unexpected error = %v", err)
				}
			}
		})
	}
}

// JSON serialization test
func TestPrivateSpaceUpgradeResponse_JSONSerialization(t *testing.T) {
	item := &PrivateSpaceUpgradeResponse{
		ScheduledUpdateTime: "2026-05-01T00:00:00Z",
		Status:              "SCHEDULED",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal PrivateSpaceUpgradeResponse: %v", err)
	}

	var decoded PrivateSpaceUpgradeResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PrivateSpaceUpgradeResponse: %v", err)
	}

	if decoded.Status != item.Status {
		t.Errorf("Unmarshaled Status = %v, want %v", decoded.Status, item.Status)
	}
	if decoded.ScheduledUpdateTime != item.ScheduledUpdateTime {
		t.Errorf("Unmarshaled ScheduledUpdateTime = %v, want %v", decoded.ScheduledUpdateTime, item.ScheduledUpdateTime)
	}
}
