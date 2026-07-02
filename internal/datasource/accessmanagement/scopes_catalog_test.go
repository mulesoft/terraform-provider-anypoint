package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewScopesCatalogDataSource(t *testing.T) {
	dataSource := NewScopesCatalogDataSource()

	if dataSource == nil {
		t.Error("NewScopesCatalogDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("ScopesCatalogDataSource does not implement DataSourceWithConfigure")
	}
}

func TestScopesCatalogDataSource_Metadata(t *testing.T) {
	dataSource := NewScopesCatalogDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_scopes_catalog" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_scopes_catalog")
	}
}

func TestScopesCatalogDataSource_Schema(t *testing.T) {
	dataSource := NewScopesCatalogDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check optional attributes
	optionalAttrs := []string{"include_internal"}
	for _, attrName := range optionalAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() {
				t.Errorf("Schema() attribute %s should be optional", attrName)
			}
		} else {
			t.Errorf("Schema() missing optional attribute: %s", attrName)
		}
	}

	// Check computed attributes
	computedAttrs := []string{"scopes"}
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

func TestScopesCatalogDataSource_Configure(t *testing.T) {
	dataSource := NewScopesCatalogDataSource().(*ScopesCatalogDataSource)

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

func TestScopesCatalogDataSourceModel_Validation(t *testing.T) {
	model := ScopesCatalogDataSourceModel{}
	_ = model.Scopes
	_ = model.IncludeInternal
}

func BenchmarkScopesCatalogDataSource_Schema(b *testing.B) {
	dataSource := NewScopesCatalogDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
