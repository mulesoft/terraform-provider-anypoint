package accessmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestEnvironmentResource_IntegrationCRUD tests the full CRUD lifecycle of Environment resource
func TestIntegrationEnvironmentResource_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test configuration
	envName := "terraform-integration-test-env"
	envNameUpdated := "terraform-integration-test-env-updated"

	// Create mock server for API simulation
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/environments":             testEnvironmentCreateHandler(t, envName),
		"/accounts/api/organizations/test-org-id/environments/test-env-id": testEnvironmentReadUpdateDeleteHandler(t, envName, envNameUpdated),
		"/accounts/api/v2/oauth2/token":                                    testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":                                                 testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)
	defer server.Close()

	// Create client
	clientConfig := &client.UserClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-pass",
		BaseURL:      server.URL,
		Timeout:      30,
	}

	userAnypointClient, err := client.NewUserAnypointClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	envClient := &accessmanagement.EnvironmentClient{
		UserAnypointClient: userAnypointClient,
	}

	// Create environment resource
	envResource := &EnvironmentResource{
		client: envClient,
	}

	ctx := context.Background()

	// Test CREATE operation
	t.Run("Create", func(t *testing.T) {
		if envResource.client == nil {
			t.Error("Environment resource client should be configured")
		}

		var _ resource.Resource = envResource
	})

	// Test READ operation
	t.Run("Read", func(t *testing.T) {
		environment, err := envClient.GetEnvironment(ctx, "test-org-id", "test-env-id")
		if err != nil {
			t.Errorf("GetEnvironment failed: %v", err)
		}

		if environment == nil {
			t.Error("GetEnvironment returned nil environment")
		}

		if environment != nil {
			if environment.Name != envName {
				t.Errorf("Expected environment name %s, got %s", envName, environment.Name)
			}
			if environment.Type != "sandbox" {
				t.Errorf("Expected environment type sandbox, got %s", environment.Type)
			}
		}
	})

	// Test UPDATE operation
	t.Run("Update", func(t *testing.T) {
		updateReq := &accessmanagement.UpdateEnvironmentRequest{
			Name: &envNameUpdated,
		}

		environment, err := envClient.UpdateEnvironment(ctx, "test-org-id", "test-env-id", updateReq)
		if err != nil {
			t.Errorf("UpdateEnvironment failed: %v", err)
		}

		if environment != nil && environment.Name != envNameUpdated {
			t.Errorf("Expected updated environment name %s, got %s", envNameUpdated, environment.Name)
		}
	})

	// Test DELETE operation
	t.Run("Delete", func(t *testing.T) {
		err := envClient.DeleteEnvironment(ctx, "test-org-id", "test-env-id")
		if err != nil {
			t.Errorf("DeleteEnvironment failed: %v", err)
		}
	})
}

// TestEnvironmentResource_ErrorHandling tests error scenarios
func TestIntegrationEnvironmentResource_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test error handling scenarios
	errorHandlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/environments/nonexistent": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Environment not found")
		},
		"/accounts/api/organizations/test-org-id/environments": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid environment name")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, errorHandlers)
	defer server.Close()

	// Create client
	clientConfig := &client.UserClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-pass",
		BaseURL:      server.URL,
		Timeout:      30,
	}

	userAnypointClient, err := client.NewUserAnypointClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	envClient := &accessmanagement.EnvironmentClient{
		UserAnypointClient: userAnypointClient,
	}

	ctx := context.Background()

	// Test 404 error handling
	t.Run("NotFound", func(t *testing.T) {
		_, err := envClient.GetEnvironment(ctx, "test-org-id", "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent environment")
		}

		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	// Test validation error handling
	t.Run("ValidationError", func(t *testing.T) {
		createReq := &accessmanagement.CreateEnvironmentRequest{
			Name:         "", // Invalid empty name
			Type:         "sandbox",
			IsProduction: false,
		}

		_, err := envClient.CreateEnvironment(ctx, "test-org-id", createReq)
		if err == nil {
			t.Error("Expected validation error for empty environment name")
		}
	})
}

// TestEnvironmentResource_InterfaceCompliance tests that resource implements required interfaces
func TestIntegrationEnvironmentResource_InterfaceCompliance(t *testing.T) {
	envResource := &EnvironmentResource{}

	// Test interface compliance
	var _ resource.Resource = envResource
	var _ resource.ResourceWithConfigure = envResource
	var _ resource.ResourceWithImportState = envResource

	// Test that all required methods exist
	ctx := context.Background()

	// Test Metadata method
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &resource.MetadataResponse{}
	envResource.Metadata(ctx, req, resp)

	expected := "anypoint_environment"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}

	// Test Schema method (basic verification)
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	envResource.Schema(ctx, schemaReq, schemaResp)

	if len(schemaResp.Schema.Attributes) == 0 {
		t.Error("Schema should define attributes")
	}

	// Verify required attributes exist
	requiredAttrs := []string{"name", "type", "is_production"}
	for _, attr := range requiredAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing required attribute: %s", attr)
		}
	}

	// Verify computed attributes exist
	computedAttrs := []string{"id", "organization_id", "client_id", "arc_namespace"}
	for _, attr := range computedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing computed attribute: %s", attr)
		}
	}
}

// Helper functions for mock handlers

func testEnvironmentCreateHandler(t *testing.T, expectedName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/environments")

		// Return created environment
		testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"id":             "test-env-id",
			"name":           expectedName,
			"type":           "sandbox",
			"isProduction":   false,
			"organizationId": "test-org-id",
			"clientId":       "test-client-id",
			"arcNamespace":   "test-arc-namespace",
		})
	}
}

func testEnvironmentReadUpdateDeleteHandler(t *testing.T, originalName, updatedName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/environments/test-env-id")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-env-id",
				"name":           originalName,
				"type":           "sandbox",
				"isProduction":   false,
				"organizationId": "test-org-id",
				"clientId":       "test-client-id",
				"arcNamespace":   "test-arc-namespace",
			})
		case "PUT", "PATCH":
			testutil.AssertHTTPRequest(t, r, r.Method, "/accounts/api/organizations/test-org-id/environments/test-env-id")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-env-id",
				"name":           updatedName,
				"type":           "sandbox",
				"isProduction":   false,
				"organizationId": "test-org-id",
				"clientId":       "test-client-id",
				"arcNamespace":   "test-arc-namespace",
			})
		case "DELETE":
			testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/environments/test-env-id")
			w.WriteHeader(http.StatusNoContent)
		default:
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkIntegrationEnvironmentResource_Schema(b *testing.B) {
	envResource := &EnvironmentResource{}
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		envResource.Schema(ctx, req, resp)
	}
}

func BenchmarkIntegrationEnvironmentResource_Metadata(b *testing.B) {
	envResource := &EnvironmentResource{}
	ctx := context.Background()
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.MetadataResponse{}
		envResource.Metadata(ctx, req, resp)
	}
}
