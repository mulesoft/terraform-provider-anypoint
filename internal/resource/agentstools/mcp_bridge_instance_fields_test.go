package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

func bridgeSchemaAttr(t *testing.T, name string) schema.StringAttribute {
	t.Helper()
	resp := &resource.SchemaResponse{}
	NewMCPBridgeResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s: expected StringAttribute, got %T", name, resp.Schema.Attributes[name])
	}
	return attr
}

// The bridge is an API Manager instance like mcp_server, agent_instance and
// api_instance, and the create wizard exposes the same instance-level fields. All three
// must be settable, not read-only, or a Terraform-managed bridge cannot reproduce what
// the UI produces.
func TestMCPBridgeSchema_InstanceFieldsAreSettable(t *testing.T) {
	for _, name := range []string{"instance_label", "approval_method", "consumer_endpoint", "provider_id"} {
		attr := bridgeSchemaAttr(t, name)
		if !attr.Optional {
			t.Errorf("%s must be Optional so it can be declared in config", name)
		}
		if !attr.Computed {
			t.Errorf("%s must also be Computed so an imported value does not read back as a diff", name)
		}
	}
}

func TestMCPBridgeSchema_ApprovalMethodRejectsUnknownValues(t *testing.T) {
	attr := bridgeSchemaAttr(t, "approval_method")
	if len(attr.Validators) == 0 {
		t.Fatal("approval_method must constrain its values; the platform only accepts \"manual\"")
	}
}

func TestBridgeInstanceFields_IsEmptyWhenNothingChanged(t *testing.T) {
	if !(agentstools.BridgeInstanceFields{}).IsEmpty() {
		t.Error("an all-nil field set must report empty so the caller skips the PATCH entirely")
	}
	v := "x"
	if (agentstools.BridgeInstanceFields{EndpointURI: &v}).IsEmpty() {
		t.Error("a populated field set must not report empty")
	}
}

// changedInstanceField drives the PATCH. Dropping an Optional+Computed attribute from
// config surfaces as unknown, which must be read as "keep what is there" rather than
// "clear it" — otherwise a plan that touches an unrelated attribute wipes the label.
func TestChangedInstanceField(t *testing.T) {
	tests := []struct {
		name    string
		planned types.String
		current types.String
		want    *string
	}{
		{"unchanged value sends nothing", types.StringValue("a"), types.StringValue("a"), nil},
		{"unknown planned keeps existing", types.StringUnknown(), types.StringValue("a"), nil},
		{"both absent sends nothing", types.StringNull(), types.StringNull(), nil},
		{"new value is sent", types.StringValue("b"), types.StringValue("a"), strPtr("b")},
		{"first value is sent", types.StringValue("b"), types.StringNull(), strPtr("b")},
		{"explicit removal clears", types.StringNull(), types.StringValue("a"), strPtr("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := changedInstanceField(tt.planned, tt.current)
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("expected no change, got %q", *got)
			case tt.want != nil && got == nil:
				t.Errorf("expected %q, got no change", *tt.want)
			case tt.want != nil && got != nil && *tt.want != *got:
				t.Errorf("got %q, want %q", *got, *tt.want)
			}
		})
	}
}

func TestKnownString(t *testing.T) {
	if got := knownString(types.StringNull()); got != "" {
		t.Errorf("null must render as empty so omitempty drops it, got %q", got)
	}
	if got := knownString(types.StringUnknown()); got != "" {
		t.Errorf("unknown must render as empty so omitempty drops it, got %q", got)
	}
	if got := knownString(types.StringValue("v1.0")); got != "v1.0" {
		t.Errorf("got %q, want v1.0", got)
	}
}

func strPtr(s string) *string { return &s }
