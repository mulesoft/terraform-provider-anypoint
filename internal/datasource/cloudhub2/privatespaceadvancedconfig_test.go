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

func TestNewPrivateSpaceAdvancedConfigDataSource(t *testing.T) {
	dataSource := NewPrivateSpaceAdvancedConfigDataSource()

	if dataSource == nil {
		t.Error("NewPrivateSpaceAdvancedConfigDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestPrivateSpaceAdvancedConfigDataSource_Metadata(t *testing.T) {
	dataSource := NewPrivateSpaceAdvancedConfigDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_privatespace_advanced_config" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_privatespace_advanced_config")
	}
}

func TestPrivateSpaceAdvancedConfigDataSource_Schema(t *testing.T) {
	dataSource := NewPrivateSpaceAdvancedConfigDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"private_space_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	computedAttrs := []string{"id", "enable_iam_role", "ingress_configuration"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}

	optionalAttrs := []string{"organization_id"}
	for _, attrName := range optionalAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() {
				t.Errorf("Schema() attribute %s should be optional", attrName)
			}
		} else {
			t.Errorf("Schema() missing optional attribute: %s", attrName)
		}
	}
}

func TestPrivateSpaceAdvancedConfigDataSource_Configure(t *testing.T) {
	dataSource := NewPrivateSpaceAdvancedConfigDataSource().(*PrivateSpaceAdvancedConfigDataSource)

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

func TestPrivateSpaceAdvancedConfigDataSourceModel_Validation(t *testing.T) {
	model := PrivateSpaceAdvancedConfigDataSourceModel{}
	_ = model.ID
}

func TestPrivateSpaceAdvancedConfigDataSource_Read(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id"

	mockPrivateSpace := ch2client.PrivateSpace{
		ID:            "test-ps-id",
		Name:          "Test Private Space",
		EnableIAMRole: true,
		IngressConfiguration: ch2client.PrivateSpaceIngressConfig{
			ReadResponseTimeout: 60000,
			Protocol:            "https",
			Logs: ch2client.PrivateSpaceIngressLogs{
				PortLogLevel: "info",
				Filters: []ch2client.PrivateSpaceLogFilter{
					{IP: "192.168.1.1", Level: "debug"},
				},
			},
			Deployment: ch2client.PrivateSpaceIngressDeployment{
				Status:            "deployed",
				LastSeenTimestamp: 1234567890,
			},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceAdvancedConfigDataSource().(*PrivateSpaceAdvancedConfigDataSource)
	ds.client = &ch2client.PrivateSpaceAdvancedConfigClient{
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
		"id":                    tftypes.NewValue(tftypes.String, nil),
		"private_space_id":      tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":       tftypes.NewValue(tftypes.String, "test-org-id"),
		"enable_iam_role":       tftypes.NewValue(tftypes.Bool, nil),
		"ingress_configuration": tftypes.NewValue(schemaResp.Schema.Attributes["ingress_configuration"].GetType().TerraformType(ctx), nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestPrivateSpaceAdvancedConfigDataSource_Read_Error(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceAdvancedConfigDataSource().(*PrivateSpaceAdvancedConfigDataSource)
	ds.client = &ch2client.PrivateSpaceAdvancedConfigClient{
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
		"id":                    tftypes.NewValue(tftypes.String, nil),
		"private_space_id":      tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":       tftypes.NewValue(tftypes.String, "test-org-id"),
		"enable_iam_role":       tftypes.NewValue(tftypes.Bool, nil),
		"ingress_configuration": tftypes.NewValue(schemaResp.Schema.Attributes["ingress_configuration"].GetType().TerraformType(ctx), nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func TestPrivateSpaceAdvancedConfigDataSource_Read_NotFound(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/nonexistent-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceAdvancedConfigDataSource().(*PrivateSpaceAdvancedConfigDataSource)
	ds.client = &ch2client.PrivateSpaceAdvancedConfigClient{
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
		"id":                    tftypes.NewValue(tftypes.String, nil),
		"private_space_id":      tftypes.NewValue(tftypes.String, "nonexistent-id"),
		"organization_id":       tftypes.NewValue(tftypes.String, "test-org-id"),
		"enable_iam_role":       tftypes.NewValue(tftypes.Bool, nil),
		"ingress_configuration": tftypes.NewValue(schemaResp.Schema.Attributes["ingress_configuration"].GetType().TerraformType(ctx), nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on not found")
	}
}

func BenchmarkPrivateSpaceAdvancedConfigDataSource_Schema(b *testing.B) {
	dataSource := NewPrivateSpaceAdvancedConfigDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
