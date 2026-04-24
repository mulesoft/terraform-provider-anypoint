package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// MockHTTPServer creates a mock HTTP server for testing
func MockHTTPServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

// CreateTestClientConfig creates a test client config for mock testing
func CreateTestClientConfig(t *testing.T, baseURL string) map[string]interface{} {
	t.Helper()

	return map[string]interface{}{
		"base_url":      baseURL,
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
		"timeout":       30,
		"token":         "mock-token",
		"org_id":        "test-org-id",
	}
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}

func IntPtr(i int) *int {
	return &i
}

// JSONResponse creates a JSON response for mock handlers
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// ErrorResponse creates an error response for mock handlers
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	JSONResponse(w, statusCode, map[string]string{
		"error":  message,
		"status": fmt.Sprintf("%d", statusCode),
	})
}

// AssertHTTPRequest validates HTTP request method and path with Bearer token
func AssertHTTPRequest(t *testing.T, r *http.Request, expectedMethod, expectedPath string) {
	t.Helper()
	AssertHTTPRequestWithAuth(t, r, expectedMethod, expectedPath, true)
}

// AssertHTTPRequestWithAuth validates HTTP request with optional auth check
func AssertHTTPRequestWithAuth(t *testing.T, r *http.Request, expectedMethod, expectedPath string, requireAuth bool) {
	t.Helper()

	if r.Method != expectedMethod {
		t.Errorf("Expected method %s, got %s", expectedMethod, r.Method)
	}

	if r.URL.Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
	}

	if requireAuth {
		// Verify authentication header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Authorization header with Bearer token, got %s", authHeader)
		}
	}
}

// AssertJSONBody validates request body is valid JSON and matches expected structure
func AssertJSONBody(t *testing.T, r *http.Request, expectedKeys ...string) map[string]interface{} {
	t.Helper()

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode JSON body: %v", err)
	}

	for _, key := range expectedKeys {
		if _, exists := body[key]; !exists {
			t.Errorf("Expected key '%s' in request body", key)
		}
	}

	return body
}

// MockAuthResponse creates a mock authentication response
func MockAuthResponse() map[string]interface{} {
	return map[string]interface{}{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
}

// MockMeResponse creates a mock user info response
func MockMeResponse() map[string]interface{} {
	return map[string]interface{}{
		"user": map[string]interface{}{
			"id":       "test-user-id",
			"username": "test-user",
			"organization": map[string]interface{}{
				"id":   "test-org-id",
				"name": "Test Organization",
			},
		},
		"client": map[string]interface{}{
			"org_id": "test-org-id",
		},
	}
}

// StandardMockHandlers returns common mock handlers for authentication
func StandardMockHandlers() map[string]func(w http.ResponseWriter, r *http.Request) {
	return map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			JSONResponse(w, http.StatusOK, MockAuthResponse())
		},
		"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}
			JSONResponse(w, http.StatusOK, MockMeResponse())
		},
	}
}

// GetTestEnvVar retrieves an environment variable for testing with a fallback default value
func GetTestEnvVar(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
