package exchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

// TestAssetResource_ModifyPlan_DeprecatedPublishGuard: the multipart publish rejects
// status "deprecated" (400 INVALID_ASSET_STATUS), and on a version-bump replace the old
// version is already deleted by then. Create/replace with deprecated must fail at plan;
// an in-place move to (or edit while) deprecated must not.
func TestAssetResource_ModifyPlan_DeprecatedPublishGuard(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	nullObj := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object), nil)
	obj := func(version, status, name string) tftypes.Value {
		return assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, "org"),
			"group_id":        tftypes.NewValue(tftypes.String, "grp"),
			"asset_id":        tftypes.NewValue(tftypes.String, "asset"),
			"version":         tftypes.NewValue(tftypes.String, version),
			"type":            tftypes.NewValue(tftypes.String, "custom"),
			"status":          statusValue(status),
			"name":            tftypes.NewValue(tftypes.String, name),
		})
	}
	tests := []struct {
		name            string
		planRaw, stateR tftypes.Value
		wantErr         bool
		cfgRaw          *tftypes.Value // nil: config equals plan
	}{
		{"create with deprecated is blocked", obj("1.0.0", "deprecated", "n"), nullObj, true, nil},
		{"version bump while deprecated is blocked", obj("1.0.1", "deprecated", "n"), obj("1.0.0", "deprecated", "n"), true, nil},
		{"in-place published->deprecated is allowed", obj("1.0.0", "deprecated", "n"), obj("1.0.0", "published", "n"), false, nil},
		{"in-place edit while deprecated is allowed", obj("1.0.0", "deprecated", "n2"), obj("1.0.0", "deprecated", "n"), false, nil},
		{"version bump with published is allowed", obj("1.0.1", "published", "n"), obj("1.0.0", "deprecated", "n"), false, nil},
		// status omitted: UseStateForUnknown plans the prior "deprecated", but the publish sends no status.
		{"version bump with status omitted (state deprecated) is allowed", obj("1.0.1", "deprecated", "n"), obj("1.0.0", "deprecated", "n"), false, cfgPtr(obj("1.0.1", "", "n"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.planRaw
			if tt.cfgRaw != nil {
				cfg = *tt.cfgRaw
			}
			req := resource.ModifyPlanRequest{
				Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: tt.planRaw},
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: tt.stateR},
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: cfg},
			}
			resp := &resource.ModifyPlanResponse{}
			res.ModifyPlan(ctx, req, resp)
			found := false
			for _, d := range resp.Diagnostics {
				if strings.Contains(d.Summary(), `Cannot publish an asset version with status "deprecated"`) {
					found = true
				}
			}
			if found != tt.wantErr {
				t.Fatalf("deprecated guard fired=%v, want %v (diags: %v)", found, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestCheckVersionCollisionAtPlan_OwnedGAVOnReplan: core plans the create half of a
// replace with a null prior state. When that call carries this instance's own GAV (the
// private marker from the first call) the existing version is ours and is destroyed
// first, so it must not be reported as a collision; any other GAV still is.
func TestCheckVersionCollisionAtPlan_OwnedGAVOnReplan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // every version "exists"
		_, _ = w.Write([]byte(`{"groupId":"grp","assetId":"asset","version":"1.0.0","type":"custom"}`))
	}))
	defer server.Close()
	res := &AssetResource{client: &exchange.AssetClient{BaseURL: server.URL, Token: "t", HTTPClient: &http.Client{}}}
	plan := &AssetResourceModel{GroupID: types.StringValue("grp"), AssetID: types.StringValue("asset"), Version: types.StringValue("1.0.0")}
	state := &AssetResourceModel{}

	for _, tc := range []struct {
		owned   string
		wantErr bool
	}{
		{"grp/asset/1.0.0", false},
		{"grp/asset/0.9.0", true},
		{"", true},
	} {
		resp := &resource.ModifyPlanResponse{}
		res.checkVersionCollisionAtPlan(context.Background(), true, tc.owned, state, plan, resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("owned=%q: error=%v, want %v (%v)", tc.owned, resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
		}
	}
}

func cfgPtr(v tftypes.Value) *tftypes.Value { return &v }

// statusValue maps "" to a null status (attribute omitted from config).
func statusValue(s string) tftypes.Value {
	if s == "" {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, s)
}
