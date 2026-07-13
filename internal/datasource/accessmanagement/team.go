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
	_ datasource.DataSource              = &TeamDataSource{}
	_ datasource.DataSourceWithConfigure = &TeamDataSource{}
)

// TeamDataSource is the data source implementation.
type TeamDataSource struct {
	client        *accessmanagement.TeamClient
	rolesClient   *accessmanagement.TeamRolesClient
	membersClient *accessmanagement.TeamMembersClient
	usersClient   *accessmanagement.RoleUsersClient
	catalogClient *accessmanagement.RolePermissionClient
}

// TeamDataSourceModel describes the data source data model.
type TeamDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ParentTeamID   types.String `tfsdk:"parent_team_id"`
	TeamType       types.String `tfsdk:"team_type"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Roles          types.List   `tfsdk:"roles"`
	Members        types.List   `tfsdk:"members"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// teamDSRoleObjectType is the object type for a role entry in the data source output.
var teamDSRoleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

// teamDSMemberObjectType is the object type for a member entry in the data source output.
var teamDSMemberObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"username":                        types.StringType,
		"membership_type":                 types.StringType,
		"is_assigned_via_external_groups": types.BoolType,
	},
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
		Description: "Fetches information about an Anypoint Platform team, including its role assignments and members.",
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
				Description: "The ID of the parent team. Null for root teams. Use this value as parent_team_id when creating child teams.",
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
			"roles": schema.ListNestedAttribute{
				Description: "The roles (permissions) assigned to this team. Excludes system/internal assignments.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The role's display name.",
							Computed:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the role (e.g., org, envId).",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"members": schema.ListNestedAttribute{
				Description: "The members of this team.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Description: "The member's username.",
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
					},
				},
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

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

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

	rolesClient, err := accessmanagement.NewTeamRolesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Roles Client",
			"An unexpected error occurred when creating the Team Roles client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	membersClient, err := accessmanagement.NewTeamMembersClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Members Client",
			"An unexpected error occurred when creating the Team Members client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	usersClient, err := accessmanagement.NewRoleUsersClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Users Client",
			"An unexpected error occurred when creating the Users client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	catalogClient, err := accessmanagement.NewRolePermissionClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Catalog Client",
			"An unexpected error occurred when creating the Role Catalog client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = teamClient
	d.rolesClient = rolesClient
	d.membersClient = membersClient
	d.usersClient = usersClient
	d.catalogClient = catalogClient
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
	data.TeamType = types.StringValue(team.TeamType)
	data.OrganizationID = types.StringValue(team.OrgID)
	data.CreatedAt = types.StringValue(team.CreatedAt)
	data.UpdatedAt = types.StringValue(team.UpdatedAt)
	// ancestor_team_ids is ordered root-first / direct-parent-LAST, so the direct
	// parent is the last element (see Team.DirectParentID).
	if parent := team.DirectParentID(); parent != "" {
		data.ParentTeamID = types.StringValue(parent)
	} else {
		data.ParentTeamID = types.StringNull()
	}

	// Populate roles (excluding internal/system assignments), labeled by display name.
	assignments, err := d.rolesClient.ListTeamRoles(ctx, orgID, team.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team roles", "Could not list team role assignments: "+err.Error())
		return
	}
	roles, err := d.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading available roles", "Could not list available roles: "+err.Error())
		return
	}
	roleIDToName := make(map[string]string, len(roles))
	for _, role := range roles {
		roleIDToName[role.RoleID] = role.Name
	}
	roleValues := make([]attr.Value, 0, len(assignments))
	for _, a := range assignments {
		if a.Internal {
			continue
		}
		// Skip platform-injected side-effect grants whose role_id is not in the
		// assignable catalog (e.g. the org-scoped "Business Group Viewer" the
		// platform auto-adds alongside an env-scoped role). These have no catalog
		// name and are not something the user assigned, so — mirroring the
		// anypoint_team resource's reconcile logic — the data source must not
		// surface them (otherwise they appear with an empty name).
		if _, inCatalog := roleIDToName[a.RoleID]; !inCatalog {
			continue
		}
		cp := types.MapNull(types.StringType)
		if len(a.ContextParams) > 0 {
			cpElems := make(map[string]attr.Value, len(a.ContextParams))
			for k, v := range a.ContextParams {
				cpElems[k] = types.StringValue(v)
			}
			cp = types.MapValueMust(types.StringType, cpElems)
		}
		obj, diags := types.ObjectValue(teamDSRoleObjectType.AttrTypes, map[string]attr.Value{
			"name":           types.StringValue(roleIDToName[a.RoleID]),
			"context_params": cp,
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		roleValues = append(roleValues, obj)
	}
	data.Roles = types.ListValueMust(teamDSRoleObjectType, roleValues)

	// Populate members. Map user IDs to usernames via the org user list.
	members, err := d.membersClient.ListTeamMembers(ctx, orgID, team.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team members", "Could not list team members: "+err.Error())
		return
	}
	orgUsers, err := d.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization users", "Could not list organization users: "+err.Error())
		return
	}
	userIDToUsername := make(map[string]string, len(orgUsers))
	for _, u := range orgUsers {
		userIDToUsername[u.ID] = u.Username
	}
	memberValues := make([]attr.Value, 0, len(members))
	for _, m := range members {
		// Skip members whose ID does not resolve to an org username. Once a team
		// gains a child team (or other hierarchy side-effects), the platform injects
		// a non-user "group identity" member; the members endpoint returns it, but it
		// has no username, isn't something the user added, and is invisible in the UI.
		// Mirroring the anypoint_team resource's reconcile (which excludes the same
		// identity), the data source must skip it — otherwise it surfaces with an
		// empty username and produces a perpetual output diff. Note this is distinct
		// from external-group members, which ARE org users and are reported as-is.
		username, ok := userIDToUsername[m.ID]
		if !ok {
			continue
		}
		obj, diags := types.ObjectValue(teamDSMemberObjectType.AttrTypes, map[string]attr.Value{
			"username":                        types.StringValue(username),
			"membership_type":                 types.StringValue(m.MembershipType),
			"is_assigned_via_external_groups": types.BoolValue(m.IsAssignedViaExternalGroups),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		memberValues = append(memberValues, obj)
	}
	data.Members = types.ListValueMust(teamDSMemberObjectType, memberValues)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
