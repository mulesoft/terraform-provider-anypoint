package cloudhub2

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &VPNConnectionDataSource{}
	_ datasource.DataSourceWithConfigure = &VPNConnectionDataSource{}
)

// VPNConnectionDataSource is the data source implementation.
type VPNConnectionDataSource struct {
	client *cloudhub2.VPNConnectionClient
}

// VPNConnectionDataSourceModel describes the data source data model.
type VPNConnectionDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	ConnectionID   types.String `tfsdk:"connection_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	VPNs           types.List   `tfsdk:"vpns"`
}

// VPNModel describes a VPN configuration.
type VPNModel struct {
	LocalASN            types.String `tfsdk:"local_asn"`
	RemoteASN           types.String `tfsdk:"remote_asn"`
	RemoteIPAddress     types.String `tfsdk:"remote_ip_address"`
	StaticRoutes        types.List   `tfsdk:"static_routes"`
	VPNTunnels          types.List   `tfsdk:"vpn_tunnels"`
	Name                types.String `tfsdk:"name"`
	ConnectionName      types.String `tfsdk:"connection_name"`
	VPNConnectionStatus types.String `tfsdk:"vpn_connection_status"`
	VPNID               types.String `tfsdk:"vpn_id"`
	ConnectionID        types.String `tfsdk:"connection_id"`
}

// VPNTunnelModel describes a VPN tunnel configuration.
type VPNTunnelModel struct {
	PSK           types.String `tfsdk:"psk"`
	PTPCidr       types.String `tfsdk:"ptp_cidr"`
	StartupAction types.String `tfsdk:"startup_action"`
	IsLogsEnabled types.Bool   `tfsdk:"is_logs_enabled"`
}

func NewVPNConnectionDataSource() datasource.DataSource {
	return &VPNConnectionDataSource{}
}

// Metadata returns the data source type name.
func (d *VPNConnectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_connection"
}

// Schema defines the schema for the data source.
func (d *VPNConnectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a CloudHub 2.0 VPN connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the VPN connection.",
				Computed:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The private space ID where the VPN connection is located.",
				Required:    true,
			},
			"connection_id": schema.StringAttribute{
				Description: "The ID of the VPN connection to look up.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the VPN connection.",
				Computed:    true,
			},
			"vpns": schema.ListNestedAttribute{
				Description: "List of VPN configurations within this connection.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"local_asn": schema.StringAttribute{
							Description: "Local Autonomous System Number.",
							Computed:    true,
						},
						"remote_asn": schema.StringAttribute{
							Description: "Remote Autonomous System Number.",
							Computed:    true,
						},
						"remote_ip_address": schema.StringAttribute{
							Description: "Remote IP address for the VPN.",
							Computed:    true,
						},
						"static_routes": schema.ListAttribute{
							Description: "List of static routes.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"vpn_tunnels": schema.ListNestedAttribute{
							Description: "List of VPN tunnels.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"psk": schema.StringAttribute{
										Description: "Pre-shared key for the VPN tunnel.",
										Computed:    true,
										Sensitive:   true,
									},
									"ptp_cidr": schema.StringAttribute{
										Description: "Point-to-point CIDR block.",
										Computed:    true,
									},
									"startup_action": schema.StringAttribute{
										Description: "Startup action for the tunnel.",
										Computed:    true,
									},
									"is_logs_enabled": schema.BoolAttribute{
										Description: "Whether logging is enabled for this tunnel.",
										Computed:    true,
									},
								},
							},
						},
						"name": schema.StringAttribute{
							Description: "The name of the VPN.",
							Computed:    true,
						},
						"connection_name": schema.StringAttribute{
							Description: "The connection name.",
							Computed:    true,
						},
						"vpn_connection_status": schema.StringAttribute{
							Description: "The status of the VPN connection.",
							Computed:    true,
						},
						"vpn_id": schema.StringAttribute{
							Description: "The VPN ID.",
							Computed:    true,
						},
						"connection_id": schema.StringAttribute{
							Description: "The connection ID.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *VPNConnectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	// Create the VPN connection client
	vpnConnectionClient, err := cloudhub2.NewVPNConnectionClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CloudHub 2.0 VPN Connection API Client",
			"An unexpected error occurred when creating the CloudHub 2.0 VPN Connection API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"CloudHub 2.0 Client Error: "+err.Error(),
		)
		return
	}

	d.client = vpnConnectionClient
}

// Read refreshes the Terraform state with the latest data.
func (d *VPNConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VPNConnectionDataSourceModel

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

	// Get the VPN connection from the API
	vpnConnection, err := d.client.GetVPNConnection(ctx, orgID, data.PrivateSpaceID.ValueString(), data.ConnectionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading VPN connection",
			"Could not read VPN connection ID "+data.ConnectionID.ValueString()+" in private space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(vpnConnection.ID)
	data.Name = types.StringValue(vpnConnection.Name)
	data.OrganizationID = types.StringValue(orgID)

	// Define the VPN tunnel object type
	vpnTunnelAttrTypes := map[string]attr.Type{
		"psk":             types.StringType,
		"ptp_cidr":        types.StringType,
		"startup_action":  types.StringType,
		"is_logs_enabled": types.BoolType,
	}

	// Define the VPN object type
	vpnAttrTypes := map[string]attr.Type{
		"local_asn":              types.StringType,
		"remote_asn":             types.StringType,
		"remote_ip_address":      types.StringType,
		"static_routes":          types.ListType{ElemType: types.StringType},
		"vpn_tunnels":            types.ListType{ElemType: types.ObjectType{AttrTypes: vpnTunnelAttrTypes}},
		"name":                   types.StringType,
		"connection_name":        types.StringType,
		"vpn_connection_status":  types.StringType,
		"vpn_id":                 types.StringType,
		"connection_id":          types.StringType,
	}

	// Map VPNs
	var vpnObjects []attr.Value
	for _, vpn := range vpnConnection.VPNs {
		// Map static routes
		var staticRouteElements []attr.Value
		for _, route := range vpn.StaticRoutes {
			staticRouteElements = append(staticRouteElements, types.StringValue(route))
		}
		staticRoutesList, diag := types.ListValue(types.StringType, staticRouteElements)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Map VPN tunnels
		var tunnelObjects []attr.Value
		for _, tunnel := range vpn.VPNTunnels {
			tunnelObj, diag := types.ObjectValue(
				vpnTunnelAttrTypes,
				map[string]attr.Value{
					"psk":             types.StringValue(tunnel.PSK),
					"ptp_cidr":        types.StringValue(tunnel.PTPCidr),
					"startup_action":  types.StringValue(tunnel.StartupAction),
					"is_logs_enabled": types.BoolValue(tunnel.IsLogsEnabled),
				},
			)
			resp.Diagnostics.Append(diag...)
			if resp.Diagnostics.HasError() {
				return
			}
			tunnelObjects = append(tunnelObjects, tunnelObj)
		}
		tunnelsList, diag := types.ListValue(types.ObjectType{AttrTypes: vpnTunnelAttrTypes}, tunnelObjects)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Create VPN object
		vpnObj, diag := types.ObjectValue(
			vpnAttrTypes,
			map[string]attr.Value{
				"local_asn":             types.StringValue(strconv.Itoa(vpn.LocalASN)),
				"remote_asn":            types.StringValue(strconv.Itoa(vpn.RemoteASN)),
				"remote_ip_address":     types.StringValue(vpn.RemoteIPAddress),
				"static_routes":         staticRoutesList,
				"vpn_tunnels":           tunnelsList,
				"name":                  types.StringValue(vpn.Name),
				"connection_name":       types.StringValue(vpn.ConnectionName),
				"vpn_connection_status": types.StringValue(vpn.VPNConnectionStatus),
				"vpn_id":                types.StringValue(vpn.VPNID),
				"connection_id":         types.StringValue(vpn.ConnectionID),
			},
		)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		vpnObjects = append(vpnObjects, vpnObj)
	}

	vpnsList, diag := types.ListValue(types.ObjectType{AttrTypes: vpnAttrTypes}, vpnObjects)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.VPNs = vpnsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
