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
	_ datasource.DataSource              = &FirewallRulesDataSource{}
	_ datasource.DataSourceWithConfigure = &FirewallRulesDataSource{}
)

// FirewallRulesDataSource is the data source implementation.
type FirewallRulesDataSource struct {
	client *cloudhub2.FirewallRulesClient
}

// FirewallRulesDataSourceModel describes the data source data model.
type FirewallRulesDataSourceModel struct {
	ID             types.String                  `tfsdk:"id"`
	PrivateSpaceID types.String                  `tfsdk:"private_space_id"`
	OrganizationID types.String                  `tfsdk:"organization_id"`
	Rules          []FirewallRuleDataSourceModel `tfsdk:"rules"`
}

// FirewallRuleDataSourceModel describes individual firewall rule data model for data source.
type FirewallRuleDataSourceModel struct {
	CidrBlock types.String `tfsdk:"cidr_block"`
	Protocol  types.String `tfsdk:"protocol"`
	FromPort  types.Int64  `tfsdk:"from_port"`
	ToPort    types.Int64  `tfsdk:"to_port"`
	Type      types.String `tfsdk:"type"`
}

func NewFirewallRulesDataSource() datasource.DataSource {
	return &FirewallRulesDataSource{}
}

// Metadata returns the data source type name.
func (d *FirewallRulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewallrules"
}

// Schema defines the schema for the data source.
func (d *FirewallRulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves firewall rules for an Anypoint Private Space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the firewall rules (same as private_space_id).",
				Computed:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space to retrieve firewall rules for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description: "List of managed firewall rules.",
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
func (d *FirewallRulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	firewallRulesClient, err := cloudhub2.NewFirewallRulesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Firewall Rules Client",
			"An unexpected error occurred when creating the Firewall Rules client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = firewallRulesClient
}

// mapFirewallRulesFromAPIForDataSource converts API firewall rules to data source models
func mapFirewallRulesFromAPIForDataSource(apiRules []cloudhub2.FirewallRule) []FirewallRuleDataSourceModel {
	rules := make([]FirewallRuleDataSourceModel, len(apiRules))
	for i, apiRule := range apiRules {
		rules[i] = FirewallRuleDataSourceModel{
			CidrBlock: types.StringValue(apiRule.CidrBlock),
			Protocol:  types.StringValue(apiRule.Protocol),
			FromPort:  types.Int64Value(int64(apiRule.FromPort)),
			ToPort:    types.Int64Value(int64(apiRule.ToPort)),
			Type:      types.StringValue(apiRule.Type),
		}
	}
	return rules
}

// Read refreshes the Terraform state with the latest data.
func (d *FirewallRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FirewallRulesDataSourceModel

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

	// Get the firewall rules from the API
	privateSpace, err := d.client.GetFirewallRules(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading firewall rules",
			"Could not read firewall rules for private space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to data source model (only managedFirewallRules)
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString())
	data.Rules = mapFirewallRulesFromAPIForDataSource(privateSpace.ManagedFirewallRules)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
