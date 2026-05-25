package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// buildNullState builds a tfsdk.State where every attribute is null.
func buildNullState(t *testing.T, r resource.Resource) (resource.SchemaResponse, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	return schemaResp, tfsdk.State{Schema: schemaResp.Schema, Raw: nullValue(t, stateType)}
}

func nullValue(t *testing.T, tfType tftypes.Type) tftypes.Value {
	t.Helper()
	switch typ := tfType.(type) {
	case tftypes.Object:
		vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for k, at := range typ.AttributeTypes {
			vals[k] = nullValue(t, at)
		}
		return tftypes.NewValue(typ, vals)
	default:
		return tftypes.NewValue(tfType, nil)
	}
}

// ── APIPolicyResource.ImportState ────────────────────────────────────────────

func TestAPIPolicyResource_ImportState_4PartID(t *testing.T) {
	r := NewAPIPolicyResource().(*APIPolicyResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	t.Run("valid 4-part ID sets all attributes", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/100/200"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got APIPolicyResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.APIInstanceID.ValueString() != "100" {
			t.Errorf("APIInstanceID = %q, want 100", got.APIInstanceID.ValueString())
		}
		if got.ID.ValueString() != "200" {
			t.Errorf("ID = %q, want 200", got.ID.ValueString())
		}
	})

	t.Run("wrong number of parts produces error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/100"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 3-part ID")
		}
	})
}

// ── KnownPolicyResource.ImportState ──────────────────────────────────────────

func TestKnownPolicyResource_ImportState_4PartID(t *testing.T) {
	newFn := NewKnownPolicyResourceFunc("http-caching")
	r := newFn().(*KnownPolicyResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	t.Run("valid 4-part ID sets all attributes", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/42/99"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got KnownPolicyResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.ID.ValueString() != "99" {
			t.Errorf("ID = %q, want 99", got.ID.ValueString())
		}
	})

	t.Run("wrong number of parts produces error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 2-part ID")
		}
	})
}

// ── SLATierResource.ImportState ───────────────────────────────────────────────

func TestSLATierResource_ImportState_NumericID(t *testing.T) {
	r := NewSLATierResource().(*SLATierResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	t.Run("numeric tier ID passes directly", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/100/42"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got SLATierResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.APIInstanceID.ValueString() != "100" {
			t.Errorf("APIInstanceID = %q, want 100", got.APIInstanceID.ValueString())
		}
		if got.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", got.ID.ValueString())
		}
	})

	t.Run("wrong number of parts produces error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org/env/api"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 3-part ID")
		}
	})

	t.Run("name-based ID with invalid api_instance_id errors", func(t *testing.T) {
		r2 := NewSLATierResource().(*SLATierResource)
		// client is nil here — the non-numeric tierRef branch tries to
		// call GetSLATierByName, but the api_instance_id parse error fires first.
		req := resource.ImportStateRequest{ID: "org/env/not-a-number/Gold"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r2.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error when api_instance_id is non-numeric")
		}
	})

	t.Run("name-based ID with valid api_instance_id resolves via client", func(t *testing.T) {
		tiersPath := "/apimanager/api/v1/organizations/org-1/environments/env-2/apis/100/tiers"
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			tiersPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"tiers": []interface{}{
						map[string]interface{}{"id": 77, "name": "Gold", "autoApprove": false, "limits": []interface{}{}},
					},
				})
			},
		}
		server := testutil.MockHTTPServer(t, handlers)
		r3 := NewSLATierResource().(*SLATierResource)
		r3.client = &apimgmtclient.SLATierClient{
			AnypointClient: &anypointclient.AnypointClient{
				BaseURL:    server.URL,
				Token:      "tok",
				HTTPClient: &http.Client{},
				OrgID:      "org-1",
			},
		}
		req := resource.ImportStateRequest{ID: "org-1/env-2/100/Gold"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
		r3.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got SLATierResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		// Tier name "Gold" should have been resolved to ID 77
		if got.ID.ValueString() != "77" {
			t.Errorf("ID = %q, want 77 (resolved from tier name Gold)", got.ID.ValueString())
		}
	})
}

// ── SLATierResource.Read error paths ─────────────────────────────────────────

func TestSLATierResource_Read_InvalidTierID(t *testing.T) {
	r := NewSLATierResource().(*SLATierResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	// Patch the state to have a non-numeric tier ID (exercises the strconv.Atoi error branch)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	patchedRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = tftypes.NewValue(at, nil)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "not-a-number")
		vals["api_instance_id"] = tftypes.NewValue(tftypes.String, "100")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "org-1")
		vals["environment_id"] = tftypes.NewValue(tftypes.String, "env-1")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: patchedRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error for non-numeric tier ID")
	}
}

func TestSLATierResource_Read_InvalidAPIInstanceID(t *testing.T) {
	r := NewSLATierResource().(*SLATierResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	patchedRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = tftypes.NewValue(at, nil)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "42")
		vals["api_instance_id"] = tftypes.NewValue(tftypes.String, "not-a-number")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "org-1")
		vals["environment_id"] = tftypes.NewValue(tftypes.String, "env-1")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: patchedRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error for non-numeric api_instance_id")
	}
}

// ── APIPolicyResource.Read error paths ───────────────────────────────────────

func TestAPIPolicyResource_Read_InvalidAPIInstanceID(t *testing.T) {
	r := NewAPIPolicyResource().(*APIPolicyResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	patchedRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = tftypes.NewValue(at, nil)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "200")
		vals["api_instance_id"] = tftypes.NewValue(tftypes.String, "not-a-number")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "org-1")
		vals["environment_id"] = tftypes.NewValue(tftypes.String, "env-1")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: patchedRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error for non-numeric api_instance_id")
	}
}

func TestAPIPolicyResource_Read_InvalidPolicyID(t *testing.T) {
	r := NewAPIPolicyResource().(*APIPolicyResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullState(t, r)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	patchedRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = tftypes.NewValue(at, nil)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "not-a-number")
		vals["api_instance_id"] = tftypes.NewValue(tftypes.String, "100")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "org-1")
		vals["environment_id"] = tftypes.NewValue(tftypes.String, "env-1")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: patchedRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error for non-numeric policy ID")
	}
}
