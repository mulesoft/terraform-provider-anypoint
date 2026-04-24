package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRoleGroupUsersResource(t *testing.T) {
	r := NewRoleGroupUsersResource()

	if r == nil {
		t.Error("NewRoleGroupUsersResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("RoleGroupUsersResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("RoleGroupUsersResource should implement ResourceWithImportState")
	}
}

func TestRoleGroupUsersResource_Metadata(t *testing.T) {
	r := NewRoleGroupUsersResource()
	testutil.TestResourceMetadata(t, r, "_rolegroup_users")
}

func TestRoleGroupUsersResource_Schema(t *testing.T) {
	res := NewRoleGroupUsersResource()

	requiredAttrs := []string{"rolegroup_id", "user_ids"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestRoleGroupUsersResource_Configure(t *testing.T) {
	res := NewRoleGroupUsersResource().(*RoleGroupUsersResource)

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

func TestRoleGroupUsersResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewRoleGroupUsersResource().(*RoleGroupUsersResource)

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

func TestRoleGroupUsersResource_ImportState(t *testing.T) {
	r := NewRoleGroupUsersResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestRoleGroupUsersResourceModel_Validation(t *testing.T) {
	model := RoleGroupUsersResourceModel{}
	_ = model.ID
}

func BenchmarkRoleGroupUsersResource_Schema(b *testing.B) {
	res := NewRoleGroupUsersResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
