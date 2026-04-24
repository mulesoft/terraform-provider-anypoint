package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewCertificatePinsetDataSource(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource()

	if dataSource == nil {
		t.Error("NewCertificatePinsetDataSource() returned nil")
	}

	var _ datasource.DataSource = dataSource
	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestCertificatePinsetDataSource_Metadata(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_secret_group_certificate_pinsets" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_secret_group_certificate_pinsets")
	}
}

func TestCertificatePinsetDataSource_Schema(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"environment_id", "secret_group_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	computedAttrs := []string{"organization_id", "certificate_pinsets"}
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

func TestCertificatePinsetDataSource_Configure(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource().(*CertificatePinsetDataSource)

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

func TestCertificatePinsetDataSourceModel_Validation(t *testing.T) {
	model := CertificatePinsetDataSourceModel{}
	_ = model.OrganizationID
}

func BenchmarkCertificatePinsetDataSource_Schema(b *testing.B) {
	dataSource := NewCertificatePinsetDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
