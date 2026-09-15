package cloudhub2

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTransitGatewayDataSource(t *testing.T) {
	ds := NewTransitGatewayDataSource()

	if ds == nil {
		t.Error("NewTransitGatewayDataSource() returned nil")
	}

	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("data source should implement DataSourceWithConfigure")
	}
}

func TestTransitGatewayDataSource_Metadata(t *testing.T) {
	ds := NewTransitGatewayDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)

	if resp.TypeName != "anypoint_transit_gateway_connections" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "anypoint_transit_gateway_connections")
	}
}

func TestTransitGatewayDataSource_Schema(t *testing.T) {
	ds := NewTransitGatewayDataSource()

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", schemaResp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"private_space_id", "organization_id"}
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

	computedAttrs := []string{"transit_gateway_connections"}
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

func TestTransitGatewayDataSource_Configure(t *testing.T) {
	ds := NewTransitGatewayDataSource().(*TransitGatewayDataSource)

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

func TestTransitGatewayDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewTransitGatewayDataSource().(*TransitGatewayDataSource)

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

func TestTransitGatewayDataSource_Read(t *testing.T) {
	tgwListPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwListPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{
					"id":   "tgw-1",
					"name": "prod-tgw",
					"spec": map[string]interface{}{
						"resourceShare": map[string]interface{}{
							"id":      "share-uuid",
							"account": "123456789012",
						},
					},
					"status": map[string]interface{}{
						"gateway":     "available",
						"attachment":  "available",
						"tgwResource": "tgw-0abc",
						"routes":      []string{"10.0.0.0/8", "172.16.0.0/12"},
					},
				},
				{
					"id":   "tgw-2",
					"name": "staging-tgw",
					"spec": map[string]interface{}{
						"resourceShare": map[string]interface{}{
							"id":      "share-uuid-2",
							"account": "987654321098",
						},
					},
					"status": map[string]interface{}{
						"gateway":     "pending",
						"attachment":  "pending",
						"tgwResource": "",
						"routes":      []string{},
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
	ds := NewTransitGatewayDataSource().(*TransitGatewayDataSource)
	ds.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	configType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build the nested type for transit_gateway_connections attribute. Routes are a
	// plain string list here, matching the resource and the singular data source.
	tgwObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":     tftypes.String,
		"name":   tftypes.String,
		"status": tftypes.String,
		"routes": tftypes.List{ElementType: tftypes.String},
	}}

	configRaw := tftypes.NewValue(configType, map[string]tftypes.Value{
		"private_space_id":            tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":             tftypes.NewValue(tftypes.String, "test-org-id"),
		"transit_gateway_connections": tftypes.NewValue(tftypes.List{ElementType: tgwObjType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if len(got.TransitGatewayConnections) != 2 {
		t.Fatalf("Expected 2 transit gateway connections, got %d", len(got.TransitGatewayConnections))
	}
	if got.TransitGatewayConnections[0].Name.ValueString() != "prod-tgw" {
		t.Errorf("Expected first TGW name 'prod-tgw', got '%s'", got.TransitGatewayConnections[0].Name.ValueString())
	}
	if got.TransitGatewayConnections[0].Status.ValueString() != "available" {
		t.Errorf("Expected first TGW status 'available', got '%s'", got.TransitGatewayConnections[0].Status.ValueString())
	}
	// Routes must surface as plain CIDR strings so a value read from this data
	// source can be passed straight into anypoint_transit_gateway_connection's
	// `routes` argument. Asserting the values (not just the count) pins the shape:
	// wrapping each CIDR in an object again would fail to decode into []string.
	routesOf := func(i int) []string {
		t.Helper()
		var routes []string
		if diags := got.TransitGatewayConnections[i].Routes.ElementsAs(ctx, &routes, false); diags.HasError() {
			t.Fatalf("routes[%d] did not decode as a list of strings: %v", i, diags.Errors())
		}
		return routes
	}

	if want := []string{"10.0.0.0/8", "172.16.0.0/12"}; !reflect.DeepEqual(routesOf(0), want) {
		t.Errorf("Expected routes %v for first TGW, got %v", want, routesOf(0))
	}
	// An empty routes array must come back as an empty list, not null, so callers
	// can iterate it without a null check.
	if got := routesOf(1); len(got) != 0 {
		t.Errorf("Expected no routes for second TGW, got %v", got)
	}
	if got.TransitGatewayConnections[1].Routes.IsNull() {
		t.Error("Expected empty routes to be an empty list, got null")
	}
}
