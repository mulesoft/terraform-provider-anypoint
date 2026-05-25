package client

import (
	"net/http"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func makeMeResponseWithOrgs(orgs []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"user": map[string]interface{}{
			"id":       "user-1",
			"username": "testuser",
			"organization": map[string]interface{}{
				"id": "org-1",
			},
			"memberOfOrganizations": orgs,
		},
		"client": map[string]interface{}{
			"org_id": "org-1",
		},
	}
}

func TestUserAnypointClient_SwitchOrganization(t *testing.T) {
	t.Run("switches to accessible org", func(t *testing.T) {
		meResp := makeMeResponseWithOrgs([]map[string]interface{}{
			{"id": "org-1", "name": "Org One"},
			{"id": "org-2", "name": "Org Two"},
		})
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, meResp)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		}

		err := c.SwitchOrganization("org-2")
		if err != nil {
			t.Fatalf("SwitchOrganization() unexpected error: %v", err)
		}
		if c.OrgID != "org-2" {
			t.Errorf("OrgID = %q, want org-2", c.OrgID)
		}
	})

	t.Run("returns error for inaccessible org", func(t *testing.T) {
		meResp := makeMeResponseWithOrgs([]map[string]interface{}{
			{"id": "org-1", "name": "Org One"},
		})
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, meResp)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		}

		err := c.SwitchOrganization("org-not-accessible")
		if err == nil {
			t.Fatal("Expected error for inaccessible org")
		}
	})

	t.Run("returns error when getMe fails", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "server error")
			},
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		}

		err := c.SwitchOrganization("org-x")
		if err == nil {
			t.Fatal("Expected error when getMe fails")
		}
	})
}

func TestUserAnypointClient_GetAccessibleOrganizations(t *testing.T) {
	t.Run("returns accessible orgs", func(t *testing.T) {
		meResp := makeMeResponseWithOrgs([]map[string]interface{}{
			{"id": "org-1", "name": "Org One"},
			{"id": "org-2", "name": "Org Two"},
		})
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, meResp)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		}

		orgs, err := c.GetAccessibleOrganizations()
		if err != nil {
			t.Fatalf("GetAccessibleOrganizations() unexpected error: %v", err)
		}
		if len(orgs) != 2 {
			t.Fatalf("Expected 2 orgs, got %d", len(orgs))
		}
		// Check that IDs and names are present
		found := false
		for _, org := range orgs {
			if org["id"] == "org-1" && org["name"] == "Org One" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find org-1 with name 'Org One'")
		}
	})

	t.Run("returns error when getMe fails", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusUnauthorized, "unauthorized")
			},
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		}

		_, err := c.GetAccessibleOrganizations()
		if err == nil {
			t.Fatal("Expected error when getMe fails")
		}
	})

	t.Run("empty org list returns empty slice", func(t *testing.T) {
		meResp := map[string]interface{}{
			"user": map[string]interface{}{
				"id":                    "user-1",
				"memberOfOrganizations": []interface{}{},
			},
			"client": map[string]interface{}{"org_id": "org-1"},
		}
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
			},
			"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, meResp)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		}

		orgs, err := c.GetAccessibleOrganizations()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(orgs) != 0 {
			t.Errorf("Expected empty orgs, got %d", len(orgs))
		}
	})
}

