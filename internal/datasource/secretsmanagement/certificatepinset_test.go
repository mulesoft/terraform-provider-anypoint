package secretsmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewCertificatePinsetDataSource(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource()

	if dataSource == nil {
		t.Error("NewCertificatePinsetDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestCertificatePinsetDataSource_Metadata(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_secret_group_certificate_pinsets" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_secret_group_certificate_pinsets")
	}
}

func TestCertificatePinsetDataSource_Schema(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"environment_id", "secret_group_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	computedAttrs := []string{"organization_id", "certificate_pinsets"}
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

func TestCertificatePinsetDataSource_Configure(t *testing.T) {
	dataSource := NewCertificatePinsetDataSource().(*CertificatePinsetDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
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

func TestCertificatePinsetDataSourceModel_Validation(t *testing.T) {
	model := CertificatePinsetDataSourceModel{}
	_ = model.OrganizationID
}

func TestCertificatePinsetDataSource_Read(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/certificatePinsets"

	mockPinsets := []secretsmgmt.CertificatePinsetResponse{
		{Name: "pinset-one", Meta: secretsmgmt.SecretGroupMeta{ID: "ps-id-1"}},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockPinsets)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewCertificatePinsetDataSource().(*CertificatePinsetDataSource)
	ds.client = &secretsmgmt.CertificatePinsetClient{
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["certificate_pinsets"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id":     tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":      tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":     tftypes.NewValue(tftypes.String, "test-sg-id"),
		"certificate_pinsets": tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got CertificatePinsetDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.CertificatePinsets) != 1 {
		t.Fatalf("Expected 1 pinset, got %d", len(got.CertificatePinsets))
	}
	if got.CertificatePinsets[0].ID.ValueString() != "ps-id-1" {
		t.Errorf("Expected ID ps-id-1, got %s", got.CertificatePinsets[0].ID.ValueString())
	}
}

func TestCertificatePinsetDataSource_Read_Error(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/certificatePinsets"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewCertificatePinsetDataSource().(*CertificatePinsetDataSource)
	ds.client = &secretsmgmt.CertificatePinsetClient{
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["certificate_pinsets"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id":     tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":      tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":     tftypes.NewValue(tftypes.String, "test-sg-id"),
		"certificate_pinsets": tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func BenchmarkCertificatePinsetDataSource_Schema(b *testing.B) {
	dataSource := NewCertificatePinsetDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
