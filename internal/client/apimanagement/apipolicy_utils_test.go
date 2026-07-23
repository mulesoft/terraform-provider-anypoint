package apimanagement

import (
	"testing"
)

func TestLookupPolicy_Known(t *testing.T) {
	known := []string{
		"rate-limiting", "spike-control", "ip-allowlist", "ip-blocklist",
		"cors", "jwt-validation", "client-id-enforcement", "message-logging",
		"header-injection", "header-removal",
	}
	for _, pt := range known {
		info, ok := LookupPolicy(pt)
		if !ok {
			t.Errorf("LookupPolicy(%q) returned false, want true", pt)
		}
		if info.AssetID == "" {
			t.Errorf("LookupPolicy(%q) returned empty AssetID", pt)
		}
		if info.GroupID == "" {
			t.Errorf("LookupPolicy(%q) returned empty GroupID", pt)
		}
	}
}

func TestLookupPolicy_Unknown(t *testing.T) {
	_, ok := LookupPolicy("non-existent-policy-xyz")
	if ok {
		t.Error("LookupPolicy() with unknown policy should return false")
	}
}

func TestCamelToSnake(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"rateLimits", "rate_limits"},
		{"keySelector", "key_selector"},
		{"exposeHeaders", "expose_headers"},
		{"jwksURL", "jwks_url"},
		{"id", "id"},
		{"", ""},
		{"alreadySnake", "already_snake"},
	}
	for _, tc := range cases {
		got := CamelToSnake(tc.input)
		if got != tc.expected {
			t.Errorf("CamelToSnake(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestSnakeToCamel(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"rate_limits", "rateLimits"},
		{"key_selector", "keySelector"},
		{"expose_headers", "exposeHeaders"},
		{"id", "id"},
		{"", ""},
		{"already", "already"},
	}
	for _, tc := range cases {
		got := SnakeToCamel(tc.input)
		if got != tc.expected {
			t.Errorf("SnakeToCamel(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestCamelSnakeRoundtrip(t *testing.T) {
	cases := []string{"rateLimits", "keySelector", "exposeHeaders", "clusterizable"}
	for _, c := range cases {
		snake := CamelToSnake(c)
		back := SnakeToCamel(snake)
		if back != c {
			t.Errorf("roundtrip %q -> %q -> %q", c, snake, back)
		}
	}
}

func TestApplyPolicyDefaults_RateLimiting(t *testing.T) {
	config := map[string]interface{}{
		"rateLimits": []interface{}{},
	}
	ApplyPolicyDefaults("rate-limiting", config)
	// exposeHeaders should be injected as default false if missing
	_ = config["exposeHeaders"] // verify no panic on access
}

func TestApplyPolicyDefaults_UnknownPolicy(t *testing.T) {
	config := map[string]interface{}{"foo": "bar"}
	// should not panic for unknown policy
	ApplyPolicyDefaults("unknown-policy-xyz", config)
}

func TestApplyPolicyDefaults_NilConfig(t *testing.T) {
	// should not panic with nil map
	ApplyPolicyDefaults("rate-limiting", nil)
}

func TestLookupPolicy_AllKnown(t *testing.T) {
	for policyType := range KnownPolicies {
		info, ok := LookupPolicy(policyType)
		if !ok {
			t.Errorf("LookupPolicy(%q) returned false for known policy", policyType)
		}
		if info.DefaultVersion == "" {
			t.Errorf("LookupPolicy(%q) has empty DefaultVersion", policyType)
		}
	}
}

// TestLookupPolicy_GraphQLAndWebSocket asserts the GraphQL Gateway and
// WebSockets & Streaming policies are registered with the coordinates pulled
// from Exchange (group 68ef9520-…, version 1.0.0, inbound). Regression guard so
// these five policy_type aliases cannot be silently dropped.
func TestLookupPolicy_GraphQLAndWebSocket(t *testing.T) {
	const muleGroup = "68ef9520-24e9-4cf2-b2f5-620025690913"
	want := []string{
		"graphql-schema-validation",
		"graphql-operation-limits",
		"graphql-introspection-control",
		"graphql-static-query-complexity",
		"websocket-connection-limit",
	}
	for _, pt := range want {
		info, ok := LookupPolicy(pt)
		if !ok {
			t.Errorf("LookupPolicy(%q) returned false, want registered", pt)
			continue
		}
		if info.GroupID != muleGroup {
			t.Errorf("LookupPolicy(%q).GroupID = %q, want %q", pt, info.GroupID, muleGroup)
		}
		if info.AssetID != pt {
			t.Errorf("LookupPolicy(%q).AssetID = %q, want %q", pt, info.AssetID, pt)
		}
		if info.DefaultVersion != "1.0.0" {
			t.Errorf("LookupPolicy(%q).DefaultVersion = %q, want 1.0.0", pt, info.DefaultVersion)
		}
		if !info.InboundPolicy {
			t.Errorf("LookupPolicy(%q) should be an inbound policy", pt)
		}
		if info.OutboundPolicy {
			t.Errorf("LookupPolicy(%q) should not be an outbound policy", pt)
		}
	}
}

// TestValidatePolicyConfiguration_StaticQueryComplexity verifies the one field
// that Exchange marks required (maximumComplexity) is enforced, and that an
// unknown field is rejected — the config schema wiring for the new policies.
func TestValidatePolicyConfiguration_StaticQueryComplexity(t *testing.T) {
	// Missing the required maximumComplexity → one error.
	errs := ValidatePolicyConfiguration("graphql-static-query-complexity", map[string]interface{}{})
	if len(errs) == 0 {
		t.Error("expected a validation error when maximumComplexity is missing, got none")
	}

	// Present + only-known fields → no error.
	errs = ValidatePolicyConfiguration("graphql-static-query-complexity", map[string]interface{}{
		"maximumComplexity": 100,
	})
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got %v", errs)
	}

	// Unknown field → rejected.
	errs = ValidatePolicyConfiguration("graphql-static-query-complexity", map[string]interface{}{
		"maximumComplexity": 100,
		"bogusField":        true,
	})
	if len(errs) == 0 {
		t.Error("expected a validation error for an unknown field, got none")
	}
}

// TestApplyPolicyDefaults_WebSocketConnectionLimit checks the int default is
// injected when the user omits it.
func TestApplyPolicyDefaults_WebSocketConnectionLimit(t *testing.T) {
	config := map[string]interface{}{}
	ApplyPolicyDefaults("websocket-connection-limit", config)
	if config["maximumConnections"] != 100 {
		t.Errorf("maximumConnections default = %v, want 100", config["maximumConnections"])
	}
}
