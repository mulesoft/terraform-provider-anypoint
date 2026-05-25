package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestAPIPolicyClient_OutboundCRUD(t *testing.T) {
	orgID := "org-123"
	envID := "env-456"
	apiID := 100
	policyID := 42

	inboundBase := "/apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies"
	outboundBase := "/apimanager/xapi/v1/organizations/org-123/environments/env-456/apis/100/policies/outbound-policies"
	itemPath := inboundBase + "/42"

	mockPolicy := APIPolicy{
		ID:       policyID,
		AssetID:  "http-transform",
		AssetVersion: "1.0.0",
	}

	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			Token:      "test-token",
			HTTPClient: &http.Client{},
		},
	}

	t.Run("CreateOutboundAPIPolicy returns single object", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
					return
				}
				testutil.JSONResponse(w, http.StatusCreated, mockPolicy)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		result, err := c.CreateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, &CreateOutboundAPIPolicyRequest{
			AssetID:           "http-transform",
			AssetVersion:      "1.0.0",
			ConfigurationData: map[string]interface{}{"key": "value"},
		})
		if err != nil {
			t.Fatalf("CreateOutboundAPIPolicy() unexpected error: %v", err)
		}
		if result.ID != policyID {
			t.Errorf("CreateOutboundAPIPolicy() ID = %d, want %d", result.ID, policyID)
		}
	})

	t.Run("CreateOutboundAPIPolicy returns array response", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, []APIPolicy{mockPolicy})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		result, err := c.CreateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, &CreateOutboundAPIPolicyRequest{
			AssetID: "http-transform",
		})
		if err != nil {
			t.Fatalf("CreateOutboundAPIPolicy() unexpected error on array response: %v", err)
		}
		if result.ID != policyID {
			t.Errorf("CreateOutboundAPIPolicy() ID = %d, want %d", result.ID, policyID)
		}
	})

	t.Run("CreateOutboundAPIPolicy empty array returns error", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, []APIPolicy{})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		_, err := c.CreateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, &CreateOutboundAPIPolicyRequest{})
		if err == nil {
			t.Fatal("Expected error for empty array response")
		}
		if !strings.Contains(err.Error(), "empty array") {
			t.Errorf("Error = %v, want containing 'empty array'", err)
		}
	})

	t.Run("CreateOutboundAPIPolicy server error", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad request")
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		_, err := c.CreateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, &CreateOutboundAPIPolicyRequest{})
		if err == nil {
			t.Fatal("Expected error for 400 response")
		}
		if !strings.Contains(err.Error(), "failed to create policy with status 400") {
			t.Errorf("Error = %v, want 'failed to create policy with status 400'", err)
		}
	})

	t.Run("UpdateOutboundAPIPolicy returns single object", func(t *testing.T) {
		updated := mockPolicy
		updated.AssetVersion = "2.0.0"
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			itemPath: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, updated)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		result, err := c.UpdateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, policyID, &UpdateOutboundAPIPolicyRequest{
			AssetVersion: "2.0.0",
		})
		if err != nil {
			t.Fatalf("UpdateOutboundAPIPolicy() unexpected error: %v", err)
		}
		if result.AssetVersion != "2.0.0" {
			t.Errorf("UpdateOutboundAPIPolicy() AssetVersion = %s, want 2.0.0", result.AssetVersion)
		}
	})

	t.Run("UpdateOutboundAPIPolicy returns array response", func(t *testing.T) {
		updated := mockPolicy
		updated.AssetVersion = "3.0.0"
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			itemPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, []APIPolicy{updated})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		result, err := c.UpdateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, policyID, &UpdateOutboundAPIPolicyRequest{})
		if err != nil {
			t.Fatalf("UpdateOutboundAPIPolicy() unexpected error on array response: %v", err)
		}
		if result.AssetVersion != "3.0.0" {
			t.Errorf("UpdateOutboundAPIPolicy() AssetVersion = %s, want 3.0.0", result.AssetVersion)
		}
	})

	t.Run("UpdateOutboundAPIPolicy not found", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			itemPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		_, err := c.UpdateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, policyID, &UpdateOutboundAPIPolicyRequest{})
		if err == nil {
			t.Fatal("Expected error for 404 response")
		}
		if !client.IsNotFound(err) {
			t.Errorf("Expected IsNotFound, got %v", err)
		}
	})

	t.Run("UpdateOutboundAPIPolicy empty array returns error", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			itemPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, []APIPolicy{})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		_, err := c.UpdateOutboundAPIPolicy(context.Background(), orgID, envID, apiID, policyID, &UpdateOutboundAPIPolicyRequest{})
		if err == nil {
			t.Fatal("Expected error for empty array response")
		}
		if !strings.Contains(err.Error(), "empty array") {
			t.Errorf("Error = %v, want containing 'empty array'", err)
		}
	})

	t.Run("DeleteOutboundAPIPolicy success", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			itemPath: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		err := c.DeleteOutboundAPIPolicy(context.Background(), orgID, envID, apiID, policyID)
		if err != nil {
			t.Fatalf("DeleteOutboundAPIPolicy() unexpected error: %v", err)
		}
	})

	t.Run("DeleteOutboundAPIPolicy not found returns error", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			itemPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		err := c.DeleteOutboundAPIPolicy(context.Background(), orgID, envID, apiID, policyID)
		if err == nil {
			t.Fatal("Expected error for 404 response")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("Expected 404 in error, got %v", err)
		}
	})

	t.Run("ListOutboundAPIPolicies returns array", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, []APIPolicy{mockPolicy})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		policies, err := c.ListOutboundAPIPolicies(context.Background(), orgID, envID, apiID)
		if err != nil {
			t.Fatalf("ListOutboundAPIPolicies() unexpected error: %v", err)
		}
		if len(policies) != 1 {
			t.Errorf("ListOutboundAPIPolicies() len = %d, want 1", len(policies))
		}
	})

	t.Run("ListOutboundAPIPolicies returns envelope", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"policies": []APIPolicy{mockPolicy},
				})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		policies, err := c.ListOutboundAPIPolicies(context.Background(), orgID, envID, apiID)
		if err != nil {
			t.Fatalf("ListOutboundAPIPolicies() unexpected error on envelope: %v", err)
		}
		if len(policies) != 1 {
			t.Errorf("ListOutboundAPIPolicies() len = %d, want 1", len(policies))
		}
	})

	t.Run("ListOutboundAPIPolicies server error", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			outboundBase: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "server error")
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		c.BaseURL = server.URL

		_, err := c.ListOutboundAPIPolicies(context.Background(), orgID, envID, apiID)
		if err == nil {
			t.Fatal("Expected error for 500 response")
		}
		if !strings.Contains(err.Error(), "failed to list outbound policies with status 500") {
			t.Errorf("Error = %v, want 'failed to list outbound policies with status 500'", err)
		}
	})
}

func TestExpandCORSConfiguration(t *testing.T) {
	t.Run("public resource returns unchanged", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource": true,
			"originGroups": []interface{}{
				map[string]interface{}{
					"methods": []interface{}{"GET", "POST"},
				},
			},
		}
		result := expandCORSConfiguration(cfg)
		groups := result["originGroups"].([]interface{})
		group := groups[0].(map[string]interface{})
		if _, hasAllowed := group["allowedMethods"]; hasAllowed {
			t.Error("Public resource should not transform methods to allowedMethods")
		}
	})

	t.Run("non-public adds name and converts methods", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource": false,
			"originGroups": []interface{}{
				map[string]interface{}{
					"methods": []interface{}{"GET", "POST"},
				},
			},
		}
		result := expandCORSConfiguration(cfg)
		groups := result["originGroups"].([]interface{})
		group := groups[0].(map[string]interface{})

		if group["name"] != "group-0" {
			t.Errorf("Expected name 'group-0', got %v", group["name"])
		}
		allowedMethods, ok := group["allowedMethods"].([]interface{})
		if !ok || len(allowedMethods) != 2 {
			t.Fatalf("Expected 2 allowedMethods, got %v", group["allowedMethods"])
		}
		m0 := allowedMethods[0].(map[string]interface{})
		if m0["methodName"] != "GET" {
			t.Errorf("Expected methodName=GET, got %v", m0["methodName"])
		}
		if m0["isAllowed"] != true {
			t.Errorf("Expected isAllowed=true, got %v", m0["isAllowed"])
		}
		if _, hasMethods := group["methods"]; hasMethods {
			t.Error("Original methods key should be removed")
		}
	})

	t.Run("non-public with existing name preserved", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource": false,
			"originGroups": []interface{}{
				map[string]interface{}{
					"name":    "my-group",
					"methods": []interface{}{"PUT"},
				},
			},
		}
		result := expandCORSConfiguration(cfg)
		groups := result["originGroups"].([]interface{})
		group := groups[0].(map[string]interface{})
		if group["name"] != "my-group" {
			t.Errorf("Expected name 'my-group', got %v", group["name"])
		}
	})

	t.Run("accessControlMaxAge moves from top to group", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource":      false,
			"accessControlMaxAge": 60,
			"originGroups": []interface{}{
				map[string]interface{}{
					"methods": []interface{}{"GET"},
				},
			},
		}
		result := expandCORSConfiguration(cfg)
		if _, hasTopLevel := result["accessControlMaxAge"]; hasTopLevel {
			t.Error("Top-level accessControlMaxAge should be removed")
		}
		groups := result["originGroups"].([]interface{})
		group := groups[0].(map[string]interface{})
		if group["accessControlMaxAge"] != 60 {
			t.Errorf("Expected group accessControlMaxAge=60, got %v", group["accessControlMaxAge"])
		}
	})

	t.Run("default accessControlMaxAge is 30", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource": false,
			"originGroups": []interface{}{
				map[string]interface{}{
					"methods": []interface{}{"GET"},
				},
			},
		}
		result := expandCORSConfiguration(cfg)
		groups := result["originGroups"].([]interface{})
		group := groups[0].(map[string]interface{})
		if group["accessControlMaxAge"] != 30 {
			t.Errorf("Expected default accessControlMaxAge=30, got %v", group["accessControlMaxAge"])
		}
	})

	t.Run("no originGroups returns unchanged", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource": false,
		}
		result := expandCORSConfiguration(cfg)
		if _, hasGroups := result["originGroups"]; hasGroups {
			t.Error("Should not add originGroups key when not present")
		}
	})

	t.Run("float64 accessControlMaxAge", func(t *testing.T) {
		cfg := map[string]interface{}{
			"publicResource":      false,
			"accessControlMaxAge": float64(45),
			"originGroups": []interface{}{
				map[string]interface{}{
					"methods": []interface{}{"GET"},
				},
			},
		}
		result := expandCORSConfiguration(cfg)
		groups := result["originGroups"].([]interface{})
		group := groups[0].(map[string]interface{})
		if group["accessControlMaxAge"] != 45 {
			t.Errorf("Expected group accessControlMaxAge=45, got %v", group["accessControlMaxAge"])
		}
	})
}

func TestAPIPolicyClient_outboundBasePath(t *testing.T) {
	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL: "https://anypoint.mulesoft.com",
		},
	}
	path := c.outboundBasePath("org-1", "env-2", 3)
	expected := "https://anypoint.mulesoft.com/apimanager/xapi/v1/organizations/org-1/environments/env-2/apis/3/policies/outbound-policies"
	if path != expected {
		t.Errorf("outboundBasePath() = %v, want %v", path, expected)
	}
}

func TestAPIPolicyClient_OutboundRequests_JSONSerialization(t *testing.T) {
	createReq := &CreateOutboundAPIPolicyRequest{
		GroupID:           "org.mule.policies",
		AssetID:           "http-transform",
		AssetVersion:      "1.0.0",
		ConfigurationData: map[string]interface{}{"requestTransformations": []interface{}{}},
		UpstreamIDs:       []string{"up-1"},
		Label:             "test-label",
	}

	data, err := json.Marshal(createReq)
	if err != nil {
		t.Fatalf("Failed to marshal CreateOutboundAPIPolicyRequest: %v", err)
	}
	var decoded CreateOutboundAPIPolicyRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if decoded.AssetID != createReq.AssetID {
		t.Errorf("AssetID = %v, want %v", decoded.AssetID, createReq.AssetID)
	}

	updateReq := &UpdateOutboundAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{"key": "val"},
		AssetVersion:      "2.0.0",
		Label:             "updated-label",
		UpstreamIDs:       []string{"up-2"},
	}
	data, err = json.Marshal(updateReq)
	if err != nil {
		t.Fatalf("Failed to marshal UpdateOutboundAPIPolicyRequest: %v", err)
	}
	var decodedUpdate UpdateOutboundAPIPolicyRequest
	if err := json.Unmarshal(data, &decodedUpdate); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if decodedUpdate.AssetVersion != updateReq.AssetVersion {
		t.Errorf("AssetVersion = %v, want %v", decodedUpdate.AssetVersion, updateReq.AssetVersion)
	}
}
