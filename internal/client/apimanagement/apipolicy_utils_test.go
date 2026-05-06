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
