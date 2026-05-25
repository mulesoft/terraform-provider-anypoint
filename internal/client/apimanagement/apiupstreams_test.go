package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAPIUpstreamsClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *client.Config
		wantErr bool
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
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
				tt.config.BaseURL = server.URL
			}
			c, err := NewAPIUpstreamsClient(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if c == nil {
					t.Error("Expected non-nil client")
				}
			}
		})
	}
}

func TestAPIUpstreamsClient_ListUpstreams(t *testing.T) {
	orgID := "org-123"
	envID := "env-456"
	apiID := "api-789"
	basePath := "/apimanager/api/v1/organizations/" + orgID + "/environments/" + envID + "/apis/" + apiID + "/upstreams"

	mockUpstreams := []APIUpstream{
		{ID: "up-1", Label: "Primary", URI: "https://primary.example.com"},
		{ID: "up-2", Label: "Secondary", URI: "https://secondary.example.com"},
	}

	tests := []struct {
		name        string
		handler     func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		wantLen     int
	}{
		{
			name: "successful list with upstreams",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"total":     2,
					"upstreams": mockUpstreams,
				})
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name: "successful list empty",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"total":     0,
					"upstreams": []APIUpstream{},
				})
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "API not found")
			},
			wantErr:     true,
			errContains: "unexpected status 404",
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
			},
			wantErr:     true,
			errContains: "unexpected status 500",
		},
		{
			name: "malformed JSON response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &APIUpstreamsClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "test-token",
					HTTPClient: &http.Client{},
				},
			}

			upstreams, err := c.ListUpstreams(context.Background(), orgID, envID, apiID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if len(upstreams) != tt.wantLen {
					t.Errorf("ListUpstreams() len = %d, want %d", len(upstreams), tt.wantLen)
				}
			}
		})
	}
}

func TestAPIUpstream_JSONSerialization(t *testing.T) {
	upstream := APIUpstream{
		ID:    "up-1",
		Label: "Primary",
		URI:   "https://primary.example.com",
	}

	data, err := json.Marshal(upstream)
	if err != nil {
		t.Fatalf("Failed to marshal APIUpstream: %v", err)
	}

	var decoded APIUpstream
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal APIUpstream: %v", err)
	}

	if decoded.ID != upstream.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, upstream.ID)
	}
	if decoded.Label != upstream.Label {
		t.Errorf("Label = %v, want %v", decoded.Label, upstream.Label)
	}
	if decoded.URI != upstream.URI {
		t.Errorf("URI = %v, want %v", decoded.URI, upstream.URI)
	}
}
