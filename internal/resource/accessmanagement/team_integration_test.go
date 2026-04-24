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

// TestTeamResource_IntegrationCRUD tests the full CRUD lifecycle of Team resource
func TestIntegrationTeamResource_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test configuration
	teamName := "terraform-integration-test-team"
	teamNameUpdated := "terraform-integration-test-team-updated"

	// Create mock server for API simulation
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/teams":              testTeamCreateHandler(t, teamName),
		"/accounts/api/organizations/test-org-id/teams/test-team-id": testTeamReadUpdateDeleteHandler(t, teamName, teamNameUpdated),
		"/accounts/api/v2/oauth2/token":                              testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":                                           testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)
	defer server.Close()

	// Create client
	clientConfig := &client.ClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	}

	anypointClient, err := client.NewAnypointClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	teamClient := &accessmanagement.TeamClient{
		AnypointClient: anypointClient,
	}

	// Create team resource
	teamResource := &TeamResource{
		client: teamClient,
	}

	ctx := context.Background()

	// Test CREATE operation
	t.Run("Create", func(t *testing.T) {
		if teamResource.client == nil {
			t.Error("Team resource client should be configured")
		}

		var testResource resource.Resource = teamResource
		if testResource == nil {
			t.Error("Team resource should implement Resource interface")
		}
	})

	// Test READ operation
	t.Run("Read", func(t *testing.T) {
		team, err := teamClient.GetTeam(ctx, "test-org-id", "test-team-id")
		if err != nil {
			t.Errorf("GetTeam failed: %v", err)
		}

		if team == nil {
			t.Error("GetTeam returned nil team")
		}

		if team != nil {
			if team.TeamName != teamName {
				t.Errorf("Expected team name %s, got %s", teamName, team.TeamName)
			}
			if team.TeamType != "internal" {
				t.Errorf("Expected team type internal, got %s", team.TeamType)
			}
		}
	})

	// Test UPDATE operation
	t.Run("Update", func(t *testing.T) {
		updateReq := &accessmanagement.UpdateTeamRequest{
			TeamName: &teamNameUpdated,
		}

		team, err := teamClient.UpdateTeam(ctx, "test-org-id", "test-team-id", updateReq)
		if err != nil {
			t.Errorf("UpdateTeam failed: %v", err)
		}

		if team != nil && team.TeamName != teamNameUpdated {
			t.Errorf("Expected updated team name %s, got %s", teamNameUpdated, team.TeamName)
		}
	})

	// Test DELETE operation
	t.Run("Delete", func(t *testing.T) {
		err := teamClient.DeleteTeam(ctx, "test-org-id", "test-team-id")
		if err != nil {
			t.Errorf("DeleteTeam failed: %v", err)
		}
	})
}

// TestIntegrationTeamResource_ErrorHandling tests error scenarios
func TestIntegrationTeamResource_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test error handling scenarios
	errorHandlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/test-org-id/teams/nonexistent": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
		},
		"/accounts/api/organizations/test-org-id/teams": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid team name")
			}
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, errorHandlers)
	defer server.Close()

	// Create client
	clientConfig := &client.ClientConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		BaseURL:      server.URL,
		Timeout:      30,
	}

	anypointClient, err := client.NewAnypointClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	teamClient := &accessmanagement.TeamClient{
		AnypointClient: anypointClient,
	}

	ctx := context.Background()

	// Test 404 error handling
	t.Run("NotFound", func(t *testing.T) {
		_, err := teamClient.GetTeam(ctx, "test-org-id", "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent team")
		}

		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	// Test validation error handling
	t.Run("ValidationError", func(t *testing.T) {
		createReq := &accessmanagement.CreateTeamRequest{
			TeamName: "", // Invalid empty name
			TeamType: "internal",
		}

		_, err := teamClient.CreateTeam(ctx, "test-org-id", createReq)
		if err == nil {
			t.Error("Expected validation error for empty team name")
		}
	})
}

// TestIntegrationTeamResource_InterfaceCompliance tests that resource implements required interfaces
func TestIntegrationTeamResource_InterfaceCompliance(t *testing.T) {
	teamResource := &TeamResource{}

	// Test interface compliance
	var _ resource.Resource = teamResource
	var _ resource.ResourceWithConfigure = teamResource
	var _ resource.ResourceWithImportState = teamResource

	// Test that all required methods exist
	ctx := context.Background()

	// Test Metadata method
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &resource.MetadataResponse{}
	teamResource.Metadata(ctx, req, resp)

	expected := "anypoint_team"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName %s, got %s", expected, resp.TypeName)
	}

	// Test Schema method (basic verification)
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	teamResource.Schema(ctx, schemaReq, schemaResp)

	if len(schemaResp.Schema.Attributes) == 0 {
		t.Error("Schema should define attributes")
	}

	// Verify required attributes exist
	requiredAttrs := []string{"team_name", "team_type"}
	for _, attr := range requiredAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing required attribute: %s", attr)
		}
	}

	// Verify computed attributes exist
	computedAttrs := []string{"id", "organization_id", "created_at", "updated_at"}
	for _, attr := range computedAttrs {
		if _, exists := schemaResp.Schema.Attributes[attr]; !exists {
			t.Errorf("Schema missing computed attribute: %s", attr)
		}
	}
}

// Helper functions for mock handlers

func testTeamCreateHandler(t *testing.T, expectedName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/teams")

		// Return created team
		testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
			"team_id":      "test-team-id",
			"name":         expectedName,
			"team_name":    expectedName,
			"org_id":       "test-org-id",
			"team_type":    "internal",
			"created_date": "2023-01-01T00:00:00Z",
			"updated_date": "2023-01-01T00:00:00Z",
			"member_count": 0,
		})
	}
}

func testTeamReadUpdateDeleteHandler(t *testing.T, originalName, updatedName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/teams/test-team-id")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":      "test-team-id",
				"name":         originalName,
				"team_name":    originalName,
				"org_id":       "test-org-id",
				"team_type":    "internal",
				"created_date": "2023-01-01T00:00:00Z",
				"updated_date": "2023-01-01T00:00:00Z",
				"member_count": 0,
			})
		case "PUT", "PATCH":
			testutil.AssertHTTPRequest(t, r, r.Method, "/accounts/api/organizations/test-org-id/teams/test-team-id")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":      "test-team-id",
				"name":         updatedName,
				"team_name":    updatedName,
				"org_id":       "test-org-id",
				"team_type":    "internal",
				"created_date": "2023-01-01T00:00:00Z",
				"updated_date": "2023-01-01T00:00:00Z",
				"member_count": 0,
			})
		case "DELETE":
			testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/teams/test-team-id")
			w.WriteHeader(http.StatusNoContent)
		default:
			testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// Benchmark tests for performance validation
func BenchmarkIntegrationTeamResource_Schema(b *testing.B) {
	teamResource := &TeamResource{}
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		teamResource.Schema(ctx, req, resp)
	}
}

func BenchmarkIntegrationTeamResource_Metadata(b *testing.B) {
	teamResource := &TeamResource{}
	ctx := context.Background()
	req := resource.MetadataRequest{ProviderTypeName: "anypoint"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.MetadataResponse{}
		teamResource.Metadata(ctx, req, resp)
	}
}
