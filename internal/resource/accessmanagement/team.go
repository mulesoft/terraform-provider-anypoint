package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &TeamResource{}
	_ resource.ResourceWithConfigure   = &TeamResource{}
	_ resource.ResourceWithImportState = &TeamResource{}
)

// TeamResource is the resource implementation.
type TeamResource struct {
	client        *accessmanagement.TeamClient
	rolesClient   *accessmanagement.TeamRolesClient
	membersClient *accessmanagement.TeamMembersClient
	usersClient   *accessmanagement.RoleUsersClient      // org user lookup (ListOrgUsers)
	catalogClient *accessmanagement.RolePermissionClient // available-roles catalog (ListAvailableRoles)
}

// TeamResourceModel describes the resource data model.
type TeamResourceModel struct {
	ID             types.String `tfsdk:"id"`
	TeamName       types.String `tfsdk:"team_name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ParentTeamID   types.String `tfsdk:"parent_team_id"`
	TeamType       types.String `tfsdk:"team_type"`
	Roles          types.Set    `tfsdk:"roles"`
	Members        types.Set    `tfsdk:"members"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

// Metadata returns the resource type name.
func (r *TeamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

// Schema defines the schema for the resource.
func (r *TeamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform team, including its inline role assignments and members.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_name": schema.StringAttribute{
				Description: "The name of the team.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the team will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_team_id": schema.StringAttribute{
				Description: "The ID of the parent team. If not specified, the provider looks up the " +
					"organization's root team (the team with no ancestors) and uses it as the parent — " +
					"mirroring the Anypoint UI, which defaults the parent to the root team. The platform " +
					"API requires a parent, so this is always populated in state after apply.",
				Optional: true,
				Computed: true,
				// Optional+Computed: when the user omits parent_team_id, the provider
				// computes it once (the org root) and stores it. Without this modifier,
				// every subsequent plan would mark the already-known parent as "known
				// after apply", which then drives Update to send an empty parent_team_id
				// (a phantom "move") and get a 400 from the platform. UseStateForUnknown
				// keeps the stored value stable across plans — matching id/created_at above.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_type": schema.StringAttribute{
				Description: "The type of the team. Optional; defaults to 'internal' (the same default the " +
					"Anypoint UI applies, where the Create Team dialog does not expose a type selector). " +
					"Changing the type requires the target type to be enabled in the organization.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("internal"),
			},
			"roles": schema.SetNestedAttribute{
				Description: "The set of roles (permissions) assigned to this team. When set, this list is " +
					"authoritative: roles not listed here are removed on apply. Omit the attribute entirely " +
					"to leave role assignments unmanaged. System (internal) assignments are never modified.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The role's display name as shown in the Anypoint UI (e.g., 'Exchange Viewer'). " +
								"Case-insensitive. Use the anypoint_available_roles data source to discover valid names.",
							Required: true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the role. Typically includes 'org' (organization ID) " +
								"and, for environment-scoped roles, 'envId'.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"members": schema.SetNestedAttribute{
				Description: "The set of members of this team. When set, this list is authoritative: members not " +
					"listed here are removed on apply. Omit the attribute entirely to leave membership unmanaged. " +
					"Members assigned via external groups (SAML/SCIM) are never modified.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Description: "The member's username. Case-insensitive; use the anypoint_users data source to discover usernames.",
							Required:    true,
						},
						"membership_type": schema.StringAttribute{
							Description: "The membership type: 'member' (default) or 'maintainer'. Maintainers can " +
								"additionally manage team membership and child teams. Omit to default to 'member'.",
							Optional: true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the team was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the team was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *TeamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
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
			"Unable to Create Team Client",
			"An unexpected error occurred when creating the Team client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	// Sub-clients for managing the team's roles and members inline. They share the
	// same cached token via userConfig, so no extra authentication is performed.
	rolesClient, err := accessmanagement.NewTeamRolesClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Roles Client",
			"An unexpected error occurred when creating the Team Roles client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
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

	usersClient, err := accessmanagement.NewRoleUsersClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Users Client",
			"An unexpected error occurred when creating the Users client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	catalogClient, err := accessmanagement.NewRolePermissionClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Catalog Client",
			"An unexpected error occurred when creating the Role Catalog client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = teamClient
	r.rolesClient = rolesClient
	r.membersClient = membersClient
	r.usersClient = usersClient
	r.catalogClient = catalogClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	manageRoles := !data.Roles.IsNull() && !data.Roles.IsUnknown()
	manageMembers := !data.Members.IsNull() && !data.Members.IsUnknown()

	// Resolve (and thereby validate) role names and usernames up front so we fail
	// before creating anything if a name is invalid — no orphaned team.
	desiredRoles, err := r.resolveTeamRoles(ctx, data.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Invalid role", err.Error())
		return
	}
	desiredMembers, err := r.resolveTeamMembers(ctx, orgID, data.Members)
	if err != nil {
		resp.Diagnostics.AddError("Invalid member", err.Error())
		return
	}

	// Resolve the parent team. The platform API REQUIRES parent_team_id, so when
	// the user omits it we default to the org's root team (the team with no
	// ancestors) — mirroring the Anypoint UI, whose Create Team dialog defaults
	// the parent to the root ("Everyone at <org>") and always sends it.
	parentTeamID := data.ParentTeamID.ValueString()
	if parentTeamID == "" {
		rootID, err := r.resolveRootTeamID(ctx, orgID)
		if err != nil {
			resp.Diagnostics.AddError("Error resolving root team", err.Error())
			return
		}
		parentTeamID = rootID
	}

	// Create the team
	teamRequest := &accessmanagement.CreateTeamRequest{
		TeamName:     data.TeamName.ValueString(),
		ParentTeamID: parentTeamID,
		TeamType:     data.TeamType.ValueString(),
	}

	team, err := r.client.CreateTeam(ctx, orgID, teamRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating team",
			"Could not create team: "+err.Error(),
		)
		return
	}

	// Map response body to schema (leaves Roles/Members untouched).
	r.mapTeamToState(team, &data, orgID)

	// Guarantee parent_team_id is recorded in state even if the create response
	// omitted ancestor_team_ids (mapTeamToState derives the parent from those).
	// Falls back to exactly what we sent — for an omitted parent that is the
	// resolved root ID, so a follow-up plan shows no drift.
	if data.ParentTeamID.IsNull() || data.ParentTeamID.IsUnknown() {
		data.ParentTeamID = types.StringValue(parentTeamID)
	}

	// Persist partial state now so a later failure doesn't orphan the team.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reconcile roles.
	if manageRoles {
		if err := r.applyTeamRoles(ctx, orgID, team.ID, desiredRoles); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &data, orgID, team.ID, manageRoles, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("Error applying roles", err.Error())
			return
		}
	}

	// Reconcile members.
	if manageMembers {
		if err := r.applyTeamMembers(ctx, orgID, team.ID, desiredMembers); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &data, orgID, team.ID, manageRoles, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("Error applying members", err.Error())
			return
		}
	}

	tflog.Trace(ctx, "created team")

	// On success, actual == desired == plan, so keep the plan's typed values verbatim
	// (guaranteeing state == config for these Optional attributes).
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Get the team from the API
	team, err := r.client.GetTeam(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading team",
			"Could not read team ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema (leaves Roles/Members as they were in state).
	r.mapTeamToState(team, &data, orgID)

	// Only refresh roles/members if they are being managed (non-null in state).
	// Omitted (null) attributes stay null so the resource leaves them unmanaged.
	if !data.Roles.IsNull() {
		roles, err := r.reconcileTeamRolesIntoState(ctx, orgID, data.ID.ValueString(), data.Roles)
		if err != nil {
			resp.Diagnostics.AddError("Error reading roles", err.Error())
			return
		}
		data.Roles = roles
	}
	if !data.Members.IsNull() {
		members, err := r.reconcileTeamMembersIntoState(ctx, orgID, data.ID.ValueString(), data.Members)
		if err != nil {
			resp.Diagnostics.AddError("Error reading members", err.Error())
			return
		}
		data.Members = members
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamResourceModel

	// Read Terraform plan and state data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	teamID := state.ID.ValueString()

	manageRoles := !plan.Roles.IsNull() && !plan.Roles.IsUnknown()
	manageMembers := !plan.Members.IsNull() && !plan.Members.IsUnknown()

	// Resolve (validate) desired roles/members before mutating anything.
	desiredRoles, err := r.resolveTeamRoles(ctx, plan.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Invalid role", err.Error())
		return
	}
	desiredMembers, err := r.resolveTeamMembers(ctx, orgID, plan.Members)
	if err != nil {
		resp.Diagnostics.AddError("Invalid member", err.Error())
		return
	}

	// Handle parent_team_id changes first (requires separate API call). Only move
	// the team when the planned parent is KNOWN, NON-EMPTY, and actually different
	// from the current parent. The unknown/empty guard is defense-in-depth: the
	// platform API rejects an empty parent_team_id with a 400, so a phantom "move"
	// (e.g. a planned unknown collapsing to "") must never reach it. With
	// parent_team_id using UseStateForUnknown this should not occur, but the guard
	// keeps a future regression from silently sending an empty parent.
	plannedParent := plan.ParentTeamID.ValueString()
	if !plan.ParentTeamID.IsUnknown() && plannedParent != "" && plannedParent != state.ParentTeamID.ValueString() {
		parentUpdateRequest := &accessmanagement.UpdateTeamParentRequest{
			ParentTeamID: plannedParent,
		}
		if err := r.client.UpdateTeamParent(ctx, orgID, teamID, parentUpdateRequest); err != nil {
			resp.Diagnostics.AddError(
				"Error updating team parent",
				"Could not update team parent: "+err.Error(),
			)
			return
		}
	}

	// Build update request for name/type — only send changed fields.
	updateRequest := &accessmanagement.UpdateTeamRequest{}
	hasChanges := false
	if !plan.TeamName.Equal(state.TeamName) {
		teamName := plan.TeamName.ValueString()
		updateRequest.TeamName = &teamName
		hasChanges = true
	}
	if !plan.TeamType.Equal(state.TeamType) {
		teamType := plan.TeamType.ValueString()
		updateRequest.TeamType = &teamType
		hasChanges = true
	}

	if hasChanges {
		team, err := r.client.UpdateTeam(ctx, orgID, teamID, updateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating team",
				"Could not update team: "+err.Error(),
			)
			return
		}
		r.mapTeamToState(team, &plan, orgID)
	} else {
		// Read the team back so computed fields (updated_at, etc.) are current.
		team, err := r.client.GetTeam(ctx, orgID, teamID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading team after update",
				"Could not read team: "+err.Error(),
			)
			return
		}
		r.mapTeamToState(team, &plan, orgID)
	}

	// Reconcile roles if managed. On failure, refresh actual state so a partial
	// apply is recorded accurately for the next run.
	if manageRoles {
		if err := r.applyTeamRoles(ctx, orgID, teamID, desiredRoles); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &plan, orgID, teamID, manageRoles, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddError("Error applying roles", err.Error())
			return
		}
	}

	if manageMembers {
		if err := r.applyTeamMembers(ctx, orgID, teamID, desiredMembers); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &plan, orgID, teamID, manageRoles, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddError("Error applying members", err.Error())
			return
		}
	}

	tflog.Trace(ctx, "updated team")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err := r.client.DeleteTeam(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted, nothing to do
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting team",
			"Could not delete team: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// refreshManagedIntoStateBestEffort re-reads the actual roles/members from the API
// into `data` after a partial-failure so the recorded state matches reality (the
// next apply then reconciles the remainder). Errors here are ignored — this is a
// best-effort cleanup path invoked only when an apply has already failed.
func (r *TeamResource) refreshManagedIntoStateBestEffort(ctx context.Context, data *TeamResourceModel, orgID, teamID string, manageRoles, manageMembers bool) {
	if manageRoles {
		if roles, err := r.reconcileTeamRolesIntoState(ctx, orgID, teamID, data.Roles); err == nil {
			data.Roles = roles
		}
	}
	if manageMembers {
		if members, err := r.reconcileTeamMembersIntoState(ctx, orgID, teamID, data.Members); err == nil {
			data.Members = members
		}
	}
}

// resolveRootTeamID returns the organization's root team ID — the team with no
// ancestors. Used when the user creates a team without specifying parent_team_id,
// so the provider can supply the parent the platform API requires (matching the
// Anypoint UI, which defaults the parent to the root team).
func (r *TeamResource) resolveRootTeamID(ctx context.Context, orgID string) (string, error) {
	teams, err := r.client.ListTeams(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("could not list teams to find the org root team: %w", err)
	}
	for _, t := range teams {
		if len(t.AncestorTeamIDs) == 0 {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf(
		"could not find a root team (a team with no ancestors) in organization %s; "+
			"set parent_team_id explicitly on the anypoint_team resource",
		orgID,
	)
}

// mapTeamToState maps an API Team response to the Terraform state model, leaving the
// Roles/Members typed values untouched (they are reconciled separately).
func (r *TeamResource) mapTeamToState(team *accessmanagement.Team, data *TeamResourceModel, orgID string) {
	data.ID = types.StringValue(team.ID)
	data.TeamName = types.StringValue(team.TeamName)
	data.TeamType = types.StringValue(team.TeamType)
	data.OrganizationID = types.StringValue(orgID)
	data.CreatedAt = types.StringValue(team.CreatedAt)
	data.UpdatedAt = types.StringValue(team.UpdatedAt)

	// Derive parent_team_id from ancestor_team_ids. The platform lists ancestors
	// root-first / direct-parent-LAST, so the direct parent is the last element
	// (see Team.DirectParentID). Using [0] would return the ROOT for any team more
	// than one level deep, flipping parent_team_id and causing "inconsistent result
	// after apply".
	if parent := team.DirectParentID(); parent != "" {
		data.ParentTeamID = types.StringValue(parent)
	} else if data.ParentTeamID.IsUnknown() {
		// No ancestors and nothing planned — normalize unknown to null so state is consistent.
		data.ParentTeamID = types.StringNull()
	}
}
