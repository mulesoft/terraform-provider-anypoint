package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &PrivateSpaceConnectionDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpaceConnectionDataSource{}
)

// PrivateSpaceConnectionDataSource is the data source implementation.
type PrivateSpaceConnectionDataSource struct {
	client *cloudhub2.PrivateSpaceConnectionClient
}

// PrivateSpaceConnectionDataSourceModel describes the data source data model.
type PrivateSpaceConnectionDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	PrivateSpaceID    types.String `tfsdk:"private_space_id"`
	OrganizationID    types.String `tfsdk:"organization_id"`
	ConnectionType    types.String `tfsdk:"connection_type"`
	Status            types.String `tfsdk:"status"`
	CustomerGatewayIP types.String `tfsdk:"customer_gateway_ip"`
	CustomerTunnelIPs types.List   `tfsdk:"customer_tunnel_ips"`
	AnypointTunnelIPs types.List   `tfsdk:"anypoint_tunnel_ips"`
	BGPAsn            types.Int64  `tfsdk:"bgp_asn"`
	AnypointBGPAsn    types.Int64  `tfsdk:"anypoint_bgp_asn"`
	CustomerNetworks  types.List   `tfsdk:"customer_networks"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func NewPrivateSpaceConnectionDataSource() datasource.DataSource {
	return &PrivateSpaceConnectionDataSource{}
}

// Metadata returns the data source type name.
func (d *PrivateSpaceConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_connection"
}

// Schema defines the schema for the data source.
func (d *PrivateSpaceConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a CloudHub 2.0 private space connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private space connection.",
				Required:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The private space ID where the connection is located.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the private space connection.",
				Computed:    true,
			},
			"connection_type": schema.StringAttribute{
				Description: "The type of connection (e.g., VPN, Direct Connect).",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the connection.",
				Computed:    true,
			},
			"customer_gateway_ip": schema.StringAttribute{
				Description: "The customer gateway IP address.",
				Computed:    true,
			},
			"customer_tunnel_ips": schema.ListAttribute{
				Description: "List of customer tunnel IP addresses.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"anypoint_tunnel_ips": schema.ListAttribute{
				Description: "List of Anypoint tunnel IP addresses.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"bgp_asn": schema.Int64Attribute{
				Description: "The BGP ASN number for the customer.",
				Computed:    true,
			},
			"anypoint_bgp_asn": schema.Int64Attribute{
				Description: "The BGP ASN number for Anypoint.",
				Computed:    true,
			},
			"customer_networks": schema.ListAttribute{
				Description: "List of customer network CIDR blocks.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the connection.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the connection.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateSpaceConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Extract the client configuration
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Create the private space connection client
	connectionClient, err := cloudhub2.NewPrivateSpaceConnectionClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CloudHub 2.0 Private Space Connection API Client",
			"An unexpected error occurred when creating the CloudHub 2.0 Private Space Connection API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"CloudHub 2.0 Client Error: "+err.Error(),
		)
		return
	}

	d.client = connectionClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateSpaceConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateSpaceConnectionDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// Get the private space connection from the API
	connection, err := d.client.GetPrivateSpaceConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private space connection",
			"Could not read private space connection ID "+data.ID.ValueString()+" in private space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(connection.ID)
	data.Name = types.StringValue(connection.Name)
	data.PrivateSpaceID = types.StringValue(connection.PrivateSpaceID)
	data.ConnectionType = types.StringValue(connection.Type) // Use Type field
	data.Status = types.StringValue(connection.Status)
	data.CreatedAt = types.StringValue(connection.CreatedAt)
	data.UpdatedAt = types.StringValue(connection.UpdatedAt)

	// These fields are not available in the actual struct, set to defaults
	data.CustomerGatewayIP = types.StringNull()
	data.BGPAsn = types.Int64Value(0)
	data.AnypointBGPAsn = types.Int64Value(0)
	data.CustomerTunnelIPs = types.ListNull(types.StringType)
	data.AnypointTunnelIPs = types.ListNull(types.StringType)
	data.CustomerNetworks = types.ListNull(types.StringType)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
