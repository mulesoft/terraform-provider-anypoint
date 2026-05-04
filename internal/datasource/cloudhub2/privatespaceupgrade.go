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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &PrivateSpaceUpgradeDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpaceUpgradeDataSource{}
)

// NewPrivateSpaceUpgradeDataSource is a helper function to simplify the provider implementation.
func NewPrivateSpaceUpgradeDataSource() datasource.DataSource {
	return &PrivateSpaceUpgradeDataSource{}
}

// PrivateSpaceUpgradeDataSource is the data source implementation.
type PrivateSpaceUpgradeDataSource struct {
	client *cloudhub2.PrivateSpaceUpgradeClient
}

// PrivateSpaceUpgradeDataSourceModel describes the data source data model.
type PrivateSpaceUpgradeDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	PrivateSpaceID      types.String `tfsdk:"private_space_id"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	ScheduledUpdateTime types.String `tfsdk:"scheduled_update_time"`
	Status              types.String `tfsdk:"status"`
}

// Metadata returns the data source type name.
func (d *PrivateSpaceUpgradeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_upgrade"
}

// Schema defines the schema for the data source.
func (d *PrivateSpaceUpgradeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves upgrade status information for a CloudHub 2.0 private space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space to get upgrade status for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
			},
			"scheduled_update_time": schema.StringAttribute{
				Description: "The scheduled update time for the upgrade.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the upgrade (e.g., QUEUED, IN_PROGRESS, COMPLETED).",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateSpaceUpgradeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	upgradeClient, err := cloudhub2.NewPrivateSpaceUpgradeClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Upgrade Client",
			"An unexpected error occurred when creating the Private Space Upgrade client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = upgradeClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateSpaceUpgradeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateSpaceUpgradeDataSourceModel

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

	// Get upgrade status from API
	upgradeStatus, err := d.client.GetPrivateSpaceUpgradeStatus(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private space upgrade status",
			"Could not read private space upgrade status for "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString()) // Use private space ID as the data source ID
	data.ScheduledUpdateTime = types.StringValue(upgradeStatus.ScheduledUpdateTime)
	data.Status = types.StringValue(upgradeStatus.Status)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
