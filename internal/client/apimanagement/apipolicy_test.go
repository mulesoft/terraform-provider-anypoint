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
		_, _ = w.Write([]byte(`{"message":"invalid configuration"}`))
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

// TestListAPIPolicies_ReturnsBothInboundAndOutbound is a regression test for
// the outbound 409-conflict recovery path. When the user re-applies a plan
// whose state was lost, Create returns 409 because the policy exists on the
// server. The old recovery used GET on /policies/outbound-policies which the
// Anypoint Platform rejects with 405 Method Not Allowed. The new recovery
// uses the universal listing endpoint at /policies (this function), which
// returns BOTH inbound and outbound policies — outbound policies are simply
// inbound-listed entries that carry a non-empty upstreamIds.
func TestListAPIPolicies_ReturnsBothInboundAndOutbound(t *testing.T) {
	orgID := "org-123"
	envID := "env-456"
	apiID := 100

	inbound := APIPolicy{
		ID:           1001,
		AssetID:      "rate-limiting",
		GroupID:      "68ef9520-24e9-4cf2-b2f5-620025690913",
		AssetVersion: "1.4.0",
		Label:        "inbound-rl",
		Order:        1,
		APIID:        apiID,
	}
	outbound := APIPolicy{
		ID:           1002,
		AssetID:      "message-logging-outbound",
		GroupID:      "68ef9520-24e9-4cf2-b2f5-620025690913",
		AssetVersion: "1.0.0",
		Label:        "outbound-logger",
		Order:        2,
		APIID:        apiID,
		UpstreamIDs:  []string{"upstream-uuid-1"},
	}

	handlers := testutil.StandardMockHandlers()
	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]APIPolicy{inbound, outbound})
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	policies, err := c.ListAPIPolicies(context.Background(), orgID, envID, apiID)
	if err != nil {
		t.Fatalf("ListAPIPolicies failed: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}

	var sawInbound, sawOutbound bool
	for _, p := range policies {
		if p.ID == 1001 && p.Label == "inbound-rl" {
			sawInbound = true
		}
		if p.ID == 1002 && p.Label == "outbound-logger" && len(p.UpstreamIDs) == 1 {
			sawOutbound = true
		}
	}
	if !sawInbound {
		t.Error("expected to find the inbound policy in the list")
	}
	if !sawOutbound {
		t.Error("expected to find the outbound policy (with upstreamIds) in the list")
	}
}

// TestListAPIPolicies_AcceptsCompactListShape covers the JSON shape the
// production Anypoint Platform actually returns from the policies LIST
// endpoint: items use `policyId` (not `id`), nest asset coordinates under
// `template`/`implementationAsset`, and use `configuration` (not
// `configurationData`). Outbound policies appear in this list with their
// `upstreamIds` echoed at the top level. If the provider's listing parser
// silently mis-decodes these fields, 409-conflict recovery prints "no
// matching policy found" even when the orphan is sitting right there.
func TestListAPIPolicies_AcceptsCompactListShape(t *testing.T) {
	const body = `[
		{
			"policyId": 6086880,
			"order": 11,
			"label": "sse-logs",
			"template": {"groupId":"68ef9520-24e9-4cf2-b2f5-620025690913","assetId":"sse-logging","assetVersion":"1.0.1"},
			"implementationAsset": {"assetId":"sse-logging-policy-flex","groupId":"68ef9520-24e9-4cf2-b2f5-620025690913","version":"1.0.3"},
			"configuration": {"logs":[]},
			"policyTemplateId": "1185"
		},
		{
			"policyId": 6086912,
			"order": 33,
			"label": "outbound-logger",
			"template": {"groupId":"68ef9520-24e9-4cf2-b2f5-620025690913","assetId":"message-logging-outbound","assetVersion":"1.0.0"},
			"upstreamIds": ["upstream-uuid-1"],
			"policyTemplateId": "1950"
		}
	]`
	handlers := testutil.StandardMockHandlers()
	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	policies, err := c.ListAPIPolicies(context.Background(), "org-123", "env-456", 100)
	if err != nil {
		t.Fatalf("ListAPIPolicies failed: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}

	var inbound, outbound *APIPolicy
	for i := range policies {
		switch policies[i].ID {
		case 6086880:
			inbound = &policies[i]
		case 6086912:
			outbound = &policies[i]
		}
	}
	if inbound == nil {
		t.Fatal("expected to find inbound id=6086880 (parsed from policyId)")
	}
	if inbound.AssetID != "sse-logging" || inbound.GroupID != "68ef9520-24e9-4cf2-b2f5-620025690913" || inbound.AssetVersion != "1.0.1" {
		t.Errorf("expected asset coords from `template`, got %+v", inbound)
	}
	if inbound.Label != "sse-logs" {
		t.Errorf("expected label=sse-logs, got %q", inbound.Label)
	}
	if outbound == nil {
		t.Fatal("expected to find outbound id=6086912 (parsed from policyId)")
	}
	if outbound.AssetID != "message-logging-outbound" {
		t.Errorf("expected outbound assetId=message-logging-outbound, got %q", outbound.AssetID)
	}
	if outbound.Label != "outbound-logger" {
		t.Errorf("expected outbound label=outbound-logger, got %q", outbound.Label)
	}
	if len(outbound.UpstreamIDs) != 1 || outbound.UpstreamIDs[0] != "upstream-uuid-1" {
		t.Errorf("expected outbound to carry upstreamIds=[upstream-uuid-1], got %v", outbound.UpstreamIDs)
	}
}

// TestListAPIPolicies_AcceptsEnvelopeShape verifies that the universal listing
// endpoint is parsed correctly when it returns the {"policies":[...]} envelope
// instead of a top-level array — the Platform has historically returned both.
func TestListAPIPolicies_AcceptsEnvelopeShape(t *testing.T) {
	handlers := testutil.StandardMockHandlers()
	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/policies"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"policies":[{"id":7,"assetId":"rate-limiting","label":"only"}]}`))
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &APIPolicyClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	policies, err := c.ListAPIPolicies(context.Background(), "org-123", "env-456", 100)
	if err != nil {
		t.Fatalf("ListAPIPolicies failed: %v", err)
	}
	if len(policies) != 1 || policies[0].ID != 7 || policies[0].Label != "only" {
		t.Fatalf("unexpected policies parsed from envelope: %#v", policies)
	}
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
