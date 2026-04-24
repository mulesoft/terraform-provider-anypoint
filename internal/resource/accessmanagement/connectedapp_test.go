package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppResource(t *testing.T) {
	r := NewConnectedAppResource()

	if r == nil {
		t.Error("NewConnectedAppResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("ConnectedAppResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("ConnectedAppResource should implement ResourceWithImportState")
	}
}

func TestConnectedAppResource_Metadata(t *testing.T) {
	r := NewConnectedAppResource()
	testutil.TestResourceMetadata(t, r, "_connected_app")
}

func TestConnectedAppResource_Schema(t *testing.T) {
	res := NewConnectedAppResource()

	requiredAttrs := []string{"client_id", "owner_org_id", "client_name", "client_secret", "grant_types", "audience"}
	optionalAttrs := []string{"public_keys", "redirect_uris", "scopes"}
	computedAttrs := []string{"enabled", "generate_iss_claim_without_token"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestConnectedAppResource_Configure(t *testing.T) {
	res := NewConnectedAppResource().(*ConnectedAppResource)

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

func TestConnectedAppResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewConnectedAppResource().(*ConnectedAppResource)

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

func TestConnectedAppResource_ImportState(t *testing.T) {
	r := NewConnectedAppResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestConnectedAppResourceModel_Validation(t *testing.T) {
	model := ConnectedAppResourceModel{}
	_ = model.ClientID
}

func BenchmarkConnectedAppResource_Schema(b *testing.B) {
	res := NewConnectedAppResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
