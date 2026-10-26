package exchange

import "testing"

func TestGraphQLTypeMapping(t *testing.T) {
	if got := storedAssetType("graphql-api"); got != "graphql" {
		t.Errorf("filter graphql-api -> %q, want graphql", got)
	}
	if got := userAssetType("graphql"); got != "graphql-api" {
		t.Errorf("stored graphql -> %q, want graphql-api", got)
	}
	for _, v := range []string{"graphql", "rest-api", "policy", "extension", ""} {
		if got := storedAssetType(v); got != v {
			t.Errorf("storedAssetType(%q) = %q, want unchanged", v, got)
		}
	}
	for _, v := range []string{"rest-api", "policy", "extension", ""} {
		if got := userAssetType(v); got != v {
			t.Errorf("userAssetType(%q) = %q, want unchanged", v, got)
		}
	}
}
