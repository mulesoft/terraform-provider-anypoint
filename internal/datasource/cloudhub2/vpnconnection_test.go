package cloudhub2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewVPNConnectionDataSource(t *testing.T) {
	dataSource := NewVPNConnectionDataSource()

	if dataSource == nil {
		t.Error("NewVPNConnectionDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestVPNConnectionDataSource_Metadata(t *testing.T) {
	dataSource := NewVPNConnectionDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_vpn_connection" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_vpn_connection")
	}
}

func TestVPNConnectionDataSource_Schema(t *testing.T) {
	dataSource := NewVPNConnectionDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"private_space_id", "connection_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	computedAttrs := []string{"id", "name", "vpns"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}

	// Verify optional + computed attributes
	optionalComputedAttrs := []string{"organization_id"}
	for _, attrName := range optionalComputedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() || !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be optional and computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing optional+computed attribute: %s", attrName)
		}
	}

	// Verify nested attributes in vpns
	vpnsAttr, exists := resp.Schema.Attributes["vpns"]
	if !exists {
		t.Fatal("Schema() missing vpns attribute")
	}

	listNestedAttr, ok := vpnsAttr.(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("vpns attribute is not a ListNestedAttribute")
	}

	vpnNestedAttrs := []string{
		"local_asn", "remote_asn", "remote_ip_address", "static_routes",
		"vpn_tunnels", "name", "connection_name", "vpn_connection_status",
		"vpn_id", "connection_id",
	}
	for _, nestedAttrName := range vpnNestedAttrs {
		if _, exists := listNestedAttr.NestedObject.Attributes[nestedAttrName]; !exists {
			t.Errorf("Schema() vpns missing nested attribute: %s", nestedAttrName)
		}
	}

	// Verify nested tunnel attributes
	tunnelsAttr, exists := listNestedAttr.NestedObject.Attributes["vpn_tunnels"]
	if !exists {
		t.Fatal("Schema() vpns missing vpn_tunnels attribute")
	}

	tunnelListNestedAttr, ok := tunnelsAttr.(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("vpn_tunnels attribute is not a ListNestedAttribute")
	}

	tunnelNestedAttrs := []string{"psk", "ptp_cidr", "startup_action", "is_logs_enabled"}
	for _, nestedAttrName := range tunnelNestedAttrs {
		if _, exists := tunnelListNestedAttr.NestedObject.Attributes[nestedAttrName]; !exists {
			t.Errorf("Schema() vpn_tunnels missing nested attribute: %s", nestedAttrName)
		}
	}

	// Verify psk is marked as sensitive
	pskAttr := tunnelListNestedAttr.NestedObject.Attributes["psk"]
	if stringAttr, ok := pskAttr.(schema.StringAttribute); ok {
		if !stringAttr.Sensitive {
			t.Error("Schema() psk attribute should be marked as sensitive")
		}
	}
}

func TestVPNConnectionDataSource_Configure(t *testing.T) {
	dataSource := NewVPNConnectionDataSource().(*VPNConnectionDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: providerData,
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}

	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestVPNConnectionDataSourceModel_Validation(t *testing.T) {
	model := VPNConnectionDataSourceModel{}
	_ = model.ID
}

func BenchmarkVPNConnectionDataSource_Schema(b *testing.B) {
	dataSource := NewVPNConnectionDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
