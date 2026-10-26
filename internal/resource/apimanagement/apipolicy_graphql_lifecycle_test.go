package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestLifecycle_GraphQLAndWebSocketPolicies runs a FULL provider-code lifecycle
// (resolve → validate → apply-defaults → Create → flatten → Read → Update →
// Delete) for every GraphQL Gateway and WebSockets & Streaming policy added to
// the KnownPolicies catalog, against a mock Anypoint Platform.
//
// Why this test exists (and why it is not redundant with the existing
// TestIntegrationAPIPolicyResource_CRUD): the pre-existing integration test
// hand-codes group_id/asset_id/asset_version and a rate-limiting config. It
// therefore never exercises the resolution seam that the 5-policy change is
// built on:
//
//	policy_type alias
//	  → resolvePolicyIdentifiers  (alias → group 68ef9520-…, asset, version 1.0.0)
//	  → validateConfigurationData (required-field + unknown-field enforcement)
//	  → ApplyPolicyDefaults       (inject the schema defaults)
//	  → CreateAPIPolicyRequest    (resolved coordinates on the wire)
//	  → flattenPolicy             (reverse-resolve policy_type back into state)
//
// A live CRUD against the real …/apis/{apiId}/policies endpoint is impossible
// for these policies in the target org: their Exchange capability metadata is
// `capabilities.assetTypes: [graphql]`, so they only attach to a GraphQL-typed
// API instance running on a Flex/Omni gateway — and that org has zero API
// instances and zero gateways in every environment. This hermetic test is the
// faithful stand-in: it drives the exact provider code the resource runs, with
// the Platform mocked, and asserts the resolved coordinates reach the request.
func TestLifecycle_GraphQLAndWebSocketPolicies(t *testing.T) {
	const (
		orgID     = "test-org-id"
		envID     = "test-env-id"
		muleGroup = "68ef9520-24e9-4cf2-b2f5-620025690913"
	)

	type policyCase struct {
		policyType string
		// userConfig is the JSON the practitioner would put in configuration_data.
		userConfig string
		// requiredField, when non-empty, is a field Exchange marks required;
		// the test asserts that omitting it fails validation.
		requiredField string
		// wantDefaults are fields ApplyPolicyDefaults must inject when the user
		// omits them, with their expected injected values.
		wantDefaults map[string]interface{}
	}

	cases := []policyCase{
		{
			policyType:    "graphql-static-query-complexity",
			userConfig:    `{"maximumComplexity": 100}`,
			requiredField: "maximumComplexity",
			wantDefaults: map[string]interface{}{
				"defaultFieldCost":     1,
				"blockOperation":       true,
				"rejectUnboundedLists": true,
			},
		},
		{
			policyType: "graphql-schema-validation",
			userConfig: `{}`,
			wantDefaults: map[string]interface{}{
				"blockOperation": true,
			},
		},
		{
			policyType: "graphql-introspection-control",
			userConfig: `{}`,
			wantDefaults: map[string]interface{}{
				"blockSchema":   false,
				"blockType":     false,
				"blockTypename": false,
			},
		},
		{
			policyType: "graphql-operation-limits",
			userConfig: `{}`,
			wantDefaults: map[string]interface{}{
				"maxDepth":      -1,
				"maxAliases":    -1,
				"maxRootFields": -1,
				"maxDirectives": -1,
			},
		},
		{
			policyType: "websocket-connection-limit",
			userConfig: `{}`,
			wantDefaults: map[string]interface{}{
				"maximumConnections": 100,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.policyType, func(t *testing.T) {
			// --- mock Platform ---------------------------------------------------
			// A pointer we fill in on POST and echo back on GET/PATCH so state
			// round-trips exactly what the provider sent.
			var stored *apimanagement.APIPolicy
			var capturedCreate apimanagement.CreateAPIPolicyRequest

			handlers := testutil.StandardMockHandlers()

			handlers["POST /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies"] = func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&capturedCreate); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				// The whole point: the resolved coordinates must be on the wire.
				if capturedCreate.GroupID != muleGroup ||
					capturedCreate.AssetID != tc.policyType ||
					capturedCreate.AssetVersion != "1.0.0" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"message":"resolved coordinates missing from create request"}`))
					return
				}
				stored = &apimanagement.APIPolicy{
					ID:                5001,
					GroupID:           capturedCreate.GroupID,
					AssetID:           capturedCreate.AssetID,
					AssetVersion:      capturedCreate.AssetVersion,
					APIID:             100,
					Order:             1,
					Disabled:          false,
					ConfigurationData: capturedCreate.ConfigurationData,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(stored)
			}

			handlers["GET /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/5001"] = func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(stored)
			}

			handlers["PATCH /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/5001"] = func(w http.ResponseWriter, r *http.Request) {
				updated := *stored
				updated.Disabled = true
				stored = &updated
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(stored)
			}

			deleted := false
			handlers["DELETE /apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/100/policies/5001"] = func(w http.ResponseWriter, r *http.Request) {
				deleted = true
				w.WriteHeader(http.StatusNoContent)
			}

			server := testutil.MockHTTPServer(t, handlers)

			res := &APIPolicyResource{
				client: &apimanagement.APIPolicyClient{
					AnypointClient: &client.AnypointClient{
						BaseURL:    server.URL,
						Token:      "test-token",
						OrgID:      orgID,
						HTTPClient: server.Client(),
					},
				},
			}

			// --- step 1: resolve policy_type → coordinates (the new seam) --------
			// Model carries ONLY the alias, exactly as a minimal HCL block would.
			data := APIPolicyResourceModel{
				OrganizationID:    types.StringValue(orgID),
				EnvironmentID:     types.StringValue(envID),
				APIInstanceID:     types.StringValue("100"),
				PolicyType:        types.StringValue(tc.policyType),
				ConfigurationData: types.StringValue(tc.userConfig),
			}

			gotGroup, gotAsset, gotVersion, err := res.resolvePolicyIdentifiers(&data)
			if err != nil {
				t.Fatalf("resolvePolicyIdentifiers(%q) error: %v", tc.policyType, err)
			}
			if gotGroup != muleGroup {
				t.Errorf("resolved group = %q, want %q", gotGroup, muleGroup)
			}
			if gotAsset != tc.policyType {
				t.Errorf("resolved asset = %q, want %q", gotAsset, tc.policyType)
			}
			if gotVersion != "1.0.0" {
				t.Errorf("resolved version = %q, want 1.0.0", gotVersion)
			}

			// --- step 2: validation — required field enforced --------------------
			if tc.requiredField != "" {
				if errs := res.validateConfigurationData(gotAsset, `{}`); len(errs) == 0 {
					t.Errorf("expected validation error for missing required %q, got none", tc.requiredField)
				}
			}
			// The practitioner's actual config must validate cleanly.
			if errs := res.validateConfigurationData(gotAsset, tc.userConfig); len(errs) != 0 {
				t.Fatalf("validateConfigurationData(%q, %s) unexpected errors: %v", gotAsset, tc.userConfig, errs)
			}

			// --- step 3: apply defaults ------------------------------------------
			var cfg map[string]interface{}
			if err := json.Unmarshal([]byte(tc.userConfig), &cfg); err != nil {
				t.Fatalf("bad userConfig JSON: %v", err)
			}
			apimanagement.ApplyPolicyDefaults(gotAsset, cfg)
			for field, want := range tc.wantDefaults {
				got, ok := cfg[field]
				if !ok {
					t.Errorf("default %q not injected", field)
					continue
				}
				if !numericOrBoolEqual(got, want) {
					t.Errorf("default %q = %v (%T), want %v (%T)", field, got, got, want, want)
				}
			}

			// --- step 4: Create (real client → mock; coords asserted server-side)
			createReq := &apimanagement.CreateAPIPolicyRequest{
				ConfigurationData: cfg,
				APIVersionID:      100,
				GroupID:           gotGroup,
				AssetID:           gotAsset,
				AssetVersion:      gotVersion,
			}
			policy, err := res.client.CreateAPIPolicy(context.Background(), orgID, envID, 100, createReq)
			if err != nil {
				t.Fatalf("CreateAPIPolicy failed: %v", err)
			}
			if policy.ID != 5001 {
				t.Errorf("created policy ID = %d, want 5001", policy.ID)
			}

			// --- step 5: flatten into state; policy_type reverse-resolves --------
			res.flattenPolicy(policy, &data, orgID, envID)
			if data.PolicyType.ValueString() != tc.policyType {
				t.Errorf("flattened policy_type = %q, want %q", data.PolicyType.ValueString(), tc.policyType)
			}
			if data.GroupID.ValueString() != muleGroup {
				t.Errorf("flattened group_id = %q, want %q", data.GroupID.ValueString(), muleGroup)
			}
			if data.AssetVersion.ValueString() != "1.0.0" {
				t.Errorf("flattened asset_version = %q, want 1.0.0", data.AssetVersion.ValueString())
			}
			if data.ID.ValueString() != "5001" {
				t.Errorf("flattened id = %q, want 5001", data.ID.ValueString())
			}

			// --- step 6: Read ----------------------------------------------------
			readPolicy, err := res.client.GetAPIPolicy(context.Background(), orgID, envID, 100, 5001)
			if err != nil {
				t.Fatalf("GetAPIPolicy failed: %v", err)
			}
			if readPolicy.AssetID != tc.policyType {
				t.Errorf("read asset_id = %q, want %q", readPolicy.AssetID, tc.policyType)
			}
			if readPolicy.ConfigurationData == nil {
				t.Error("read configurationData is nil")
			}

			// --- step 7: Update (disable) ----------------------------------------
			disabled := true
			updated, err := res.client.UpdateAPIPolicy(context.Background(), orgID, envID, 100, 5001, &apimanagement.UpdateAPIPolicyRequest{
				Disabled: &disabled,
			})
			if err != nil {
				t.Fatalf("UpdateAPIPolicy failed: %v", err)
			}
			if !updated.Disabled {
				t.Error("expected policy to be disabled after update")
			}

			// --- step 8: Delete --------------------------------------------------
			if err := res.client.DeleteAPIPolicy(context.Background(), orgID, envID, 100, 5001); err != nil {
				t.Fatalf("DeleteAPIPolicy failed: %v", err)
			}
			if !deleted {
				t.Error("DELETE handler was never invoked")
			}
		})
	}
}

// numericOrBoolEqual compares an injected default (which may be an int in the
// pre-marshal map) against the expected value, tolerating int/float64 spread.
func numericOrBoolEqual(got, want interface{}) bool {
	if gb, ok := got.(bool); ok {
		wb, ok := want.(bool)
		return ok && gb == wb
	}
	gf, gok := toFloat(got)
	wf, wok := toFloat(want)
	return gok && wok && gf == wf
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
