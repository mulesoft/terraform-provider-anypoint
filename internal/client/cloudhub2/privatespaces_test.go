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

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }

func TestNewPrivateSpacesClient(t *testing.T) {
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
			// Create mock server
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

			if tt.config != nil {
				tt.config.BaseURL = server.URL
			}

			client, err := NewPrivateSpacesClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewPrivateSpacesClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPrivateSpacesClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPrivateSpacesClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewPrivateSpacesClient() returned nil client")
				}
			}
		})
	}
}

func TestPrivateSpacesClient_CreatePrivateSpace(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:                 "test-space-id",
		Name:               "Test Space",
		Version:            "1.0",
		Status:             "ACTIVE",
		Region:             "us-east-1",
		OrganizationID:     "test-org-id",
		RootOrganizationID: "root-org-id",
		Provisioning: PrivateSpaceProvisioning{
			Status:  "SUCCESS",
			Message: "Private space created successfully",
		},
		EnableEgress:           true,
		EnableNetworkIsolation: false,
	}

	tests := []struct {
		name        string
		request     *CreatePrivateSpaceRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		expectedPS  *PrivateSpace
	}{
		{
			name: "successful creation",
			request: &CreatePrivateSpaceRequest{
				Name:         "Test Space",
				Region:       "us-east-1",
				EnableEgress: boolPtr(true),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/runtimefabric/api/organizations/test-org-id/privatespaces")

				body := testutil.AssertJSONBody(t, r, "name", "region")

				if body["name"] != "Test Space" {
					t.Errorf("Expected name 'Test Space', got %v", body["name"])
				}
				if body["region"] != "us-east-1" {
					t.Errorf("Expected region 'us-east-1', got %v", body["region"])
				}

				testutil.JSONResponse(w, http.StatusCreated, mockPrivateSpace)
			},
			wantErr:    false,
			expectedPS: mockPrivateSpace,
		},
		{
			name: "server returns error",
			request: &CreatePrivateSpaceRequest{
				Name:   "Test Space",
				Region: "us-east-1",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid region")
			},
			wantErr:     true,
			errContains: "failed to create private space with status 400",
		},
		{
			name: "malformed response",
			request: &CreatePrivateSpaceRequest{
				Name:   "Test Space",
				Region: "us-east-1",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
		{
			name:    "nil request",
			request: nil,
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				t.Error("Handler should not be called for nil request")
			},
			wantErr:     true,
			errContains: "failed to marshal private space data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/runtimefabric/api/organizations/test-org-id/privatespaces": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			ps, err := client.CreatePrivateSpace(context.Background(), "test-org-id", tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreatePrivateSpace() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreatePrivateSpace() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreatePrivateSpace() unexpected error = %v", err)
				}
				if ps == nil {
					t.Errorf("CreatePrivateSpace() returned nil private space")
				}

				// Validate returned private space
				if ps != nil && tt.expectedPS != nil {
					if ps.ID != tt.expectedPS.ID {
						t.Errorf("CreatePrivateSpace() ID = %v, want %v", ps.ID, tt.expectedPS.ID)
					}
					if ps.Name != tt.expectedPS.Name {
						t.Errorf("CreatePrivateSpace() Name = %v, want %v", ps.Name, tt.expectedPS.Name)
					}
					if ps.Region != tt.expectedPS.Region {
						t.Errorf("CreatePrivateSpace() Region = %v, want %v", ps.Region, tt.expectedPS.Region)
					}
					if ps.Status != tt.expectedPS.Status {
						t.Errorf("CreatePrivateSpace() Status = %v, want %v", ps.Status, tt.expectedPS.Status)
					}
				}
			}
		})
	}
}

func TestPrivateSpacesClient_GetPrivateSpace(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:                 "test-space-id",
		Name:               "Test Space",
		Version:            "1.0",
		Status:             "ACTIVE",
		Region:             "us-east-1",
		OrganizationID:     "test-org-id",
		RootOrganizationID: "root-org-id",
		Network: NetworkConfig{
			CidrBlock: "10.0.0.0/16",
			DNSTarget: "test.dns.target",
		},
		FirewallRules: []FirewallRule{
			{
				CidrBlock: "0.0.0.0/0",
				Protocol:  "TCP",
				FromPort:  80,
				ToPort:    80,
				Type:      "INBOUND",
			},
		},
	}

	tests := []struct {
		name        string
		spaceID     string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		expectedPS  *PrivateSpace
	}{
		{
			name:    "successful get",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr:    false,
			expectedPS: mockPrivateSpace,
		},
		{
			name:    "private space not found",
			spaceID: "nonexistent-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/runtimefabric/api/organizations/test-org-id/privatespaces/nonexistent-space-id")
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
		{
			name:    "server error",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get private space with status 500",
		},
		{
			name:    "malformed response",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/test-org-id/privatespaces/%s", tt.spaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			ps, err := client.GetPrivateSpace(context.Background(), "test-org-id", tt.spaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPrivateSpace() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetPrivateSpace() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetPrivateSpace() unexpected error = %v", err)
				}
				if ps == nil {
					t.Errorf("GetPrivateSpace() returned nil private space")
				}

				if ps != nil && tt.expectedPS != nil {
					if ps.ID != tt.expectedPS.ID {
						t.Errorf("GetPrivateSpace() ID = %v, want %v", ps.ID, tt.expectedPS.ID)
					}
					if ps.Name != tt.expectedPS.Name {
						t.Errorf("GetPrivateSpace() Name = %v, want %v", ps.Name, tt.expectedPS.Name)
					}
					if ps.Region != tt.expectedPS.Region {
						t.Errorf("GetPrivateSpace() Region = %v, want %v", ps.Region, tt.expectedPS.Region)
					}
					if ps.Network.CidrBlock != tt.expectedPS.Network.CidrBlock {
						t.Errorf("GetPrivateSpace() Network.CidrBlock = %v, want %v",
							ps.Network.CidrBlock, tt.expectedPS.Network.CidrBlock)
					}
					if len(ps.FirewallRules) != len(tt.expectedPS.FirewallRules) {
						t.Errorf("GetPrivateSpace() FirewallRules length = %v, want %v",
							len(ps.FirewallRules), len(tt.expectedPS.FirewallRules))
					}
				}
			}
		})
	}
}

func TestPrivateSpacesClient_UpdatePrivateSpace(t *testing.T) {
	mockPrivateSpace := &PrivateSpace{
		ID:     "test-space-id",
		Name:   "Updated Test Space",
		Status: "ACTIVE",
		Region: "us-east-1",
	}

	tests := []struct {
		name        string
		spaceID     string
		request     *UpdatePrivateSpaceRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful update",
			spaceID: "test-space-id",
			request: &UpdatePrivateSpaceRequest{
				Name:         stringPtr("Updated Test Space"),
				EnableEgress: boolPtr(true),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")

				body := testutil.AssertJSONBody(t, r, "name")
				if body["name"] != "Updated Test Space" {
					t.Errorf("Expected name 'Updated Test Space', got %v", body["name"])
				}

				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr: false,
		},
		{
			name:    "private space not found",
			spaceID: "nonexistent-space-id",
			request: &UpdatePrivateSpaceRequest{
				Name: stringPtr("Updated Test Space"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
		{
			name:    "server error",
			spaceID: "test-space-id",
			request: &UpdatePrivateSpaceRequest{
				Name: stringPtr("Updated Test Space"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to update private space with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/test-org-id/privatespaces/%s", tt.spaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			ps, err := client.UpdatePrivateSpace(context.Background(), "test-org-id", tt.spaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdatePrivateSpace() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdatePrivateSpace() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePrivateSpace() unexpected error = %v", err)
				}
				if ps == nil {
					t.Errorf("UpdatePrivateSpace() returned nil private space")
				}
			}
		})
	}
}

func TestPrivateSpacesClient_DeletePrivateSpace(t *testing.T) {
	tests := []struct {
		name        string
		spaceID     string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful deletion",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/runtimefabric/api/organizations/test-org-id/privatespaces/test-space-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:    "private space not found",
			spaceID: "nonexistent-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "failed to delete private space with status 404",
		},
		{
			name:    "server error",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete private space with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/runtimefabric/api/organizations/test-org-id/privatespaces/%s", tt.spaceID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeletePrivateSpace(context.Background(), "test-org-id", tt.spaceID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeletePrivateSpace() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeletePrivateSpace() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeletePrivateSpace() unexpected error = %v", err)
				}
			}
		})
	}
}

// Test data structures marshaling/unmarshaling
func TestPrivateSpace_JSONSerialization(t *testing.T) {
	ps := &PrivateSpace{
		ID:     "test-space-id",
		Name:   "Test Space",
		Status: "ACTIVE",
		Region: "us-east-1",
		Network: NetworkConfig{
			CidrBlock: "10.0.0.0/16",
			DNSTarget: "test.dns.target",
		},
		FirewallRules: []FirewallRule{
			{
				CidrBlock: "0.0.0.0/0",
				Protocol:  "TCP",
				FromPort:  80,
				ToPort:    80,
				Type:      "INBOUND",
			},
		},
		Provisioning: PrivateSpaceProvisioning{
			Status:  "SUCCESS",
			Message: "Successfully provisioned",
		},
	}

	// Test marshaling
	data, err := json.Marshal(ps)
	if err != nil {
		t.Fatalf("Failed to marshal private space: %v", err)
	}

	// Test unmarshaling
	var decoded PrivateSpace
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal private space: %v", err)
	}

	// Validate key fields
	if decoded.ID != ps.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, ps.ID)
	}
	if decoded.Name != ps.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", decoded.Name, ps.Name)
	}
	if decoded.Network.CidrBlock != ps.Network.CidrBlock {
		t.Errorf("Unmarshaled Network.CidrBlock = %v, want %v", decoded.Network.CidrBlock, ps.Network.CidrBlock)
	}
	if len(decoded.FirewallRules) != len(ps.FirewallRules) {
		t.Errorf("Unmarshaled FirewallRules length = %v, want %v", len(decoded.FirewallRules), len(ps.FirewallRules))
	}
	if decoded.Provisioning.Status != ps.Provisioning.Status {
		t.Errorf("Unmarshaled Provisioning.Status = %v, want %v", decoded.Provisioning.Status, ps.Provisioning.Status)
	}
}
