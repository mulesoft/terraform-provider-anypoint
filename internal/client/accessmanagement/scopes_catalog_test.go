package accessmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewScopesCatalogClient(t *testing.T) {
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
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.Config{
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

			client, err := NewScopesCatalogClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewScopesCatalogClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewScopesCatalogClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewScopesCatalogClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewScopesCatalogClient() returned nil client")
				}
			}
		})
	}
}

func TestScopesCatalogClient_ListScopesCatalog(t *testing.T) {
	mockScopes := []ScopeCatalogEntry{
		{
			Scope:        "create:generations",
			DisplayName:  "Mule Developer Generative AI User",
			Description:  "Access to AI generation features",
			ProductLabel: "AI",
			Internal:     false,
		},
		{
			Scope:        "read:exchange",
			DisplayName:  "Exchange Viewer",
			Description:  "Read access to Exchange assets",
			ProductLabel: "Exchange",
			Internal:     false,
		},
		{
			Scope:        "internal:admin",
			DisplayName:  "Internal Admin",
			Description:  "Internal system administration",
			ProductLabel: "System",
			Internal:     true,
		},
	}

	tests := []struct {
		name          string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
		expectedCount int
	}{
		{
			name: "successful list",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/cs/scopes")
				testutil.JSONResponse(w, http.StatusOK, mockScopes)
			},
			wantErr:       false,
			expectedCount: 3,
		},
		{
			name: "server error",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to list scopes catalog with status 500",
		},
		{
			name: "malformed response",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/cs/scopes": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &ScopesCatalogClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			scopes, err := client.ListScopesCatalog(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListScopesCatalog() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ListScopesCatalog() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListScopesCatalog() unexpected error = %v", err)
				}
				if len(scopes) != tt.expectedCount {
					t.Errorf("ListScopesCatalog() returned %d scopes, want %d", len(scopes), tt.expectedCount)
				}
				if len(scopes) > 0 && scopes[0].Scope != mockScopes[0].Scope {
					t.Errorf("ListScopesCatalog() first scope = %v, want %v", scopes[0].Scope, mockScopes[0].Scope)
				}
			}
		})
	}
}
