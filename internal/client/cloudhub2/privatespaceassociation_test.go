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

func TestNewPrivateSpaceAssociationClient(t *testing.T) {
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

			client, err := NewPrivateSpaceAssociationClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPrivateSpaceAssociationClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPrivateSpaceAssociationClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPrivateSpaceAssociationClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewPrivateSpaceAssociationClient() returned nil client")
				}
			}
		})
	}
}

func TestPrivateSpaceAssociationClient_CreatePrivateSpaceAssociations(t *testing.T) {
	mockAssociations := []PrivateSpaceAssociation{
		{
			ID: "test-PrivateSpaceAssociation-id",
		},
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *CreatePrivateSpaceAssociationRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful creation",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreatePrivateSpaceAssociationRequest{},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusCreated, mockAssociations)
			},
			wantErr: false,
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request:        &CreatePrivateSpaceAssociationRequest{},
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/associations", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceAssociationClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreatePrivateSpaceAssociations(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreatePrivateSpaceAssociations() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreatePrivateSpaceAssociations() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreatePrivateSpaceAssociations() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceAssociationClient_GetPrivateSpaceAssociations(t *testing.T) {
	mockAssociations := []PrivateSpaceAssociation{
		{
			ID: "test-PrivateSpaceAssociation-id",
		},
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
				testutil.JSONResponse(w, http.StatusOK, mockAssociations)
			},
			wantErr: false,
		},
		{
			name:           "not found returns empty slice",
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/associations", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceAssociationClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetPrivateSpaceAssociations(context.Background(), tt.orgID, tt.privateSpaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrivateSpaceAssociations() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetPrivateSpaceAssociations() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetPrivateSpaceAssociations() returned nil")
				}
			}
		})
	}
}

func TestPrivateSpaceAssociationClient_DeletePrivateSpaceAssociation(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		associationID  string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:           "successful deletion",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			associationID:  "test-PrivateSpaceAssociation-id",
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
			associationID:  "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s/associations/%s", tt.orgID, tt.privateSpaceID, tt.associationID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpaceAssociationClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeletePrivateSpaceAssociation(context.Background(), tt.orgID, tt.privateSpaceID, tt.associationID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeletePrivateSpaceAssociation() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeletePrivateSpaceAssociation() unexpected error = %v", err)
				}
			}
		})
	}
}

// JSON serialization test
func TestPrivateSpaceAssociation_JSONSerialization(t *testing.T) {
	item := &PrivateSpaceAssociation{
		ID: "test-id",
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal PrivateSpaceAssociation: %v", err)
	}

	// Test unmarshaling
	var decoded PrivateSpaceAssociation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PrivateSpaceAssociation: %v", err)
	}

	// Validate key fields
	if decoded.ID != item.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, item.ID)
	}
}
