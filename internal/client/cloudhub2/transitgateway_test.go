package cloudhub2

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTransitGatewayClient(t *testing.T) {
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

			client, err := NewTransitGatewayClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTransitGatewayClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewTransitGatewayClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewTransitGatewayClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewTransitGatewayClient() returned nil client")
				}
			}
		})
	}
}

func TestTransitGatewayClient_CreateTransitGateway(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		privateSpaceID string
		request        *CreateTransitGatewayRequest
		wantErr        bool
		expectedName   string
		expectedID     string
	}{
		{
			name:           "successful creation returns hardcoded response",
			orgID:          "test-org-id",
			privateSpaceID: "test-space-id",
			request: &CreateTransitGatewayRequest{
				Name:                 "Test Transit Gateway",
				ResourceShareID:      "test-resource-share-id",
				ResourceShareAccount: "123456789",
				Routes:               []string{"10.0.0.0/16"},
			},
			wantErr:      false,
			expectedName: "Test Transit Gateway",
			expectedID:   "83d77850-04ee-4368-8122-192f760913de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &TransitGatewayClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    "http://unused",
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreateTransitGateway(context.Background(), tt.orgID, tt.privateSpaceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateTransitGateway() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreateTransitGateway() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreateTransitGateway() returned nil transit gateway")
				}

				if result != nil {
					if result.ID != tt.expectedID {
						t.Errorf("CreateTransitGateway() ID = %v, want %v", result.ID, tt.expectedID)
					}
					if result.Name != tt.expectedName {
						t.Errorf("CreateTransitGateway() Name = %v, want %v", result.Name, tt.expectedName)
					}
				}
			}
		})
	}
}

func TestTransitGateway_JSONSerialization(t *testing.T) {
	tgw := &TransitGateway{
		ID:   "test-tgw-id",
		Name: "Test Transit Gateway",
		Spec: TransitGatewaySpec{
			ResourceShare: ResourceShare{
				ID:      "test-resource-share-id",
				Account: "123456789",
			},
			Region:    "us-east-1",
			SpaceName: "test-space",
		},
		Status: TransitGatewayStatus{
			Gateway:     "attached",
			Attachment:  "available",
			TGWResource: "test-tgw-resource",
			Routes:      []string{"10.0.0.0/16"},
		},
	}

	// Test marshaling
	data, err := json.Marshal(tgw)
	if err != nil {
		t.Fatalf("Failed to marshal TransitGateway: %v", err)
	}

	// Test unmarshaling
	var decoded TransitGateway
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TransitGateway: %v", err)
	}

	// Validate key fields
	if decoded.ID != tgw.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, tgw.ID)
	}
	if decoded.Name != tgw.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", decoded.Name, tgw.Name)
	}
	if decoded.Spec.Region != tgw.Spec.Region {
		t.Errorf("Unmarshaled Spec.Region = %v, want %v", decoded.Spec.Region, tgw.Spec.Region)
	}
}
