package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTLSContextResource(t *testing.T) {
	r := NewTLSContextResource()

	if r == nil {
		t.Error("NewTLSContextResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TLSContextResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TLSContextResource should implement ResourceWithImportState")
	}
}

func TestTLSContextResource_Metadata(t *testing.T) {
	r := NewTLSContextResource()
	testutil.TestResourceMetadata(t, r, "_flex_tls_context")
}

func TestTLSContextResource_Schema(t *testing.T) {
	res := NewTLSContextResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id"}
	optionalAttrs := []string{"organization_id", "keystore_id", "truststore_id", "min_tls_version", "max_tls_version", "alpn_protocols", "cipher_suites", "enable_client_cert_validation", "skip_server_cert_validation"}
	computedAttrs := []string{"id", "target", "min_tls_version", "max_tls_version", "enable_client_cert_validation", "skip_server_cert_validation", "organization_id", "expiration_date"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTLSContextResource_Configure(t *testing.T) {
	res := NewTLSContextResource().(*TLSContextResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestTLSContextResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTLSContextResource().(*TLSContextResource)

	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data",
	}
	resp := &resource.ConfigureResponse{}

	res.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}

	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestTLSContextResource_ImportState(t *testing.T) {
	r := NewTLSContextResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTLSContextResourceModel_Validation(t *testing.T) {
	model := TLSContextResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.Target
	_ = model.KeystoreID
	_ = model.TruststoreID
	_ = model.MinTLSVersion
	_ = model.MaxTLSVersion
	_ = model.AlpnProtocols
	_ = model.CipherSuites
	_ = model.EnableClientCertValidation
	_ = model.SkipServerCertValidation
	_ = model.ExpirationDate
}

func BenchmarkTLSContextResource_Schema(b *testing.B) {
	res := NewTLSContextResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
