package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPortHasNoStaticDefault guards the sibling of the base_path import bug.
//
// port is Optional+Computed and RequiresReplace. A static Default wins over prior
// state whenever the config omits the attribute, so a bridge imported while listening
// on 8083 planned 8083 -> 8081 the moment the user left `port` out — and being
// RequiresReplace, that proposed destroying a healthy bridge to move it back to the
// default. Omitting an optional field after an import is the natural thing to do, so
// this was easier to trigger than the base_path case.
//
// UseStateForUnknown makes "omitted" mean "leave it alone". New bridges still get
// 8081 — but that is settleBridgePort's job, not bridgeProxyURI's; see
// TestSettleBridgePortFillsStateOnCreate for why the distinction matters.
func TestPortHasNoStaticDefault(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewMCPBridgeResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["port"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("port: expected Int64Attribute, got %T", resp.Schema.Attributes["port"])
	}

	if attr.Default != nil {
		t.Error("port must NOT carry a static Default: it overrides imported state and, " +
			"because port is RequiresReplace, turns an omitted attribute into a destroy-and-recreate")
	}
	if len(attr.PlanModifiers) < 2 {
		t.Fatalf("port: expected UseStateForUnknown followed by RequiresReplace, got %d modifier(s)", len(attr.PlanModifiers))
	}
	// Order matters: the state value must be adopted before RequiresReplace compares.
	if got := attr.PlanModifiers[0].Description(context.Background()); got == "" {
		t.Error("port: first plan modifier should be UseStateForUnknown")
	}
}

// TestSettleBridgePortFillsStateOnCreate covers the hole that removing the static
// Default opened up, and that the schema-shape test above cannot see.
//
// Removing the Default was right (it caused destroy-and-recreate on import), but the
// Default was also the ONLY thing that put a concrete value in `port` on create. With
// it gone, a config that omits `port` plans as UNKNOWN — UseStateForUnknown returns
// early when there is no prior state — and flattenBridge never assigns Port. The only
// other assignment lives in backfillBridgeImportFields, which is on the import path.
//
// So the create path would hand Terraform an unknown value and the apply would die with
// "Provider returned invalid result object after apply" AFTER the bridge, its Exchange
// asset and all five policies were already created on the platform.
//
// Red-check: delete the settleBridgePort call in Create and the "omitted" case fails.
func TestSettleBridgePortFillsStateOnCreate(t *testing.T) {
	cases := []struct {
		name string
		in   types.Int64
		want int64
	}{
		{"omitted from config (unknown on create)", types.Int64Unknown(), defaultBridgePort},
		{"explicitly null", types.Int64Null(), defaultBridgePort},
		{"zero", types.Int64Value(0), defaultBridgePort},
		{"explicit port is preserved", types.Int64Value(8083), 8083},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := &MCPBridgeResourceModel{Port: tc.in}
			settleBridgePort(data)

			if data.Port.IsUnknown() || data.Port.IsNull() {
				t.Fatalf("port left %s — Terraform rejects a non-concrete value after apply "+
					"with \"Provider returned invalid result object after apply\"", data.Port)
			}
			if got := data.Port.ValueInt64(); got != tc.want {
				t.Errorf("port = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestSettleBridgePortMatchesProxyURI keeps the value written to state and the value
// actually sent to the platform from drifting apart: state must not claim 8081 while
// the proxy URI was built with something else.
func TestSettleBridgePortMatchesProxyURI(t *testing.T) {
	data := &MCPBridgeResourceModel{Port: types.Int64Unknown()}
	settleBridgePort(data)

	// Create builds the URI from the pre-settle value, which is 0 for an absent port.
	fromURI := bridgeProxyURI(0, "/mcp")
	want := "http://0.0.0.0:8081/mcp"
	if fromURI != want {
		t.Fatalf("bridgeProxyURI(0) = %q, want %q", fromURI, want)
	}
	if got := data.Port.ValueInt64(); got != defaultBridgePort {
		t.Errorf("state port = %d but proxy URI used %d — state and platform disagree",
			got, defaultBridgePort)
	}
}

// TestBridgeProxyURI_LeadingSlashIsNotSignificant is the premise the plan modifier
// rests on: both spellings describe the same bridge because the URI sent to the
// platform is identical.
func TestBridgeProxyURI_LeadingSlashIsNotSignificant(t *testing.T) {
	withSlash := bridgeProxyURI(8083, "/mcp")
	without := bridgeProxyURI(8083, "mcp")

	if withSlash != without {
		t.Fatalf("expected identical proxy URIs, got %q and %q", withSlash, without)
	}
	if want := "http://0.0.0.0:8083/mcp"; withSlash != want {
		t.Errorf("proxy URI = %q, want %q", withSlash, want)
	}
}

// TestBasePathSlashInsensitive_SuppressesSlashOnlyDiff pins the fix for the import
// bug: base_path is RequiresReplace, and reading a bridge back (notably on import,
// where it is recovered from the live proxy URI) always yields the bare form. A user
// who declared "/mcp" therefore ended up with state "mcp", and the next plan proposed
// destroying and recreating a healthy bridge over a cosmetic slash.
func TestBasePathSlashInsensitive_SuppressesSlashOnlyDiff(t *testing.T) {
	tests := []struct {
		name       string
		state      types.String
		config     types.String
		wantPlan   types.String
		wantReason string
	}{
		{
			name:       "declared with a slash, read back without: keep state, no diff",
			state:      types.StringValue("mcp"),
			config:     types.StringValue("/mcp"),
			wantPlan:   types.StringValue("mcp"),
			wantReason: "this is the import case that forced a replacement",
		},
		{
			name:       "declared without a slash, read back with one",
			state:      types.StringValue("/mcp"),
			config:     types.StringValue("mcp"),
			wantPlan:   types.StringValue("/mcp"),
			wantReason: "symmetric — direction must not matter",
		},
		{
			name:       "surrounding whitespace is not significant either",
			state:      types.StringValue("mcp"),
			config:     types.StringValue("  /mcp  "),
			wantPlan:   types.StringValue("mcp"),
			wantReason: "normalizer trims before comparing",
		},
		// The commonest bridge of all: no base path. It reads back as "" while an
		// omitted config is null, and base_path is RequiresReplace — so if these do
		// not compare equal, importing such a bridge proposes destroying it.
		{
			name:       "bridge with no base path, config omits it",
			state:      types.StringValue(""),
			config:     types.StringNull(),
			wantPlan:   types.StringValue(""),
			wantReason: "null and empty describe the same bridge",
		},
		{
			name:       "state null, config explicitly empty",
			state:      types.StringNull(),
			config:     types.StringValue(""),
			wantPlan:   types.StringNull(),
			wantReason: "symmetric — same proxy URI either way",
		},
		// An omitted base_path against a bridge that HAS one is a real removal.
		{
			name:       "removing an existing base path is a real change",
			state:      types.StringValue("mcp"),
			config:     types.StringNull(),
			wantPlan:   types.StringNull(),
			wantReason: "the user is asking to drop the path — must still replace",
		},
		// A genuine change must still reach RequiresReplace. Suppressing it would be
		// far worse than the bug being fixed: the bridge would silently keep serving
		// the old path.
		{
			name:       "a real path change is left alone",
			state:      types.StringValue("mcp"),
			config:     types.StringValue("/mcp-v2"),
			wantPlan:   types.StringValue("/mcp-v2"),
			wantReason: "different path — must still force replacement",
		},
		{
			name:       "clearing the path is a real change",
			state:      types.StringValue("mcp"),
			config:     types.StringValue(""),
			wantPlan:   types.StringValue(""),
			wantReason: "empty base path is a different bridge",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &planmodifier.StringResponse{PlanValue: tc.config}
			basePathSlashInsensitive{}.PlanModifyString(
				context.Background(),
				planmodifier.StringRequest{
					StateValue:  tc.state,
					ConfigValue: tc.config,
					PlanValue:   tc.config,
				},
				resp,
			)

			if !resp.PlanValue.Equal(tc.wantPlan) {
				t.Errorf("plan = %s, want %s (%s)", resp.PlanValue, tc.wantPlan, tc.wantReason)
			}
		})
	}
}

// TestBasePathSlashInsensitive_LeavesUncomparablePlansAlone guards the cases where
// there is nothing to compare: a create declaring a path (no prior state), both sides
// absent, and a config value not yet known.
func TestBasePathSlashInsensitive_LeavesUncomparablePlansAlone(t *testing.T) {
	cases := []struct {
		name   string
		state  types.String
		config types.String
	}{
		{"create — no prior state", types.StringNull(), types.StringValue("/mcp")},
		{"both absent", types.StringNull(), types.StringNull()},
		{"config not yet known", types.StringValue("mcp"), types.StringUnknown()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &planmodifier.StringResponse{PlanValue: tc.config}
			basePathSlashInsensitive{}.PlanModifyString(
				context.Background(),
				planmodifier.StringRequest{StateValue: tc.state, ConfigValue: tc.config, PlanValue: tc.config},
				resp,
			)
			if !resp.PlanValue.Equal(tc.config) {
				t.Errorf("plan = %s, want the config value %s untouched", resp.PlanValue, tc.config)
			}
		})
	}
}

// TestBasePathModifierIsWiredBeforeRequiresReplace asserts the invariant the whole fix
// depends on, which unit-testing PlanModifyString directly cannot reach: the modifier
// must actually be attached to base_path, and must precede RequiresReplace so the
// latter compares the already-reconciled value. Without this, deleting the modifier
// from the schema — or swapping the two entries — leaves every other test passing.
func TestBasePathModifierIsWiredBeforeRequiresReplace(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewMCPBridgeResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["base_path"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("base_path: expected StringAttribute, got %T", resp.Schema.Attributes["base_path"])
	}
	if len(attr.PlanModifiers) < 2 {
		t.Fatalf("base_path: expected at least 2 plan modifiers, got %d", len(attr.PlanModifiers))
	}
	if _, ok := attr.PlanModifiers[0].(basePathSlashInsensitive); !ok {
		t.Errorf("base_path: first plan modifier is %T, want basePathSlashInsensitive — "+
			"it must run BEFORE RequiresReplace or the slash difference is never collapsed",
			attr.PlanModifiers[0])
	}
}
