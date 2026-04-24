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

func TestIntegrationAPIInstanceResource_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockInstance := &apimanagement.APIInstance{
		ID:             19876543,
		AssetID:        "myhealth-test",
		AssetVersion:   "1.0.0",
		ProductVersion: "v1",
		GroupID:        "test-org-id",
		Technology:     "flexGateway",
		Status:         "active",
		Spec: &apimanagement.APIInstanceSpec{
			AssetID: "myhealth-test",
			GroupID: "test-org-id",
			Version: "1.0.0",
		},
		Endpoint: &apimanagement.APIInstanceEndpoint{
			DeploymentType: "HY",
			Type:           "http",
			ProxyURI:       testutil.StringPtr("http://0.0.0.0:8081/gw-test"),
		},
		Routing: []apimanagement.APIInstanceRoute{
			{
				Label: "gw-1-route-1",
				Upstreams: []apimanagement.APIInstanceUpstream{
					{Weight: 50, URI: "http://www.google.com", Label: "Google"},
					{Weight: 50, URI: "http://www.yahoo.com", Label: "Yahoo"},
				},
				Rules: &apimanagement.APIInstanceRules{Methods: "GET"},
			},
			{
				Label: "gw-1-route-2",
				Upstreams: []apimanagement.APIInstanceUpstream{
					{Weight: 100, URI: "http://www.google.com", Label: "Google1"},
				},
				Rules: &apimanagement.APIInstanceRules{Methods: "POST"},
			},
		},
		Deployment: &apimanagement.APIInstanceDeployment{
			EnvironmentID:  "test-env-id",
			Type:           "HY",
			ExpectedStatus: "deployed",
			TargetID:       "target-gw-id",
			TargetName:     "gw-1",
			GatewayVersion: "1.0.0",
		},
	}

	updatedInstance := *mockInstance
	updatedInstance.Routing = []apimanagement.APIInstanceRoute{
		{
			Label: "updated-route",
			Upstreams: []apimanagement.APIInstanceUpstream{
				{Weight: 100, URI: "http://www.example.com", Label: "Example"},
			},
			Rules: &apimanagement.APIInstanceRules{Methods: "GET,POST"},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.JSONResponse(w, http.StatusCreated, mockInstance)
			} else {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		"/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/19876543": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.JSONResponse(w, http.StatusOK, mockInstance)
			case "PATCH":
				testutil.JSONResponse(w, http.StatusOK, &updatedInstance)
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

	anypointClient, err := client.NewAnypointClient(&client.ClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	apiClient := &apimanagement.APIInstanceClient{AnypointClient: anypointClient}
	apiResource := &APIInstanceResource{client: apiClient}

	t.Run("Create", func(t *testing.T) {
		if apiResource.client == nil {
			t.Fatal("Resource client should be configured")
		}
		var _ resource.Resource = apiResource
	})

	t.Run("Read", func(t *testing.T) {
		inst, err := apiClient.GetAPIInstance(context.Background(), "test-org-id", "test-env-id", 19876543)
		if err != nil {
			t.Fatalf("GetAPIInstance failed: %v", err)
		}
		if inst.ID != 19876543 {
			t.Errorf("Expected ID 19876543, got %d", inst.ID)
		}
		if inst.Technology != "flexGateway" {
			t.Errorf("Expected Technology flexGateway, got %s", inst.Technology)
		}
		if len(inst.Routing) != 2 {
			t.Errorf("Expected 2 routes, got %d", len(inst.Routing))
		}
		if inst.Routing[0].Label != "gw-1-route-1" {
			t.Errorf("Expected first route label 'gw-1-route-1', got %s", inst.Routing[0].Label)
		}
		if inst.Deployment.TargetName != "gw-1" {
			t.Errorf("Expected TargetName gw-1, got %s", inst.Deployment.TargetName)
		}
	})

	t.Run("Update", func(t *testing.T) {
		updateReq := &apimanagement.UpdateAPIInstanceRequest{
			Routing: []apimanagement.APIInstanceRoute{
				{
					Label: "updated-route",
					Upstreams: []apimanagement.APIInstanceUpstream{
						{Weight: 100, URI: "http://www.example.com", Label: "Example"},
					},
					Rules: &apimanagement.APIInstanceRules{Methods: "GET,POST"},
				},
			},
		}
		inst, err := apiClient.UpdateAPIInstance(context.Background(), "test-org-id", "test-env-id", 19876543, updateReq)
		if err != nil {
			t.Fatalf("UpdateAPIInstance failed: %v", err)
		}
		if len(inst.Routing) != 1 {
			t.Errorf("Expected 1 route after update, got %d", len(inst.Routing))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := apiClient.DeleteAPIInstance(context.Background(), "test-org-id", "test-env-id", 19876543)
		if err != nil {
			t.Fatalf("DeleteAPIInstance failed: %v", err)
		}
	})
}

func TestIntegrationAPIInstanceResource_InterfaceCompliance(t *testing.T) {
	apiResource := &APIInstanceResource{}

	var _ resource.Resource = apiResource
	var _ resource.ResourceWithConfigure = apiResource
	var _ resource.ResourceWithImportState = apiResource

	ctx := context.Background()

	metaReq := resource.MetadataRequest{ProviderTypeName: "anypoint"}
	metaResp := &resource.MetadataResponse{}
	apiResource.Metadata(ctx, metaReq, metaResp)

	if metaResp.TypeName != "anypoint_api_instance" {
		t.Errorf("Expected TypeName anypoint_api_instance, got %s", metaResp.TypeName)
	}

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	apiResource.Schema(ctx, schemaReq, schemaResp)

	if len(schemaResp.Schema.Attributes) == 0 {
		t.Fatal("Schema should define attributes")
	}

	requiredAttrs := []string{"environment_id", "spec"}
	for _, attr := range requiredAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing required attribute: %s", attr)
		}
	}

	computedAttrs := []string{"id", "organization_id", "status", "asset_id", "asset_version"}
	for _, attr := range computedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing computed attribute: %s", attr)
		}
	}

	nestedAttrs := []string{"spec", "endpoint", "deployment", "routing"}
	for _, attr := range nestedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing nested attribute: %s", attr)
		}
	}
}

func BenchmarkIntegrationAPIInstanceResource_Schema(b *testing.B) {
	apiResource := &APIInstanceResource{}
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		apiResource.Schema(ctx, req, resp)
	}
}
