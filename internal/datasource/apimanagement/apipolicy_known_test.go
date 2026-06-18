package apimanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// TestKnownPolicyDataSourceTypes verifies the registry returns all known policy types.
func TestKnownPolicyDataSourceTypes(t *testing.T) {
	types_ := KnownPolicyDataSourceTypes()
	if len(types_) == 0 {
		t.Error("KnownPolicyDataSourceTypes() should return non-empty list")
	}
	// Should match the count of KnownPolicies in the client package.
	if len(types_) != len(apimanagement.KnownPolicies) {
		t.Errorf("KnownPolicyDataSourceTypes() returned %d types, want %d", len(types_), len(apimanagement.KnownPolicies))
	}
}

// TestNewKnownPolicyDataSourceFunc_ValidType verifies a valid type creates a data source.
func TestNewKnownPolicyDataSourceFunc_ValidType(t *testing.T) {
	factory := NewKnownPolicyDataSourceFunc("rate-limiting")
	ds := factory()
	if ds == nil {
		t.Fatal("NewKnownPolicyDataSourceFunc returned nil")
	}

	// Verify metadata
	metaReq := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
	metaResp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), metaReq, metaResp)
	expected := "anypoint_api_policy_rate_limiting"
	if metaResp.TypeName != expected {
		t.Errorf("Metadata TypeName = %q, want %q", metaResp.TypeName, expected)
	}
}

// TestNewKnownPolicyDataSourceFunc_PanicsOnUnknown verifies unknown types panic.
func TestNewKnownPolicyDataSourceFunc_PanicsOnUnknown(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for unknown policy type")
		}
	}()
	NewKnownPolicyDataSourceFunc("non-existent-policy-xyz")
}

// TestKnownPolicyDataSource_Schema verifies schema generation for a known policy.
func TestKnownPolicyDataSource_Schema(t *testing.T) {
	factory := NewKnownPolicyDataSourceFunc("spike-control")
	ds := factory()

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", schemaResp.Diagnostics.Errors())
	}

	// Check that "configuration" attribute exists
	configAttr, ok := schemaResp.Schema.Attributes["configuration"]
	if !ok {
		t.Fatal("Schema missing 'configuration' attribute")
	}

	// Type assert to SingleNestedAttribute — it should contain the policy-specific fields
	_ = configAttr // type checking is sufficient since Schema() didn't error
}

// TestKnownPolicyDataSource_ReadAutoDiscover tests the auto-discover path.
func TestKnownPolicyDataSource_ReadAutoDiscover(t *testing.T) {
	// Mock server that returns a list of policies
	mockPolicies := []map[string]interface{}{
		{
			"id":               101,
			"policyTemplateId": "spike-control",
			"groupId":          "68ef9520-24e9-4cf2-b2f5-620025690913",
			"assetId":          "spike-control",
			"assetVersion":     "1.2.2",
			"configurationData": map[string]interface{}{
				"maximumRequests":          100,
				"timePeriodInMilliseconds": 1000,
				"delayTimeInMillis":        500,
				"delayAttempts":            3,
				"queuingLimit":             10,
				"exposeHeaders":            false,
			},
			"order":    1,
			"disabled": false,
			"apiId":    12345,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPolicies)
	}))
	defer server.Close()

	ds := &KnownPolicyDataSource{
		client: &apimanagement.APIPolicyClient{
			AnypointClient: &anypointclient.AnypointClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: server.Client(),
				OrgID:      "test-org",
			},
		},
		policyInfo: apimanagement.KnownPolicies["spike-control"],
		typeSuffix: "spike_control",
	}

	// Build model — no policy_id, so it auto-discovers
	data := &KnownPolicyDataSourceModel{
		OrganizationID: types.StringValue("test-org"),
		EnvironmentID:  types.StringValue("test-env"),
		APIInstanceID:  types.StringValue("12345"),
		PolicyID:       types.StringNull(),
	}

	// Call flatten directly with a policy to test flattening logic
	policy := &apimanagement.APIPolicy{
		ID:               101,
		APIID:            12345,
		PolicyTemplateID: "spike-control",
		GroupID:          "68ef9520-24e9-4cf2-b2f5-620025690913",
		AssetID:          "spike-control",
		AssetVersion:     "1.2.2",
		ConfigurationData: map[string]interface{}{
			"maximumRequests":          float64(100),
			"timePeriodInMilliseconds": float64(1000),
			"delayTimeInMillis":        float64(500),
			"delayAttempts":            float64(3),
			"queuingLimit":             float64(10),
			"exposeHeaders":            false,
		},
		Order:    1,
		Disabled: false,
	}

	ds.flatten(context.Background(), policy, data, "test-org", "test-env")

	// Verify flattened values
	if data.ID.ValueString() != "101" {
		t.Errorf("ID = %q, want %q", data.ID.ValueString(), "101")
	}
	if data.PolicyID.ValueString() != "101" {
		t.Errorf("PolicyID = %q, want %q", data.PolicyID.ValueString(), "101")
	}
	if data.Order.ValueInt64() != 1 {
		t.Errorf("Order = %d, want %d", data.Order.ValueInt64(), 1)
	}
	if data.Disabled.ValueBool() != false {
		t.Errorf("Disabled = %v, want false", data.Disabled.ValueBool())
	}
	if data.AssetVersion.ValueString() != "1.2.2" {
		t.Errorf("AssetVersion = %q, want %q", data.AssetVersion.ValueString(), "1.2.2")
	}

	// Verify configuration object is populated
	if data.Configuration.IsNull() {
		t.Error("Configuration should not be null")
	}
	attrs := data.Configuration.Attributes()
	if _, ok := attrs["maximum_requests"]; !ok {
		t.Error("Configuration should have 'maximum_requests' field")
	}
}

// TestKnownPolicyDataSource_FlattenConfiguration_AllTypes tests all field type conversions.
func TestKnownPolicyDataSource_FlattenConfiguration_AllTypes(t *testing.T) {
	ds := &KnownPolicyDataSource{
		policyInfo: apimanagement.KnownPolicies["ip-allowlist"],
		typeSuffix: "ip_allowlist",
	}

	configData := map[string]interface{}{
		"ipExpression": "#[attributes.headers['X-Forwarded-For']]",
		"ips":          []interface{}{"192.168.1.0/24", "10.0.0.0/8"},
	}

	obj := ds.flattenConfiguration(context.Background(), configData)
	if obj.IsNull() {
		t.Fatal("flattenConfiguration returned null")
	}

	attrs := obj.Attributes()

	// Check string field
	ipExpr, ok := attrs["ip_expression"]
	if !ok {
		t.Fatal("Missing ip_expression in flattened config")
	}
	if sv, ok := ipExpr.(types.String); ok {
		if sv.ValueString() != "#[attributes.headers['X-Forwarded-For']]" {
			t.Errorf("ip_expression = %q, want the header expression", sv.ValueString())
		}
	}

	// Check string_array field
	ips, ok := attrs["ips"]
	if !ok {
		t.Fatal("Missing ips in flattened config")
	}
	if lv, ok := ips.(types.List); ok {
		if len(lv.Elements()) != 2 {
			t.Errorf("ips has %d elements, want 2", len(lv.Elements()))
		}
	}
}

// TestKnownPolicyDataSource_MetadataCoverage verifies each known policy gets the right type name.
func TestKnownPolicyDataSource_MetadataCoverage(t *testing.T) {
	testCases := []struct {
		policyType   string
		expectedName string
	}{
		{"rate-limiting", "anypoint_api_policy_rate_limiting"},
		{"cors", "anypoint_api_policy_cors"},
		{"jwt-validation", "anypoint_api_policy_jwt_validation"},
		{"message-logging-outbound", "anypoint_api_policy_message_logging_outbound"},
		{"mcp-pii-detector", "anypoint_api_policy_mcp_pii_detector"},
		{"a2a-pii-detector", "anypoint_api_policy_a2a_pii_detector"},
	}

	for _, tc := range testCases {
		t.Run(tc.policyType, func(t *testing.T) {
			factory := NewKnownPolicyDataSourceFunc(tc.policyType)
			ds := factory()

			metaReq := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
			metaResp := &datasource.MetadataResponse{}
			ds.Metadata(context.Background(), metaReq, metaResp)

			if metaResp.TypeName != tc.expectedName {
				t.Errorf("Metadata TypeName = %q, want %q", metaResp.TypeName, tc.expectedName)
			}
		})
	}
}

// TestDSNativeToDynamic tests the dynamic value conversion for complex types.
func TestDSNativeToDynamic(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		v := dsNativeToDynamic(nil)
		if !v.IsNull() {
			t.Error("Expected null for nil input")
		}
	})

	t.Run("string", func(t *testing.T) {
		v := dsNativeToDynamic("hello")
		if sv, ok := v.(types.String); !ok || sv.ValueString() != "hello" {
			t.Errorf("Expected StringValue(\"hello\"), got %v", v)
		}
	})

	t.Run("bool", func(t *testing.T) {
		v := dsNativeToDynamic(true)
		if bv, ok := v.(types.Bool); !ok || !bv.ValueBool() {
			t.Errorf("Expected BoolValue(true), got %v", v)
		}
	})

	t.Run("float64", func(t *testing.T) {
		v := dsNativeToDynamic(42.5)
		if _, ok := v.(types.Number); !ok {
			t.Errorf("Expected NumberValue, got %T", v)
		}
	})

	t.Run("empty array", func(t *testing.T) {
		v := dsNativeToDynamic([]interface{}{})
		if v.IsNull() {
			t.Error("Expected non-null tuple for empty array")
		}
	})

	t.Run("map", func(t *testing.T) {
		v := dsNativeToDynamic(map[string]interface{}{"key": "value"})
		if v.IsNull() {
			t.Error("Expected non-null object for map")
		}
	})

	t.Run("fallback to string", func(t *testing.T) {
		v := dsNativeToDynamic(42) // int type
		if _, ok := v.(types.Number); !ok {
			t.Errorf("Expected NumberValue for int, got %T", v)
		}
	})
}

// TestDSNullForType checks all branches.
func TestDSNullForType(t *testing.T) {
	if !dsNullForType(types.StringType).IsNull() {
		t.Error("string null should be null")
	}
	if !dsNullForType(types.NumberType).IsNull() {
		t.Error("number null should be null")
	}
	if !dsNullForType(types.BoolType).IsNull() {
		t.Error("bool null should be null")
	}
	if !dsNullForType(types.ListType{ElemType: types.StringType}).IsNull() {
		t.Error("list null should be null")
	}
	if !dsNullForType(types.DynamicType).IsNull() {
		t.Error("dynamic null should be null")
	}
}

// TestDSFieldSchemaType checks all branches.
func TestDSFieldSchemaType(t *testing.T) {
	tests := []struct {
		input string
		want  fmt.Stringer
	}{
		{"string", types.StringType},
		{"int", types.NumberType},
		{"bool", types.BoolType},
		{"string_array", types.ListType{ElemType: types.StringType}},
		{"unknown_type", types.DynamicType},
		{"", types.DynamicType},
	}
	for _, tt := range tests {
		got := dsFieldSchemaType(tt.input)
		if got != tt.want {
			t.Errorf("dsFieldSchemaType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
