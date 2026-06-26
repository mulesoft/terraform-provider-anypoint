package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &TeamMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &TeamMembersDataSource{}
)

// TeamMembersDataSource is the data source implementation.
type TeamMembersDataSource struct {
	client *accessmanagement.TeamMembersClient
}

// TeamMembersDataSourceModel describes the data source data model.
type TeamMembersDataSourceModel struct {
	TeamID         types.String `tfsdk:"team_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Members        types.List   `tfsdk:"members"`
	Total          types.Int64  `tfsdk:"total"`
}

func NewTeamMembersDataSource() datasource.DataSource {
	return &TeamMembersDataSource{}
}

// Metadata returns the data source type name.
func (d *TeamMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

// Schema defines the schema for the data source.
func (d *TeamMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all members of a specific team.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Description: "The ID of the team to list members for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"members": schema.ListNestedAttribute{
				Description: "List of members in the team.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The user ID.",
							Computed:    true,
						},
						"membership_type": schema.StringAttribute{
							Description: "The membership type (member or maintainer).",
							Computed:    true,
						},
						"is_assigned_via_external_groups": schema.BoolAttribute{
							Description: "Whether the membership was assigned via external groups (e.g., SAML/SCIM).",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "When the membership was created.",
							Computed:    true,
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				Description: "Total number of members in the team.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TeamMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
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

	membersClient, err := accessmanagement.NewTeamMembersClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Members Client",
			"An unexpected error occurred when creating the Team Members client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = membersClient
}

// Read refreshes the Terraform state with the latest data.
func (d *TeamMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamMembersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// List all members for the team
	members, err := d.client.ListTeamMembers(ctx, orgID, data.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading team members",
			"Could not list members for team "+data.TeamID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Build the list of member objects
	memberAttrTypes := map[string]attr.Type{
		"id":                              types.StringType,
		"membership_type":                 types.StringType,
		"is_assigned_via_external_groups": types.BoolType,
		"created_at":                      types.StringType,
	}
	memberObjType := types.ObjectType{AttrTypes: memberAttrTypes}

	memberValues := make([]attr.Value, len(members))
	for i, m := range members {
		obj, diags := types.ObjectValue(memberAttrTypes, map[string]attr.Value{
			"id":                              types.StringValue(m.ID),
			"membership_type":                 types.StringValue(m.MembershipType),
			"is_assigned_via_external_groups": types.BoolValue(m.IsAssignedViaExternalGroups),
			"created_at":                      types.StringValue(m.CreatedAt),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		memberValues[i] = obj
	}

	membersList, diags := types.ListValue(memberObjType, memberValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Members = membersList
	data.Total = types.Int64Value(int64(len(members)))
	data.OrganizationID = types.StringValue(orgID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
