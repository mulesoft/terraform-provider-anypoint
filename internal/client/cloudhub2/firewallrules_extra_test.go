package cloudhub2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestFirewallRulesClient_GetFirewallRules(t *testing.T) {
	mockPS := &PrivateSpace{
		ID:   "ps-1",
		Name: "test-space",
		FirewallRules: []FirewallRule{
			{CidrBlock: "10.0.0.0/16", Protocol: "TCP", FromPort: 443, ToPort: 443, Type: "INBOUND"},
		},
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		handler        func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful get",
			orgID:          "org-1",
			privateSpaceID: "ps-1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockPS)
			},
			wantErr: false,
		},
		{
			name:           "not found returns NotFoundError",
			orgID:          "org-1",
			privateSpaceID: "missing-ps",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
		{
			name:           "server error",
			orgID:          "org-1",
			privateSpaceID: "ps-1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
			},
			wantErr:     true,
			errContains: "failed to get firewall rules with status 500",
		},
		{
			name:           "malformed JSON response",
			orgID:          "org-1",
			privateSpaceID: "ps-1",
			handler: func(w http.ResponseWriter, r *http.Request) {
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &FirewallRulesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := c.GetFirewallRules(context.Background(), tt.orgID, tt.privateSpaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFirewallRules() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetFirewallRules() error = %v, want containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetFirewallRules() unexpected error = %v", err)
				}
				if result == nil {
					t.Fatal("GetFirewallRules() returned nil")
				}
				if result.ID != mockPS.ID {
					t.Errorf("GetFirewallRules() ID = %v, want %v", result.ID, mockPS.ID)
				}
			}
		})
	}
}
