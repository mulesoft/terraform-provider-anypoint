package accessmanagement

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Scopes verified live to need envId: with only org they passed plan, then the
// /scopes PUT returned 400 after the app was created (app left tainted). They must
// fail at plan time, and pass once envId is supplied.
func TestValidateScopeContextParams_LiveVerifiedEnvScopes(t *testing.T) {
	for _, s := range []string{
		"read:api_query", "read:data_gateway", "view:clients", "view:destinations",
		"read:activity", "read:host_partners", "read:store", "read:store_clients",
	} {
		orgOnly := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{{scope: s, params: map[string]attr.Value{"org": types.StringValue("o1")}}})
		if !validateScopeContextParams(orgOnly).HasError() {
			t.Errorf("%s with only org: expected a plan-time envId error", s)
		}
		withEnv := makeScopeSet(t, []struct {
			scope  string
			params map[string]attr.Value
		}{{scope: s, params: map[string]attr.Value{"org": types.StringValue("o1"), "envId": types.StringValue("e1")}}})
		if d := validateScopeContextParams(withEnv); d.HasError() {
			t.Errorf("%s with org+envId: unexpected error %v", s, d)
		}
	}
}
