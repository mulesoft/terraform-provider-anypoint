package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewPrivateSpacesDataSource(t *testing.T) {
	ds := NewPrivateSpacesDataSource()

	if ds == nil {
		t.Error("NewPrivateSpacesDataSource() returned nil")
	}

	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("data source should implement DataSourceWithConfigure")
	}
}

func TestPrivateSpacesDataSource_Metadata(t *testing.T) {
	ds := NewPrivateSpacesDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)

	if resp.TypeName != "anypoint_private_spaces" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "anypoint_private_spaces")
	}
}

func TestPrivateSpacesDataSource_Schema(t *testing.T) {
	ds := NewPrivateSpacesDataSource()

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", schemaResp.Diagnostics.Errors())
	}

	// organization_id is the only required input.
	orgAttr, ok := schemaResp.Schema.Attributes["organization_id"]
	if !ok {
		t.Fatal("Missing required attribute: organization_id")
	}
	if !orgAttr.IsRequired() {
		t.Error("Attribute organization_id should be required")
	}

	// private_spaces is the computed output list.
	psAttr, ok := schemaResp.Schema.Attributes["private_spaces"]
	if !ok {
		t.Fatal("Missing computed attribute: private_spaces")
	}
	if !psAttr.IsComputed() {
		t.Error("Attribute private_spaces should be computed")
	}
}

func TestPrivateSpacesDataSource_Configure(t *testing.T) {
	ds := NewPrivateSpacesDataSource().(*PrivateSpacesDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: providerData}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure() reported errors: %v", resp.Diagnostics.Errors())
	}
	if ds.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestPrivateSpacesDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewPrivateSpacesDataSource().(*PrivateSpacesDataSource)

	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: "invalid-data"}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}
	if ds.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestPrivateSpacesDataSource_Read(t *testing.T) {
	listPath := "/runtimefabric/api/organizations/test-org-id/privatespaces"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "GET", listPath)
			// The live API returns an OBJECT ENVELOPE wrapping the array (a
			// bare-array assumption broke the terraform plan). Region shape
			// differs by payload, so cover BOTH here:
			//   ps-1: region at the TOP LEVEL — the real LIST wire shape
			//         (per the privateSpaceList data type).
			//   ps-2: region nested under network.region — the single-space
			//         shape, exercising the fallback branch.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"id":                 "ps-1",
						"name":               "prod-space",
						"status":             "ACTIVE",
						"statusMessage":      "running",
						"region":             "us-east-1",
						"organizationId":     "test-org-id",
						"rootOrganizationId": "root-org",
					},
					{
						"id":                 "ps-2",
						"name":               "staging-space",
						"status":             "CREATING",
						"statusMessage":      "provisioning",
						"network":            map[string]interface{}{"region": "us-west-2"},
						"organizationId":     "test-org-id",
						"rootOrganizationId": "root-org",
					},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	ds := NewPrivateSpacesDataSource().(*PrivateSpacesDataSource)
	ds.client = &ch2client.PrivateSpacesClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	configType := schemaResp.Schema.Type().TerraformType(ctx)

	// Nested object type for each private_spaces entry.
	psObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":                   tftypes.String,
		"name":                 tftypes.String,
		"status":               tftypes.String,
		"status_message":       tftypes.String,
		"region":               tftypes.String,
		"organization_id":      tftypes.String,
		"root_organization_id": tftypes.String,
	}}

	configRaw := tftypes.NewValue(configType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"private_spaces":  tftypes.NewValue(tftypes.List{ElementType: psObjType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got PrivateSpacesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if len(got.PrivateSpaces) != 2 {
		t.Fatalf("Expected 2 private spaces, got %d", len(got.PrivateSpaces))
	}
	if got.PrivateSpaces[0].ID.ValueString() != "ps-1" {
		t.Errorf("Expected first PS id 'ps-1', got '%s'", got.PrivateSpaces[0].ID.ValueString())
	}
	if got.PrivateSpaces[0].Name.ValueString() != "prod-space" {
		t.Errorf("Expected first PS name 'prod-space', got '%s'", got.PrivateSpaces[0].Name.ValueString())
	}
	if got.PrivateSpaces[0].Status.ValueString() != "ACTIVE" {
		t.Errorf("Expected first PS status 'ACTIVE', got '%s'", got.PrivateSpaces[0].Status.ValueString())
	}
	// ps-1 region comes from the TOP-LEVEL field (the real list wire shape).
	if got.PrivateSpaces[0].Region.ValueString() != "us-east-1" {
		t.Errorf("Expected first PS region 'us-east-1', got '%s'", got.PrivateSpaces[0].Region.ValueString())
	}
	if got.PrivateSpaces[1].ID.ValueString() != "ps-2" {
		t.Errorf("Expected second PS id 'ps-2', got '%s'", got.PrivateSpaces[1].ID.ValueString())
	}
	// ps-2 has no top-level region, so it must fall back to network.region.
	if got.PrivateSpaces[1].Region.ValueString() != "us-west-2" {
		t.Errorf("Expected second PS region 'us-west-2', got '%s'", got.PrivateSpaces[1].Region.ValueString())
	}
	if got.PrivateSpaces[1].RootOrganizationID.ValueString() != "root-org" {
		t.Errorf("Expected second PS root org 'root-org', got '%s'", got.PrivateSpaces[1].RootOrganizationID.ValueString())
	}
}
