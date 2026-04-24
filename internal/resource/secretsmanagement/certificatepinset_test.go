package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewCertificatePinsetResource(t *testing.T) {
	r := NewCertificatePinsetResource()

	if r == nil {
		t.Error("NewCertificatePinsetResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("CertificatePinsetResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("CertificatePinsetResource should implement ResourceWithImportState")
	}
}

func TestCertificatePinsetResource_Metadata(t *testing.T) {
	r := NewCertificatePinsetResource()
	testutil.TestResourceMetadata(t, r, "_secret_group_certificate_pinset")
}

func TestCertificatePinsetResource_Schema(t *testing.T) {
	res := NewCertificatePinsetResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id", "certificate_pinset_base64"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "expiration_date", "algorithm", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestCertificatePinsetResource_Configure(t *testing.T) {
	res := NewCertificatePinsetResource().(*CertificatePinsetResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.ClientConfig{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestCertificatePinsetResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewCertificatePinsetResource().(*CertificatePinsetResource)

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

func TestCertificatePinsetResource_ImportState(t *testing.T) {
	r := NewCertificatePinsetResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestCertificatePinsetResourceModel_Validation(t *testing.T) {
	model := CertificatePinsetResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.PinsetB64
	_ = model.ExpirationDate
	_ = model.Algorithm
}

func BenchmarkCertificatePinsetResource_Schema(b *testing.B) {
	res := NewCertificatePinsetResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
