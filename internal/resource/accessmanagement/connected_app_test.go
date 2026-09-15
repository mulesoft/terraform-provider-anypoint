package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppResource(t *testing.T) {
	r := NewConnectedAppResource()

	if r == nil {
		t.Error("NewConnectedAppResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("ConnectedAppResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("ConnectedAppResource should implement ResourceWithImportState")
	}
}

func TestConnectedAppResource_Metadata(t *testing.T) {
	r := NewConnectedAppResource()
	testutil.TestResourceMetadata(t, r, "_connected_app")
}

func TestConnectedAppResource_Schema(t *testing.T) {
	res := NewConnectedAppResource()

	requiredAttrs := []string{"name", "grant_types"}
	optionalAttrs := []string{"redirect_uris", "public_keys", "audience", "client_uri", "enabled", "organization_id"}
	computedAttrs := []string{"id", "client_secret", "owner_user_id", "created_at", "updated_at"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestConnectedAppResource_Configure(t *testing.T) {
	res := NewConnectedAppResource().(*ConnectedAppResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestConnectedAppResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewConnectedAppResource().(*ConnectedAppResource)

	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data",
	}
	resp := &resource.ConfigureResponse{}

	res.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}

	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestConnectedAppResourceModel_Validation(t *testing.T) {
	model := ConnectedAppResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.GrantTypes
}

// TestConnectedAppResource_Scopes_UseStateForUnknown asserts the churn fix on the
// inline `scopes` set: on an unrelated in-place update (config omits scopes -> plan
// value is unknown), the plan modifier reuses the prior state instead of rendering
// "(known after apply)". Safe because Update gates the scope sync on
// !plan.Scopes.Equal(state.Scopes) (an explicit [] still differs from a non-empty
// state and still clears), and because Update does not otherwise re-derive scopes in
// the omit path — so reusing prior state also prevents an unknown-after-apply error.
// Proven to FAIL if the modifier is removed (the len==0 Fatalf fires).
func TestConnectedAppResource_Scopes_UseStateForUnknown(t *testing.T) {
	res := NewConnectedAppResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	attrs := schemaResp.Schema.Attributes

	// Non-null resource state so UseStateForUnknown does not bail (it bails only when the
	// WHOLE resource state is null, i.e. create).
	nonNullState := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")}

	a := attrs["scopes"]
	sna, ok := a.(schema.SetNestedAttribute)
	if !ok {
		t.Fatalf("scopes: expected SetNestedAttribute, got %T", a)
	}
	if len(sna.PlanModifiers) == 0 {
		t.Fatalf("scopes: NO plan modifiers — churn regression (missing UseStateForUnknown)")
	}
	st, ok := a.GetType().(types.SetType)
	if !ok {
		t.Fatalf("scopes: expected set type, got %T", a.GetType())
	}
	elemType := st.ElementType()

	priorState := types.SetValueMust(elemType, []attr.Value{}) // known empty set
	req := planmodifier.SetRequest{
		State:       nonNullState,
		StateValue:  priorState,
		PlanValue:   types.SetUnknown(elemType), // framework marks computed unknown on update
		ConfigValue: types.SetNull(elemType),    // unconfigured
	}
	resp := &planmodifier.SetResponse{PlanValue: req.PlanValue}
	for _, m := range sna.PlanModifiers {
		m.PlanModifySet(ctx, req, resp)
	}
	if resp.PlanValue.IsUnknown() {
		t.Errorf("scopes: still (known after apply) after plan modifiers — churn NOT fixed")
	}
	if !resp.PlanValue.Equal(priorState) {
		t.Errorf("scopes: expected plan to reuse prior state %v, got %v", priorState, resp.PlanValue)
	}
}

func BenchmarkConnectedAppResource_Schema(b *testing.B) {
	res := NewConnectedAppResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
