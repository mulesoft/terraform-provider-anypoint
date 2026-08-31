package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewManagedOmniGatewayResource(t *testing.T) {
	r := NewManagedOmniGatewayResource()
	if r == nil {
		t.Error("NewManagedOmniGatewayResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("ManagedOmniGatewayResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("ManagedOmniGatewayResource should implement ResourceWithImportState")
	}
}

func TestManagedOmniGatewayResource_Metadata(t *testing.T) {
	r := NewManagedOmniGatewayResource()
	testutil.TestResourceMetadata(t, r, "_managed_omni_gateway")
}

func TestManagedOmniGatewayResource_Schema(t *testing.T) {
	r := NewManagedOmniGatewayResource()
	requiredAttrs := []string{"name", "environment_id", "target_id"}
	optionalAttrs := []string{"organization_id", "runtime_version", "release_channel", "size", "ingress", "properties", "logging", "tracing"}
	computedAttrs := []string{"id", "status", "organization_id", "target_type"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestManagedOmniGatewayResource_Configure(t *testing.T) {
	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	testutil.TestResourceConfigure(t, res, providerData)
	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestManagedOmniGatewayResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: "invalid"}
	resp := &resource.ConfigureResponse{}
	res.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestManagedOmniGatewayResource_ImportState(t *testing.T) {
	r := NewManagedOmniGatewayResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestManagedOmniGatewayResourceModel_Validation(t *testing.T) {
	model := ManagedOmniGatewayResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.TargetID
	_ = model.RuntimeVersion
	_ = model.ReleaseChannel
	_ = model.Size
	_ = model.Status
	_ = model.Ingress
	_ = model.Properties
	_ = model.Logging
	_ = model.Tracing
}

func TestManagedOmniGatewayResource_Read(t *testing.T) {
	basePath := "/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/test-gw-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-gw-id",
				"name":           "test-gateway",
				"targetId":       "test-target-id",
				"runtimeVersion": "1.6.0",
				"releaseChannel": "LTS",
				"size":           "SMALL",
				"status":         "Active",
				"configuration": map[string]interface{}{
					"ingress":    map[string]interface{}{"publicUrl": "", "internalUrl": ""},
					"properties": map[string]interface{}{},
					"logging":    map[string]interface{}{},
					"tracing":    map[string]interface{}{},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	res.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-gw-id"),
		"name":            tftypes.NewValue(tftypes.String, "test-gateway"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, "test-target-id"),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, "1.6.0"),
		"release_channel": tftypes.NewValue(tftypes.String, "LTS"),
		"size":            tftypes.NewValue(tftypes.String, "SMALL"),
		"status":          tftypes.NewValue(tftypes.String, "Active"),
		"ingress":         tftypes.NewValue(ingressObjType, nil),
		"properties":      tftypes.NewValue(propertiesObjType, nil),
		"logging":         tftypes.NewValue(loggingObjType, nil),
		"tracing":         tftypes.NewValue(tracingObjType, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got ManagedOmniGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-gateway" {
		t.Errorf("Expected Name 'test-gateway', got %s", got.Name.ValueString())
	}
}

func TestManagedOmniGatewayResource_Read_NotFound(t *testing.T) {
	basePath := "/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/test-gw-id"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	res.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-gw-id"),
		"name":            tftypes.NewValue(tftypes.String, "test-gateway"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, "test-target-id"),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, "1.6.0"),
		"release_channel": tftypes.NewValue(tftypes.String, "LTS"),
		"size":            tftypes.NewValue(tftypes.String, "SMALL"),
		"status":          tftypes.NewValue(tftypes.String, "Active"),
		"ingress":         tftypes.NewValue(ingressObjType, nil),
		"properties":      tftypes.NewValue(propertiesObjType, nil),
		"logging":         tftypes.NewValue(loggingObjType, nil),
		"tracing":         tftypes.NewValue(tracingObjType, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

// TestManagedOmniGatewayResource_Update_RetainsPriorStatusOnAsyncTransition verifies
// that after a successful PUT the provider writes the prior state status (e.g. RUNNING)
// rather than the transient async status (e.g. APPLYING) returned by the API. This
// prevents "provider produced inconsistent result" errors from the Terraform framework.
func TestManagedOmniGatewayResource_Update_RetainsPriorStatusOnAsyncTransition(t *testing.T) {
	putPath := "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/test-gw-id"
	domainPath := "/runtimefabric/api/organizations/test-org-id/targets/test-target-id/environments/test-env-id/domains"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		putPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			// Platform accepted the size change and re-entered async provisioning.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-gw-id",
				"name":           "test-gateway",
				"targetId":       "test-target-id",
				"runtimeVersion": "1.6.0",
				"releaseChannel": "lts",
				"size":           "large",
				"status":         "APPLYING",
				"configuration": map[string]interface{}{
					"ingress": map[string]interface{}{
						"publicUrl":         "https://test-gateway.example.com",
						"internalUrl":       "https://test-gateway.internal.example.com",
						"forwardSslSession": true,
						"lastMileSecurity":  true,
					},
					"properties": map[string]interface{}{
						"upstreamResponseTimeout": 15,
						"connectionIdleTimeout":   60,
					},
					"logging": map[string]interface{}{
						"level":       "info",
						"forwardLogs": true,
					},
					"tracing": map[string]interface{}{
						"enabled":  false,
						"sampling": 1,
					},
				},
			})
		},
		domainPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"domains": []string{"*.example.com"},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	res.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	buildState := func(size, status string) tftypes.Value {
		return tftypes.NewValue(stateType, map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "test-gw-id"),
			"name":            tftypes.NewValue(tftypes.String, "test-gateway"),
			"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
			"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
			"target_id":       tftypes.NewValue(tftypes.String, "test-target-id"),
			"target_type":     tftypes.NewValue(tftypes.String, nil),
			"runtime_version": tftypes.NewValue(tftypes.String, "1.6.0"),
			"release_channel": tftypes.NewValue(tftypes.String, "lts"),
			"size":            tftypes.NewValue(tftypes.String, size),
			"status":          tftypes.NewValue(tftypes.String, status),
			"ingress":         tftypes.NewValue(ingressObjType, nil),
			"properties":      tftypes.NewValue(propertiesObjType, nil),
			"logging":         tftypes.NewValue(loggingObjType, nil),
			"tracing":         tftypes.NewValue(tracingObjType, nil),
		})
	}

	priorStateRaw := buildState("small", "RUNNING")
	planRaw := buildState("large", "RUNNING") // plan inherits RUNNING via UseStateForUnknown

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() reported unexpected errors: %v", resp.Diagnostics.Errors())
	}

	var got ManagedOmniGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.Size.ValueString() != "large" {
		t.Errorf("Expected Size 'large', got %q", got.Size.ValueString())
	}
	// The provider must write RUNNING (prior state), not APPLYING (async API transition).
	if got.Status.ValueString() != "RUNNING" {
		t.Errorf("Expected Status 'RUNNING' (prior state retained), got %q — async status race not handled", got.Status.ValueString())
	}
}

// TestManagedOmniGatewayResource_Create_SharedSpace is the regression test for
// W-23490209. The original bug: Create always derived ingress URLs from the
// target's domains, but a shared space (CloudHub 2.0) exposes NO domains — the
// domains endpoint returns HTTP 200 with an empty array — so the len(derived)==0
// guard aborted the create before the POST ever fired. Shared-space gateways
// could never be created.
//
// The fix: detect the target type up front via /targets, and for a shared space
// skip domain derivation entirely, letting the platform assign the public URL.
//
// This test asserts:
//  1. Create does NOT error for a shared-space target with no domains.
//  2. GetDomains is NEVER called for a shared-space target (derivation bypassed).
//  3. The platform-assigned public_url round-trips into state.
//  4. target_type resolves to "shared-space" even though the create (api/v1)
//     response omits targetType — it falls back to the detected value.
func TestManagedOmniGatewayResource_Create_SharedSpace(t *testing.T) {
	const sharedRegionSlug = "cloudhub-us-east-1"
	const platformAssignedURL = "https://shared-gw-x7k2p9.usa-e2.stgx.cloudhub.io"

	createPath := "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways"
	targetsPath := "/runtimefabric/api/organizations/test-org-id/targets"
	// If the provider ever tried to derive URLs for a shared space it would GET
	// this path. Registering it as a hard failure locks in the "skip derivation"
	// behavior — the crux of the fix.
	domainsPath := "/runtimefabric/api/organizations/test-org-id/targets/" + sharedRegionSlug + "/environments/test-env-id/domains"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		targetsPath: func(w http.ResponseWriter, r *http.Request) {
			// Bare JSON array, mixing a private space (UUID) and the shared space
			// (region slug) — the exact shape returned live by stgx.
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{"id": "0a1b2c3d-1111-2222-3333-444455556666", "name": "some-private-space", "type": "private-space"},
				{"id": sharedRegionSlug, "name": "US East (N. Virginia)", "type": "shared-space"},
			})
		},
		domainsPath: func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("GetDomains must NOT be called for a shared-space target — URL derivation should be bypassed")
			// Return the empty-domains payload the live shared-space endpoint sends,
			// so that if the guard regressed we'd reproduce the original abort.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{"domains": []string{}})
		},
		createPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			// The create (api/v1) response for a shared space: the platform has
			// assigned a random-slug public URL, there is NO internal URL, and
			// targetType is omitted (verified live).
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
				"id":             "gw-shared-1",
				"name":           "shared-gw",
				"targetId":       sharedRegionSlug,
				"runtimeVersion": "1.9.9",
				"releaseChannel": "lts",
				"size":           "small",
				"status":         "PROVISIONING",
				"configuration": map[string]interface{}{
					"ingress":    map[string]interface{}{"publicUrl": platformAssignedURL},
					"properties": map[string]interface{}{},
					"logging":    map[string]interface{}{},
					"tracing":    map[string]interface{}{},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	res.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	// runtime_version is pre-set so Create skips the versions lookup.
	//
	// The nested blocks (ingress/properties/logging/tracing) are Optional+Computed
	// with UseStateForUnknown. When the user OMITS them from the config — the exact
	// shared-space scenario — the framework hands Create an UNKNOWN value for each
	// (there is no prior state to copy on Create). This mirrors that reality: seeding
	// them as tftypes.UnknownValue rather than nil reproduces the real apply. Seeding
	// nil (a concrete null) would mask the bug where reconcileTracing returned the
	// unknown plan value, leaving a Computed attribute unknown after apply.
	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":            tftypes.NewValue(tftypes.String, "shared-gw"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, sharedRegionSlug),
		"target_type":     tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"runtime_version": tftypes.NewValue(tftypes.String, "1.9.9"),
		"release_channel": tftypes.NewValue(tftypes.String, "lts"),
		"size":            tftypes.NewValue(tftypes.String, "small"),
		"status":          tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"ingress":         tftypes.NewValue(ingressObjType, tftypes.UnknownValue),
		"properties":      tftypes.NewValue(propertiesObjType, tftypes.UnknownValue),
		"logging":         tftypes.NewValue(loggingObjType, tftypes.UnknownValue),
		"tracing":         tftypes.NewValue(tracingObjType, tftypes.UnknownValue),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: planRaw}}
	res.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors for a shared-space target (the original bug): %v", resp.Diagnostics.Errors())
	}

	var got ManagedOmniGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.ID.ValueString() != "gw-shared-1" {
		t.Errorf("Expected ID 'gw-shared-1', got %q", got.ID.ValueString())
	}
	if got.TargetType.ValueString() != "shared-space" {
		t.Errorf("Expected target_type 'shared-space' (from detection fallback), got %q", got.TargetType.ValueString())
	}
	if got.Ingress.IsNull() {
		t.Fatal("Ingress should not be null after create")
	}
	publicURL := got.Ingress.Attributes()["public_url"].(types.String).ValueString()
	if publicURL != platformAssignedURL {
		t.Errorf("Expected platform-assigned public_url %q to round-trip, got %q", platformAssignedURL, publicURL)
	}

	// Post-apply invariant — the crux of this regression. The Terraform framework
	// rejects any state whose Computed attributes are still unknown after apply
	// with "Provider returned invalid result object after apply". Because the
	// user omitted `tracing` (a Computed nested object), reconcileTracing must
	// resolve it to a KNOWN value, not echo back the unknown plan value. Assert
	// EVERY value in the returned state is fully known — this catches `tracing`
	// (the field that regressed) and any sibling that ever drifts unknown.
	if resp.State.Raw.IsFullyKnown() {
		// good — every attribute settled to a concrete value
	} else {
		// Pinpoint the offending attribute(s) so a future regression is obvious.
		var offenders []string
		for _, name := range []string{"ingress", "properties", "logging", "tracing"} {
			var obj types.Object
			switch name {
			case "ingress":
				obj = got.Ingress
			case "properties":
				obj = got.Properties
			case "logging":
				obj = got.Logging
			case "tracing":
				obj = got.Tracing
			}
			if obj.IsUnknown() {
				offenders = append(offenders, name)
			}
		}
		t.Fatalf("state has unknown values after Create (framework would reject with "+
			"\"invalid result object after apply\"); unknown nested object(s): %v", offenders)
	}
	// Belt-and-suspenders: tracing specifically must be known AND carry its
	// computed defaults (enabled=false, sampling=1) from the flattened API value.
	if got.Tracing.IsUnknown() || got.Tracing.IsNull() {
		t.Fatalf("tracing must be a known, non-null object after Create; got unknown=%v null=%v",
			got.Tracing.IsUnknown(), got.Tracing.IsNull())
	}
	if s := got.Tracing.Attributes()["sampling"].(types.Int64).ValueInt64(); s != 1 {
		t.Errorf("expected computed tracing.sampling default 1, got %d", s)
	}
}

// TestManagedOmniGatewayResource_Create_PrivateSpace is the companion regression
// test to _Create_SharedSpace. Because the shared-space fix added a branch to the
// SHARED code path in Create, this test locks in that the PRIVATE-space path is
// unchanged: the provider must still resolve the target as private-space, fetch
// its domains, derive the public/internal ingress URLs, and send them in the POST
// body. (A shared-space regression that accidentally skipped derivation for every
// target would slip past _Create_SharedSpace but fail here.)
//
// This test asserts:
//  1. Create does NOT error for a private-space target.
//  2. GetDomains IS called (derivation path taken).
//  3. The derived public_url AND internal_url land in the POST request body.
//  4. target_type resolves to "private-space".
func TestManagedOmniGatewayResource_Create_PrivateSpace(t *testing.T) {
	const privateSpaceID = "0a1b2c3d-1111-2222-3333-444455556666"
	const gwName = "private-gw"
	const domainWildcard = "*.hey4z8.usa-e2.stgx.cloudhub.io"
	// BuildIngressURLs strips the "*." prefix, then:
	//   public   = https://<gw>.<base>
	//   internal = https://<gw>.internal-<base>
	const wantPublicURL = "https://private-gw.hey4z8.usa-e2.stgx.cloudhub.io"
	const wantInternalURL = "https://private-gw.internal-hey4z8.usa-e2.stgx.cloudhub.io"

	createPath := "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways"
	targetsPath := "/runtimefabric/api/organizations/test-org-id/targets"
	domainsPath := "/runtimefabric/api/organizations/test-org-id/targets/" + privateSpaceID + "/environments/test-env-id/domains"

	domainsCalled := false
	var capturedCreateBody map[string]interface{}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		targetsPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{"id": privateSpaceID, "name": "some-private-space", "type": "private-space"},
				{"id": "cloudhub-us-east-1", "name": "US East (N. Virginia)", "type": "shared-space"},
			})
		},
		domainsPath: func(w http.ResponseWriter, r *http.Request) {
			domainsCalled = true
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"domains": []string{domainWildcard},
			})
		},
		createPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			// Capture the request body so the main test body can assert that the
			// derived ingress URLs were actually sent to the platform.
			_ = json.NewDecoder(r.Body).Decode(&capturedCreateBody)
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
				"id":             "gw-private-1",
				"name":           gwName,
				"targetId":       privateSpaceID,
				"runtimeVersion": "1.9.9",
				"releaseChannel": "lts",
				"size":           "small",
				"status":         "PROVISIONING",
				"configuration": map[string]interface{}{
					"ingress": map[string]interface{}{
						"publicUrl":   wantPublicURL,
						"internalUrl": wantInternalURL,
					},
					"properties": map[string]interface{}{},
					"logging":    map[string]interface{}{},
					"tracing":    map[string]interface{}{},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	res.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	// runtime_version pre-set (skip versions lookup); nested blocks null so the
	// user supplied no URLs — forcing the derivation path.
	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"name":            tftypes.NewValue(tftypes.String, gwName),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, privateSpaceID),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, "1.9.9"),
		"release_channel": tftypes.NewValue(tftypes.String, "lts"),
		"size":            tftypes.NewValue(tftypes.String, "small"),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"ingress":         tftypes.NewValue(ingressObjType, nil),
		"properties":      tftypes.NewValue(propertiesObjType, nil),
		"logging":         tftypes.NewValue(loggingObjType, nil),
		"tracing":         tftypes.NewValue(tracingObjType, nil),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: planRaw}}
	res.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors for a private-space target: %v", resp.Diagnostics.Errors())
	}
	if !domainsCalled {
		t.Error("GetDomains MUST be called for a private-space target — the derivation path was skipped")
	}

	// The derived URLs must have been sent in the POST body.
	gotPublic, gotInternal := ingressURLsFromBody(t, capturedCreateBody)
	if gotPublic != wantPublicURL {
		t.Errorf("POST body ingress.publicUrl = %q, want derived %q", gotPublic, wantPublicURL)
	}
	if gotInternal != wantInternalURL {
		t.Errorf("POST body ingress.internalUrl = %q, want derived %q", gotInternal, wantInternalURL)
	}

	var got ManagedOmniGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.TargetType.ValueString() != "private-space" {
		t.Errorf("Expected target_type 'private-space' (from detection fallback), got %q", got.TargetType.ValueString())
	}
	if publicURL := got.Ingress.Attributes()["public_url"].(types.String).ValueString(); publicURL != wantPublicURL {
		t.Errorf("Expected derived public_url %q to round-trip into state, got %q", wantPublicURL, publicURL)
	}
}

// TestManagedOmniGatewayResource_Update_SharedSpace verifies the Update path for a
// shared-space gateway. Update also gained a shared-space branch (it keys off
// state.TargetType instead of making another /targets call), so this test locks in:
//  1. Update does NOT error for a shared-space gateway.
//  2. GetDomains is NEVER called during a shared-space update (no derivation, and
//     — critically — no extra API round-trip; the type is read from prior state).
//  3. The platform-assigned public_url is preserved and sent in the PUT body.
func TestManagedOmniGatewayResource_Update_SharedSpace(t *testing.T) {
	const sharedRegionSlug = "cloudhub-us-east-1"
	const platformAssignedURL = "https://shared-gw-x7k2p9.usa-e2.stgx.cloudhub.io"

	putPath := "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-shared-1"
	// Any GetDomains call for a shared-space update is a regression — fail hard.
	domainsPath := "/runtimefabric/api/organizations/test-org-id/targets/" + sharedRegionSlug + "/environments/test-env-id/domains"

	var capturedPutBody map[string]interface{}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		domainsPath: func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("GetDomains must NOT be called during a shared-space update — type comes from prior state")
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{"domains": []string{}})
		},
		putPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			_ = json.NewDecoder(r.Body).Decode(&capturedPutBody)
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "gw-shared-1",
				"name":           "shared-gw",
				"targetId":       sharedRegionSlug,
				"runtimeVersion": "1.9.9",
				"releaseChannel": "lts",
				"size":           "large",
				"status":         "APPLYING",
				"configuration": map[string]interface{}{
					"ingress":    map[string]interface{}{"publicUrl": platformAssignedURL},
					"properties": map[string]interface{}{},
					"logging":    map[string]interface{}{},
					"tracing":    map[string]interface{}{},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	res.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	// A shared-space gateway already in state: target_type=shared-space, the
	// platform-assigned public_url set, and NO internal_url (shared space has none).
	ingressVal := func() tftypes.Value {
		return tftypes.NewValue(ingressObjType, map[string]tftypes.Value{
			"public_url":          tftypes.NewValue(tftypes.String, platformAssignedURL),
			"internal_url":        tftypes.NewValue(tftypes.String, ""),
			"forward_ssl_session": tftypes.NewValue(tftypes.Bool, true),
			"last_mile_security":  tftypes.NewValue(tftypes.Bool, true),
		})
	}

	buildState := func(size, status string) tftypes.Value {
		return tftypes.NewValue(stateType, map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "gw-shared-1"),
			"name":            tftypes.NewValue(tftypes.String, "shared-gw"),
			"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
			"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
			"target_id":       tftypes.NewValue(tftypes.String, sharedRegionSlug),
			"target_type":     tftypes.NewValue(tftypes.String, "shared-space"),
			"runtime_version": tftypes.NewValue(tftypes.String, "1.9.9"),
			"release_channel": tftypes.NewValue(tftypes.String, "lts"),
			"size":            tftypes.NewValue(tftypes.String, size),
			"status":          tftypes.NewValue(tftypes.String, status),
			"ingress":         ingressVal(),
			"properties":      tftypes.NewValue(propertiesObjType, nil),
			"logging":         tftypes.NewValue(loggingObjType, nil),
			"tracing":         tftypes.NewValue(tracingObjType, nil),
		})
	}

	priorStateRaw := buildState("small", "RUNNING")
	planRaw := buildState("large", "RUNNING") // size change; public_url carried forward via UseStateForUnknown

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors for a shared-space gateway: %v", resp.Diagnostics.Errors())
	}

	// The platform-assigned public_url must be preserved in the PUT body.
	gotPublic, _ := ingressURLsFromBody(t, capturedPutBody)
	if gotPublic != platformAssignedURL {
		t.Errorf("PUT body ingress.publicUrl = %q, want preserved platform URL %q", gotPublic, platformAssignedURL)
	}

	var got ManagedOmniGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Size.ValueString() != "large" {
		t.Errorf("Expected Size 'large', got %q", got.Size.ValueString())
	}
	if publicURL := got.Ingress.Attributes()["public_url"].(types.String).ValueString(); publicURL != platformAssignedURL {
		t.Errorf("Expected platform-assigned public_url %q preserved in state, got %q", platformAssignedURL, publicURL)
	}
}

// ingressURLsFromBody pulls configuration.ingress.publicUrl / internalUrl out of a
// decoded request body, guarding every step so a shape mismatch fails the test
// cleanly instead of panicking the HTTP handler goroutine.
func ingressURLsFromBody(t *testing.T, body map[string]interface{}) (publicURL, internalURL string) {
	t.Helper()
	if body == nil {
		t.Fatal("request body was not captured (nil)")
	}
	cfg, ok := body["configuration"].(map[string]interface{})
	if !ok {
		t.Fatalf("request body missing configuration object: %#v", body)
	}
	ingress, ok := cfg["ingress"].(map[string]interface{})
	if !ok {
		t.Fatalf("request body missing configuration.ingress object: %#v", cfg)
	}
	if v, ok := ingress["publicUrl"].(string); ok {
		publicURL = v
	}
	if v, ok := ingress["internalUrl"].(string); ok {
		internalURL = v
	}
	return publicURL, internalURL
}
