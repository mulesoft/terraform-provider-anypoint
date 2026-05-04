package client

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewUserAnypointClient(t *testing.T) {
	// Save original env vars to restore later
	originalUsername := os.Getenv("ANYPOINT_ADMIN_USERNAME")
	originalPassword := os.Getenv("ANYPOINT_ADMIN_PASSWORD")
	defer func() {
		_ = os.Setenv("ANYPOINT_ADMIN_USERNAME", originalUsername)
		_ = os.Setenv("ANYPOINT_ADMIN_PASSWORD", originalPassword)
	}()

	tests := []struct {
		name        string
		config      *UserClientConfig
		envVars     map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with explicit credentials",
			config: &UserClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr: false,
		},
		{
			name: "valid config with env var credentials",
			config: &UserClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				// Username and Password empty - should come from env vars
			},
			envVars: map[string]string{
				"ANYPOINT_ADMIN_USERNAME": "env-user",
				"ANYPOINT_ADMIN_PASSWORD": "env-password",
			},
			wantErr: false,
		},
		{
			name: "custom baseURL and timeout",
			config: &UserClientConfig{
				BaseURL:      "https://custom.anypoint.mulesoft.com",
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
				Timeout:      60,
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &UserClientConfig{
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &UserClientConfig{
				ClientID: "test-client-id",
				Username: "test-user",
				Password: "test-password",
			},
			wantErr:     true,
			errContains: "client_secret is required",
		},
		{
			name: "missing username - no env var",
			config: &UserClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Password:     "test-password",
			},
			envVars: map[string]string{
				"ANYPOINT_ADMIN_USERNAME": "",
			},
			wantErr:     true,
			errContains: "username is required",
		},
		{
			name: "missing password - no env var",
			config: &UserClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
			},
			envVars: map[string]string{
				"ANYPOINT_ADMIN_PASSWORD": "",
			},
			wantErr:     true,
			errContains: "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			if tt.envVars != nil {
				for key, value := range tt.envVars {
					_ = os.Setenv(key, value)
				}
				defer func() {
					for key := range tt.envVars {
						_ = os.Unsetenv(key)
					}
				}()
			}

			// Create mock server for authentication
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
					testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
				},
				"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
					testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			// Override baseURL to use mock server if not testing custom baseURL
			if tt.config != nil && tt.config.BaseURL == "" {
				tt.config.BaseURL = server.URL
			} else if tt.config != nil && tt.config.BaseURL != "" {
				// For custom baseURL tests, create new config with mock server
				testConfig := *tt.config
				testConfig.BaseURL = server.URL
				tt.config = &testConfig
			}

			client, err := NewUserAnypointClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewUserAnypointClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewUserAnypointClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewUserAnypointClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewUserAnypointClient() returned nil client")
				}

				// Validate client configuration
				if client != nil {
					if client.ClientID != tt.config.ClientID {
						t.Errorf("NewUserAnypointClient() ClientID = %v, want %v", client.ClientID, tt.config.ClientID)
					}
					if client.ClientSecret != tt.config.ClientSecret {
						t.Errorf("NewUserAnypointClient() ClientSecret = %v, want %v", client.ClientSecret, tt.config.ClientSecret)
					}
					if client.Token == "" {
						t.Errorf("NewUserAnypointClient() Token should be set after authentication")
					}
					if client.OrgID == "" {
						t.Errorf("NewUserAnypointClient() OrgID should be set after authentication")
					}

					// Test timeout setting
					expectedTimeout := 10 * time.Minute
					if tt.config.Timeout > 0 {
						expectedTimeout = time.Duration(tt.config.Timeout) * time.Second
					}
					if client.HTTPClient.Timeout != expectedTimeout {
						t.Errorf("NewUserAnypointClient() HTTPClient.Timeout = %v, want %v",
							client.HTTPClient.Timeout, expectedTimeout)
					}

					// Validate username from config or env var
					expectedUsername := tt.config.Username
					if expectedUsername == "" && tt.envVars != nil {
						expectedUsername = tt.envVars["ANYPOINT_ADMIN_USERNAME"]
					}
					if client.Username != expectedUsername {
						t.Errorf("NewUserAnypointClient() Username = %v, want %v", client.Username, expectedUsername)
					}
				}
			}
		})
	}
}

func TestUserAnypointClient_authenticate(t *testing.T) {
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

				body := testutil.AssertJSONBody(t, r, "grant_type", "client_id", "client_secret", "username", "password")

				if body["grant_type"] != "password" {
					t.Errorf("Expected grant_type 'password', got %v", body["grant_type"])
				}
				if body["username"] != "test-user" {
					t.Errorf("Expected username 'test-user', got %v", body["username"])
				}

				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			wantErr: false,
		},
		{
			name: "authentication failure - invalid credentials",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusUnauthorized, "Invalid username or password")
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
			name: "malformed response",
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
				"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
					testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
				},
			}

			server := testutil.MockHTTPServer(t, handlers)

			client := &UserAnypointClient{
				BaseURL:      server.URL,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
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

func TestUserAnypointClient_getMe(t *testing.T) {
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

			client := &UserAnypointClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
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

func TestUserAnypointClient_extractOrgID(t *testing.T) {
	client := &UserAnypointClient{}

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

func TestUserAnypointClient_ConfigDefaults(t *testing.T) {
	config := &UserClientConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
		},
		"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
		},
	})
	config.BaseURL = server.URL

	client, err := NewUserAnypointClient(config)
	if err != nil {
		t.Fatalf("NewUserAnypointClient() error = %v", err)
	}

	// Test default timeout
	expectedTimeout := 10 * time.Minute
	if client.HTTPClient.Timeout != expectedTimeout {
		t.Errorf("NewUserAnypointClient() default timeout = %v, want %v", client.HTTPClient.Timeout, expectedTimeout)
	}
}

func TestUserAnypointClient_AuthenticationFlow(t *testing.T) {
	authCalled := false
	meCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			authCalled = true

			// Verify it's using password grant
			body := make(map[string]interface{})
			_ = json.NewDecoder(r.Body).Decode(&body)

			if body["grant_type"] != "password" {
				t.Errorf("Expected password grant, got %v", body["grant_type"])
			}

			testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
		},
		"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
			meCalled = true
			testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
		},
	}

	server := testutil.MockHTTPServer(t, handlers)

	config := &UserClientConfig{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	client, err := NewUserAnypointClient(config)
	if err != nil {
		t.Fatalf("NewUserAnypointClient() error = %v", err)
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
	if client.Username != "test-user" {
		t.Errorf("Client Username = %v, want %v", client.Username, "test-user")
	}
}

func TestUserAnypointClient_EnvironmentVariableHandling(t *testing.T) {
	// Save original env vars
	originalUsername := os.Getenv("ANYPOINT_ADMIN_USERNAME")
	originalPassword := os.Getenv("ANYPOINT_ADMIN_PASSWORD")
	defer func() {
		_ = os.Setenv("ANYPOINT_ADMIN_USERNAME", originalUsername)
		_ = os.Setenv("ANYPOINT_ADMIN_PASSWORD", originalPassword)
	}()

	// Set test env vars
	_ = os.Setenv("ANYPOINT_ADMIN_USERNAME", "env-user")
	_ = os.Setenv("ANYPOINT_ADMIN_PASSWORD", "env-password")

	config := &UserClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		// Username and Password intentionally empty to test env var fallback
	}

	server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			// Verify username from env var is used
			body := make(map[string]interface{})
			_ = json.NewDecoder(r.Body).Decode(&body)

			if body["username"] != "env-user" {
				t.Errorf("Expected username from env var 'env-user', got %v", body["username"])
			}
			if body["password"] != "env-password" {
				t.Errorf("Expected password from env var 'env-password', got %v", body["password"])
			}

			testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
		},
		"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
		},
	})
	config.BaseURL = server.URL

	client, err := NewUserAnypointClient(config)
	if err != nil {
		t.Fatalf("NewUserAnypointClient() error = %v", err)
	}

	// Verify client uses environment variables
	if client.Username != "env-user" {
		t.Errorf("Client Username = %v, want %v", client.Username, "env-user")
	}
	if client.Password != "env-password" {
		t.Errorf("Client Password = %v, want %v", client.Password, "env-password")
	}
}
