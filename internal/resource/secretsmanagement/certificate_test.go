package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewCertificateResource(t *testing.T) {
	r := NewCertificateResource()

	if r == nil {
		t.Error("NewCertificateResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("CertificateResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("CertificateResource should implement ResourceWithImportState")
	}
}

func TestCertificateResource_Metadata(t *testing.T) {
	r := NewCertificateResource()
	testutil.TestResourceMetadata(t, r, "_secret_group_certificate")
}

func TestCertificateResource_Schema(t *testing.T) {
	res := NewCertificateResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id", "certificate_base64"}
	optionalAttrs := []string{"type", "organization_id"}
	computedAttrs := []string{"id", "expiration_date", "algorithm", "type", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestCertificateResource_Configure(t *testing.T) {
	res := NewCertificateResource().(*CertificateResource)

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

func TestCertificateResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewCertificateResource().(*CertificateResource)

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

func TestCertificateResource_ImportState(t *testing.T) {
	r := NewCertificateResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestCertificateResourceModel_Validation(t *testing.T) {
	model := CertificateResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.Type
	_ = model.CertStoreB64
	_ = model.ExpirationDate
	_ = model.Algorithm
}

func BenchmarkCertificateResource_Schema(b *testing.B) {
	res := NewCertificateResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
