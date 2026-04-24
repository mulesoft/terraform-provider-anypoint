package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewManagedFlexGatewayClient(t *testing.T) {
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
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				handlers := testutil.StandardMockHandlers()
				server := testutil.MockHTTPServer(t, handlers)
				tt.config.BaseURL = server.URL
			}

			_, err := NewManagedFlexGatewayClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewManagedFlexGatewayClient() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewManagedFlexGatewayClient() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestManagedFlexGatewayClient_CRUD(t *testing.T) {
	gwName := "test-gateway"
	gwNameUpdated := "test-gateway-updated"

	mockGateway := &ManagedFlexGateway{
		ID:             "gw-123",
		Name:           gwName,
		TargetID:       "target-abc",
		RuntimeVersion: "1.9.9",
		ReleaseChannel: "lts",
		Size:           "small",
		Status:         "running",
		Configuration: ManagedFlexGatewayConfig{
			Ingress: IngressConfig{
				PublicURL:         "https://test-gateway-hey4z8.usa-e2.stgx.cloudhub.io",
				InternalURL:       "https://test-gateway-.internal-hey4z8.usa-e2.stgx.cloudhub.io",
				ForwardSSLSession: true,
				LastMileSecurity:  true,
			},
			Properties: PropertiesConfig{
				UpstreamResponseTimeout: 15,
				ConnectionIdleTimeout:   60,
			},
			Logging: LoggingConfig{
				Level:       "info",
				ForwardLogs: true,
			},
			Tracing: TracingConfig{
				Enabled: false,
			},
		},
	}

	updatedGateway := *mockGateway
	updatedGateway.Name = gwNameUpdated

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways")
				testutil.JSONResponse(w, http.StatusCreated, mockGateway)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-123": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-123")
				testutil.JSONResponse(w, http.StatusOK, mockGateway)
			case "PUT":
				testutil.AssertHTTPRequest(t, r, "PUT", "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-123")
				testutil.JSONResponse(w, http.StatusOK, &updatedGateway)
			case "DELETE":
				testutil.AssertHTTPRequest(t, r, "DELETE", "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-123")
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	anypointClient, err := client.NewAnypointClient(&client.ClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create anypoint client: %v", err)
	}

	gwClient := &ManagedFlexGatewayClient{AnypointClient: anypointClient}

	t.Run("Create", func(t *testing.T) {
		createReq := &CreateManagedFlexGatewayRequest{
			Name:           gwName,
			TargetID:       "target-abc",
			RuntimeVersion: "1.9.9",
			ReleaseChannel: "lts",
			Size:           "small",
			Configuration: ManagedFlexGatewayConfig{
				Ingress: IngressConfig{
					PublicURL:         "https://test-gateway-hey4z8.usa-e2.stgx.cloudhub.io",
					InternalURL:       "https://test-gateway-.internal-hey4z8.usa-e2.stgx.cloudhub.io",
					ForwardSSLSession: true,
					LastMileSecurity:  true,
				},
				Properties: PropertiesConfig{UpstreamResponseTimeout: 15, ConnectionIdleTimeout: 60},
				Logging:    LoggingConfig{Level: "info", ForwardLogs: true},
				Tracing:    TracingConfig{Enabled: false},
			},
		}

		gw, err := gwClient.CreateManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", createReq)
		if err != nil {
			t.Fatalf("CreateManagedFlexGateway() unexpected error: %v", err)
		}
		if gw.ID != "gw-123" {
			t.Errorf("Expected ID gw-123, got %s", gw.ID)
		}
		if gw.Name != gwName {
			t.Errorf("Expected Name %s, got %s", gwName, gw.Name)
		}
		if gw.Configuration.Ingress.ForwardSSLSession != true {
			t.Error("Expected ForwardSSLSession true")
		}
		if gw.Configuration.Ingress.PublicURL != "https://test-gateway-hey4z8.usa-e2.stgx.cloudhub.io" {
			t.Errorf("Unexpected PublicURL: %s", gw.Configuration.Ingress.PublicURL)
		}
	})

	t.Run("Read", func(t *testing.T) {
		gw, err := gwClient.GetManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "gw-123")
		if err != nil {
			t.Fatalf("GetManagedFlexGateway() unexpected error: %v", err)
		}
		if gw.ID != "gw-123" {
			t.Errorf("Expected ID gw-123, got %s", gw.ID)
		}
		if gw.Status != "running" {
			t.Errorf("Expected Status running, got %s", gw.Status)
		}
	})

	t.Run("Update", func(t *testing.T) {
		updateReq := &UpdateManagedFlexGatewayRequest{
			Name: gwNameUpdated,
		}

		gw, err := gwClient.UpdateManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "gw-123", updateReq)
		if err != nil {
			t.Fatalf("UpdateManagedFlexGateway() unexpected error: %v", err)
		}
		if gw.Name != gwNameUpdated {
			t.Errorf("Expected Name %s, got %s", gwNameUpdated, gw.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := gwClient.DeleteManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "gw-123")
		if err != nil {
			t.Fatalf("DeleteManagedFlexGateway() unexpected error: %v", err)
		}
	})
}

func TestManagedFlexGatewayClient_GetDomains(t *testing.T) {
	mockDomains := &DomainsResponse{
		Domains:     []string{"*.hey4z8.usa-e2.stgx.cloudhub.io"},
		AppUniqueID: "60fef",
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/runtimefabric/api/organizations/test-org-id/targets/target-abc/environments/test-env-id/domains": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockDomains)
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	anypointClient, err := client.NewAnypointClient(&client.ClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	gwClient := &ManagedFlexGatewayClient{AnypointClient: anypointClient}

	resp, err := gwClient.GetDomains(context.Background(), "test-org-id", "target-abc", "test-env-id")
	if err != nil {
		t.Fatalf("GetDomains() unexpected error: %v", err)
	}

	if len(resp.Domains) != 1 {
		t.Fatalf("Expected 1 domain, got %d", len(resp.Domains))
	}
	if resp.Domains[0] != "*.hey4z8.usa-e2.stgx.cloudhub.io" {
		t.Errorf("Unexpected domain: %s", resp.Domains[0])
	}
	if resp.AppUniqueID != "60fef" {
		t.Errorf("Expected AppUniqueID 60fef, got %s", resp.AppUniqueID)
	}
}

func TestBuildIngressURLs(t *testing.T) {
	tests := []struct {
		name           string
		gwName         string
		domains        []string
		wantPublicURLs []string
		wantInternal   string
	}{
		{
			name:           "single domain",
			gwName:         "my-gw",
			domains:        []string{"*.hey4z8.usa-e2.stgx.cloudhub.io"},
			wantPublicURLs: []string{"https://my-gw.hey4z8.usa-e2.stgx.cloudhub.io"},
			wantInternal:   "https://my-gw.internal-hey4z8.usa-e2.stgx.cloudhub.io",
		},
		{
			name:   "multiple domains",
			gwName: "gw-1",
			domains: []string{
				"*.hey4z8.usa-e2.stgx.cloudhub.io",
				"*.abc123.eu-w1.prod.cloudhub.io",
			},
			wantPublicURLs: []string{
				"https://gw-1.hey4z8.usa-e2.stgx.cloudhub.io",
				"https://gw-1.abc123.eu-w1.prod.cloudhub.io",
			},
			wantInternal: "https://gw-1.internal-hey4z8.usa-e2.stgx.cloudhub.io",
		},
		{
			name:           "empty domains",
			gwName:         "gw-empty",
			domains:        []string{},
			wantPublicURLs: nil,
			wantInternal:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubURLs, intURL := BuildIngressURLs(tt.gwName, tt.domains)
			if len(pubURLs) != len(tt.wantPublicURLs) {
				t.Fatalf("Expected %d public URLs, got %d", len(tt.wantPublicURLs), len(pubURLs))
			}
			for i, want := range tt.wantPublicURLs {
				if pubURLs[i] != want {
					t.Errorf("PublicURLs[%d] = %s, want %s", i, pubURLs[i], want)
				}
			}
			if intURL != tt.wantInternal {
				t.Errorf("InternalURL = %s, want %s", intURL, tt.wantInternal)
			}
		})
	}
}

func TestManagedFlexGatewayClient_ErrorHandling(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/nonexistent": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Gateway not found")
		},
		"/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid gateway configuration")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	anypointClient, err := client.NewAnypointClient(&client.ClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	gwClient := &ManagedFlexGatewayClient{AnypointClient: anypointClient}

	t.Run("NotFound", func(t *testing.T) {
		_, err := gwClient.GetManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent gateway")
		}
	})

	t.Run("BadRequest", func(t *testing.T) {
		createReq := &CreateManagedFlexGatewayRequest{
			Name: "",
		}
		_, err := gwClient.CreateManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", createReq)
		if err == nil {
			t.Error("Expected error for invalid request")
		}
	})
}

func TestManagedFlexGateway_JSONSerialization(t *testing.T) {
	gw := &ManagedFlexGateway{
		ID:             "gw-123",
		Name:           "test-gw",
		TargetID:       "target-abc",
		RuntimeVersion: "1.9.9",
		ReleaseChannel: "lts",
		Size:           "small",
		Status:         "running",
		Configuration: ManagedFlexGatewayConfig{
			Ingress: IngressConfig{
				PublicURL:         "https://test-gw-hey4z8.usa-e2.stgx.cloudhub.io",
				InternalURL:       "https://test-gw-.internal-hey4z8.usa-e2.stgx.cloudhub.io",
				ForwardSSLSession: true,
				LastMileSecurity:  true,
			},
			Properties: PropertiesConfig{
				UpstreamResponseTimeout: 15,
				ConnectionIdleTimeout:   60,
			},
			Logging: LoggingConfig{Level: "info", ForwardLogs: true},
			Tracing: TracingConfig{Enabled: false},
		},
	}

	data, err := json.Marshal(gw)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ManagedFlexGateway
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != gw.ID {
		t.Errorf("Expected ID %s, got %s", gw.ID, decoded.ID)
	}
	if decoded.Configuration.Ingress.PublicURL != gw.Configuration.Ingress.PublicURL {
		t.Errorf("PublicURL mismatch: got %s, want %s", decoded.Configuration.Ingress.PublicURL, gw.Configuration.Ingress.PublicURL)
	}
	if decoded.Configuration.Ingress.InternalURL != gw.Configuration.Ingress.InternalURL {
		t.Errorf("InternalURL mismatch")
	}
	if decoded.Configuration.Ingress.ForwardSSLSession != true {
		t.Error("Expected ForwardSSLSession true after round-trip")
	}
	if decoded.Configuration.Properties.UpstreamResponseTimeout != 15 {
		t.Errorf("Expected UpstreamResponseTimeout 15, got %d", decoded.Configuration.Properties.UpstreamResponseTimeout)
	}
	if decoded.Configuration.Tracing.Enabled != false {
		t.Error("Expected Tracing.Enabled false after round-trip")
	}
}
