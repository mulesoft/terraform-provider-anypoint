package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// buildNullState builds a tfsdk.State filled with null values from the resource schema.
func buildNullStateForResource(t *testing.T, r resource.Resource) (resource.SchemaResponse, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	rawState := buildNullTFValue(t, stateType)
	return schemaResp, tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}
}

func buildNullTFValue(t *testing.T, tfType tftypes.Type) tftypes.Value {
	t.Helper()
	switch typ := tfType.(type) {
	case tftypes.Object:
		vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for k, v := range typ.AttributeTypes {
			vals[k] = buildNullTFValue(t, v)
		}
		return tftypes.NewValue(typ, vals)
	default:
		return tftypes.NewValue(tfType, nil)
	}
}

// ── ConnectedAppScopesResource ────────────────────────────────────────────────

func TestConnectedAppScopesResource_ImportState_SetsAppID(t *testing.T) {
	r := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullStateForResource(t, r)

	req := resource.ImportStateRequest{ID: "my-connected-app-id"}
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
	}
	var got ConnectedAppScopesResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ConnectedAppID.ValueString() != "my-connected-app-id" {
		t.Errorf("ConnectedAppID = %q, want my-connected-app-id", got.ConnectedAppID.ValueString())
	}
	if got.ID.ValueString() != "my-connected-app-id" {
		t.Errorf("ID = %q, want my-connected-app-id", got.ID.ValueString())
	}
}

// ── EnvironmentResource ───────────────────────────────────────────────────────

func TestEnvironmentResource_ImportState_PassthroughID(t *testing.T) {
	r := NewEnvironmentResource().(*EnvironmentResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullStateForResource(t, r)

	req := resource.ImportStateRequest{ID: "env-uuid-abc"}
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
	}
	var got EnvironmentResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "env-uuid-abc" {
		t.Errorf("ID = %q, want env-uuid-abc", got.ID.ValueString())
	}
}

// ── TeamResource ─────────────────────────────────────────────────────────────

func TestTeamResource_ImportState_PassthroughID(t *testing.T) {
	r := NewTeamResource().(*TeamResource)
	ctx := context.Background()
	schemaResp, rawState := buildNullStateForResource(t, r)

	req := resource.ImportStateRequest{ID: "team-uuid-xyz"}
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState.Raw}}
	r.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
	}
	var got TeamResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "team-uuid-xyz" {
		t.Errorf("ID = %q, want team-uuid-xyz", got.ID.ValueString())
	}
}
