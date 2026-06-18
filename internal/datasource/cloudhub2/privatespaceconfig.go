package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &PrivateSpaceConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpaceConfigDataSource{}
)

// NewPrivateSpaceConfigDataSource is a helper function to simplify the provider implementation.
func NewPrivateSpaceConfigDataSource() datasource.DataSource {
	return &PrivateSpaceConfigDataSource{}
}

// PrivateSpaceConfigDataSource is the data source implementation.
type PrivateSpaceConfigDataSource struct {
	client *cloudhub2.PrivateSpacesClient
}

// PrivateSpaceConfigDataSourceModel describes the data source data model.
type PrivateSpaceConfigDataSourceModel struct {
	ID                      types.String                           `tfsdk:"id"`
	OrganizationID          types.String                           `tfsdk:"organization_id"`
	Name                    types.String                           `tfsdk:"name"`
	Status                  types.String                           `tfsdk:"status"`
	RootOrganizationID      types.String                           `tfsdk:"root_organization_id"`
	MuleAppDeploymentCount  types.Int64                            `tfsdk:"mule_app_deployment_count"`
	DaysLeftForRelaxedQuota types.Int64                            `tfsdk:"days_left_for_relaxed_quota"`
	VPCMigrationInProgress  types.Bool                             `tfsdk:"vpc_migration_in_progress"`
	Network                 *NetworkConfigDataSourceModel          `tfsdk:"network"`
	FirewallRules           []FirewallRuleDataSourceModel          `tfsdk:"firewall_rules"`
}

// NetworkConfigDataSourceModel describes the network configuration data model.
type NetworkConfigDataSourceModel struct {
	Region                   types.String `tfsdk:"region"`
	CidrBlock                types.String `tfsdk:"cidr_block"`
	ReservedCIDRs            types.List   `tfsdk:"reserved_cidrs"`
	InboundStaticIPs         types.List   `tfsdk:"inbound_static_ips"`
	InboundInternalStaticIPs types.List   `tfsdk:"inbound_internal_static_ips"`
	OutboundStaticIPs        types.List   `tfsdk:"outbound_static_ips"`
	DNSTarget                types.String `tfsdk:"dns_target"`
}

// FirewallRuleDataSourceModel describes the firewall rule data model.
type FirewallRuleDataSourceModel struct {
	CidrBlock types.String `tfsdk:"cidr_block"`
	Protocol  types.String `tfsdk:"protocol"`
	FromPort  types.Int64  `tfsdk:"from_port"`
	ToPort    types.Int64  `tfsdk:"to_port"`
	Type      types.String `tfsdk:"type"`
}

// Metadata returns the data source type name.
func (d *PrivateSpaceConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_config"
}

// Schema defines the schema for the data source.
func (d *PrivateSpaceConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves configuration information for a CloudHub 2.0 private space, including network and firewall settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the private space to look up.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the private space.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the private space.",
				Computed:    true,
			},
			"root_organization_id": schema.StringAttribute{
				Description: "The root organization ID of the private space.",
				Computed:    true,
			},
			"mule_app_deployment_count": schema.Int64Attribute{
				Description: "The number of Mule apps deployed in the private space.",
				Computed:    true,
			},
			"days_left_for_relaxed_quota": schema.Int64Attribute{
				Description: "The number of days left for relaxed quota.",
				Computed:    true,
			},
			"vpc_migration_in_progress": schema.BoolAttribute{
				Description: "Whether a VPC migration is in progress.",
				Computed:    true,
			},
			"network": schema.SingleNestedAttribute{
				Description: "Network configuration for the private space.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"region": schema.StringAttribute{
						Description: "The AWS region for the private network.",
						Computed:    true,
					},
					"cidr_block": schema.StringAttribute{
						Description: "The CIDR block for the private network.",
						Computed:    true,
					},
					"reserved_cidrs": schema.ListAttribute{
						Description: "Reserved CIDR blocks for the private network.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"inbound_static_ips": schema.ListAttribute{
						Description: "Inbound static IPs assigned to the private network.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"inbound_internal_static_ips": schema.ListAttribute{
						Description: "Inbound internal static IPs assigned to the private network.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"outbound_static_ips": schema.ListAttribute{
						Description: "Outbound static IPs assigned to the private network.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"dns_target": schema.StringAttribute{
						Description: "The DNS target for the private network.",
						Computed:    true,
					},
				},
			},
			"firewall_rules": schema.ListNestedAttribute{
				Description: "Firewall rules configured for the private space.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr_block": schema.StringAttribute{
							Description: "The CIDR block for the firewall rule.",
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for the firewall rule (tcp, udp, icmp).",
							Computed:    true,
						},
						"from_port": schema.Int64Attribute{
							Description: "The starting port for the firewall rule.",
							Computed:    true,
						},
						"to_port": schema.Int64Attribute{
							Description: "The ending port for the firewall rule.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the firewall rule (inbound, outbound).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateSpaceConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	spaceClient, err := cloudhub2.NewPrivateSpacesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Spaces Client",
			"An unexpected error occurred when creating the Private Spaces client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = spaceClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateSpaceConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateSpaceConfigDataSourceModel

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

	// Get private space configuration from API
	tflog.Debug(ctx, "Reading private space config", map[string]interface{}{
		"org_id":   orgID,
		"space_id": data.ID.ValueString(),
	})

	space, err := d.client.GetPrivateSpace(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private space config",
			"Could not read private space config for "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to model
	data.Name = types.StringValue(space.Name)
	data.Status = types.StringValue(space.Status)
	data.RootOrganizationID = types.StringValue(space.RootOrganizationID)
	data.MuleAppDeploymentCount = types.Int64Value(int64(space.MuleAppDeploymentCount))
	data.DaysLeftForRelaxedQuota = types.Int64Value(int64(space.DaysLeftForRelaxedQuota))
	data.VPCMigrationInProgress = types.BoolValue(space.VPCMigrationInProgress)

	// Map network configuration if present
	if space.Network.Region != "" {
		networkModel := &NetworkConfigDataSourceModel{
			Region:    types.StringValue(space.Network.Region),
			CidrBlock: types.StringValue(space.Network.CidrBlock),
			DNSTarget: types.StringValue(space.Network.DNSTarget),
		}

		// Convert string slices to types.List
		var diags diag.Diagnostics

		networkModel.InboundStaticIPs, diags = types.ListValueFrom(ctx, types.StringType, space.Network.InboundStaticIPs)
		resp.Diagnostics.Append(diags...)

		networkModel.InboundInternalStaticIPs, diags = types.ListValueFrom(ctx, types.StringType, space.Network.InboundInternalStaticIPs)
		resp.Diagnostics.Append(diags...)

		networkModel.OutboundStaticIPs, diags = types.ListValueFrom(ctx, types.StringType, space.Network.OutboundStaticIPs)
		resp.Diagnostics.Append(diags...)

		networkModel.ReservedCIDRs, diags = types.ListValueFrom(ctx, types.StringType, space.Network.ReservedCIDRs)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() {
			return
		}

		data.Network = networkModel
	}

	// Map firewall rules if present
	if len(space.ManagedFirewallRules) > 0 {
		firewallRules := make([]FirewallRuleDataSourceModel, 0, len(space.ManagedFirewallRules))
		for _, rule := range space.ManagedFirewallRules {
			firewallRules = append(firewallRules, FirewallRuleDataSourceModel{
				CidrBlock: types.StringValue(rule.CidrBlock),
				Protocol:  types.StringValue(rule.Protocol),
				FromPort:  types.Int64Value(int64(rule.FromPort)),
				ToPort:    types.Int64Value(int64(rule.ToPort)),
				Type:      types.StringValue(rule.Type),
			})
		}
		data.FirewallRules = firewallRules
	}

	tflog.Trace(ctx, "Successfully read private space config", map[string]interface{}{
		"space_id": data.ID.ValueString(),
		"name":     data.Name.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
