package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
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

	requiredAttrs := []string{"private_space_id"}
	optionalAttrs := []string{"organization_id", "associations"}
	computedAttrs := []string{"id", "associations"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestPrivateSpaceAssociationResource_Configure(t *testing.T) {
	res := NewPrivateSpaceAssociationResource().(*PrivateSpaceAssociationResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
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

func TestPrivateSpaceAssociationResource_ImportState_SimpleID(t *testing.T) {
	res := NewPrivateSpaceAssociationResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{ID: "test-space-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}

	res.(resource.ResourceWithImportState).ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with simple ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var model PrivateSpaceAssociationResourceModel
	resp.State.Get(ctx, &model)
	if model.ID.ValueString() != "test-space-id-associations" {
		t.Errorf("expected id = %q, got %q", "test-space-id-associations", model.ID.ValueString())
	}
	if model.PrivateSpaceID.ValueString() != "test-space-id" {
		t.Errorf("expected private_space_id = %q, got %q", "test-space-id", model.PrivateSpaceID.ValueString())
	}
	if !model.OrganizationID.IsNull() {
		t.Errorf("expected organization_id to be null for simple import, got %q", model.OrganizationID.ValueString())
	}
}

func TestPrivateSpaceAssociationResource_ImportState_CompositeID(t *testing.T) {
	res := NewPrivateSpaceAssociationResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{ID: "test-org-id/test-space-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}

	res.(resource.ResourceWithImportState).ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with composite ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var model PrivateSpaceAssociationResourceModel
	resp.State.Get(ctx, &model)
	if model.ID.ValueString() != "test-space-id-associations" {
		t.Errorf("expected id = %q, got %q", "test-space-id-associations", model.ID.ValueString())
	}
	if model.PrivateSpaceID.ValueString() != "test-space-id" {
		t.Errorf("expected private_space_id = %q, got %q", "test-space-id", model.PrivateSpaceID.ValueString())
	}
	if model.OrganizationID.ValueString() != "test-org-id" {
		t.Errorf("expected organization_id = %q, got %q", "test-org-id", model.OrganizationID.ValueString())
	}
}

func TestPrivateSpaceAssociationResourceModel_Validation(t *testing.T) {
	model := PrivateSpaceAssociationResourceModel{}
	_ = model.ID
}

func TestPrivateSpaceAssociationResource_Read(t *testing.T) {
	assocPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/associations"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		assocPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{"id": "assoc-1", "organizationId": "test-org-id", "environmentId": "env-a"},
				{"id": "assoc-2", "organizationId": "test-org-id", "environmentId": "env-b"},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceAssociationResource().(*PrivateSpaceAssociationResource)
	res.client = &ch2client.PrivateSpaceAssociationClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	assocElemType := objType.AttributeTypes["associations"].(tftypes.List).ElementType
	createdElemType := objType.AttributeTypes["created_associations"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "test-ps-id-associations"),
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
		t.Errorf("PrivateSpaceID = %q, want test-ps-id", got.PrivateSpaceID.ValueString())
	}

	var assocs []struct {
		OrganizationID string `tfsdk:"organization_id"`
		Environment    string `tfsdk:"environment"`
	}
	if diags := got.Associations.ElementsAs(ctx, &assocs, false); diags.HasError() {
		t.Fatalf("ElementsAs associations errors: %v", diags.Errors())
	}
	if len(assocs) != 2 {
		t.Fatalf("associations len = %d, want 2", len(assocs))
	}
	if assocs[0].Environment != "env-a" {
		t.Errorf("assocs[0].Environment = %q, want env-a", assocs[0].Environment)
	}
	if assocs[1].Environment != "env-b" {
		t.Errorf("assocs[1].Environment = %q, want env-b", assocs[1].Environment)
	}

	var created []struct {
		ID             string `tfsdk:"id"`
		OrganizationID string `tfsdk:"organization_id"`
		Environment    string `tfsdk:"environment"`
	}
	if diags := got.CreatedAssociations.ElementsAs(ctx, &created, false); diags.HasError() {
		t.Fatalf("ElementsAs created_associations errors: %v", diags.Errors())
	}
	if len(created) != 2 {
		t.Fatalf("created_associations len = %d, want 2", len(created))
	}
	if created[0].ID != "assoc-1" {
		t.Errorf("created[0].ID = %q, want assoc-1", created[0].ID)
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
