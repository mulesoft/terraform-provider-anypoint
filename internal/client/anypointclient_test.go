package client

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAnypointClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with defaults",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			wantErr: false,
		},
		{
			name: "valid config with custom baseURL and timeout",
			config: &Config{
				BaseURL:      "https://custom.anypoint.mulesoft.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Timeout:      60,
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &Config{
				ClientSecret: "test-client-secret",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &Config{
				ClientID: "test-client-id",
			},
			wantErr:     true,
			errContains: "client_secret is required",
		},
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server for authentication
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

			// Override baseURL to use mock server
			if tt.config != nil && tt.config.BaseURL == "" {
				tt.config.BaseURL = server.URL
			} else if tt.config != nil {
				// Create a new config with server URL for custom baseURL tests
				testConfig := *tt.config
				testConfig.BaseURL = server.URL
				tt.config = &testConfig
			}

			client, err := NewAnypointClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewAnypointClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewAnypointClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewAnypointClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewAnypointClient() returned nil client")
				}

				// Validate client configuration
				if client != nil {
					if client.ClientID != tt.config.ClientID {
						t.Errorf("NewAnypointClient() ClientID = %v, want %v", client.ClientID, tt.config.ClientID)
					}
					if client.ClientSecret != tt.config.ClientSecret {
						t.Errorf("NewAnypointClient() ClientSecret = %v, want %v", client.ClientSecret, tt.config.ClientSecret)
					}
					if client.Token == "" {
						t.Errorf("NewAnypointClient() Token should be set after authentication")
					}
					if client.OrgID == "" {
						t.Errorf("NewAnypointClient() OrgID should be set after authentication")
					}

					// Test timeout setting
					expectedTimeout := 10 * time.Minute
					if tt.config.Timeout > 0 {
						expectedTimeout = time.Duration(tt.config.Timeout) * time.Second
					}
					if client.HTTPClient.Timeout != expectedTimeout {
						t.Errorf("NewAnypointClient() HTTPClient.Timeout = %v, want %v", client.HTTPClient.Timeout, expectedTimeout)
					}
				}
			}
		})
	}
}

func TestAnypointClient_authenticate(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful authentication",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequestWithAuth(t, r, "POST", "/accounts/api/v2/oauth2/token", false)

				body := testutil.AssertJSONBody(t, r, "client_id", "client_secret", "grant_type")

				if body["grant_type"] != "client_credentials" {
					t.Errorf("Expected grant_type 'client_credentials', got %v", body["grant_type"])
				}

				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			wantErr: false,
		},
		{
			name: "authentication failure - invalid credentials",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
			},
			wantErr:     true,
			errContains: "authentication failed with status 401",
		},
		{
			name: "authentication failure - server error",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Server error")
			},
			wantErr:     true,
			errContains: "authentication failed with status 500",
		},
		{
			name: "authentication failure - malformed response",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": "json"`))
			},
			wantErr:     true,
			errContains: "failed to decode auth response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/v2/oauth2/token": tt.mockHandler,
				"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
			}

			server := testutil.MockHTTPServer(t, handlers)

			client := &AnypointClient{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				HTTPClient:   &http.Client{},
			}

			err := client.authenticate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("authenticate() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("authenticate() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("authenticate() unexpected error = %v", err)
				}
				if client.Token == "" {
					t.Errorf("authenticate() Token should be set")
				}
				if client.OrgID == "" {
					t.Errorf("authenticate() OrgID should be set")
				}
			}
		})
	}
}

func TestAnypointClient_getMe(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful get me",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/me")
				testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
			},
			wantErr: false,
		},
		{
			name: "unauthorized request",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
			},
			wantErr:     true,
			errContains: "failed to get user info with status 401",
		},
		{
			name: "malformed JSON response",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/me": tt.mockHandler,
			})

			client := &AnypointClient{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Token:        "mock-access-token",
				OrgID:        "test-org-id",
				HTTPClient:   &http.Client{},
			}

			me, err := client.getMe()

			if tt.wantErr {
				if err == nil {
					t.Errorf("getMe() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("getMe() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("getMe() unexpected error = %v", err)
				}
				if me == nil {
					t.Errorf("getMe() returned nil response")
				}

				// Validate response structure
				if user, ok := me["user"].(map[string]interface{}); ok {
					if userID, exists := user["id"].(string); !exists || userID == "" {
						t.Errorf("getMe() missing or empty user.id")
					}
				} else {
					t.Errorf("getMe() response missing user object")
				}
			}
		})
	}
}

func TestAnypointClient_extractOrgID(t *testing.T) {
	client := &AnypointClient{}

	tests := []struct {
		name          string
		meResponse    map[string]interface{}
		expectedOrgID string
		wantErr       bool
	}{
		{
			name: "extract from connected app (priority 1)",
			meResponse: map[string]interface{}{
				"client": map[string]interface{}{
					"org_id": "client-org-id",
				},
				"user": map[string]interface{}{
					"organization": map[string]interface{}{
						"id": "user-org-id",
					},
					"properties": map[string]interface{}{
						"cs_auth": map[string]interface{}{
							"activeOrganizationId": "active-org-id",
						},
					},
				},
			},
			expectedOrgID: "client-org-id",
			wantErr:       false,
		},
		{
			name: "extract from user organization (priority 2)",
			meResponse: map[string]interface{}{
				"user": map[string]interface{}{
					"organization": map[string]interface{}{
						"id": "user-org-id",
					},
					"properties": map[string]interface{}{
						"cs_auth": map[string]interface{}{
							"activeOrganizationId": "active-org-id",
						},
					},
				},
			},
			expectedOrgID: "user-org-id",
			wantErr:       false,
		},
		{
			name: "extract from active organization (priority 3)",
			meResponse: map[string]interface{}{
				"user": map[string]interface{}{
					"properties": map[string]interface{}{
						"cs_auth": map[string]interface{}{
							"activeOrganizationId": "active-org-id",
						},
					},
				},
			},
			expectedOrgID: "active-org-id",
			wantErr:       false,
		},
		{
			name: "no organization ID found",
			meResponse: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "test-user",
				},
			},
			expectedOrgID: "",
			wantErr:       true,
		},
		{
			name:          "empty response",
			meResponse:    map[string]interface{}{},
			expectedOrgID: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgID, err := client.extractOrgID(tt.meResponse)

			if tt.wantErr {
				if err == nil {
					t.Errorf("extractOrgID() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("extractOrgID() unexpected error = %v", err)
				}
				if orgID != tt.expectedOrgID {
					t.Errorf("extractOrgID() = %v, want %v", orgID, tt.expectedOrgID)
				}
			}
		})
	}
}

func TestAnypointClient_ConfigDefaults(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
	}

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	config.BaseURL = server.URL

	client, err := NewAnypointClient(config)
	if err != nil {
		t.Fatalf("NewAnypointClient() error = %v", err)
	}

	// Test default BaseURL (should be overridden by our test)
	if !strings.Contains(client.BaseURL, server.URL) {
		t.Errorf("NewAnypointClient() BaseURL = %v, want %v", client.BaseURL, server.URL)
	}

	// Test default timeout
	expectedTimeout := 10 * time.Minute
	if client.HTTPClient.Timeout != expectedTimeout {
		t.Errorf("NewAnypointClient() default timeout = %v, want %v", client.HTTPClient.Timeout, expectedTimeout)
	}
}

func TestAnypointClient_AuthenticationFlow(t *testing.T) {
	authCalled := false
	meCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
		},
		"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
			meCalled = true
			testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
		},
	}

	server := testutil.MockHTTPServer(t, handlers)

	config := &Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	client, err := NewAnypointClient(config)
	if err != nil {
		t.Fatalf("NewAnypointClient() error = %v", err)
	}

	// Verify both authentication endpoints were called
	if !authCalled {
		t.Error("Authentication endpoint was not called")
	}
	if !meCalled {
		t.Error("Me endpoint was not called")
	}

	// Verify client is properly configured
	if client.Token != "mock-access-token" {
		t.Errorf("Client token = %v, want %v", client.Token, "mock-access-token")
	}
	if client.OrgID != "test-org-id" {
		t.Errorf("Client OrgID = %v, want %v", client.OrgID, "test-org-id")
	}
}
