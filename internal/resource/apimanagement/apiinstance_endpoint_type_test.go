package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// endpointTypeValidators reaches into the resource schema and returns the
// string validators attached to the nested endpoint.type attribute.
func endpointTypeValidators(t *testing.T) []validator.String {
	t.Helper()

	r := NewAPIInstanceResource()
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}

	epAttr, ok := resp.Schema.Attributes["endpoint"]
	if !ok {
		t.Fatal("expected `endpoint` attribute on api_instance schema")
	}
	epNested, ok := epAttr.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected `endpoint` to be a SingleNestedAttribute, got %T", epAttr)
	}
	typeAttr, ok := epNested.Attributes["type"]
	if !ok {
		t.Fatal("expected `type` attribute inside `endpoint`")
	}
	typeStr, ok := typeAttr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected endpoint.type to be a StringAttribute, got %T", typeAttr)
	}
	return typeStr.StringValidators()
}

// runEndpointTypeValidators evaluates every validator on endpoint.type against
// a candidate value and reports whether any validator raised an error.
func runEndpointTypeValidators(t *testing.T, validators []validator.String, value string) bool {
	t.Helper()
	ctx := context.Background()
	for _, v := range validators {
		req := validator.StringRequest{
			Path:        path.Root("endpoint").AtName("type"),
			ConfigValue: types.StringValue(value),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			return true // rejected
		}
	}
	return false // accepted
}

// TestAPIInstanceEndpointType_AcceptsAllPlatformTypes is the regression guard
// for Bug D (W-23170647): the endpoint.type validator originally allowed only
// {http, rest, raml}, which made it impossible to create GraphQL/WebSocket/gRPC
// API instances via Terraform even though the platform requires those exact
// endpoint types for the matching Exchange asset types. The full recognized set
// was live-enumerated on production (2026-07-21): http, rest, raml, wsdl,
// graphql, grpc, websocket. graphql in particular is REQUIRED to front a
// graphql Exchange asset, which is the prerequisite for the 5 new GraphQL/
// WebSocket policies. This test pins every recognized value as valid so a future
// edit cannot silently re-narrow the enum and re-break the feature.
func TestAPIInstanceEndpointType_AcceptsAllPlatformTypes(t *testing.T) {
	validators := endpointTypeValidators(t)
	if len(validators) == 0 {
		t.Fatal("expected at least one validator on endpoint.type")
	}

	valid := []string{"http", "rest", "raml", "wsdl", "graphql", "grpc", "websocket"}
	for _, v := range valid {
		if rejected := runEndpointTypeValidators(t, validators, v); rejected {
			t.Errorf("endpoint.type %q should be accepted by the validator but was rejected", v)
		}
	}
}

// TestAPIInstanceEndpointType_RejectsUnknownType ensures the validator is still
// a genuine allow-list and did not get widened into a no-op. The probe values
// were confirmed on production to return "Invalid endpoint type" (400), i.e. the
// platform does not recognize them either.
func TestAPIInstanceEndpointType_RejectsUnknownType(t *testing.T) {
	validators := endpointTypeValidators(t)

	invalid := []string{"soap", "asyncapi", "openapi", "proxy", "", "HTTP"}
	for _, v := range invalid {
		if rejected := runEndpointTypeValidators(t, validators, v); !rejected {
			t.Errorf("endpoint.type %q should be rejected by the validator but was accepted", v)
		}
	}
}
