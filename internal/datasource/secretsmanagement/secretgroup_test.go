package secretsmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewSecretGroupDataSource(t *testing.T) {
	dataSource := NewSecretGroupDataSource()

	if dataSource == nil {
		t.Error("NewSecretGroupDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestSecretGroupDataSource_Metadata(t *testing.T) {
	dataSource := NewSecretGroupDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_secret_groups" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_secret_groups")
	}
}

func TestSecretGroupDataSource_Schema(t *testing.T) {
	dataSource := NewSecretGroupDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"environment_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	computedAttrs := []string{"organization_id", "secret_groups"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}
}

func TestSecretGroupDataSource_Configure(t *testing.T) {
	dataSource := NewSecretGroupDataSource().(*SecretGroupDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: providerData,
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}

	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestSecretGroupDataSourceModel_Validation(t *testing.T) {
	model := SecretGroupDataSourceModel{}
	_ = model.OrganizationID
}

func TestSecretGroupDataSource_Read(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups"

	mockGroups := []secretsmgmt.SecretGroupResponse{
		{
			Name:         "group-one",
			Downloadable: true,
			Meta:         secretsmgmt.SecretGroupMeta{ID: "sg-id-1"},
			CurrentState: "ACTIVE",
		},
		{
			Name:         "group-two",
			Downloadable: false,
			Meta:         secretsmgmt.SecretGroupMeta{ID: "sg-id-2"},
			CurrentState: "CLEAR",
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockGroups)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewSecretGroupDataSource().(*SecretGroupDataSource)
	ds.client = &secretsmgmt.SecretGroupClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_groups":   tftypes.NewValue(tftypes.List{ElementType: stateType.(tftypes.Object).AttributeTypes["secret_groups"].(tftypes.List).ElementType}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw},
	}

	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SecretGroupDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if len(got.SecretGroups) != 2 {
		t.Fatalf("Expected 2 secret groups, got %d", len(got.SecretGroups))
	}
	if got.SecretGroups[0].ID.ValueString() != "sg-id-1" {
		t.Errorf("Expected first ID sg-id-1, got %s", got.SecretGroups[0].ID.ValueString())
	}
	if got.SecretGroups[1].Name.ValueString() != "group-two" {
		t.Errorf("Expected second name group-two, got %s", got.SecretGroups[1].Name.ValueString())
	}
	if got.OrganizationID.ValueString() != "test-org-id" {
		t.Errorf("Expected org test-org-id, got %s", got.OrganizationID.ValueString())
	}
}

func TestSecretGroupDataSource_Read_Error(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewSecretGroupDataSource().(*SecretGroupDataSource)
	ds.client = &secretsmgmt.SecretGroupClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_groups":   tftypes.NewValue(tftypes.List{ElementType: stateType.(tftypes.Object).AttributeTypes["secret_groups"].(tftypes.List).ElementType}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
	}
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw},
	}

	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func BenchmarkSecretGroupDataSource_Schema(b *testing.B) {
	dataSource := NewSecretGroupDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
