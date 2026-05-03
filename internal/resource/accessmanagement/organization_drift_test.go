package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	am "github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestOrganizationDriftScenario_ManagedGatewayLarge simulates the bug report:
//
//  1. Apply with `managed_gateway_large = { assigned = 0 }` → Create.
//  2. User changes the entitlement to 1 in the Anypoint UI → server now has 1.
//  3. Re-apply with the original config → Terraform should reset server to 0.
//
// This test exercises the provider-side flatten/expand layers end-to-end so
// we can assert the exact bytes that flow Read→State and State→Update.
func TestOrganizationDriftScenario_ManagedGatewayLarge(t *testing.T) {
	ctx := context.Background()

	// Step 1 — server currently reports assigned = 1 (post UI change).
	serverSide := am.Entitlements{
		CreateSubOrgs:      false,
		CreateEnvironments: true,
		ManagedGatewayLarge: &am.AssignedEntitlement{
			Assigned: 1,
		},
		ManagedGatewaySmall: &am.AssignedEntitlement{
			Assigned: 3,
		},
	}

	// Flatten — this is what Read would put into state on refresh.
	stateObj, diags := flattenEntitlements(ctx, serverSide)
	if diags.HasError() {
		t.Fatalf("flatten diagnostics: %s", diags)
	}

	gwLarge, ok := stateObj.Attributes()["managed_gateway_large"].(types.Object)
	if !ok {
		t.Fatalf("managed_gateway_large is not an Object: %T", stateObj.Attributes()["managed_gateway_large"])
	}
	if gwLarge.IsNull() || gwLarge.IsUnknown() {
		t.Fatalf("expected managed_gateway_large to be a known Object, got null/unknown")
	}
	assignedVal := gwLarge.Attributes()["assigned"].(types.Int64).ValueInt64()
	if assignedVal != 1 {
		t.Errorf("Step 2 refresh drift: want state.managed_gateway_large.assigned = 1, got %d", assignedVal)
	}

	// Step 3 — user's config says assigned = 0. Build a plan Object with the
	// user's desired value (= 0), while keeping managed_gateway_small from state
	// (the user didn't touch that one so its plan value equals state).
	wantAssigned := int64(0)
	planGwLarge, planDiags := types.ObjectValue(
		map[string]attr.Type{"assigned": types.Int64Type},
		map[string]attr.Value{"assigned": types.Int64Value(wantAssigned)},
	)
	if planDiags.HasError() {
		t.Fatalf("plan object diagnostics: %s", planDiags)
	}

	// Replace managed_gateway_large in the flattened state with user's intent.
	entAttrs := map[string]attr.Value{}
	for k, v := range stateObj.Attributes() {
		entAttrs[k] = v
	}
	entAttrs["managed_gateway_large"] = planGwLarge

	planObj, diags := types.ObjectValue(getEntitlementsAttributeTypes(), entAttrs)
	if diags.HasError() {
		t.Fatalf("plan-level entitlements: %s", diags)
	}

	// Expand — this is what Update would send to UpdateOrganization.
	expanded, diags := expandEntitlements(ctx, planObj)
	if diags.HasError() {
		t.Fatalf("expand diagnostics: %s", diags)
	}
	if expanded.ManagedGatewayLarge == nil {
		t.Fatalf("expanded ManagedGatewayLarge is nil — the 0 value would be omitted from PUT, server would not reset")
	}
	if expanded.ManagedGatewayLarge.Assigned != 0 {
		t.Errorf("expanded ManagedGatewayLarge.Assigned = %d, want 0", expanded.ManagedGatewayLarge.Assigned)
	}
	// Preserved from state:
	if expanded.ManagedGatewaySmall == nil || expanded.ManagedGatewaySmall.Assigned != 3 {
		t.Errorf("expanded ManagedGatewaySmall should be preserved at 3 (got %+v)", expanded.ManagedGatewaySmall)
	}

	// Step 4 — verify the full PUT body actually includes the 0.
	req := &am.UpdateOrganizationRequest{
		ID:           "abc",
		Name:         "Demo",
		OwnerID:      "owner",
		Properties:   map[string]interface{}{},
		Entitlements: expanded,
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, `"managedGatewayLarge":{"assigned":0}`) {
		t.Errorf("PUT body missing managedGatewayLarge:{assigned:0}: %s", got)
	}
}

// TestUpdatePayload_MatchesUIContract pins the "fully-declared HCL" PUT body
// against the wire format the Anypoint UI expects — envelope shape, camelCase
// field names, AssignedEntitlement (no `reassigned` key) for
// managedGatewaySmall/Large, and absence of parentOrganizationId/staticIps/vpns.
//
// NOTE: in production the Update handler feeds req.Config (not req.Plan) into
// expandEntitlements, so the real PUT only contains fields the user actually
// declared in HCL. See
// TestUpdatePayload_BusinessGroupOmitsMasterOrgOnlyEntitlements for that path.
// This test exercises the marshal pipeline when every optional entitlement
// happens to be declared, to guard the envelope shape independently. Reference
// curl captured from the UI:
//
//	PUT /accounts/api/organizations/{id}
//	{"id":"…","name":"qwqq","ownerId":"…","properties":{"flow_designer":{}},
//	 "entitlements":{"globalDeployment":false,"createSubOrgs":true,
//	                 "createEnvironments":false,
//	                 "networkConnections":{"assigned":0},
//	                 "managedGatewaySmall":{"assigned":1}}}
//
// We assert the JSON the provider marshals has the same top-level envelope
// (id / name / ownerId / properties / entitlements and NOT parentOrganizationId),
// round-trips the server's `properties` blob unchanged, and uses the exact
// `managedGatewaySmall":{"assigned":1}` / `managedGatewayLarge":{"assigned":N}`
// wire format — i.e. AssignedEntitlement (just `assigned`, no `reassigned`).
func TestUpdatePayload_MatchesUIContract(t *testing.T) {
	ctx := context.Background()

	// Build a plan that matches the UI's edit session: user bumped
	// managed_gateway_small from 0 to 1 and left managed_gateway_large at 0.
	plan := map[string]attr.Value{
		"create_sub_orgs":         types.BoolValue(true),
		"create_environments":     types.BoolValue(false),
		"global_deployment":       types.BoolValue(false),
		"runtime_fabric":          types.BoolValue(false),
		"vcores_production":       zeroVCore(),
		"vcores_sandbox":          zeroVCore(),
		"vcores_design":           zeroVCore(),
		"vpcs":                    zeroVCore(),
		"network_connections":     zeroVCore(),
		"hybrid":                  zeroEnabled(),
		"flex_gateway":            zeroEnabled(),
		"worker_logging_override": zeroEnabled(),
		"service_mesh":            zeroEnabled(),
		"mq_messages": types.ObjectValueMust(
			map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type},
			map[string]attr.Value{"base": types.Int64Value(0), "add_on": types.Int64Value(0)},
		),
		"mq_requests": types.ObjectValueMust(
			map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type},
			map[string]attr.Value{"base": types.Int64Value(0), "add_on": types.Int64Value(0)},
		),
		"gateways":      zeroAssigned(),
		"load_balancer": zeroAssigned(),
		"design_center": types.ObjectValueMust(
			map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType},
			map[string]attr.Value{"api": types.BoolValue(false), "mozart": types.BoolValue(false)},
		),
		"managed_gateway_small": types.ObjectValueMust(
			map[string]attr.Type{"assigned": types.Int64Type},
			map[string]attr.Value{"assigned": types.Int64Value(1)},
		),
		"managed_gateway_large": zeroAssigned(),
	}
	planObj, diags := types.ObjectValue(getEntitlementsAttributeTypes(), plan)
	if diags.HasError() {
		t.Fatalf("plan entitlements: %s", diags)
	}

	expanded, diags := expandEntitlements(ctx, planObj)
	if diags.HasError() {
		t.Fatalf("expand: %s", diags)
	}

	// `properties` is round-tripped from GetOrganization in the Update
	// handler; we simulate that here with the same `flow_designer` shape
	// the UI emits.
	req := &am.UpdateOrganizationRequest{
		ID:      "bc823f4d-8cdf-4040-8fc4-4abcd7f70d2d",
		Name:    "qwqq",
		OwnerID: "f7f43384-b33e-470c-ad4c-285aa0c01212",
		Properties: map[string]interface{}{
			"flow_designer": map[string]interface{}{},
		},
		Entitlements: expanded,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	t.Logf("provider PUT body:\n%s", got)

	// ── Envelope shape matches the UI contract ──────────────────────────
	for _, want := range []string{
		`"id":"bc823f4d-8cdf-4040-8fc4-4abcd7f70d2d"`,
		`"name":"qwqq"`,
		`"ownerId":"f7f43384-b33e-470c-ad4c-285aa0c01212"`,
		`"properties":{"flow_designer":{}}`,
		`"entitlements":{`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PUT body missing %s\n  got: %s", want, got)
		}
	}
	// parentOrganizationId is immutable on the server — the UI never sends it
	// and neither should we.
	if strings.Contains(got, `"parentOrganizationId"`) {
		t.Errorf("PUT body unexpectedly includes parentOrganizationId: %s", got)
	}

	// ── Exact managed_gateway wire format ───────────────────────────────
	// Must be AssignedEntitlement (no `reassigned` key).
	if !strings.Contains(got, `"managedGatewaySmall":{"assigned":1}`) {
		t.Errorf("managedGatewaySmall wire format wrong\n  got: %s", got)
	}
	if !strings.Contains(got, `"managedGatewayLarge":{"assigned":0}`) {
		t.Errorf("managedGatewayLarge wire format wrong\n  got: %s", got)
	}

	// ── Top-level booleans (always present, like the UI) ────────────────
	for _, want := range []string{
		`"createSubOrgs":true`,
		`"createEnvironments":false`,
		`"globalDeployment":false`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PUT body missing %s", want)
		}
	}

	// ── networkConnections uses VCoreEntitlement; when reassigned=0 it must
	//    serialise as {"assigned":0} per the UI contract (omitempty drops
	//    reassigned).
	if !strings.Contains(got, `"networkConnections":{"assigned":0}`) {
		t.Errorf("networkConnections wire format wrong\n  got: %s", got)
	}

	// ── static_ips and vpns must not appear — they are not provider-managed.
	if strings.Contains(got, `"staticIps"`) || strings.Contains(got, `"vpns"`) {
		t.Errorf("PUT body unexpectedly includes staticIps/vpns: %s", got)
	}
}

// TestUpdatePayload_BusinessGroupOmitsMasterOrgOnlyEntitlements pins the
// business-group PUT body to the exact shape the Anypoint UI sends, which
// is the ONLY shape the Access Management endpoint accepts for sub-orgs.
//
// Bug:
//  1. User declares `vcores_production = { assigned = 0 }` (and a few other
//     fields) on a business-group organization.
//  2. User bumps vCoresProduction to 1 via the Anypoint UI.
//  3. Terraform detects drift and tries to PUT back to 0.
//  4. Provider ships a "full" entitlements blob including
//     `hybrid:{enabled:false}`, `flexGateway:{enabled:false}`,
//     `runtimeFabric:false`, etc. — filled in from state via
//     UseStateForUnknown.
//  5. Server 403s: "Can not enable entitlement on a business group. It can
//     only be set for a master organization."
//
// Meanwhile the UI successfully PUTs the same reset with only the fields the
// user touched:
//
//	{"entitlements":{"globalDeployment":false,"createSubOrgs":false,
//	                 "createEnvironments":true,
//	                 "vCoresProduction":{"assigned":0},
//	                 "managedGatewaySmall":{"assigned":0}}}
//
// The fix is to build the PUT from req.Config (user HCL) rather than
// req.Plan (which carries UseStateForUnknown fills). Config-level null ==
// "no opinion" == omit from PUT. This test exercises expandEntitlements on
// a config object that only declares the three top-level booleans plus
// vcores_production and managed_gateway_small, and asserts the marshalled
// PUT body matches the UI's payload byte-for-byte on the entitlement set.
func TestUpdatePayload_BusinessGroupOmitsMasterOrgOnlyEntitlements(t *testing.T) {
	ctx := context.Background()

	// Config (what the user literally wrote in HCL) — master-org-only
	// entitlements are *not* declared so they must be absent from the PUT.
	configEntitlements := map[string]attr.Value{
		"create_sub_orgs":     types.BoolValue(false),
		"create_environments": types.BoolValue(true),
		"global_deployment":   types.BoolValue(false),
		// runtime_fabric intentionally null: user didn't declare it.
		"runtime_fabric": types.BoolNull(),
		"vcores_production": types.ObjectValueMust(
			getVCoreEntitlementAttributeTypes(),
			map[string]attr.Value{
				"assigned":   types.Float64Value(0),
				"reassigned": types.Float64Null(),
			},
		),
		"managed_gateway_small": types.ObjectValueMust(
			map[string]attr.Type{"assigned": types.Int64Type},
			map[string]attr.Value{"assigned": types.Int64Value(0)},
		),
		// All master-org-only / untouched entitlements are null — the user
		// didn't mention them in HCL.
		"vcores_sandbox":          types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vcores_design":           types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vpcs":                    types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"network_connections":     types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"hybrid":                  types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType}),
		"flex_gateway":            types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType}),
		"worker_logging_override": types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType}),
		"service_mesh":            types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType}),
		"mq_messages":             types.ObjectNull(map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type}),
		"mq_requests":             types.ObjectNull(map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type}),
		"gateways":                types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"load_balancer":           types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"design_center":           types.ObjectNull(map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType}),
		"managed_gateway_large":   types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
	}
	configObj, diags := types.ObjectValue(getEntitlementsAttributeTypes(), configEntitlements)
	if diags.HasError() {
		t.Fatalf("config entitlements: %s", diags)
	}

	expanded, diags := expandEntitlements(ctx, configObj)
	if diags.HasError() {
		t.Fatalf("expand: %s", diags)
	}

	req := &am.UpdateOrganizationRequest{
		ID:      "a02fab4f-4695-4325-882e-f326d1cef704",
		Name:    "terraform-suborg-example-renamed",
		OwnerID: "f7f43384-b33e-470c-ad4c-285aa0c01212",
		Properties: map[string]interface{}{
			"flow_designer": map[string]interface{}{},
		},
		Entitlements: expanded,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(body)
	t.Logf("provider PUT body (business-group):\n%s", got)

	// ── Expected on-wire entitlements (exact UI contract) ───────────────
	expectedPresent := []string{
		`"createSubOrgs":false`,
		`"createEnvironments":true`,
		`"globalDeployment":false`,
		`"vCoresProduction":{"assigned":0}`,
		`"managedGatewaySmall":{"assigned":0}`,
	}
	for _, want := range expectedPresent {
		if !strings.Contains(got, want) {
			t.Errorf("PUT body missing expected fragment %q\n  got: %s", want, got)
		}
	}

	// ── Master-org-only entitlements MUST NOT be present ───────────────
	// The server returns 403 on any business-group PUT that mentions these,
	// even with value false/zero.
	forbidden := []string{
		`"runtimeFabric"`,
		`"hybrid"`,
		`"flexGateway"`,
		`"serviceMesh"`,
		`"workerLoggingOverride"`,
	}
	for _, bad := range forbidden {
		if strings.Contains(got, bad) {
			t.Errorf("PUT body unexpectedly includes master-org-only entitlement %s\n  got: %s", bad, got)
		}
	}

	// ── Untouched quota entitlements MUST NOT be present either ────────
	// The user didn't declare them → config null → we should omit them so
	// the server preserves current values.
	undeclared := []string{
		`"vCoresSandbox"`,
		`"vCoresDesign"`,
		`"vpcs"`,
		`"networkConnections"`,
		`"mqMessages"`,
		`"mqRequests"`,
		`"gateways"`,
		`"loadBalancer"`,
		`"designCenter"`,
		`"managedGatewayLarge"`,
	}
	for _, bad := range undeclared {
		if strings.Contains(got, bad) {
			t.Errorf("PUT body unexpectedly includes undeclared entitlement %s\n  got: %s", bad, got)
		}
	}
}

// TestMergeEntitlements_PreservesPlanWhenApiOmitsMasterOrgFields covers the
// "Provider produced inconsistent result after apply" regression for sub-orgs.
//
// Bug trace:
//  1. User apply succeeds; PUT body (built from config) only contains the
//     fields the user declared, so the server is happy (no 403).
//  2. The PUT response contains the server's current view of the org.
//     For a business group that view typically OMITS `service_mesh` /
//     `flex_gateway` / `hybrid` / `worker_logging_override` because those are
//     master-org-only flags inherited from the parent.
//  3. If we overwrite state with just the PUT response's entitlements blob,
//     every previously-known value becomes null — but the plan (via
//     UseStateForUnknown) still carries the prior concrete values, so
//     Terraform fails with e.g. `.entitlements.service_mesh.enabled: was
//     cty.False, but now cty.True` on the next apply.
//
// The fix is the merge helper: for each entitlement attribute, the API value
// wins when it is concrete, and we fall back to the plan value when the API
// response omitted it (null). That keeps the post-apply state consistent
// with Terraform's computed plan while still picking up real server changes.
func TestMergeEntitlements_PreservesPlanWhenApiOmitsMasterOrgFields(t *testing.T) {
	ctx := context.Background()

	// Plan side: what Terraform computed before calling the API. Reflects the
	// prior state for fields the user didn't touch via UseStateForUnknown.
	planAttrs := map[string]attr.Value{
		"create_sub_orgs":     types.BoolValue(false),
		"create_environments": types.BoolValue(true),
		"global_deployment":   types.BoolValue(false),
		"runtime_fabric":      types.BoolNull(),
		"vcores_production": types.ObjectValueMust(
			getVCoreEntitlementAttributeTypes(),
			map[string]attr.Value{"assigned": types.Float64Value(0), "reassigned": types.Float64Value(0)},
		),
		"vcores_sandbox":      types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vcores_design":       types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vpcs":                types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"network_connections": types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"hybrid":              types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType}),
		// These two had concrete false in plan (e.g. carried forward via
		// UseStateForUnknown from an older zero-filled state). The merge must
		// preserve them because the API response omits them.
		"flex_gateway": types.ObjectValueMust(
			map[string]attr.Type{"enabled": types.BoolType},
			map[string]attr.Value{"enabled": types.BoolValue(false)},
		),
		"service_mesh": types.ObjectValueMust(
			map[string]attr.Type{"enabled": types.BoolType},
			map[string]attr.Value{"enabled": types.BoolValue(false)},
		),
		"worker_logging_override": types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType}),
		"mq_messages":             types.ObjectNull(map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type}),
		"mq_requests":             types.ObjectNull(map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type}),
		"gateways":                types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"load_balancer":           types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"design_center":           types.ObjectNull(map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType}),
		// User changed managed_gateway_small in this apply session.
		"managed_gateway_small": types.ObjectValueMust(
			map[string]attr.Type{"assigned": types.Int64Type},
			map[string]attr.Value{"assigned": types.Int64Value(1)},
		),
		"managed_gateway_large": types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
	}
	planObj, diags := types.ObjectValue(getEntitlementsAttributeTypes(), planAttrs)
	if diags.HasError() {
		t.Fatalf("plan obj: %s", diags)
	}

	// API response: the PUT body returned — reflects server's actual view but
	// OMITS the master-org-only entitlements on this business group.
	apiResponse := am.Entitlements{
		CreateSubOrgs:      false,
		CreateEnvironments: true,
		GlobalDeployment:   false,
		ManagedGatewaySmall: &am.AssignedEntitlement{
			Assigned: 1,
		},
		// flex_gateway, service_mesh, hybrid, worker_logging_override,
		// design_center intentionally nil — the server didn't return them.
	}
	apiObj, diags := flattenEntitlements(ctx, apiResponse)
	if diags.HasError() {
		t.Fatalf("flatten: %s", diags)
	}

	// Merge API response into plan. Plan values win when API is null; API wins
	// when concrete.
	merged := mergeEntitlementsPreservingPlan(ctx, planObj, apiObj, &diags)
	if diags.HasError() {
		t.Fatalf("merge: %s", diags)
	}

	// ── flex_gateway and service_mesh MUST carry the plan's false value
	//    through; otherwise Terraform fails with "inconsistent result after
	//    apply".
	for _, name := range []string{"flex_gateway", "service_mesh"} {
		obj, ok := merged.Attributes()[name].(types.Object)
		if !ok {
			t.Errorf("%s: not an Object", name)
			continue
		}
		if obj.IsNull() {
			t.Errorf("%s: merge should have preserved the plan's concrete value, got null", name)
			continue
		}
		got := obj.Attributes()["enabled"].(types.Bool).ValueBool()
		if got != false {
			t.Errorf("%s.enabled: want false from plan, got %v", name, got)
		}
	}

	// ── managed_gateway_small: API concrete value must win (it matches plan
	//    here, but in general a drift-detection update needs the API value to
	//    override plan).
	gwSmall := merged.Attributes()["managed_gateway_small"].(types.Object)
	if gwSmall.IsNull() {
		t.Fatal("managed_gateway_small: API returned concrete value, merge should not be null")
	}
	if gwSmall.Attributes()["assigned"].(types.Int64).ValueInt64() != 1 {
		t.Errorf("managed_gateway_small.assigned: want 1 from API, got %v", gwSmall.Attributes()["assigned"])
	}

	// ── Top-level booleans: API wins (they're always present).
	if merged.Attributes()["create_environments"].(types.Bool).ValueBool() != true {
		t.Errorf("create_environments: want true from API")
	}
}

// TestMergeEntitlements_PreservesPlanNullAgainstConcreteApi covers the second
// regression slug:
//
//	Provider produced inconsistent result after apply ...
//	.entitlements.flex_gateway: was null, but now cty.ObjectVal({"enabled":true})
//
// This happens when the user has not declared flex_gateway / service_mesh in
// HCL and the prior state for that attribute is null (because flatten now
// stores null for omitted enabled-style entitlements rather than zero-filling).
// Terraform's planning then proposes plan = null. If the server's PUT response
// returns a concrete value for that field — which IS what happens on
// business-group orgs because the master inherits an `enabled:true` value —
// the merge must NOT let the concrete API value through; doing so violates
// Terraform's "post-apply equals plan" contract. The next refresh will pick up
// the server's value and the following plan will see it via UseStateForUnknown.
func TestMergeEntitlements_PreservesPlanNullAgainstConcreteApi(t *testing.T) {
	ctx := context.Background()

	// Plan attributes — user did NOT declare flex_gateway or service_mesh,
	// state was null (post-fix flatten leaves null for omitted enabled-style
	// entitlements), so Terraform's plan is null for them.
	enabledType := map[string]attr.Type{"enabled": types.BoolType}
	planAttrs := map[string]attr.Value{
		"create_sub_orgs":         types.BoolValue(false),
		"create_environments":     types.BoolValue(true),
		"global_deployment":       types.BoolValue(false),
		"runtime_fabric":          types.BoolNull(),
		"vcores_production":       types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vcores_sandbox":          types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vcores_design":           types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"vpcs":                    types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"network_connections":     types.ObjectNull(getVCoreEntitlementAttributeTypes()),
		"hybrid":                  types.ObjectNull(enabledType),
		"flex_gateway":            types.ObjectNull(enabledType), // ← plan null
		"worker_logging_override": types.ObjectNull(enabledType),
		"service_mesh":            types.ObjectNull(enabledType), // ← plan null
		"mq_messages":             types.ObjectNull(map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type}),
		"mq_requests":             types.ObjectNull(map[string]attr.Type{"base": types.Int64Type, "add_on": types.Int64Type}),
		"gateways":                types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"load_balancer":           types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"design_center":           types.ObjectNull(map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType}),
		"managed_gateway_small":   types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
		"managed_gateway_large":   types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type}),
	}
	planObj, diags := types.ObjectValue(getEntitlementsAttributeTypes(), planAttrs)
	if diags.HasError() {
		t.Fatalf("plan obj: %s", diags)
	}

	// API response includes flex_gateway and service_mesh as concrete
	// `{enabled:true}` (master-org-inherited). This is exactly what triggered
	// the user's "was null, but now cty.ObjectVal(...)" diagnostic.
	apiResponse := am.Entitlements{
		CreateSubOrgs:      false,
		CreateEnvironments: true,
		GlobalDeployment:   false,
		FlexGateway:        &am.EnabledEntitlement{Enabled: true},
		ServiceMesh:        &am.EnabledEntitlement{Enabled: true},
	}
	apiObj, diags := flattenEntitlements(ctx, apiResponse)
	if diags.HasError() {
		t.Fatalf("flatten: %s", diags)
	}

	merged := mergeEntitlementsPreservingPlan(ctx, planObj, apiObj, &diags)
	if diags.HasError() {
		t.Fatalf("merge: %s", diags)
	}

	// flex_gateway and service_mesh MUST stay null — Terraform's post-apply
	// must equal plan even though the server returned a concrete value.
	for _, name := range []string{"flex_gateway", "service_mesh"} {
		obj, ok := merged.Attributes()[name].(types.Object)
		if !ok {
			t.Errorf("%s: not an Object", name)
			continue
		}
		if !obj.IsNull() {
			t.Errorf(
				"%s: plan was null but merge let the concrete API value through (%v); "+
					"Terraform will reject with 'was null, but now cty.ObjectVal(...)'",
				name, obj,
			)
		}
	}
}

// TestFlatten_ServerOmitsManagedGatewayLarge verifies behaviour when the server
// omits managed_gateway_large from a GET response (e.g. treating 0 as default).
//
// The fix in this commit makes flatten return a concrete {assigned:0} object
// rather than null. A null state against a `{assigned = 0}` HCL config would
// cause perpetual drift on every plan.
func TestFlatten_ServerOmitsManagedGatewayLarge(t *testing.T) {
	ctx := context.Background()
	serverSide := am.Entitlements{
		CreateEnvironments: true,
		// ManagedGatewayLarge intentionally nil — simulate omitted
	}
	stateObj, diags := flattenEntitlements(ctx, serverSide)
	if diags.HasError() {
		t.Fatalf("flatten diagnostics: %s", diags)
	}
	gwLarge := stateObj.Attributes()["managed_gateway_large"].(types.Object)
	if gwLarge.IsNull() {
		t.Fatalf("flatten should return a concrete zero-valued Object for an omitted managed_gateway_large, got null")
	}
	if gwLarge.IsUnknown() {
		t.Fatalf("flatten should return a known Object, got unknown")
	}
	assigned := gwLarge.Attributes()["assigned"].(types.Int64).ValueInt64()
	if assigned != 0 {
		t.Errorf("flatten should default omitted managed_gateway_large.assigned to 0, got %d", assigned)
	}

	// Quota-style entitlements (assigned / base+addOn) must still zero-fill
	// when the server omits them — that's the fix for the original
	// `managed_gateway_large.assigned = 0` drift bug. A concrete {assigned:0}
	// in state matches a user HCL `= { assigned = 0 }` and stops the
	// perpetual-diff loop.
	zeroFillChecks := map[string]map[string]attr.Value{
		"gateways":              {"assigned": types.Int64Value(0)},
		"load_balancer":         {"assigned": types.Int64Value(0)},
		"managed_gateway_small": {"assigned": types.Int64Value(0)},
		"managed_gateway_large": {"assigned": types.Int64Value(0)},
	}
	for name, want := range zeroFillChecks {
		obj, ok := stateObj.Attributes()[name].(types.Object)
		if !ok {
			t.Errorf("%s: not an Object", name)
			continue
		}
		if obj.IsNull() {
			t.Errorf("%s: expected concrete zero-value Object, got null", name)
			continue
		}
		for k, v := range want {
			if got := obj.Attributes()[k]; !got.Equal(v) {
				t.Errorf("%s.%s: want %v, got %v", name, k, v, got)
			}
		}
	}

	// Enabled-style entitlements (hybrid / flex_gateway / worker_logging_override
	// / service_mesh) and design_center MUST be null when the server omits
	// them. On sub-orgs these are typically inherited from the master org and
	// may be returned as `{enabled:true}` on a PUT response; hardcoding false
	// here would cause Terraform's "was cty.False, but now cty.True"
	// inconsistent-result diagnostic on apply.
	nullChecks := []string{
		"hybrid",
		"flex_gateway",
		"worker_logging_override",
		"service_mesh",
		"design_center",
	}
	for _, name := range nullChecks {
		obj, ok := stateObj.Attributes()[name].(types.Object)
		if !ok {
			t.Errorf("%s: not an Object", name)
			continue
		}
		if !obj.IsNull() {
			t.Errorf("%s: expected null Object when server omits, got %v", name, obj)
		}
	}
}

// TestOrganizationResource_EntitlementsSchema_DefaultsAndOptionality asserts
// the schema-level contract that `entitlements` (and every sub-attribute the
// Anypoint API does not require) is Optional+Computed, with 0/false defaults
// so users can omit them from HCL without Terraform complaining.
//
// Guards against regressions of the UX fix: "entitlements is not required in
// the API call, so it shouldn't be required in the provider schema either".
func TestOrganizationResource_EntitlementsSchema_DefaultsAndOptionality(t *testing.T) {
	ctx := context.Background()
	res := NewOrganizationResource().(*OrganizationResource)

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}

	entAttr, ok := schemaResp.Schema.Attributes["entitlements"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("entitlements: expected SingleNestedAttribute, got %T", schemaResp.Schema.Attributes["entitlements"])
	}
	if !entAttr.IsOptional() {
		t.Errorf("entitlements must be Optional so users can omit the whole block")
	}
	if !entAttr.IsComputed() {
		t.Errorf("entitlements must be Computed so the server-reported view populates state when omitted")
	}
	if entAttr.IsRequired() {
		t.Errorf("entitlements must not be Required")
	}

	// Top-level booleans: Optional+Computed with Default(false).
	for _, name := range []string{"create_sub_orgs", "create_environments", "global_deployment"} {
		a, ok := entAttr.Attributes[name].(schema.BoolAttribute)
		if !ok {
			t.Errorf("entitlements.%s: expected BoolAttribute, got %T", name, entAttr.Attributes[name])
			continue
		}
		if !a.IsOptional() || !a.IsComputed() || a.IsRequired() {
			t.Errorf("entitlements.%s: want Optional+Computed, got Optional=%t Computed=%t Required=%t",
				name, a.IsOptional(), a.IsComputed(), a.IsRequired())
		}
		if a.Default == nil {
			t.Errorf("entitlements.%s: missing Default (expected booldefault.StaticBool(false))", name)
			continue
		}
		defResp := defaults.BoolResponse{}
		a.Default.DefaultBool(ctx, defaults.BoolRequest{}, &defResp)
		if defResp.PlanValue.ValueBool() != false {
			t.Errorf("entitlements.%s: Default should be false, got %v", name, defResp.PlanValue)
		}
	}

	// vCore quotas: assigned must be Optional+Computed with Default(0).
	for _, name := range []string{"vcores_production", "vcores_sandbox", "vcores_design", "vpcs", "network_connections"} {
		nested, ok := entAttr.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Errorf("entitlements.%s: expected SingleNestedAttribute, got %T", name, entAttr.Attributes[name])
			continue
		}
		assigned, ok := nested.Attributes["assigned"].(schema.Float64Attribute)
		if !ok {
			t.Errorf("entitlements.%s.assigned: expected Float64Attribute, got %T", name, nested.Attributes["assigned"])
			continue
		}
		if !assigned.IsOptional() || !assigned.IsComputed() || assigned.IsRequired() {
			t.Errorf("entitlements.%s.assigned: want Optional+Computed, got Optional=%t Computed=%t Required=%t",
				name, assigned.IsOptional(), assigned.IsComputed(), assigned.IsRequired())
		}
		if assigned.Default == nil {
			t.Errorf("entitlements.%s.assigned: missing Default (expected float64default.StaticFloat64(0))", name)
			continue
		}
		fr := defaults.Float64Response{}
		assigned.Default.DefaultFloat64(ctx, defaults.Float64Request{}, &fr)
		if fr.PlanValue.ValueFloat64() != 0 {
			t.Errorf("entitlements.%s.assigned: Default should be 0, got %v", name, fr.PlanValue)
		}
	}

	// MQ entitlements: base & add_on must be Optional+Computed with Default(0).
	for _, name := range []string{"mq_messages", "mq_requests"} {
		nested, ok := entAttr.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Errorf("entitlements.%s: expected SingleNestedAttribute, got %T", name, entAttr.Attributes[name])
			continue
		}
		for _, field := range []string{"base", "add_on"} {
			ia, ok := nested.Attributes[field].(schema.Int64Attribute)
			if !ok {
				t.Errorf("entitlements.%s.%s: expected Int64Attribute, got %T", name, field, nested.Attributes[field])
				continue
			}
			if !ia.IsOptional() || !ia.IsComputed() || ia.IsRequired() {
				t.Errorf("entitlements.%s.%s: want Optional+Computed, got Optional=%t Computed=%t Required=%t",
					name, field, ia.IsOptional(), ia.IsComputed(), ia.IsRequired())
			}
			if ia.Default == nil {
				t.Errorf("entitlements.%s.%s: missing Default (expected int64default.StaticInt64(0))", name, field)
				continue
			}
			ir := defaults.Int64Response{}
			ia.Default.DefaultInt64(ctx, defaults.Int64Request{}, &ir)
			if ir.PlanValue.ValueInt64() != 0 {
				t.Errorf("entitlements.%s.%s: Default should be 0, got %v", name, field, ir.PlanValue)
			}
		}
	}
}

// TestExpandEntitlements_NullObjectYieldsZeroPayload asserts that passing a
// null entitlements Object (the shape the plan carries when the user omits
// the whole `entitlements = { ... }` block in HCL) produces a zero-valued
// Entitlements struct — not a panic, not nil — and that the JSON payload the
// Anypoint POST/PUT receives is `{create_sub_orgs:false,...}` without any
// master-org-only fields like `runtime_fabric`.
func TestExpandEntitlements_NullObjectYieldsZeroPayload(t *testing.T) {
	ctx := context.Background()
	null := types.ObjectNull(getEntitlementsAttributeTypes())

	ent, diags := expandEntitlements(ctx, null)
	if diags.HasError() {
		t.Fatalf("expandEntitlements(null) diagnostics: %s", diags)
	}
	if ent.CreateSubOrgs || ent.CreateEnvironments || ent.GlobalDeployment {
		t.Errorf("expected all top-level booleans to default to false when entitlements is null, got %+v", ent)
	}
	if ent.RuntimeFabric != nil {
		t.Errorf("runtime_fabric must stay nil when user doesn't declare entitlements (master-org-only attribute); got %v", *ent.RuntimeFabric)
	}
	// Nested entitlements should all be nil so omitempty drops them.
	for name, v := range map[string]interface{}{
		"vcores_production":     ent.VCoresProduction,
		"vcores_sandbox":        ent.VCoresSandbox,
		"managed_gateway_large": ent.ManagedGatewayLarge,
		"hybrid":                ent.Hybrid,
		"flex_gateway":          ent.FlexGateway,
		"design_center":         ent.DesignCenter,
		"mq_messages":           ent.MqMessages,
	} {
		if !reflect.ValueOf(v).IsNil() {
			t.Errorf("entitlements.%s: expected nil for null input, got %v", name, v)
		}
	}

	// Serialise to JSON and assert the payload is the minimal shape.
	body, err := json.Marshal(am.CreateOrganizationRequest{
		Name: "x", ParentOrganizationID: "p", OwnerID: "o", Entitlements: ent,
	})
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	got := string(body)
	// Master-org-only / nested entitlements must all be omitted by `omitempty`.
	forbidden := []string{
		"runtimeFabric", "vCoresProduction", "managedGatewayLarge",
		"hybrid", "flexGateway", "designCenter", "mqMessages", "vpns", "staticIps",
	}
	for _, f := range forbidden {
		if strings.Contains(got, f) {
			t.Errorf("JSON payload should omit %q when entitlements is null; got:\n%s", f, got)
		}
	}
	for _, required := range []string{`"createSubOrgs":false`, `"createEnvironments":false`, `"globalDeployment":false`} {
		if !strings.Contains(got, required) {
			t.Errorf("JSON payload missing %q; got:\n%s", required, got)
		}
	}
}

// TestOrganizationResource_ImportState_PassthroughID verifies that running
// `terraform import anypoint_organization.<name> <org-id>` seeds state with
// the given ID so the framework's follow-up Read() call can hydrate the rest
// of the attributes from the Anypoint API.
func TestOrganizationResource_ImportState_PassthroughID(t *testing.T) {
	ctx := context.Background()
	res := NewOrganizationResource()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %s", schemaResp.Diagnostics)
	}
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	importable, ok := res.(resource.ResourceWithImportState)
	if !ok {
		t.Fatal("OrganizationResource does not implement ResourceWithImportState")
	}

	req := resource.ImportStateRequest{ID: "a02fab4f-4695-4325-882e-f326d1cef704"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(stateType, nil),
		},
	}

	importable.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState diagnostics: %s", resp.Diagnostics)
	}

	var got OrganizationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("resp.State.Get: %s", diags)
	}
	if got.ID.ValueString() != req.ID {
		t.Errorf("state.id: want %q, got %q", req.ID, got.ID.ValueString())
	}
}

// TestOrganizationResource_Read_DerivesParentOrgIDOnImport asserts that the
// first Read() after `terraform import` (when state.parent_organization_id is
// null) populates the attribute from the tail of the server's
// parentOrganizationIds chain. Without this, the next `terraform plan` would
// show `null -> "<user value>"`, which — combined with the RequiresReplace
// plan modifier on parent_organization_id — would destroy+recreate the just-
// imported resource.
func TestOrganizationResource_Read_DerivesParentOrgIDOnImport(t *testing.T) {
	const (
		orgID          = "a02fab4f-4695-4325-882e-f326d1cef704"
		masterParent   = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
		intermediateID = "11111111-2222-3333-4444-555555555555"
	)

	// Server returns the full ancestor chain root-to-parent; the tail is the
	// immediate parent the user declares as `parent_organization_id` in HCL.
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/" + orgID: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "unexpected method "+r.Method)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                    orgID,
				"name":                  "imported-sub-org",
				"createdAt":             "2026-04-29T10:00:00.000Z",
				"updatedAt":             "2026-04-29T11:00:00.000Z",
				"ownerId":               "owner-uuid",
				"clientId":              "client-uuid",
				"idprovider_id":         "idp",
				"isFederated":           false,
				"parentOrganizationIds": []string{masterParent, intermediateID},
				"subOrganizationIDs":    []string{},
				"tenantOrganizationIds": []string{},
				"mfaRequired":           "",
				"orgType":               "business",
				"gdotId":                nil,
				"deletedAt":             nil,
				"domain":                nil,
				"isRoot":                false,
				"isMaster":              false,
				"properties":            map[string]interface{}{},
				"entitlements":          map[string]interface{}{},
				"subscription":          map[string]interface{}{},
			})
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}
	server := testutil.MockHTTPServer(t, handlers)

	orgClient := &am.OrganizationClient{
		UserAnypointClient: &anypointclient.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      masterParent,
		},
	}
	orgResource := &OrganizationResource{client: orgClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build a post-import state: only `id` is known, everything else null —
	// this is exactly what ImportStatePassthroughID produces.
	importedState := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(stateType, nil),
	}
	if setDiags := importedState.SetAttribute(ctx, path.Root("id"), orgID); setDiags.HasError() {
		t.Fatalf("seed id into state: %s", setDiags)
	}

	req := resource.ReadRequest{State: importedState}
	resp := &resource.ReadResponse{State: importedState}

	orgResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %s", resp.Diagnostics)
	}

	var got OrganizationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("resp.State.Get: %s", diags)
	}

	if got.ParentOrganizationID.ValueString() != intermediateID {
		t.Errorf("state.parent_organization_id: want immediate parent %q (tail of chain), got %q",
			intermediateID, got.ParentOrganizationID.ValueString())
	}
	if got.Name.ValueString() != "imported-sub-org" {
		t.Errorf("state.name: want %q, got %q", "imported-sub-org", got.Name.ValueString())
	}
	if got.OwnerID.ValueString() != "owner-uuid" {
		t.Errorf("state.owner_id: want %q, got %q", "owner-uuid", got.OwnerID.ValueString())
	}
}

// TestOrganizationResource_Read_PreservesExistingParentOrgID guards the second
// half of the contract: on a *regular* refresh (after Create), state already
// carries parent_organization_id as the user declared it. The server's chain
// tail may or may not equal that value, but either way Read must leave the
// state value alone — rewriting it would cause a RequiresReplace churn.
func TestOrganizationResource_Read_PreservesExistingParentOrgID(t *testing.T) {
	const (
		orgID      = "a02fab4f-4695-4325-882e-f326d1cef704"
		userParent = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
		tailID     = "99999999-9999-9999-9999-999999999999"
	)

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/" + orgID: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                    orgID,
				"name":                  "stable-sub-org",
				"createdAt":             "2026-04-29T10:00:00.000Z",
				"updatedAt":             "2026-04-29T11:00:00.000Z",
				"ownerId":               "owner-uuid",
				"clientId":              "client-uuid",
				"idprovider_id":         "idp",
				"isFederated":           false,
				"parentOrganizationIds": []string{userParent, tailID}, // tail differs on purpose
				"subOrganizationIDs":    []string{},
				"tenantOrganizationIds": []string{},
				"mfaRequired":           "",
				"orgType":               "business",
				"gdotId":                nil,
				"deletedAt":             nil,
				"domain":                nil,
				"isRoot":                false,
				"isMaster":              false,
				"properties":            map[string]interface{}{},
				"entitlements":          map[string]interface{}{},
				"subscription":          map[string]interface{}{},
			})
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}
	server := testutil.MockHTTPServer(t, handlers)

	orgClient := &am.OrganizationClient{
		UserAnypointClient: &anypointclient.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      userParent,
		},
	}
	orgResource := &OrganizationResource{client: orgClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	orgResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorState := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)}
	for p, v := range map[string]interface{}{
		"id":                     orgID,
		"parent_organization_id": userParent,
	} {
		if diags := priorState.SetAttribute(ctx, path.Root(p), v); diags.HasError() {
			t.Fatalf("seed %s: %s", p, diags)
		}
	}

	req := resource.ReadRequest{State: priorState}
	resp := &resource.ReadResponse{State: priorState}

	orgResource.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %s", resp.Diagnostics)
	}

	var got OrganizationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("resp.State.Get: %s", diags)
	}
	if got.ParentOrganizationID.ValueString() != userParent {
		t.Errorf("state.parent_organization_id: Read must NOT overwrite a value the user set via Create; want %q, got %q",
			userParent, got.ParentOrganizationID.ValueString())
	}
}
