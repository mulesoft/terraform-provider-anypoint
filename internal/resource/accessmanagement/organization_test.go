package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewOrganizationResource(t *testing.T) {
	r := NewOrganizationResource()

	if r == nil {
		t.Error("NewOrganizationResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("OrganizationResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("OrganizationResource must implement ResourceWithImportState so `terraform import` works")
	}
}

func TestOrganizationResource_Metadata(t *testing.T) {
	r := NewOrganizationResource()
	testutil.TestResourceMetadata(t, r, "_organization")
}

func TestOrganizationResource_Schema(t *testing.T) {
	res := NewOrganizationResource()

	requiredAttrs := []string{"name", "parent_organization_id", "owner_id"}
	optionalAttrs := []string{"entitlements"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestOrganizationResource_Configure(t *testing.T) {
	res := NewOrganizationResource().(*OrganizationResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestOrganizationResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewOrganizationResource().(*OrganizationResource)

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

func TestOrganizationResourceModel_Validation(t *testing.T) {
	model := OrganizationResourceModel{}
	_ = model.ID
}

func BenchmarkOrganizationResource_Schema(b *testing.B) {
	res := NewOrganizationResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
