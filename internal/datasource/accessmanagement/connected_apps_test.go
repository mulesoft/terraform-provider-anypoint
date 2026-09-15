package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppsDataSource(t *testing.T) {
	dataSource := NewConnectedAppsDataSource()

	if dataSource == nil {
		t.Error("NewConnectedAppsDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("ConnectedAppsDataSource does not implement DataSourceWithConfigure")
	}
}

func TestConnectedAppsDataSource_Metadata(t *testing.T) {
	dataSource := NewConnectedAppsDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_connected_apps" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_connected_apps")
	}
}

func TestConnectedAppsDataSource_Schema(t *testing.T) {
	dataSource := NewConnectedAppsDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check optional and computed attributes
	optionalComputedAttrs := []string{"organization_id"}
	for _, attrName := range optionalComputedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() && !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be optional and computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing attribute: %s", attrName)
		}
	}

	// Check computed attributes
	computedAttrs := []string{"apps"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}
}

func TestConnectedAppsDataSource_Configure(t *testing.T) {
	dataSource := NewConnectedAppsDataSource().(*ConnectedAppsDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
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

func BenchmarkConnectedAppsDataSource_Schema(b *testing.B) {
	dataSource := NewConnectedAppsDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
