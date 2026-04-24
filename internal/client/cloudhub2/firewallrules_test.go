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

func TestNewFirewallRulesClient(t *testing.T) {
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

			client, err := NewFirewallRulesClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewFirewallRulesClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewFirewallRulesClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewFirewallRulesClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewFirewallRulesClient() returned nil client")
				}
			}
		})
	}
}

func TestFirewallRulesClient_UpdateFirewallRules(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:   "test-space-id",
		Name: "Test Private Space",
		FirewallRules: []FirewallRule{
			{
				CidrBlock: "10.0.0.0/16",
				Protocol:  "TCP",
				FromPort:  80,
				ToPort:    80,
				Type:      "INBOUND",
			},
		},
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *UpdateFirewallRulesRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		expectedResult *PrivateSpace
	}{
		{
			name:           "successful update",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &UpdateFirewallRulesRequest{
				ManagedFirewallRules: []FirewallRule{
					{
						CidrBlock: "10.0.0.0/16",
						Protocol:  "TCP",
						FromPort:  80,
						ToPort:    80,
						Type:      "INBOUND",
					},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PATCH", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")
				
				body := testutil.AssertJSONBody(t, r, "managedFirewallRules")
				rules, ok := body["managedFirewallRules"].([]interface{})
				if !ok || len(rules) == 0 {
					t.Error("Expected managedFirewallRules array in request body")
				}
				
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr:        false,
			expectedResult: mockPrivateSpace,
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &UpdateFirewallRulesRequest{
				ManagedFirewallRules: []FirewallRule{
					{
						CidrBlock: "10.0.0.0/16",
						Protocol:  "TCP",
						FromPort:  80,
						ToPort:    80,
						Type:      "INBOUND",
					},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to update firewall rules with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &FirewallRulesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdateFirewallRules(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateFirewallRules() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateFirewallRules() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateFirewallRules() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdateFirewallRules() returned nil private space")
				}
				
				// Validate returned private space
				if result != nil && tt.expectedResult != nil {
					if result.ID != tt.expectedResult.ID {
						t.Errorf("UpdateFirewallRules() ID = %v, want %v", result.ID, tt.expectedResult.ID)
					}
					if result.Name != tt.expectedResult.Name {
						t.Errorf("UpdateFirewallRules() Name = %v, want %v", result.Name, tt.expectedResult.Name)
					}
				}
			}
		})
	}
}

// JSON serialization test
func TestUpdateFirewallRulesRequest_JSONSerialization(t *testing.T) {
	req := &UpdateFirewallRulesRequest{
		ManagedFirewallRules: []FirewallRule{
			{
				CidrBlock: "10.0.0.0/16",
				Protocol:  "TCP",
				FromPort:  80,
				ToPort:    80,
				Type:      "INBOUND",
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal UpdateFirewallRulesRequest: %v", err)
	}

	// Test unmarshaling
	var decoded UpdateFirewallRulesRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal UpdateFirewallRulesRequest: %v", err)
	}

	// Validate key fields
	if len(decoded.ManagedFirewallRules) != len(req.ManagedFirewallRules) {
		t.Errorf("Unmarshaled ManagedFirewallRules length = %v, want %v", len(decoded.ManagedFirewallRules), len(req.ManagedFirewallRules))
	}
	if len(decoded.ManagedFirewallRules) > 0 {
		if decoded.ManagedFirewallRules[0].CidrBlock != req.ManagedFirewallRules[0].CidrBlock {
			t.Errorf("Unmarshaled CidrBlock = %v, want %v", decoded.ManagedFirewallRules[0].CidrBlock, req.ManagedFirewallRules[0].CidrBlock)
		}
		if decoded.ManagedFirewallRules[0].Protocol != req.ManagedFirewallRules[0].Protocol {
			t.Errorf("Unmarshaled Protocol = %v, want %v", decoded.ManagedFirewallRules[0].Protocol, req.ManagedFirewallRules[0].Protocol)
		}
	}
}