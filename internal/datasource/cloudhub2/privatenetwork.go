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
	_ datasource.DataSource              = &PrivateNetworkDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateNetworkDataSource{}
)

// PrivateNetworkDataSource is the data source implementation.
type PrivateNetworkDataSource struct {
	client *cloudhub2.PrivateNetworkClient
}

// PrivateNetworkDataSourceModel describes the data source data model.
type PrivateNetworkDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Region         types.String `tfsdk:"region"`
	CIDR           types.String `tfsdk:"cidr"`
	IsFirewalled   types.Bool   `tfsdk:"is_firewalled"`
	DNSServers     types.List   `tfsdk:"dns_servers"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewPrivateNetworkDataSource() datasource.DataSource {
	return &PrivateNetworkDataSource{}
}

// Metadata returns the data source type name.
func (d *PrivateNetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

// Schema defines the schema for the data source.
func (d *PrivateNetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a CloudHub 2.0 private network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private network.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the private network.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region where the private network is deployed.",
				Computed:    true,
			},
			"cidr": schema.StringAttribute{
				Description: "The CIDR block for the private network.",
				Computed:    true,
			},
			"is_firewalled": schema.BoolAttribute{
				Description: "Whether the private network is firewalled.",
				Computed:    true,
			},
			"dns_servers": schema.ListAttribute{
				Description: "List of DNS server IP addresses.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the private network.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the private network.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the private network.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateNetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Extract the client configuration
	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Create the private network client
	privateNetworkClient, err := cloudhub2.NewPrivateNetworkClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CloudHub 2.0 Private Network API Client",
			"An unexpected error occurred when creating the CloudHub 2.0 Private Network API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"CloudHub 2.0 Client Error: "+err.Error(),
		)
		return
	}

	d.client = privateNetworkClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateNetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateNetworkDataSourceModel

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

	// Get the private network from the API
	privateNetwork, err := d.client.GetPrivateNetwork(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private network",
			"Could not read private network ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(privateNetwork.ID)
	data.Name = types.StringValue(privateNetwork.Name)
	data.Region = types.StringNull()                                           // Not available in PrivateSpace
	data.CIDR = types.StringNull()                                             // Not available in PrivateSpace
	data.IsFirewalled = types.BoolValue(len(privateNetwork.FirewallRules) > 0) // Derived from firewall rules
	data.Status = types.StringValue(privateNetwork.Status)
	data.CreatedAt = types.StringNull() // Not available in PrivateSpace
	data.UpdatedAt = types.StringNull() // Not available in PrivateSpace

	// DNS servers not available in PrivateSpace
	data.DNSServers = types.ListNull(types.StringType)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
