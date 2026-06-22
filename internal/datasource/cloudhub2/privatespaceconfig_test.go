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

func TestNewPrivateSpaceConfigDataSource(t *testing.T) {
	dataSource := NewPrivateSpaceConfigDataSource()

	if dataSource == nil {
		t.Error("NewPrivateSpaceConfigDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestPrivateSpaceConfigDataSource_Metadata(t *testing.T) {
	dataSource := NewPrivateSpaceConfigDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_private_space_config" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_private_space_config")
	}
}

func TestPrivateSpaceConfigDataSource_Schema(t *testing.T) {
	dataSource := NewPrivateSpaceConfigDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Verify required attributes
	requiredAttrs := []string{"id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	// Verify computed attributes
	computedAttrs := []string{
		"name",
		"status",
		"root_organization_id",
		"mule_app_deployment_count",
		"days_left_for_relaxed_quota",
		"vpc_migration_in_progress",
		"network",
		"firewall_rules",
	}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}

	// Verify optional attributes
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

func TestPrivateSpaceConfigDataSource_Configure(t *testing.T) {
	dataSource := NewPrivateSpaceConfigDataSource().(*PrivateSpaceConfigDataSource)

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

func TestPrivateSpaceConfigDataSourceModel_Validation(t *testing.T) {
	model := PrivateSpaceConfigDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.Name
}

func TestPrivateSpaceConfigDataSource_Read(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id"

	mockResp := ch2client.PrivateSpace{
		ID:                      "test-ps-id",
		Name:                    "test-private-space",
		Status:                  "active",
		RootOrganizationID:      "root-org-id",
		MuleAppDeploymentCount:  5,
		DaysLeftForRelaxedQuota: 30,
		VPCMigrationInProgress:  false,
		Network: ch2client.NetworkConfig{
			Region:                   "us-east-1",
			CidrBlock:                "10.0.0.0/16",
			ReservedCIDRs:            []string{"10.0.1.0/24"},
			InboundStaticIPs:         []string{"1.2.3.4"},
			InboundInternalStaticIPs: []string{"10.0.1.5"},
			OutboundStaticIPs:        []string{"5.6.7.8"},
			DNSTarget:                "dns.example.com",
		},
		ManagedFirewallRules: []ch2client.FirewallRule{
			{
				CidrBlock: "0.0.0.0/0",
				Protocol:  "tcp",
				FromPort:  443,
				ToPort:    443,
				Type:      "inbound",
			},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockResp)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceConfigDataSource().(*PrivateSpaceConfigDataSource)
	ds.client = &ch2client.PrivateSpacesClient{
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
		"id":                         tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":            tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":                       tftypes.NewValue(tftypes.String, nil),
		"status":                     tftypes.NewValue(tftypes.String, nil),
		"root_organization_id":       tftypes.NewValue(tftypes.String, nil),
		"mule_app_deployment_count":  tftypes.NewValue(tftypes.Number, nil),
		"days_left_for_relaxed_quota": tftypes.NewValue(tftypes.Number, nil),
		"vpc_migration_in_progress":  tftypes.NewValue(tftypes.Bool, nil),
		"network": tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"region":                      tftypes.String,
				"cidr_block":                  tftypes.String,
				"reserved_cidrs":              tftypes.List{ElementType: tftypes.String},
				"inbound_static_ips":          tftypes.List{ElementType: tftypes.String},
				"inbound_internal_static_ips": tftypes.List{ElementType: tftypes.String},
				"outbound_static_ips":         tftypes.List{ElementType: tftypes.String},
				"dns_target":                  tftypes.String,
			},
		}, nil),
		"firewall_rules": tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"cidr_block": tftypes.String,
				"protocol":   tftypes.String,
				"from_port":  tftypes.Number,
				"to_port":    tftypes.Number,
				"type":       tftypes.String,
			},
		}}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got PrivateSpaceConfigDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	// Verify basic fields
	if got.Name.ValueString() != "test-private-space" {
		t.Errorf("Expected Name test-private-space, got %s", got.Name.ValueString())
	}
	if got.Status.ValueString() != "active" {
		t.Errorf("Expected Status active, got %s", got.Status.ValueString())
	}
	if got.MuleAppDeploymentCount.ValueInt64() != 5 {
		t.Errorf("Expected MuleAppDeploymentCount 5, got %d", got.MuleAppDeploymentCount.ValueInt64())
	}

	// Verify network configuration
	if got.Network == nil {
		t.Error("Expected Network to be populated")
	} else {
		if got.Network.Region.ValueString() != "us-east-1" {
			t.Errorf("Expected Network.Region us-east-1, got %s", got.Network.Region.ValueString())
		}
		if got.Network.CidrBlock.ValueString() != "10.0.0.0/16" {
			t.Errorf("Expected Network.CidrBlock 10.0.0.0/16, got %s", got.Network.CidrBlock.ValueString())
		}
	}

	// Verify firewall rules
	if len(got.FirewallRules) != 1 {
		t.Errorf("Expected 1 firewall rule, got %d", len(got.FirewallRules))
	} else {
		if got.FirewallRules[0].Protocol.ValueString() != "tcp" {
			t.Errorf("Expected FirewallRule Protocol tcp, got %s", got.FirewallRules[0].Protocol.ValueString())
		}
		if got.FirewallRules[0].FromPort.ValueInt64() != 443 {
			t.Errorf("Expected FirewallRule FromPort 443, got %d", got.FirewallRules[0].FromPort.ValueInt64())
		}
	}
}

func TestPrivateSpaceConfigDataSource_Read_Error(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceConfigDataSource().(*PrivateSpaceConfigDataSource)
	ds.client = &ch2client.PrivateSpacesClient{
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
		"id":                         tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":            tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":                       tftypes.NewValue(tftypes.String, nil),
		"status":                     tftypes.NewValue(tftypes.String, nil),
		"root_organization_id":       tftypes.NewValue(tftypes.String, nil),
		"mule_app_deployment_count":  tftypes.NewValue(tftypes.Number, nil),
		"days_left_for_relaxed_quota": tftypes.NewValue(tftypes.Number, nil),
		"vpc_migration_in_progress":  tftypes.NewValue(tftypes.Bool, nil),
		"network": tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"region":                      tftypes.String,
				"cidr_block":                  tftypes.String,
				"reserved_cidrs":              tftypes.List{ElementType: tftypes.String},
				"inbound_static_ips":          tftypes.List{ElementType: tftypes.String},
				"inbound_internal_static_ips": tftypes.List{ElementType: tftypes.String},
				"outbound_static_ips":         tftypes.List{ElementType: tftypes.String},
				"dns_target":                  tftypes.String,
			},
		}, nil),
		"firewall_rules": tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"cidr_block": tftypes.String,
				"protocol":   tftypes.String,
				"from_port":  tftypes.Number,
				"to_port":    tftypes.Number,
				"type":       tftypes.String,
			},
		}}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func BenchmarkPrivateSpaceConfigDataSource_Schema(b *testing.B) {
	dataSource := NewPrivateSpaceConfigDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
