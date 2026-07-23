package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// --- reconcileTracing ---

func TestReconcileTracing(t *testing.T) {
	makeTracingObj := func(enabled bool, sampling int64) types.Object {
		labelsList, _ := types.ListValue(
			types.ObjectType{AttrTypes: tracingLabelAttrTypes},
			[]attr.Value{},
		)
		obj, _ := types.ObjectValue(tracingAttrTypes, map[string]attr.Value{
			"enabled":  types.BoolValue(enabled),
			"sampling": types.Int64Value(sampling),
			"labels":   labelsList,
		})
		return obj
	}

	t.Run("null fromAPI returns plan", func(t *testing.T) {
		plan := makeTracingObj(true, 5)
		result := reconcileTracing(plan, types.ObjectNull(tracingAttrTypes))
		if !result.Equal(plan) {
			t.Error("Expected plan when fromAPI is null")
		}
	})

	t.Run("api returned false but plan wanted true — use plan", func(t *testing.T) {
		plan := makeTracingObj(true, 5)
		fromAPI := makeTracingObj(false, 1)
		result := reconcileTracing(plan, fromAPI)
		attrs := result.Attributes()
		if !attrs["enabled"].(types.Bool).ValueBool() {
			t.Error("Expected enabled=true (plan) when API silently dropped tracing")
		}
	})

	t.Run("api returned true — use api value", func(t *testing.T) {
		plan := makeTracingObj(true, 5)
		fromAPI := makeTracingObj(true, 10)
		result := reconcileTracing(plan, fromAPI)
		attrs := result.Attributes()
		if attrs["sampling"].(types.Int64).ValueInt64() != 10 {
			t.Errorf("Expected sampling=10 (API value), got %v", attrs["sampling"])
		}
	})

	t.Run("unknown fromAPI returns plan", func(t *testing.T) {
		plan := makeTracingObj(true, 5)
		result := reconcileTracing(plan, types.ObjectUnknown(tracingAttrTypes))
		if !result.Equal(plan) {
			t.Error("Expected plan when fromAPI is unknown")
		}
	})

	// REGRESSION (tracing-unknown-after-apply): the user OMITS `tracing`, an
	// Optional+Computed nested object. On Create the framework hands us an UNKNOWN
	// plan value (nothing in prior state for UseStateForUnknown to copy). The old
	// code hit the `planAttrs == nil → return plan` branch and returned that
	// UNKNOWN plan, leaving a Computed attribute unknown after apply — which the
	// framework rejects with "Provider returned invalid result object after apply".
	// The known flattened API value must win.
	t.Run("unknown plan returns known API value (omitted tracing on create)", func(t *testing.T) {
		fromAPI := makeTracingObj(false, 1) // flatten output: computed defaults
		result := reconcileTracing(types.ObjectUnknown(tracingAttrTypes), fromAPI)
		if result.IsUnknown() || result.IsNull() {
			t.Fatalf("result must be concrete when plan is unknown (unknown=%v null=%v); "+
				"an unknown Computed attr after apply is a framework error",
				result.IsUnknown(), result.IsNull())
		}
		if !result.Equal(fromAPI) {
			t.Errorf("expected the known API value to win over an unknown plan; got %#v", result)
		}
	})

	// A null plan (block explicitly absent, no computed backfill) similarly must
	// not become the authoritative value when the API produced a concrete object.
	t.Run("null plan returns known API value", func(t *testing.T) {
		fromAPI := makeTracingObj(false, 1)
		result := reconcileTracing(types.ObjectNull(tracingAttrTypes), fromAPI)
		if result.IsUnknown() || result.IsNull() {
			t.Fatalf("result must be concrete when plan is null; got unknown=%v null=%v",
				result.IsUnknown(), result.IsNull())
		}
		if !result.Equal(fromAPI) {
			t.Errorf("expected the known API value to win over a null plan; got %#v", result)
		}
	})
}

// --- flattenGateway ---

func TestManagedOmniGatewayResource_flattenGateway(t *testing.T) {
	r := &ManagedOmniGatewayResource{}

	t.Run("basic gateway is flattened", func(t *testing.T) {
		gw := &apimgmtclient.ManagedOmniGateway{
			ID:             "gw-1",
			Name:           "My Gateway",
			TargetID:       "target-1",
			RuntimeVersion: "1.6.0",
			ReleaseChannel: "LTS",
			Size:           "small",
			Status:         "Running",
			Configuration: apimgmtclient.ManagedOmniGatewayConfig{
				Ingress: apimgmtclient.IngressConfig{
					PublicURL:         "https://pub.example.com",
					InternalURL:       "https://int.example.com",
					ForwardSSLSession: true,
					LastMileSecurity:  false,
				},
				Properties: apimgmtclient.PropertiesConfig{
					UpstreamResponseTimeout: 15,
					ConnectionIdleTimeout:   60,
				},
				Logging: apimgmtclient.LoggingConfig{
					Level:       "info",
					ForwardLogs: true,
				},
				Tracing: apimgmtclient.TracingConfig{
					Enabled:  false,
					Sampling: 1,
				},
			},
		}
		data := &ManagedOmniGatewayResourceModel{OrganizationID: types.StringNull()}
		r.flattenGateway(gw, data, "org-1", "env-2")

		if data.ID.ValueString() != "gw-1" {
			t.Errorf("ID = %q, want gw-1", data.ID.ValueString())
		}
		if data.Name.ValueString() != "My Gateway" {
			t.Errorf("Name = %q, want My Gateway", data.Name.ValueString())
		}
		if data.Status.ValueString() != "Running" {
			t.Errorf("Status = %q, want Running", data.Status.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.Ingress.IsNull() {
			t.Error("Ingress should not be null")
		}
		if data.Properties.IsNull() {
			t.Error("Properties should not be null")
		}
		if data.Logging.IsNull() {
			t.Error("Logging should not be null")
		}
		if data.Tracing.IsNull() {
			t.Error("Tracing should not be null")
		}
	})

	t.Run("non-zero tracing sampling is preserved", func(t *testing.T) {
		gw := &apimgmtclient.ManagedOmniGateway{
			ID: "gw-2",
			Configuration: apimgmtclient.ManagedOmniGatewayConfig{
				Tracing: apimgmtclient.TracingConfig{Enabled: true, Sampling: 50},
			},
		}
		data := &ManagedOmniGatewayResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenGateway(gw, data, "org-1", "env-1")

		if data.Tracing.IsNull() {
			t.Fatal("Tracing should not be null")
		}
		attrs := data.Tracing.Attributes()
		if attrs["sampling"].(types.Int64).ValueInt64() != 50 {
			t.Errorf("Tracing.sampling = %d, want 50", attrs["sampling"].(types.Int64).ValueInt64())
		}
	})

	t.Run("zero sampling is defaulted to 1", func(t *testing.T) {
		gw := &apimgmtclient.ManagedOmniGateway{
			ID: "gw-3",
			Configuration: apimgmtclient.ManagedOmniGatewayConfig{
				Tracing: apimgmtclient.TracingConfig{Enabled: false, Sampling: 0},
			},
		}
		data := &ManagedOmniGatewayResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenGateway(gw, data, "org-1", "env-1")
		attrs := data.Tracing.Attributes()
		if attrs["sampling"].(types.Int64).ValueInt64() != 1 {
			t.Errorf("Tracing.sampling = %d, want 1 (default for zero)", attrs["sampling"].(types.Int64).ValueInt64())
		}
	})

	t.Run("existing org_id is preserved", func(t *testing.T) {
		gw := &apimgmtclient.ManagedOmniGateway{ID: "gw-4"}
		data := &ManagedOmniGatewayResourceModel{OrganizationID: types.StringValue("existing-org")}
		r.flattenGateway(gw, data, "new-org", "env-1")
		if data.OrganizationID.ValueString() != "existing-org" {
			t.Errorf("OrganizationID = %q, should not be overwritten", data.OrganizationID.ValueString())
		}
	})
}

// --- expandConfiguration ---

func TestManagedOmniGatewayResource_expandConfiguration(t *testing.T) {
	r := &ManagedOmniGatewayResource{}

	t.Run("null ingress uses defaults", func(t *testing.T) {
		data := ManagedOmniGatewayResourceModel{
			Ingress:    types.ObjectNull(ingressAttrTypes),
			Properties: types.ObjectNull(propertiesAttrTypes),
			Logging:    types.ObjectNull(loggingAttrTypes),
			Tracing:    types.ObjectNull(tracingAttrTypes),
		}
		cfg := r.expandConfiguration(data)
		if !cfg.Ingress.ForwardSSLSession {
			t.Error("Default ForwardSSLSession should be true")
		}
		if !cfg.Ingress.LastMileSecurity {
			t.Error("Default LastMileSecurity should be true")
		}
	})

	t.Run("null properties uses defaults", func(t *testing.T) {
		data := ManagedOmniGatewayResourceModel{
			Ingress:    types.ObjectNull(ingressAttrTypes),
			Properties: types.ObjectNull(propertiesAttrTypes),
			Logging:    types.ObjectNull(loggingAttrTypes),
			Tracing:    types.ObjectNull(tracingAttrTypes),
		}
		cfg := r.expandConfiguration(data)
		if cfg.Properties.UpstreamResponseTimeout != 15 {
			t.Errorf("Default UpstreamResponseTimeout = %d, want 15", cfg.Properties.UpstreamResponseTimeout)
		}
		if cfg.Properties.ConnectionIdleTimeout != 60 {
			t.Errorf("Default ConnectionIdleTimeout = %d, want 60", cfg.Properties.ConnectionIdleTimeout)
		}
	})

	t.Run("null logging uses defaults", func(t *testing.T) {
		data := ManagedOmniGatewayResourceModel{
			Ingress:    types.ObjectNull(ingressAttrTypes),
			Properties: types.ObjectNull(propertiesAttrTypes),
			Logging:    types.ObjectNull(loggingAttrTypes),
			Tracing:    types.ObjectNull(tracingAttrTypes),
		}
		cfg := r.expandConfiguration(data)
		if cfg.Logging.Level != "info" {
			t.Errorf("Default Logging.Level = %q, want info", cfg.Logging.Level)
		}
		if !cfg.Logging.ForwardLogs {
			t.Error("Default ForwardLogs should be true")
		}
	})

	t.Run("null tracing uses defaults", func(t *testing.T) {
		data := ManagedOmniGatewayResourceModel{
			Ingress:    types.ObjectNull(ingressAttrTypes),
			Properties: types.ObjectNull(propertiesAttrTypes),
			Logging:    types.ObjectNull(loggingAttrTypes),
			Tracing:    types.ObjectNull(tracingAttrTypes),
		}
		cfg := r.expandConfiguration(data)
		if cfg.Tracing.Enabled {
			t.Error("Default Tracing.Enabled should be false")
		}
		if cfg.Tracing.Sampling != 1 {
			t.Errorf("Default Tracing.Sampling = %d, want 1", cfg.Tracing.Sampling)
		}
	})
}

// --- ManagedOmniGatewayResource.ImportState ---

func TestManagedOmniGatewayResource_ImportState_IDParsing(t *testing.T) {
	r := NewManagedOmniGatewayResource().(*ManagedOmniGatewayResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	rawState := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"environment_id":  tftypes.NewValue(tftypes.String, nil),
		"target_id":       tftypes.NewValue(tftypes.String, nil),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, nil),
		"release_channel": tftypes.NewValue(tftypes.String, nil),
		"size":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"ingress":         tftypes.NewValue(objType.AttributeTypes["ingress"], nil),
		"properties":      tftypes.NewValue(objType.AttributeTypes["properties"], nil),
		"logging":         tftypes.NewValue(objType.AttributeTypes["logging"], nil),
		"tracing":         tftypes.NewValue(objType.AttributeTypes["tracing"], nil),
	})

	t.Run("valid 3-part ID", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/gw-abc"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got ManagedOmniGatewayResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.ID.ValueString() != "gw-abc" {
			t.Errorf("ID = %q, want gw-abc", got.ID.ValueString())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", got.EnvironmentID.ValueString())
		}
	})

	t.Run("valid 2-part ID sets environment_id and id", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "env-2/gw-abc"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors for 2-part ID: %v", resp.Diagnostics.Errors())
		}
		var got ManagedOmniGatewayResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.ID.ValueString() != "gw-abc" {
			t.Errorf("ID = %q, want gw-abc", got.ID.ValueString())
		}
		if !got.OrganizationID.IsNull() {
			t.Errorf("OrganizationID should be null for 2-part import, got %q", got.OrganizationID.ValueString())
		}
	})

	t.Run("invalid ID format (4-part)", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org/env/gw/extra"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 4-part ID")
		}
	})

	t.Run("invalid ID format (1-part)", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "only-one-part"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 1-part ID")
		}
	})
}

// --- ManagedOmniGatewayResource.Read with server error ---

func TestManagedOmniGatewayResource_Read_Error(t *testing.T) {
	basePath := "/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-fail"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "gw-fail"),
		"name":            tftypes.NewValue(tftypes.String, "gw"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, "t"),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, nil),
		"release_channel": tftypes.NewValue(tftypes.String, nil),
		"size":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"ingress":         tftypes.NewValue(objType.AttributeTypes["ingress"], nil),
		"properties":      tftypes.NewValue(objType.AttributeTypes["properties"], nil),
		"logging":         tftypes.NewValue(objType.AttributeTypes["logging"], nil),
		"tracing":         tftypes.NewValue(objType.AttributeTypes["tracing"], nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}
