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

func TestNewPrivateNetworkClient(t *testing.T) {
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
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.Config{
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

			client, err := NewPrivateNetworkClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPrivateNetworkClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPrivateNetworkClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPrivateNetworkClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewPrivateNetworkClient() returned nil client")
				}
			}
		})
	}
}

func TestPrivateNetworkClient_CreatePrivateNetwork(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:     "test-space-id",
		Name:   "Test Space",
		Region: "us-east-1",
		Status: "ACTIVE",
		Network: NetworkConfig{
			CidrBlock:     "10.0.0.0/16",
			DNSTarget:     "test.dns.target",
			ReservedCIDRs: []string{"10.0.1.0/24"},
		},
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *CreatePrivateNetworkRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		expectedSpace  *PrivateSpace
	}{
		{
			name:           "successful creation",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &CreatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:        "us-east-1",
					CidrBlock:     "10.0.0.0/16",
					ReservedCIDRs: []string{"10.0.1.0/24"},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PATCH", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")

				body := testutil.AssertJSONBody(t, r, "network")

				network, ok := body["network"].(map[string]interface{})
				if !ok {
					t.Error("Expected network object in request body")
					return
				}

				if network["region"] != "us-east-1" {
					t.Errorf("Expected region 'us-east-1', got %v", network["region"])
				}
				if network["cidrBlock"] != "10.0.0.0/16" {
					t.Errorf("Expected cidrBlock '10.0.0.0/16', got %v", network["cidrBlock"])
				}

				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr:       false,
			expectedSpace: mockPrivateSpace,
		},
		{
			name:           "private space not found",
			orgID:          "test-org-id",
			privateSpaceID: "nonexistent-space-id",
			request: &CreatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:    "us-east-1",
					CidrBlock: "10.0.0.0/16",
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
		{
			name:           "invalid CIDR block",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &CreatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:    "us-east-1",
					CidrBlock: "invalid-cidr",
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid CIDR block")
			},
			wantErr:     true,
			errContains: "failed to create private network with status 400",
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &CreatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:    "us-east-1",
					CidrBlock: "10.0.0.0/16",
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create private network with status 500",
		},
		{
			name:           "malformed response",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &CreatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:    "us-east-1",
					CidrBlock: "10.0.0.0/16",
				},
			},
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
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateNetworkClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			privateSpace, err := client.CreatePrivateNetwork(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreatePrivateNetwork() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreatePrivateNetwork() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreatePrivateNetwork() unexpected error = %v", err)
				}
				if privateSpace == nil {
					t.Errorf("CreatePrivateNetwork() returned nil private space")
				}

				// Validate returned private space
				if privateSpace != nil && tt.expectedSpace != nil {
					if privateSpace.ID != tt.expectedSpace.ID {
						t.Errorf("CreatePrivateNetwork() ID = %v, want %v", privateSpace.ID, tt.expectedSpace.ID)
					}
					if privateSpace.Network.CidrBlock != tt.expectedSpace.Network.CidrBlock {
						t.Errorf("CreatePrivateNetwork() Network.CidrBlock = %v, want %v",
							privateSpace.Network.CidrBlock, tt.expectedSpace.Network.CidrBlock)
					}
					if len(privateSpace.Network.ReservedCIDRs) != len(tt.expectedSpace.Network.ReservedCIDRs) {
						t.Errorf("CreatePrivateNetwork() Network.ReservedCIDRs length = %v, want %v",
							len(privateSpace.Network.ReservedCIDRs), len(tt.expectedSpace.Network.ReservedCIDRs))
					}
				}
			}
		})
	}
}

func TestPrivateNetworkClient_UpdatePrivateNetwork(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:     "test-space-id",
		Name:   "Test Space",
		Region: "us-east-1",
		Status: "ACTIVE",
		Network: NetworkConfig{
			CidrBlock:     "10.0.0.0/16",
			DNSTarget:     "updated.dns.target",
			ReservedCIDRs: []string{"10.0.1.0/24", "10.0.2.0/24"},
		},
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *UpdatePrivateNetworkRequest
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful update",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &UpdatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:        "us-east-1",
					CidrBlock:     "10.0.0.0/16",
					ReservedCIDRs: []string{"10.0.1.0/24", "10.0.2.0/24"},
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PATCH", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr: false,
		},
		{
			name:           "private space not found",
			orgID:          "test-org-id",
			privateSpaceID: "nonexistent-space-id",
			request: &UpdatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:    "us-east-1",
					CidrBlock: "10.0.0.0/16",
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &UpdatePrivateNetworkRequest{
				Network: NetworkConfiguration{
					Region:    "us-east-1",
					CidrBlock: "10.0.0.0/16",
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to update private network with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateNetworkClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			privateSpace, err := client.UpdatePrivateNetwork(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdatePrivateNetwork() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdatePrivateNetwork() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePrivateNetwork() unexpected error = %v", err)
				}
				if privateSpace == nil {
					t.Errorf("UpdatePrivateNetwork() returned nil private space")
				}
			}
		})
	}
}

func TestPrivateNetworkClient_GetPrivateNetwork(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:     "test-space-id",
		Name:   "Test Space",
		Region: "us-east-1",
		Status: "ACTIVE",
		Network: NetworkConfig{
			CidrBlock:         "10.0.0.0/16",
			DNSTarget:         "test.dns.target",
			ReservedCIDRs:     []string{"10.0.1.0/24"},
			InboundStaticIPs:  []string{"10.0.0.10"},
			OutboundStaticIPs: []string{"10.0.0.20"},
		},
	}

	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		expectedSpace  *PrivateSpace
	}{
		{
			name:           "successful get",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr:       false,
			expectedSpace: mockPrivateSpace,
		},
		{
			name:           "private space not found",
			orgID:          "test-org-id",
			privateSpaceID: "nonexistent-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get private network with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/%s/privatespaces/%s", tt.orgID, tt.privateSpaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateNetworkClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			privateSpace, err := client.GetPrivateNetwork(context.Background(), tt.orgID, tt.privateSpaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrivateNetwork() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetPrivateNetwork() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetPrivateNetwork() unexpected error = %v", err)
				}
				if privateSpace == nil {
					t.Errorf("GetPrivateNetwork() returned nil private space")
				}

				// Validate returned private space
				if privateSpace != nil && tt.expectedSpace != nil {
					if privateSpace.ID != tt.expectedSpace.ID {
						t.Errorf("GetPrivateNetwork() ID = %v, want %v", privateSpace.ID, tt.expectedSpace.ID)
					}
					if privateSpace.Network.CidrBlock != tt.expectedSpace.Network.CidrBlock {
						t.Errorf("GetPrivateNetwork() Network.CidrBlock = %v, want %v",
							privateSpace.Network.CidrBlock, tt.expectedSpace.Network.CidrBlock)
					}
					if privateSpace.Network.DNSTarget != tt.expectedSpace.Network.DNSTarget {
						t.Errorf("GetPrivateNetwork() Network.DNSTarget = %v, want %v",
							privateSpace.Network.DNSTarget, tt.expectedSpace.Network.DNSTarget)
					}
				}
			}
		})
	}
}

func TestNetworkConfiguration_JSONSerialization(t *testing.T) {
	config := &NetworkConfiguration{
		Region:        "us-east-1",
		CidrBlock:     "10.0.0.0/16",
		ReservedCIDRs: []string{"10.0.1.0/24", "10.0.2.0/24"},
	}

	// Test marshaling
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal network configuration: %v", err)
	}

	// Test unmarshaling
	var decoded NetworkConfiguration
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal network configuration: %v", err)
	}

	// Validate key fields
	if decoded.Region != config.Region {
		t.Errorf("Unmarshaled Region = %v, want %v", decoded.Region, config.Region)
	}
	if decoded.CidrBlock != config.CidrBlock {
		t.Errorf("Unmarshaled CidrBlock = %v, want %v", decoded.CidrBlock, config.CidrBlock)
	}
	if len(decoded.ReservedCIDRs) != len(config.ReservedCIDRs) {
		t.Errorf("Unmarshaled ReservedCIDRs length = %v, want %v", len(decoded.ReservedCIDRs), len(config.ReservedCIDRs))
	}
}

func TestCreatePrivateNetworkRequest_JSONSerialization(t *testing.T) {
	req := &CreatePrivateNetworkRequest{
		Network: NetworkConfiguration{
			Region:        "us-east-1",
			CidrBlock:     "10.0.0.0/16",
			ReservedCIDRs: []string{"10.0.1.0/24"},
		},
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal create request: %v", err)
	}

	// Test unmarshaling
	var decoded CreatePrivateNetworkRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal create request: %v", err)
	}

	// Validate fields
	if decoded.Network.Region != req.Network.Region {
		t.Errorf("Unmarshaled Network.Region = %v, want %v", decoded.Network.Region, req.Network.Region)
	}
	if decoded.Network.CidrBlock != req.Network.CidrBlock {
		t.Errorf("Unmarshaled Network.CidrBlock = %v, want %v", decoded.Network.CidrBlock, req.Network.CidrBlock)
	}
}

// Test edge cases and error scenarios
func TestPrivateNetworkClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *PrivateNetworkClient
		operation   func(client *PrivateNetworkClient) error
		wantErr     bool
	}{
		{
			name: "nil request to CreatePrivateNetwork",
			setupClient: func() *PrivateNetworkClient {
				return &PrivateNetworkClient{
					AnypointClient: &client.AnypointClient{
						BaseURL:    "http://test.com",
						Token:      "mock-token",
						HTTPClient: &http.Client{},
					},
				}
			},
			operation: func(client *PrivateNetworkClient) error {
				_, err := client.CreatePrivateNetwork(context.Background(), "org-id", "space-id", nil)
				return err
			},
			wantErr: true,
		},
		{
			name: "empty organization ID",
			setupClient: func() *PrivateNetworkClient {
				return &PrivateNetworkClient{
					AnypointClient: &client.AnypointClient{
						BaseURL:    "http://test.com",
						Token:      "mock-token",
						HTTPClient: &http.Client{},
					},
				}
			},
			operation: func(client *PrivateNetworkClient) error {
				req := &CreatePrivateNetworkRequest{
					Network: NetworkConfiguration{
						Region:    "us-east-1",
						CidrBlock: "10.0.0.0/16",
					},
				}
				_, err := client.CreatePrivateNetwork(context.Background(), "", "space-id", req)
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			err := tt.operation(client)

			if tt.wantErr && err == nil {
				t.Errorf("%s expected error, got nil", tt.name)
			} else if !tt.wantErr && err != nil {
				t.Errorf("%s unexpected error = %v", tt.name, err)
			}
		})
	}
}
