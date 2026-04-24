package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &TeamDataSource{}
	_ datasource.DataSourceWithConfigure = &TeamDataSource{}
)

// TeamDataSource is the data source implementation.
type TeamDataSource struct {
	client *accessmanagement.TeamClient
}

// TeamDataSourceModel describes the data source data model.
type TeamDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ParentTeamID   types.String `tfsdk:"parent_team_id"`
	TeamType       types.String `tfsdk:"team_type"`
	OrganizationID types.String `tfsdk:"organization_id"`
	CreatedDate    types.String `tfsdk:"created_date"`
	UpdatedDate    types.String `tfsdk:"updated_date"`
	MemberCount    types.Int64  `tfsdk:"member_count"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

// Metadata returns the data source type name.
func (d *TeamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

// Schema defines the schema for the data source.
func (d *TeamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an Anypoint Platform team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the team.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the team.",
				Computed:    true,
			},
			"parent_team_id": schema.StringAttribute{
				Description: "The parent team ID.",
				Computed:    true,
			},
			"team_type": schema.StringAttribute{
				Description: "The type of the team.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the team is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"created_date": schema.StringAttribute{
				Description: "The creation date of the team.",
				Computed:    true,
			},
			"updated_date": schema.StringAttribute{
				Description: "The last update date of the team.",
				Computed:    true,
			},
			"member_count": schema.Int64Attribute{
				Description: "The number of members in the team.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the team was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the team was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TeamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	// Create the team client
	teamClient, err := accessmanagement.NewTeamClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anypoint Team API Client",
			"An unexpected error occurred when creating the Anypoint Team API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	d.client = teamClient
}

// Read refreshes the Terraform state with the latest data.
func (d *TeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamDataSourceModel

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

	// Get the team from the API
	team, err := d.client.GetTeam(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading team",
			"Could not read team ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(team.ID)
	data.Name = types.StringValue(team.TeamName)
	data.ParentTeamID = types.StringNull() // ParentTeamID not in the team struct
	data.TeamType = types.StringValue(team.TeamType)
	data.OrganizationID = types.StringValue(team.OrgID)
	data.CreatedDate = types.StringValue(team.CreatedAt)
	data.UpdatedDate = types.StringValue(team.UpdatedAt)
	data.MemberCount = types.Int64Value(0) // MemberCount not in the team struct

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
