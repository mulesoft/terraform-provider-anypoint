package cloudhub2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewPrivateSpaceAssociationResource(t *testing.T) {
	r := NewPrivateSpaceAssociationResource()

	if r == nil {
		t.Error("NewPrivateSpaceAssociationResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestPrivateSpaceAssociationResource_Metadata(t *testing.T) {
	r := NewPrivateSpaceAssociationResource()
	testutil.TestResourceMetadata(t, r, "_private_space_association")
}

func TestPrivateSpaceAssociationResource_Schema(t *testing.T) {
	res := NewPrivateSpaceAssociationResource()

	requiredAttrs := []string{"private_space_id", "associations"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestPrivateSpaceAssociationResource_Configure(t *testing.T) {
	res := NewPrivateSpaceAssociationResource().(*PrivateSpaceAssociationResource)

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

func TestPrivateSpaceAssociationResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewPrivateSpaceAssociationResource().(*PrivateSpaceAssociationResource)

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

func TestPrivateSpaceAssociationResource_ImportState(t *testing.T) {
	res := NewPrivateSpaceAssociationResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-id",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(stateType, nil),
		},
	}

	if importableResource, ok := res.(resource.ResourceWithImportState); ok {
		importableResource.ImportState(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("ImportState() has errors: %v", resp.Diagnostics.Errors())
		}
	} else {
		t.Error("Resource does not implement ResourceWithImportState")
	}
}

func TestPrivateSpaceAssociationResourceModel_Validation(t *testing.T) {
	model := PrivateSpaceAssociationResourceModel{}
	_ = model.ID
}

func TestPrivateSpaceAssociationResource_Read(t *testing.T) {
	res := NewPrivateSpaceAssociationResource().(*PrivateSpaceAssociationResource)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	assocElemType := objType.AttributeTypes["associations"].(tftypes.List).ElementType
	createdElemType := objType.AttributeTypes["created_associations"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"private_space_id":     tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"associations":         tftypes.NewValue(tftypes.List{ElementType: assocElemType}, nil),
		"created_associations": tftypes.NewValue(tftypes.List{ElementType: createdElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got PrivateSpaceAssociationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.PrivateSpaceID.ValueString() != "test-ps-id" {
		t.Errorf("Expected PrivateSpaceID test-ps-id, got %s", got.PrivateSpaceID.ValueString())
	}
}

func BenchmarkPrivateSpaceAssociationResource_Schema(b *testing.B) {
	res := NewPrivateSpaceAssociationResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
