package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAPIPolicyClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *client.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &client.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
			},
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
				tt.config.BaseURL = server.URL
			}
			_, err := NewAPIPolicyClient(tt.config)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestAPIPolicyClient_CRUD(t *testing.T) {
	orgID := "org-123"
	envID := "env-456"
	apiID := 100

	policyResp := APIPolicy{
		ID:               1001,
		PolicyTemplateID: "rate-limiting",
		GroupID:          "68ef9520-24e9-4cf2-b2f5-620025690913",
		AssetID:          "rate-limiting",
		AssetVersion:     "1.4.0",
		Order:            1,
		Disabled:         false,
		APIID:            apiID,
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

	handlers := testutil.StandardMockHandlers()

	handlers["POST /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(policyResp)
	}

	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies/1001"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(policyResp)
	}

	updatedResp := policyResp
	updatedResp.Disabled = true
	handlers["PATCH /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies/1001"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updatedResp)
	}

	handlers["DELETE /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies/1001"] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}

	server := testutil.MockHTTPServer(t, handlers)

	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	t.Run("Create", func(t *testing.T) {
		order := 1
		req := &CreateAPIPolicyRequest{
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

		policy, err := c.CreateAPIPolicy(context.Background(), orgID, envID, apiID, req)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if policy.ID != 1001 {
			t.Errorf("Expected ID 1001, got %d", policy.ID)
		}
		if policy.AssetID != "rate-limiting" {
			t.Errorf("Expected assetId rate-limiting, got %s", policy.AssetID)
		}
	})

	t.Run("Read", func(t *testing.T) {
		policy, err := c.GetAPIPolicy(context.Background(), orgID, envID, apiID, 1001)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if policy.ID != 1001 {
			t.Errorf("Expected ID 1001, got %d", policy.ID)
		}
		if policy.ConfigurationData == nil {
			t.Error("Expected configurationData, got nil")
		}
	})

	t.Run("Update", func(t *testing.T) {
		disabled := true
		req := &UpdateAPIPolicyRequest{
			Disabled: &disabled,
		}
		policy, err := c.UpdateAPIPolicy(context.Background(), orgID, envID, apiID, 1001, req)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if !policy.Disabled {
			t.Error("Expected policy to be disabled")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := c.DeleteAPIPolicy(context.Background(), orgID, envID, apiID, 1001)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})
}

func TestAPIPolicyClient_ErrorHandling(t *testing.T) {
	handlers := testutil.StandardMockHandlers()

	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies/9999"] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}

	handlers["POST /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"invalid configuration"}`))
	}

	server := testutil.MockHTTPServer(t, handlers)

	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	t.Run("NotFound", func(t *testing.T) {
		_, err := c.GetAPIPolicy(context.Background(), "org-123", "env-456", 100, 9999)
		if err == nil {
			t.Error("Expected error for not-found policy")
		}
	})

	t.Run("BadRequest", func(t *testing.T) {
		req := &CreateAPIPolicyRequest{
			ConfigurationData: map[string]interface{}{},
			GroupID:           "test",
			AssetID:           "bad-policy",
			AssetVersion:      "1.0.0",
		}
		_, err := c.CreateAPIPolicy(context.Background(), "org-123", "env-456", 100, req)
		if err == nil {
			t.Error("Expected error for bad request")
		}
	})
}

func TestValidatePolicyConfiguration(t *testing.T) {
	t.Run("valid rate-limiting config", func(t *testing.T) {
		errs := ValidatePolicyConfiguration("rate-limiting", map[string]interface{}{
			"rateLimits":    []interface{}{},
			"exposeHeaders": true,
		})
		if len(errs) > 0 {
			t.Errorf("Expected no errors, got %v", errs)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		errs := ValidatePolicyConfiguration("rate-limiting", map[string]interface{}{
			"exposeHeaders": true,
		})
		if len(errs) != 1 {
			t.Errorf("Expected 1 error, got %d: %v", len(errs), errs)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		errs := ValidatePolicyConfiguration("rate-limiting", map[string]interface{}{
			"rateLimits":   []interface{}{},
			"unknownField": "value",
		})
		if len(errs) != 1 {
			t.Errorf("Expected 1 error, got %d: %v", len(errs), errs)
		}
	})

	t.Run("unknown policy skips validation", func(t *testing.T) {
		errs := ValidatePolicyConfiguration("my-custom-policy", map[string]interface{}{
			"anyField": "anyValue",
		})
		if len(errs) != 0 {
			t.Errorf("Expected 0 errors for unknown policy, got %d", len(errs))
		}
	})

	t.Run("spike-control missing all required", func(t *testing.T) {
		errs := ValidatePolicyConfiguration("spike-control", map[string]interface{}{})
		if len(errs) != 4 {
			t.Errorf("Expected 4 errors for missing spike-control required fields, got %d: %v", len(errs), errs)
		}
	})

	t.Run("ip-allowlist valid", func(t *testing.T) {
		errs := ValidatePolicyConfiguration("ip-allowlist", map[string]interface{}{
			"ipExpression": "#[attributes.headers['x-forwarded-for']]",
			"ips":          []interface{}{"192.168.1.0/24"},
		})
		if len(errs) > 0 {
			t.Errorf("Expected no errors, got %v", errs)
		}
	})
}

func TestAPIPolicy_JSONSerialization(t *testing.T) {
	order := 1
	req := &CreateAPIPolicyRequest{
		ConfigurationData: map[string]interface{}{
			"rateLimits": []interface{}{
				map[string]interface{}{
					"maximumRequests":          100,
					"timePeriodInMilliseconds": 60000,
				},
			},
		},
		GroupID:      "68ef9520-24e9-4cf2-b2f5-620025690913",
		AssetID:      "rate-limiting",
		AssetVersion: "1.4.0",
		Order:        &order,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded CreateAPIPolicyRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.AssetID != "rate-limiting" {
		t.Errorf("Expected assetId rate-limiting, got %s", decoded.AssetID)
	}
	if decoded.ConfigurationData == nil {
		t.Error("Expected configurationData, got nil")
	}
}
