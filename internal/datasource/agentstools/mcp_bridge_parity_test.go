package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"

	resourceagents "github.com/mulesoft/terraform-provider-anypoint/internal/resource/agentstools"
)

// nestedAttrNames returns the attribute names one level below the given path, for either
// schema flavour, so the resource and the data source can be compared directly.
func resourceToolAttrs(t *testing.T) map[string]struct{} {
	t.Helper()
	resp := &resource.SchemaResponse{}
	resourceagents.NewMCPBridgeResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	sources, ok := resp.Schema.Attributes["source_apis"].(rschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("source_apis: unexpected type %T", resp.Schema.Attributes["source_apis"])
	}
	tools, ok := sources.NestedObject.Attributes["tools"].(rschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("tools: unexpected type %T", sources.NestedObject.Attributes["tools"])
	}
	out := map[string]struct{}{}
	for n := range tools.NestedObject.Attributes {
		out[n] = struct{}{}
	}
	return out
}

func dataSourceSchema(t *testing.T) dschema.Schema {
	t.Helper()
	resp := &datasource.SchemaResponse{}
	NewMCPBridgeDataSource().Schema(context.Background(), datasource.SchemaRequest{}, resp)
	return resp.Schema
}

// Anything the resource can express and the platform stores should also be readable
// through the data source, otherwise a practitioner can create a bridge with a setting
// they cannot then inspect. input_schema is the documented exception: it lives only in
// the generated asset metadata, which reconstruction does not read.
func TestBridgeDataSource_CoversResourceToolSurface(t *testing.T) {
	const notReadableFromGateway = "input_schema"

	ds := dataSourceSchema(t)
	sources := ds.Attributes["source_apis"].(dschema.ListNestedAttribute)
	tools := sources.NestedObject.Attributes["tools"].(dschema.ListNestedAttribute)

	for name := range resourceToolAttrs(t) {
		if name == notReadableFromGateway {
			continue
		}
		if _, ok := tools.NestedObject.Attributes[name]; !ok {
			t.Errorf("source_apis.tools.%s exists on the resource but cannot be read through the data source", name)
		}
	}
}

// tls_context_id is stored on the upstream and comes back on read, so a bridge secured
// with a TLS context must not look unsecured through the data source.
func TestBridgeDataSource_ExposesSourceTLSContext(t *testing.T) {
	ds := dataSourceSchema(t)
	sources := ds.Attributes["source_apis"].(dschema.ListNestedAttribute)

	if _, ok := sources.NestedObject.Attributes["tls_context_id"]; !ok {
		t.Error("source_apis.tls_context_id is missing from the data source")
	}
}

// The instance-level settings are all plain reads off the API Manager instance.
func TestBridgeDataSource_ExposesInstanceFields(t *testing.T) {
	ds := dataSourceSchema(t)

	for _, name := range []string{"instance_label", "approval_method", "consumer_endpoint", "provider_id"} {
		if _, ok := ds.Attributes[name]; !ok {
			t.Errorf("%s is settable on the resource but missing from the data source", name)
		}
	}
}
