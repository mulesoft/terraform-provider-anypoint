package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// endpointPortAttribute digs the endpoint.port attribute out of the live resource schema.
func endpointPortAttribute(t *testing.T) schema.Int64Attribute {
	t.Helper()

	resp := &resource.SchemaResponse{}
	NewAPIInstanceResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	endpoint, ok := resp.Schema.Attributes["endpoint"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("endpoint: expected SingleNestedAttribute, got %T", resp.Schema.Attributes["endpoint"])
	}
	port, ok := endpoint.Attributes["port"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("endpoint.port: expected Int64Attribute, got %T", endpoint.Attributes["port"])
	}
	return port
}

// TestEndpointPortHasNoStaticDefault guards a data-loss bug that every existing
// endpoint.port test misses, because they all exercise helpers rather than the plan.
//
// endpoint.port is a NEW attribute, so no existing user config can possibly mention it.
// The framework applies a schema Default whenever the CONFIG value is null - overwriting
// the proposed-new value that came from prior state. With a static Default of 8081 that
// means, on the first apply after a provider upgrade:
//
//	refresh reads the instance's real port     -> endpoint.port = 8088
//	config omits port (the attribute is new)   -> Default overrides            -> 8081
//	plan proposes 8088 -> 8081, apply PATCHes proxyUri to :8081
//
// i.e. upgrading the provider silently moves a live API proxy's listener port, breaking
// every consumer of it and potentially colliding with whatever already owns 8081 on that
// gateway. The repo's own state-upgrade fixture is an instance on 8088.
//
// UseStateForUnknown makes an omitted port mean "leave it alone"; new instances still
// land on 8081 via buildProxyURI/flattenInstance, which is what TestBuildProxyURI and
// TestAPIInstanceResource_flattenInstance_port already pin down.
func TestEndpointPortHasNoStaticDefault(t *testing.T) {
	port := endpointPortAttribute(t)

	if port.Default != nil {
		t.Error("endpoint.port must NOT carry a static Default: it overrides the port read " +
			"back from a live instance, so upgrading the provider would re-PATCH the proxy " +
			"URI and move a production listener onto 8081")
	}
	if len(port.PlanModifiers) == 0 {
		t.Fatal("endpoint.port: expected UseStateForUnknown so an omitted port keeps the live value")
	}
}

// TestEndpointPortStaysOptionalComputed keeps the fix from being "corrected" into a
// different bug: dropping Computed would make an omitted port plan as null and wipe the
// value, and dropping Optional would stop users choosing a port at all.
func TestEndpointPortStaysOptionalComputed(t *testing.T) {
	port := endpointPortAttribute(t)

	if !port.Optional {
		t.Error("endpoint.port must stay Optional so a distinct port can be requested")
	}
	if !port.Computed {
		t.Error("endpoint.port must stay Computed so the live port can be read back")
	}
}
