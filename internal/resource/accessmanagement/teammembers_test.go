package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTeamMembersResource(t *testing.T) {
	r := NewTeamMembersResource()

	if r == nil {
		t.Error("NewTeamMembersResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TeamMembersResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TeamMembersResource should implement ResourceWithImportState")
	}
}

func TestTeamMembersResource_Metadata(t *testing.T) {
	r := NewTeamMembersResource()
	testutil.TestResourceMetadata(t, r, "_team_members")
}

func TestTeamMembersResource_Schema(t *testing.T) {
	res := NewTeamMembersResource()

	requiredAttrs := []string{"team_id", "members"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTeamMembersResource_Configure(t *testing.T) {
	res := NewTeamMembersResource().(*TeamMembersResource)

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

func TestTeamMembersResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTeamMembersResource().(*TeamMembersResource)

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

func TestTeamMembersResource_ImportState(t *testing.T) {
	r := NewTeamMembersResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTeamMembersResourceModel_Validation(t *testing.T) {
	model := TeamMembersResourceModel{}
	_ = model.ID
}

func BenchmarkTeamMembersResource_Schema(b *testing.B) {
	res := NewTeamMembersResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
