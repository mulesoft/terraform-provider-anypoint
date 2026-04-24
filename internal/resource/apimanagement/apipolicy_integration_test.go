package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestIntegrationAPIPolicyResource_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	policyResp := &apimanagement.APIPolicy{
		ID:               2001,
		PolicyTemplateID: "rate-limiting",
		GroupID:          "68ef9520-24e9-4cf2-b2f5-620025690913",
		AssetID:          "rate-limiting",
		AssetVersion:     "1.4.0",
		Order:            1,
		Disabled:         false,
		APIID:            100,
		ConfigurationData: map[string]interface{}{
			"rateLimits": []interface{}{
				map[string]interface{}{
					"maximumRequests":          float64(100),
					"timePeriodInMilliseconds": float64(60000),
				},
			},
			"exposeHeaders": true,
		},
	}

	updatedResp := *policyResp
	updatedResp.Disabled = true
	updatedResp.ConfigurationData = map[string]interface{}{
		"rateLimits": []interface{}{
			map[string]interface{}{
				"maximumRequests":          float64(200),
				"timePeriodInMilliseconds": float64(30000),
			},
		},
		"exposeHeaders": false,
	}

	handlers := testutil.StandardMockHandlers()

	handlers["POST /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies"] =
		func(w http.ResponseWriter, r *http.Request) {
			var body apimanagement.CreateAPIPolicyRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body.AssetID == "" || body.GroupID == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"missing required fields"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policyResp)
		}

	handlers["GET /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/2001"] =
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(policyResp)
		}

	handlers["PATCH /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/2001"] =
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(updatedResp)
		}

	handlers["DELETE /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/2001"] =
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}

	server := testutil.MockHTTPServer(t, handlers)

	policyClient := &apimanagement.APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			OrgID:      "test-org-id",
			HTTPClient: server.Client(),
		},
	}

	res := &APIPolicyResource{client: policyClient}

	t.Run("Create", func(t *testing.T) {
		order := 1
		createReq := &apimanagement.CreateAPIPolicyRequest{
			ConfigurationData: map[string]interface{}{
				"rateLimits": []interface{}{
					map[string]interface{}{
						"maximumRequests":          100,
						"timePeriodInMilliseconds": 60000,
					},
				},
				"exposeHeaders": true,
			},
			GroupID:      "68ef9520-24e9-4cf2-b2f5-620025690913",
			AssetID:      "rate-limiting",
			AssetVersion: "1.4.0",
			Order:        &order,
		}

		policy, err := res.client.CreateAPIPolicy(context.Background(), "test-org-id", "test-env-id", 100, createReq)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if policy.ID != 2001 {
			t.Errorf("Expected ID 2001, got %d", policy.ID)
		}
		if policy.AssetID != "rate-limiting" {
			t.Errorf("Expected assetId rate-limiting, got %s", policy.AssetID)
		}
	})

	t.Run("Read", func(t *testing.T) {
		policy, err := res.client.GetAPIPolicy(context.Background(), "test-org-id", "test-env-id", 100, 2001)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if policy.ID != 2001 {
			t.Errorf("Expected ID 2001, got %d", policy.ID)
		}
		if policy.ConfigurationData == nil {
			t.Error("Expected configurationData, got nil")
		}
		rl, ok := policy.ConfigurationData["rateLimits"]
		if !ok || rl == nil {
			t.Error("Expected rateLimits in configurationData")
		}
	})

	t.Run("Update", func(t *testing.T) {
		disabled := true
		updateReq := &apimanagement.UpdateAPIPolicyRequest{
			ConfigurationData: map[string]interface{}{
				"rateLimits": []interface{}{
					map[string]interface{}{
						"maximumRequests":          200,
						"timePeriodInMilliseconds": 30000,
					},
				},
				"exposeHeaders": false,
			},
			Disabled: &disabled,
		}
		policy, err := res.client.UpdateAPIPolicy(context.Background(), "test-org-id", "test-env-id", 100, 2001, updateReq)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if !policy.Disabled {
			t.Error("Expected policy to be disabled")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := res.client.DeleteAPIPolicy(context.Background(), "test-org-id", "test-env-id", 100, 2001)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})

	_ = res
}

func TestIntegrationAPIPolicyResource_InterfaceCompliance(t *testing.T) {
	res := NewAPIPolicyResource()

	if _, ok := res.(resource.Resource); !ok {
		t.Error("APIPolicyResource does not implement resource.Resource")
	}
	if _, ok := res.(resource.ResourceWithConfigure); !ok {
		t.Error("APIPolicyResource does not implement ResourceWithConfigure")
	}
	if _, ok := res.(resource.ResourceWithImportState); !ok {
		t.Error("APIPolicyResource does not implement ResourceWithImportState")
	}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	expectedAttrs := []string{
		"id", "organization_id", "environment_id", "api_instance_id",
		"group_id", "asset_id", "asset_version", "configuration_data",
		"order", "disabled", "policy_template_id",
	}
	for _, attr := range expectedAttrs {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("Missing expected attribute: %s", attr)
		}
	}
}

func TestIntegrationAPIPolicyResource_ValidationOnCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	errs := apimanagement.ValidatePolicyConfiguration("rate-limiting", map[string]interface{}{
		"exposeHeaders": true,
	})
	if len(errs) == 0 {
		t.Error("Expected validation error for missing rateLimits")
	}

	errs = apimanagement.ValidatePolicyConfiguration("rate-limiting", map[string]interface{}{
		"rateLimits":    []interface{}{},
		"exposeHeaders": true,
	})
	if len(errs) != 0 {
		t.Errorf("Expected no validation errors, got: %v", errs)
	}

	errs = apimanagement.ValidatePolicyConfiguration("custom-policy-xyz", map[string]interface{}{
		"anything": "goes",
	})
	if len(errs) != 0 {
		t.Error("Unknown policies should skip validation")
	}
}
