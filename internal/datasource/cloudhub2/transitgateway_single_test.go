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

func TestNewTransitGatewaySingleDataSource(t *testing.T) {
	ds := NewTransitGatewaySingleDataSource()

	if ds == nil {
		t.Error("NewTransitGatewaySingleDataSource() returned nil")
	}

	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("data source should implement DataSourceWithConfigure")
	}
}

func TestTransitGatewaySingleDataSource_Metadata(t *testing.T) {
	ds := NewTransitGatewaySingleDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)

	// The SINGULAR data source shares the resource's type name — the plural list DS
	// is anypoint_transit_gateway_connections (with the trailing 's').
	if resp.TypeName != "anypoint_transit_gateway_connection" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "anypoint_transit_gateway_connection")
	}
}

func TestTransitGatewaySingleDataSource_Schema(t *testing.T) {
	ds := NewTransitGatewaySingleDataSource()

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", schemaResp.Diagnostics.Errors())
	}

	// id/org/ps are the lookup key and must be required.
	requiredAttrs := []string{"id", "organization_id", "private_space_id"}
	for _, attr := range requiredAttrs {
		a, ok := schemaResp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Missing required attribute: %s", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Attribute %s should be required", attr)
		}
	}

	// The richer field set the plural DS does not expose must all be computed.
	computedAttrs := []string{
		"name", "aws_transit_gateway_id", "resource_share_id",
		"resource_share_account", "region", "status", "attachment", "routes",
	}
	for _, attr := range computedAttrs {
		a, ok := schemaResp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Missing computed attribute: %s", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Attribute %s should be computed", attr)
		}
	}
}

func TestTransitGatewaySingleDataSource_Configure(t *testing.T) {
	ds := NewTransitGatewaySingleDataSource().(*TransitGatewaySingleDataSource)

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

func TestTransitGatewaySingleDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewTransitGatewaySingleDataSource().(*TransitGatewaySingleDataSource)

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

func TestTransitGatewaySingleDataSource_Read(t *testing.T) {
	// Singular DS reads ONE connection by id via the by-id GET primitive.
	tgwGetPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwGetPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "tgw-1",
				"name": "prod-tgw",
				"spec": map[string]interface{}{
					"resourceShare": map[string]interface{}{
						"id":      "share-uuid",
						"account": "123456789012",
					},
					"region": "us-east-1",
				},
				"status": map[string]interface{}{
					"gateway":     "available",
					"attachment":  "attached",
					"tgwResource": "tgw-0abc123",
					"routes":      []string{"10.0.0.0/8", "172.16.0.0/12"},
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
	ds := NewTransitGatewaySingleDataSource().(*TransitGatewaySingleDataSource)
	ds.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	configType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(configType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "tgw-1"),
		"organization_id":        tftypes.NewValue(tftypes.String, "test-org-id"),
		"private_space_id":       tftypes.NewValue(tftypes.String, "test-ps-id"),
		"name":                   tftypes.NewValue(tftypes.String, nil),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, nil),
		"resource_share_id":      tftypes.NewValue(tftypes.String, nil),
		"resource_share_account": tftypes.NewValue(tftypes.String, nil),
		"region":                 tftypes.NewValue(tftypes.String, nil),
		"status":                 tftypes.NewValue(tftypes.String, nil),
		"attachment":             tftypes.NewValue(tftypes.String, nil),
		"routes":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewaySingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.Name.ValueString() != "prod-tgw" {
		t.Errorf("Name = %q, want prod-tgw", got.Name.ValueString())
	}
	// The headline richer fields the plural DS omits:
	if got.AwsTransitGatewayID.ValueString() != "tgw-0abc123" {
		t.Errorf("AwsTransitGatewayID = %q, want tgw-0abc123", got.AwsTransitGatewayID.ValueString())
	}
	if got.ResourceShareID.ValueString() != "share-uuid" {
		t.Errorf("ResourceShareID = %q, want share-uuid", got.ResourceShareID.ValueString())
	}
	if got.ResourceShareAccount.ValueString() != "123456789012" {
		t.Errorf("ResourceShareAccount = %q, want 123456789012", got.ResourceShareAccount.ValueString())
	}
	if got.Region.ValueString() != "us-east-1" {
		t.Errorf("Region = %q, want us-east-1", got.Region.ValueString())
	}
	if got.Status.ValueString() != "available" {
		t.Errorf("Status = %q, want available", got.Status.ValueString())
	}
	if got.Attachment.ValueString() != "attached" {
		t.Errorf("Attachment = %q, want attached", got.Attachment.ValueString())
	}

	var routes []string
	if diags := got.Routes.ElementsAs(ctx, &routes, false); diags.HasError() {
		t.Fatalf("routes ElementsAs errors: %v", diags.Errors())
	}
	if len(routes) != 2 || routes[0] != "10.0.0.0/8" || routes[1] != "172.16.0.0/12" {
		t.Errorf("routes = %v, want [10.0.0.0/8 172.16.0.0/12]", routes)
	}
}

// A 404 from the by-id GET must surface as an error diagnostic (a singular data
// source referencing a non-existent id is a configuration error, not empty state).
func TestTransitGatewaySingleDataSource_Read_NotFound(t *testing.T) {
	tgwGetPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/missing"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwGetPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "transit gateway not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	ds := NewTransitGatewaySingleDataSource().(*TransitGatewaySingleDataSource)
	ds.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	configType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(configType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "missing"),
		"organization_id":        tftypes.NewValue(tftypes.String, "test-org-id"),
		"private_space_id":       tftypes.NewValue(tftypes.String, "test-ps-id"),
		"name":                   tftypes.NewValue(tftypes.String, nil),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, nil),
		"resource_share_id":      tftypes.NewValue(tftypes.String, nil),
		"resource_share_account": tftypes.NewValue(tftypes.String, nil),
		"region":                 tftypes.NewValue(tftypes.String, nil),
		"status":                 tftypes.NewValue(tftypes.String, nil),
		"attachment":             tftypes.NewValue(tftypes.String, nil),
		"routes":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read() on a missing transit gateway should report an error, got none")
	}
}
