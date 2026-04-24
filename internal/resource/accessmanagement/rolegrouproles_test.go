package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRoleGroupRolesResource(t *testing.T) {
	r := NewRoleGroupRolesResource()

	if r == nil {
		t.Error("NewRoleGroupRolesResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("RoleGroupRolesResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("RoleGroupRolesResource should implement ResourceWithImportState")
	}
}

func TestRoleGroupRolesResource_Metadata(t *testing.T) {
	r := NewRoleGroupRolesResource()
	testutil.TestResourceMetadata(t, r, "_rolegroup_roles")
}

func TestRoleGroupRolesResource_Schema(t *testing.T) {
	res := NewRoleGroupRolesResource()

	requiredAttrs := []string{"rolegroup_id", "roles"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestRoleGroupRolesResource_Configure(t *testing.T) {
	res := NewRoleGroupRolesResource().(*RoleGroupRolesResource)

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

func TestRoleGroupRolesResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewRoleGroupRolesResource().(*RoleGroupRolesResource)

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

func TestRoleGroupRolesResource_ImportState(t *testing.T) {
	r := NewRoleGroupRolesResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestRoleGroupRolesResourceModel_Validation(t *testing.T) {
	model := RoleGroupRolesResourceModel{}
	_ = model.ID
}

func BenchmarkRoleGroupRolesResource_Schema(b *testing.B) {
	res := NewRoleGroupRolesResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
