package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestIntegrationManagedFlexGatewayResource_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	gwName := "integration-test-gateway"
	gwNameUpdated := "integration-test-gateway-updated"

	mockGateway := &apimanagement.ManagedFlexGateway{
		ID:             "gw-integration-123",
		Name:           gwName,
		TargetID:       "target-integration-abc",
		RuntimeVersion: "1.9.9",
		ReleaseChannel: "lts",
		Size:           "small",
		Status:         "running",
		Configuration: apimanagement.ManagedFlexGatewayConfig{
			Ingress: apimanagement.IngressConfig{
				PublicURL:         "https://integration-test-gateway-hey4z8.usa-e2.stgx.cloudhub.io",
				InternalURL:       "https://integration-test-gateway-.internal-hey4z8.usa-e2.stgx.cloudhub.io",
				ForwardSSLSession: true,
				LastMileSecurity:  true,
			},
			Properties: apimanagement.PropertiesConfig{
				UpstreamResponseTimeout: 15,
				ConnectionIdleTimeout:   60,
			},
			Logging: apimanagement.LoggingConfig{Level: "info", ForwardLogs: true},
			Tracing: apimanagement.TracingConfig{Enabled: false},
		},
	}

	updatedGateway := *mockGateway
	updatedGateway.Name = gwNameUpdated

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.AssertHTTPRequest(t, r, "POST", "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways")
				testutil.JSONResponse(w, http.StatusCreated, mockGateway)
			} else {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-integration-123": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.JSONResponse(w, http.StatusOK, mockGateway)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-integration-123": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "PUT":
				testutil.JSONResponse(w, http.StatusOK, &updatedGateway)
			case "DELETE":
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	anypointClient, err := client.NewAnypointClient(&client.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	gwClient := &apimanagement.ManagedFlexGatewayClient{AnypointClient: anypointClient}
	gwResource := &ManagedFlexGatewayResource{client: gwClient}

	t.Run("Create", func(t *testing.T) {
		if gwResource.client == nil {
			t.Fatal("Resource client should be configured")
		}
	})

	t.Run("Read", func(t *testing.T) {
		gw, err := gwClient.GetManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "gw-integration-123")
		if err != nil {
			t.Fatalf("GetManagedFlexGateway failed: %v", err)
		}
		if gw.ID != "gw-integration-123" {
			t.Errorf("Expected ID gw-integration-123, got %s", gw.ID)
		}
		if gw.Name != gwName {
			t.Errorf("Expected Name %s, got %s", gwName, gw.Name)
		}
		if gw.Configuration.Ingress.ForwardSSLSession != true {
			t.Error("Expected ForwardSSLSession true")
		}
		if gw.Configuration.Ingress.PublicURL == "" {
			t.Fatal("Expected non-empty PublicURL")
		}
	})

	t.Run("Update", func(t *testing.T) {
		updateReq := &apimanagement.UpdateManagedFlexGatewayRequest{
			Name: gwNameUpdated,
		}

		gw, err := gwClient.UpdateManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "gw-integration-123", updateReq)
		if err != nil {
			t.Fatalf("UpdateManagedFlexGateway failed: %v", err)
		}
		if gw.Name != gwNameUpdated {
			t.Errorf("Expected Name %s, got %s", gwNameUpdated, gw.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := gwClient.DeleteManagedFlexGateway(context.Background(), "test-org-id", "test-env-id", "gw-integration-123")
		if err != nil {
			t.Fatalf("DeleteManagedFlexGateway failed: %v", err)
		}
	})
}

func TestIntegrationManagedFlexGatewayResource_InterfaceCompliance(t *testing.T) {
	gwResource := &ManagedFlexGatewayResource{}

	var _ resource.ResourceWithConfigure = gwResource
	var _ resource.ResourceWithImportState = gwResource

	ctx := context.Background()

	metaReq := resource.MetadataRequest{ProviderTypeName: "anypoint"}
	metaResp := &resource.MetadataResponse{}
	gwResource.Metadata(ctx, metaReq, metaResp)

	if metaResp.TypeName != "anypoint_managed_flexgateway" {
		t.Errorf("Expected TypeName anypoint_managed_flexgateway, got %s", metaResp.TypeName)
	}

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	gwResource.Schema(ctx, schemaReq, schemaResp)

	if len(schemaResp.Schema.Attributes) == 0 {
		t.Error("Schema should define attributes")
	}

	requiredAttrs := []string{"name", "environment_id", "target_id", "runtime_version"}
	for _, attr := range requiredAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing required attribute: %s", attr)
		}
	}

	computedAttrs := []string{"id", "organization_id", "status"}
	for _, attr := range computedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing computed attribute: %s", attr)
		}
	}

	nestedAttrs := []string{"ingress", "properties", "logging", "tracing"}
	for _, attr := range nestedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing nested attribute: %s", attr)
		}
	}
}

func BenchmarkIntegrationManagedFlexGatewayResource_Schema(b *testing.B) {
	gwResource := &ManagedFlexGatewayResource{}
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		gwResource.Schema(ctx, req, resp)
	}
}

func BenchmarkIntegrationManagedFlexGatewayResource_Metadata(b *testing.B) {
	gwResource := &ManagedFlexGatewayResource{}
	ctx := context.Background()
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.MetadataResponse{}
		gwResource.Metadata(ctx, req, resp)
	}
}
