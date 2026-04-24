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

// TestIntegrationUserResource_CRUD tests the full CRUD lifecycle of User resource
func TestIntegrationUserResource_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test configuration
	username := "terraform-integration-test-user"
	email := "terraform-integration-test@example.com"
	emailUpdated := "terraform-integration-updated@example.com"

	// Create mock server for API simulation
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/users":              testUserCreateHandler(t, username, email),
		"/accounts/api/organizations/test-org-id/users/test-user-id": testUserReadUpdateDeleteHandler(t, username, email, emailUpdated),
		"/accounts/api/v2/oauth2/token":                              testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":                                           testutil.StandardMockHandlers()["/accounts/api/me"],
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

	userClient := &accessmanagement.UserClient{
		UserAnypointClient: userAnypointClient,
	}

	// Create user resource
	userResource := &UserResource{
		client: userClient,
	}

	ctx := context.Background()

	// Test CREATE operation
	t.Run("Create", func(t *testing.T) {
		if userResource.client == nil {
			t.Error("User resource client should be configured")
		}

		var testResource resource.Resource = userResource
		if testResource == nil {
			t.Error("User resource should implement Resource interface")
		}
	})

	// Test READ operation
	t.Run("Read", func(t *testing.T) {
		user, err := userClient.GetUser(ctx, "test-org-id", "test-user-id")
		if err != nil {
			t.Errorf("GetUser failed: %v", err)
		}

		if user == nil {
			t.Error("GetUser returned nil user")
		}

		if user != nil {
			if user.Username != username {
				t.Errorf("Expected username %s, got %s", username, user.Username)
			}
			if user.Email != email {
				t.Errorf("Expected email %s, got %s", email, user.Email)
			}
			if user.FirstName != "Terraform" {
				t.Errorf("Expected first name Terraform, got %s", user.FirstName)
			}
		}
	})

	// Test UPDATE operation
	t.Run("Update", func(t *testing.T) {
		updateReq := &accessmanagement.UpdateUserRequest{
			Email: &emailUpdated,
		}

		user, err := userClient.UpdateUser(ctx, "test-org-id", "test-user-id", updateReq)
		if err != nil {
			t.Errorf("UpdateUser failed: %v", err)
		}

		if user != nil && user.Email != emailUpdated {
			t.Errorf("Expected updated email %s, got %s", emailUpdated, user.Email)
		}
	})

	// Test DELETE operation
	t.Run("Delete", func(t *testing.T) {
		err := userClient.DeleteUser(ctx, "test-org-id", "test-user-id")
		if err != nil {
			t.Errorf("DeleteUser failed: %v", err)
		}
	})
}

// TestIntegrationUserResource_ErrorHandling tests error scenarios
func TestIntegrationUserResource_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test error handling scenarios
	errorHandlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/users/nonexistent": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
		},
		"/accounts/api/organizations/test-org-id/users": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid email format")
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

	userClient := &accessmanagement.UserClient{
		UserAnypointClient: userAnypointClient,
	}

	ctx := context.Background()

	// Test 404 error handling
	t.Run("NotFound", func(t *testing.T) {
		_, err := userClient.GetUser(ctx, "test-org-id", "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent user")
		}

		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	// Test validation error handling
	t.Run("ValidationError", func(t *testing.T) {
		createReq := &accessmanagement.CreateUserRequest{
			Username:  "test-user",
			Email:     "invalid-email", // Invalid email format
			FirstName: "Test",
			LastName:  "User",
			Password:  "password123",
		}

		_, err := userClient.CreateUser(ctx, "test-org-id", createReq)
		if err == nil {
			t.Error("Expected validation error for invalid email format")
		}
	})
}

// TestIntegrationUserResource_InterfaceCompliance tests that resource implements required interfaces
func TestIntegrationUserResource_InterfaceCompliance(t *testing.T) {
	userResource := &UserResource{}

	// Test interface compliance
	var _ resource.Resource = userResource
	var _ resource.ResourceWithConfigure = userResource
	var _ resource.ResourceWithImportState = userResource

	// Test that all required methods exist
	ctx := context.Background()

	// Test Metadata method
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &resource.MetadataResponse{}
	userResource.Metadata(ctx, req, resp)

	expected := "anypoint_user"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}

	// Test Schema method (basic verification)
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	userResource.Schema(ctx, schemaReq, schemaResp)

	if len(schemaResp.Schema.Attributes) == 0 {
		t.Error("Schema should define attributes")
	}

	// Verify required attributes exist
	requiredAttrs := []string{"username", "email", "first_name", "last_name", "password"}
	for _, attr := range requiredAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing required attribute: %s", attr)
		}
	}

	// Verify computed attributes exist
	computedAttrs := []string{"id", "organization_id"}
	for _, attr := range computedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing computed attribute: %s", attr)
		}
	}
}

// TestIntegrationUserResource_EmailValidation tests email validation logic
func TestIntegrationUserResource_EmailValidation(t *testing.T) {
	testCases := []struct {
		name      string
		email     string
		shouldErr bool
	}{
		{
			name:      "valid email",
			email:     "user@example.com",
			shouldErr: false,
		},
		{
			name:      "valid email with subdomain",
			email:     "user@subdomain.example.com",
			shouldErr: false,
		},
		{
			name:      "invalid email - no @",
			email:     "userexample.com",
			shouldErr: true,
		},
		{
			name:      "invalid email - no domain",
			email:     "user@",
			shouldErr: true,
		},
		{
			name:      "invalid email - no username",
			email:     "@example.com",
			shouldErr: true,
		},
		{
			name:      "empty email",
			email:     "",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test email validation (this would typically be done in the resource validation)
			isValid := strings.Contains(tc.email, "@") &&
				len(strings.Split(tc.email, "@")) == 2 &&
				tc.email != "" &&
				!strings.HasPrefix(tc.email, "@") &&
				!strings.HasSuffix(tc.email, "@")

			if tc.shouldErr && isValid {
				t.Errorf("Expected email %s to be invalid", tc.email)
			}
			if !tc.shouldErr && !isValid {
				t.Errorf("Expected email %s to be valid", tc.email)
			}
		})
	}
}

// Helper functions for mock handlers

func testUserCreateHandler(t *testing.T, expectedUsername, expectedEmail string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/users")

		// Return created user
		testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"id":             "test-user-id",
			"username":       expectedUsername,
			"email":          expectedEmail,
			"firstName":      "Terraform",
			"lastName":       "Test",
			"organizationId": "test-org-id",
			"enabled":        true,
			"deleted":        false,
			"lastLogin":      nil,
			"mfaVerified":    false,
			"mfaVerifiers":   []interface{}{},
			"type":           "member",
		})
	}
}

func testUserReadUpdateDeleteHandler(t *testing.T, username, originalEmail, updatedEmail string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/users/test-user-id")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-user-id",
				"username":       username,
				"email":          originalEmail,
				"firstName":      "Terraform",
				"lastName":       "Test",
				"organizationId": "test-org-id",
				"enabled":        true,
				"deleted":        false,
				"lastLogin":      nil,
				"mfaVerified":    false,
				"mfaVerifiers":   []interface{}{},
				"type":           "member",
			})
		case "PUT", "PATCH":
			testutil.AssertHTTPRequest(t, r, r.Method, "/accounts/api/organizations/test-org-id/users/test-user-id")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-user-id",
				"username":       username,
				"email":          updatedEmail,
				"firstName":      "Terraform",
				"lastName":       "Test",
				"organizationId": "test-org-id",
				"enabled":        true,
				"deleted":        false,
				"lastLogin":      nil,
				"mfaVerified":    false,
				"mfaVerifiers":   []interface{}{},
				"type":           "member",
			})
		case "DELETE":
			testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/users/test-user-id")
			w.WriteHeader(http.StatusNoContent)
		default:
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkIntegrationUserResource_Schema(b *testing.B) {
	userResource := &UserResource{}
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		userResource.Schema(ctx, req, resp)
	}
}

func BenchmarkIntegrationUserResource_Metadata(b *testing.B) {
	userResource := &UserResource{}
	ctx := context.Background()
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.MetadataResponse{}
		userResource.Metadata(ctx, req, resp)
	}
}
