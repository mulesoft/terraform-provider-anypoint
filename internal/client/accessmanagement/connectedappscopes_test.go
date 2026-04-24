package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppScopesClient(t *testing.T) {
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
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.UserClientConfig{
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

			client, err := NewConnectedAppScopesClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewConnectedAppScopesClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewConnectedAppScopesClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewConnectedAppScopesClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewConnectedAppScopesClient() returned nil client")
				}
			}
		})
	}
}


func TestConnectedAppScopesClient_UpdateConnectedAppScopes(t *testing.T) {
	mockConnectedAppScopes := &ConnectedAppScopes{
		Scopes: []Scope{
			{Scope: "read:scopes", ContextParams: map[string]interface{}{}},
		},
	}

	tests := []struct {
		name        string
		request     *UpdateConnectedAppScopesRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful update",
			request: &UpdateConnectedAppScopesRequest{
				Scopes: []Scope{
					{Scope: "read:scopes", ContextParams: map[string]interface{}{}},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "PATCH" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				if r.Method == "GET" {
					testutil.JSONResponse(w, http.StatusOK, mockConnectedAppScopes)
					return
				}
				t.Errorf("Unexpected method %s", r.Method)
			},
			wantErr: false,
		},
		{
			name: "server error",
			request: &UpdateConnectedAppScopesRequest{
				Scopes: []Scope{},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "API request failed with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/connectedApplications/test-app-id/scopes": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppScopesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdateConnectedAppScopes(context.Background(), "test-app-id", tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateConnectedAppScopes() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateConnectedAppScopes() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdateConnectedAppScopes() returned nil")
				}
			}
		})
	}
}



func TestConnectedAppScopesClient_GetConnectedAppScopes(t *testing.T) {
	mockConnectedAppScopes := &ConnectedAppScopes{
		Scopes: []Scope{},
	}

	tests := []struct {
		name        string
		id          string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful get",
			id:   "test-ConnectedAppScopes-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockConnectedAppScopes)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "nonexistent-id",
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
				"/accounts/api/connectedApplications/" + tt.id + "/scopes": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ConnectedAppScopesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetConnectedAppScopes(context.Background(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetConnectedAppScopes() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetConnectedAppScopes() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetConnectedAppScopes() returned nil")
				}
			}
		})
	}
}






// JSON serialization test
func TestConnectedAppScopes_JSONSerialization(t *testing.T) {
	item := &ConnectedAppScopes{
		Scopes: []Scope{
			{Scope: "read:scopes", ContextParams: map[string]interface{}{}},
		},
	}

	// Test marshaling
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal ConnectedAppScopes: %v", err)
	}

	// Test unmarshaling
	var decoded ConnectedAppScopes
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ConnectedAppScopes: %v", err)
	}

	// Validate key fields
	if len(decoded.Scopes) != len(item.Scopes) {
		t.Errorf("Unmarshaled Scopes length = %v, want %v", len(decoded.Scopes), len(item.Scopes))
	}
}
