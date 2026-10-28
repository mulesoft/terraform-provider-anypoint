package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestSettleEndpointPort pins the fix for "Provider returned invalid result object
// after apply" when a config omits endpoint.port: the planned endpoint (port unknown)
// is restored into state, so the port must be settled from the API response first.
// The null case is a state written before port existed (provider upgrade).
func TestSettleEndpointPort(t *testing.T) {
	str := types.StringValue
	api8082 := endpointObjWithPort("HY", "http", str("my-api"), types.Int64Value(8082), types.Int64Null())

	tests := []struct {
		name    string
		kept    types.Object
		fromAPI types.Object
		want    types.Int64
	}{
		{"unknown port takes API port", endpointObjWithPort("HY", "http", str("my-api"), types.Int64Unknown(), types.Int64Null()), api8082, types.Int64Value(8082)},
		{"unknown port, no API endpoint -> default", endpointObjWithPort("HY", "http", str("my-api"), types.Int64Unknown(), types.Int64Null()), types.ObjectNull(endpointAttrTypes), types.Int64Value(defaultProxyPort)},
		{"null port from old state takes API port", endpointObjWithPort("HY", "http", str("my-api"), types.Int64Null(), types.Int64Null()), api8082, types.Int64Value(8082)},
		{"known port is never touched", endpointObjWithPort("HY", "http", str("my-api"), types.Int64Value(9090), types.Int64Null()), api8082, types.Int64Value(9090)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := endpointFromObject(settleEndpointPort(tt.kept, tt.fromAPI))
			if got == nil {
				t.Fatal("settled endpoint is null")
			}
			if !got.Port.Equal(tt.want) {
				t.Errorf("port = %s, want %s", got.Port, tt.want)
			}
			if got.BasePath.ValueString() != "my-api" || got.Type.ValueString() != "http" {
				t.Errorf("other attributes must be kept from the planned endpoint, got %+v", got)
			}
		})
	}

	if got := settleEndpointPort(types.ObjectNull(endpointAttrTypes), api8082); !got.IsNull() {
		t.Errorf("null endpoint must stay null, got %s", got)
	}
}

// TestAPIInstanceStatusNotCopiedFromState: the platform moves status on its own
// (unregistered -> active), so an update plan must not promise the prior value.
func TestAPIInstanceStatusNotCopiedFromState(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewAPIInstanceResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	status, ok := resp.Schema.Attributes["status"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("status: unexpected type %T", resp.Schema.Attributes["status"])
	}
	if !status.Computed || len(status.PlanModifiers) != 0 {
		t.Errorf("status must be Computed with no UseStateForUnknown, got computed=%v modifiers=%d", status.Computed, len(status.PlanModifiers))
	}
}
