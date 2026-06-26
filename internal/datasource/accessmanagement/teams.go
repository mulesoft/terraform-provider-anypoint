package accessmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &TeamsDataSource{}
	_ datasource.DataSourceWithConfigure = &TeamsDataSource{}
)

// TeamsDataSource is the data source implementation.
type TeamsDataSource struct {
	client *accessmanagement.TeamClient
}

// TeamsDataSourceModel describes the data source data model.
type TeamsDataSourceModel struct {
	OrganizationID types.String     `tfsdk:"organization_id"`
	NameFilter     types.String     `tfsdk:"name_filter"`
	Teams          []TeamItemModel  `tfsdk:"teams"`
}

// TeamItemModel describes a single team in the list.
type TeamItemModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	TeamType        types.String `tfsdk:"team_type"`
	AncestorTeamIDs types.List   `tfsdk:"ancestor_team_ids"`
	IsRootTeam      types.Bool   `tfsdk:"is_root_team"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func NewTeamsDataSource() datasource.DataSource {
	return &TeamsDataSource{}
}

// Metadata returns the data source type name.
func (d *TeamsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teams"
}

// Schema defines the schema for the data source.
func (d *TeamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all teams in the organization. Use this to find the root team " +
			"(for use as parent_team_id when creating new teams) or to look up a team by name.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider's org.",
				Optional:    true,
				Computed:    true,
			},
			"name_filter": schema.StringAttribute{
				Description: "Optional filter to match teams by name (case-insensitive substring match). " +
					"If not provided, all teams are returned.",
				Optional: true,
			},
			"teams": schema.ListNestedAttribute{
				Description: "List of teams matching the filter.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique team ID. Use this as team_id or parent_team_id in other resources.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The team name.",
							Computed:    true,
						},
						"team_type": schema.StringAttribute{
							Description: "The type of team (e.g. 'internal').",
							Computed:    true,
						},
						"ancestor_team_ids": schema.ListAttribute{
							Description: "List of ancestor team IDs (empty for the root team).",
							Computed:    true,
							ElementType: types.StringType,
						},
						"is_root_team": schema.BoolAttribute{
							Description: "True if this is the organization's root team (has no ancestors). " +
								"Use this team's ID as parent_team_id when creating top-level teams.",
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Description: "When the team was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "When the team was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TeamsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	userConfig := &client.UserClientConfig{
		BaseURL:      config.BaseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
		Timeout:      config.Timeout,
	}

	teamClient, err := accessmanagement.NewTeamClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Teams Client",
			"An unexpected error occurred when creating the client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = teamClient
}

// Read refreshes the Terraform state with the latest data.
func (d *TeamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	teams, err := d.client.ListTeams(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing teams",
			"Could not list teams: "+err.Error(),
		)
		return
	}

	// Apply name filter if provided
	nameFilter := strings.ToLower(data.NameFilter.ValueString())

	var teamItems []TeamItemModel
	for _, t := range teams {
		if nameFilter != "" && !strings.Contains(strings.ToLower(t.TeamName), nameFilter) {
			continue
		}

		// Build ancestor_team_ids list
		ancestorList, diags := types.ListValueFrom(ctx, types.StringType, t.AncestorTeamIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		teamItems = append(teamItems, TeamItemModel{
			ID:              types.StringValue(t.ID),
			Name:            types.StringValue(t.TeamName),
			TeamType:        types.StringValue(t.TeamType),
			AncestorTeamIDs: ancestorList,
			IsRootTeam:      types.BoolValue(len(t.AncestorTeamIDs) == 0),
			CreatedAt:       types.StringValue(t.CreatedAt),
			UpdatedAt:       types.StringValue(t.UpdatedAt),
		})
	}

	data.OrganizationID = types.StringValue(orgID)
	data.Teams = teamItems

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
