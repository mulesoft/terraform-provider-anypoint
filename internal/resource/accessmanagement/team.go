package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Name           types.String `tfsdk:"name"`
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
			"name": schema.StringAttribute{
				Description: "The name of the team (shown as 'Team Name' in the UI).",
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
				Description: "The ID of the parent team. If not specified, defaults to the organization's " +
					"root team (mirroring the Anypoint UI default). Use the anypoint_teams data source " +
					"to look up team IDs by name, or use the root team's ID (the one with is_root_team = true).",
				Optional: true,
				Computed: true,
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
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The role's display name as shown in the Anypoint UI (e.g., 'Exchange Viewer'). " +
								"Case-insensitive. Use the anypoint_available_permissions data source to discover valid names.",
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
				Computed: true,
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

	teamClient, err := accessmanagement.NewTeamClient(config)
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
	// same cached token via config, so no extra authentication is performed.
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
	// the user omits it we default to the org's root team — mirroring the Anypoint
	// UI, whose Create Team dialog defaults the parent to the root ("Everyone at
	// <org>") and always sends it.
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
		TeamName:     data.Name.ValueString(),
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
	actualParentID := r.mapTeamToState(team, &data, orgID)
	if actualParentID == "" {
		actualParentID = parentTeamID // fallback to what we sent
	}

	// Populate parent_team_id in state. If the user specified it, keep their value;
	// if they didn't (defaulted to root), populate from the resolved parent.
	if data.ParentTeamID.IsNull() || data.ParentTeamID.IsUnknown() || data.ParentTeamID.ValueString() == "" {
		if actualParentID != "" {
			data.ParentTeamID = types.StringValue(actualParentID)
		} else {
			data.ParentTeamID = types.StringValue(parentTeamID)
		}
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

	// For managed attributes, plan values are already correct. For unmanaged attributes
	// (config omitted → plan is unknown), populate from API so state is concrete.
	if !manageRoles {
		roles, rolesErr := r.reconcileTeamRolesIntoState(ctx, orgID, team.ID, data.Roles)
		if rolesErr != nil {
			resp.Diagnostics.AddError("Error reading roles after create", rolesErr.Error())
			return
		}
		data.Roles = roles
	}
	if !manageMembers {
		members, membersErr := r.reconcileTeamMembersIntoState(ctx, orgID, team.ID, data.Members)
		if membersErr != nil {
			resp.Diagnostics.AddError("Error reading members after create", membersErr.Error())
			return
		}
		data.Members = members
	}

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
	parentID := r.mapTeamToState(team, &data, orgID)

	// Populate parent_team_id from the API response's ancestor chain.
	if parentID != "" {
		data.ParentTeamID = types.StringValue(parentID)
	} else {
		// Root team has no parent
		data.ParentTeamID = types.StringNull()
	}

	// Always read roles/members from API into state. With Optional+Computed,
	// Terraform handles the diff: config sets them → authoritative; config omits → accepts API reality.
	roles, err := r.reconcileTeamRolesIntoState(ctx, orgID, data.ID.ValueString(), data.Roles)
	if err != nil {
		resp.Diagnostics.AddError("Error reading roles", err.Error())
		return
	}
	data.Roles = roles

	members, err := r.reconcileTeamMembersIntoState(ctx, orgID, data.ID.ValueString(), data.Members)
	if err != nil {
		resp.Diagnostics.AddError("Error reading members", err.Error())
		return
	}
	data.Members = members

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

	// Handle parent team changes first (requires separate API call).
	// Compare planned parent_team_id to state parent_team_id to detect moves.
	plannedParent := plan.ParentTeamID.ValueString()
	stateParent := state.ParentTeamID.ValueString()
	if plannedParent != "" && plannedParent != stateParent {
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
	if !plan.Name.Equal(state.Name) {
		teamName := plan.Name.ValueString()
		updateRequest.TeamName = &teamName
		hasChanges = true
	}
	if !plan.TeamType.Equal(state.TeamType) {
		teamType := plan.TeamType.ValueString()
		updateRequest.TeamType = &teamType
		hasChanges = true
	}

	var updateParentID string
	if hasChanges {
		team, err := r.client.UpdateTeam(ctx, orgID, teamID, updateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating team",
				"Could not update team: "+err.Error(),
			)
			return
		}
		updateParentID = r.mapTeamToState(team, &plan, orgID)
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
		updateParentID = r.mapTeamToState(team, &plan, orgID)
	}

	// Populate parent_team_id in state from the API response.
	if updateParentID != "" {
		plan.ParentTeamID = types.StringValue(updateParentID)
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

	// For unmanaged attributes (config omitted → plan is unknown), populate from API.
	if !manageRoles {
		roles, rolesErr := r.reconcileTeamRolesIntoState(ctx, orgID, teamID, plan.Roles)
		if rolesErr != nil {
			resp.Diagnostics.AddError("Error reading roles after update", rolesErr.Error())
			return
		}
		plan.Roles = roles
	}
	if !manageMembers {
		members, membersErr := r.reconcileTeamMembersIntoState(ctx, orgID, teamID, plan.Members)
		if membersErr != nil {
			resp.Diagnostics.AddError("Error reading members after update", membersErr.Error())
			return
		}
		plan.Members = members
	}

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
// Roles and members will be populated by the Read that follows ImportState.
// With Optional+Computed schema, the API-read values become state, and
// Terraform's diff engine handles authoritative enforcement if config sets them.
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
// Returns the direct parent team ID (from ancestor_team_ids) for callers that need it.
func (r *TeamResource) mapTeamToState(team *accessmanagement.Team, data *TeamResourceModel, orgID string) string {
	data.ID = types.StringValue(team.ID)
	data.Name = types.StringValue(team.TeamName)
	data.TeamType = types.StringValue(team.TeamType)
	data.OrganizationID = types.StringValue(orgID)
	data.CreatedAt = types.StringValue(team.CreatedAt)
	data.UpdatedAt = types.StringValue(team.UpdatedAt)

	// Derive parent team ID from ancestor_team_ids. The platform lists ancestors
	// root-first / direct-parent-LAST, so the direct parent is the last element
	// (see Team.DirectParentID).
	return team.DirectParentID()
}

// teamRoleObjectType is the Terraform object type for a single role entry in the
// anypoint_team resource's `roles` set. It mirrors the role resource's permission
// entry (a display name plus optional context params).
var teamRoleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

// teamMemberObjectType is the Terraform object type for a single member entry in
// the anypoint_team resource's `members` set: a username and an optional
// membership_type ("member" or "maintainer").
var teamMemberObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"username":        types.StringType,
		"membership_type": types.StringType,
	},
}

// desiredTeamRole is a resolved role ready to be sent to the API.
type desiredTeamRole struct {
	roleID        string
	typedName     string            // exactly what the user wrote (preserved in state)
	contextParams map[string]string // resolved context params
	contextKey    string            // canonical JSON of contextParams for matching
}

// typedTeamRole holds the user's original (typed) representation of a role so Read
// can preserve casing / context_params form and avoid perpetual diffs.
type typedTeamRole struct {
	name types.String
	cp   types.Map
}

// desiredTeamMember is a resolved member ready to be sent to the API.
type desiredTeamMember struct {
	userID         string
	typedUsername  string // exactly what the user wrote (preserved in state)
	membershipType string // resolved membership type ("member" when omitted)
}

// typedTeamMember holds the user's original (typed) representation of a member so
// Read can preserve username casing and the (possibly null) membership_type.
type typedTeamMember struct {
	username       types.String
	membershipType types.String
}

// --- roles -------------------------------------------------------------------

// resolveTeamRoles resolves each role's display name to a role_id using the
// available-roles catalog (case-insensitive). Returns an error listing the
// offending name if it is unknown or ambiguous. Returns (nil, nil) when unmanaged.
func (r *TeamResource) resolveTeamRoles(ctx context.Context, roleSet types.Set) ([]desiredTeamRole, error) {
	if roleSet.IsNull() || roleSet.IsUnknown() {
		return nil, nil
	}

	roles, err := r.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list available roles for name resolution: %w", err)
	}

	// Build a case-insensitive name -> []role_id map so we can detect ambiguity.
	nameToIDs := make(map[string][]string, len(roles))
	for _, role := range roles {
		key := normalizeName(role.Name)
		nameToIDs[key] = append(nameToIDs[key], role.RoleID)
	}

	out := make([]desiredTeamRole, 0, len(roleSet.Elements()))
	for _, el := range roleSet.Elements() {
		obj := el.(types.Object)
		attrs := obj.Attributes()
		typedName := attrs["name"].(types.String).ValueString()
		cpMap := mapToStringMap(attrs["context_params"].(types.Map))

		ids := nameToIDs[normalizeName(typedName)]
		if len(ids) == 0 {
			return nil, fmt.Errorf(
				"role %q is not a valid role name; use the anypoint_available_permissions data source to discover valid role names",
				typedName,
			)
		}
		if len(ids) > 1 {
			return nil, fmt.Errorf(
				"role name %q is ambiguous (matches %d roles); the platform has multiple roles with this name so it cannot be resolved by name alone",
				typedName, len(ids),
			)
		}

		out = append(out, desiredTeamRole{
			roleID:        ids[0],
			typedName:     typedName,
			contextParams: cpMap,
			contextKey:    canonicalContextParams(cpMap),
		})
	}
	return out, nil
}

// applyTeamRoles reconciles the team's role assignments to exactly `desired`.
// It adds missing assignments and removes extras, skipping internal (system) ones.
func (r *TeamResource) applyTeamRoles(ctx context.Context, orgID, teamID string, desired []desiredTeamRole) error {
	assignments, err := r.rolesClient.ListTeamRoles(ctx, orgID, teamID)
	if err != nil {
		return fmt.Errorf("could not list current team role assignments: %w", err)
	}

	// Build the set of catalog (assignable) role IDs. Any assignment whose role_id
	// is not in the catalog is a platform-injected side-effect grant (e.g. the
	// org-scoped "Business Group Viewer" auto-added alongside an env-scoped role)
	// and must never be removed — the user can't express it in config, so removing
	// it would fight the platform on every apply.
	catalog, err := r.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		return fmt.Errorf("could not list available roles for assignment reconciliation: %w", err)
	}
	catalogIDs := make(map[string]struct{}, len(catalog))
	for _, role := range catalog {
		catalogIDs[role.RoleID] = struct{}{}
	}

	actual := make(map[string]accessmanagement.TeamRoleAssignment)
	for _, a := range assignments {
		if a.Internal {
			continue // never touch system-managed assignments
		}
		if _, inCatalog := catalogIDs[a.RoleID]; !inCatalog {
			continue // never touch platform-injected side-effect grants
		}
		actual[a.RoleID+"|"+canonicalContextParams(a.ContextParams)] = a
	}

	desiredByKey := make(map[string]desiredTeamRole, len(desired))
	for _, d := range desired {
		desiredByKey[d.roleID+"|"+d.contextKey] = d
	}

	// Add anything desired that isn't already present.
	for key, d := range desiredByKey {
		if _, ok := actual[key]; ok {
			continue
		}
		if _, err := r.rolesClient.AssignTeamRole(ctx, orgID, teamID, &accessmanagement.AssignTeamRoleRequest{
			RoleID:        d.roleID,
			ContextParams: d.contextParams,
		}); err != nil {
			return fmt.Errorf("failed to assign role %q: %w", d.typedName, err)
		}
	}

	// Remove anything present that is no longer desired.
	for key, a := range actual {
		if _, ok := desiredByKey[key]; ok {
			continue
		}
		if err := r.rolesClient.UnassignTeamRole(ctx, orgID, teamID, &accessmanagement.AssignTeamRoleRequest{
			RoleID:        a.RoleID,
			ContextParams: a.ContextParams,
		}); err != nil {
			return fmt.Errorf("failed to unassign role %q: %w", a.Name, err)
		}
	}

	return nil
}

// reconcileTeamRolesIntoState reads the actual (non-internal) role assignments from
// the API and builds a roles set. For each assignment matching an entry in
// typedSource, the user's original name/context_params are preserved (so casing
// differences don't cause perpetual diffs). Unmatched assignments (drift or import)
// are labeled with the canonical role name from the catalog.
func (r *TeamResource) reconcileTeamRolesIntoState(ctx context.Context, orgID, teamID string, typedSource types.Set) (types.Set, error) {
	assignments, err := r.rolesClient.ListTeamRoles(ctx, orgID, teamID)
	if err != nil {
		return types.SetNull(teamRoleObjectType), fmt.Errorf("could not list team role assignments: %w", err)
	}

	roles, err := r.catalogClient.ListAvailableRoles(ctx)
	if err != nil {
		return types.SetNull(teamRoleObjectType), fmt.Errorf("could not list available roles: %w", err)
	}
	roleIDToName := make(map[string]string, len(roles))
	nameToID := make(map[string]string, len(roles))
	for _, role := range roles {
		roleIDToName[role.RoleID] = role.Name
		nameToID[normalizeName(role.Name)] = role.RoleID
	}

	// Index the user's typed roles by (role_id | context) so we can preserve their
	// exact representation for matched assignments.
	typedByKey := map[string]typedTeamRole{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			obj := el.(types.Object)
			attrs := obj.Attributes()
			typedName := attrs["name"].(types.String)
			cpTypes := attrs["context_params"].(types.Map)
			rid := nameToID[normalizeName(typedName.ValueString())]
			key := rid + "|" + canonicalContextParams(mapToStringMap(cpTypes))
			typedByKey[key] = typedTeamRole{name: typedName, cp: cpTypes}
		}
	}

	objs := make([]attr.Value, 0, len(assignments))
	for _, a := range assignments {
		if a.Internal {
			continue
		}
		// Skip platform-injected side-effect grants that are not in the assignable
		// catalog (e.g. the org-scoped "Business Group Viewer" the platform auto-adds
		// when any env-scoped role is assigned to a team). The user cannot express
		// these in config — resolveTeamRoles only accepts catalog role names — so
		// treating them as managed would surface a phantom `name = ""` entry and a
		// perpetual diff. Mirror the internal-skip above.
		if _, inCatalog := roleIDToName[a.RoleID]; !inCatalog {
			continue
		}
		key := a.RoleID + "|" + canonicalContextParams(a.ContextParams)

		var nameVal types.String
		var cpVal types.Map
		if tr, ok := typedByKey[key]; ok {
			nameVal = tr.name
			cpVal = tr.cp
		} else {
			nameVal = types.StringValue(roleIDToName[a.RoleID])
			cpVal = stringMapToTypesMap(a.ContextParams)
		}

		obj, diags := types.ObjectValue(teamRoleObjectType.AttrTypes, map[string]attr.Value{
			"name":           nameVal,
			"context_params": cpVal,
		})
		if diags.HasError() {
			return types.SetNull(teamRoleObjectType), fmt.Errorf("failed to build team role object")
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(teamRoleObjectType, objs)
	if diags.HasError() {
		return types.SetNull(teamRoleObjectType), fmt.Errorf("failed to build team roles set")
	}
	return set, nil
}

// --- members -----------------------------------------------------------------

// resolveTeamMembers resolves each username to a user_id (case-insensitive) using
// the org user list. Returns a map of user_id -> desiredTeamMember (carrying the
// typed username and resolved membership type). Returns an error listing the
// offending username if it is unknown. Returns (nil, nil) when unmanaged.
func (r *TeamResource) resolveTeamMembers(ctx context.Context, orgID string, memberSet types.Set) (map[string]desiredTeamMember, error) {
	if memberSet.IsNull() || memberSet.IsUnknown() {
		return nil, nil
	}

	users, err := r.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not list organization users for username resolution: %w", err)
	}
	usernameToID := make(map[string]string, len(users))
	for _, u := range users {
		usernameToID[normalizeName(u.Username)] = u.ID
	}

	out := make(map[string]desiredTeamMember)
	for _, el := range memberSet.Elements() {
		obj := el.(types.Object)
		attrs := obj.Attributes()
		typedUsername := attrs["username"].(types.String).ValueString()

		membershipType := "member"
		if mt, ok := attrs["membership_type"].(types.String); ok && !mt.IsNull() && !mt.IsUnknown() && mt.ValueString() != "" {
			membershipType = mt.ValueString()
		}

		id, ok := usernameToID[normalizeName(typedUsername)]
		if !ok {
			return nil, fmt.Errorf(
				"member %q is not a valid username in organization %s; use the anypoint_users data source to discover usernames",
				typedUsername, orgID,
			)
		}
		out[id] = desiredTeamMember{
			userID:         id,
			typedUsername:  typedUsername,
			membershipType: membershipType,
		}
	}
	return out, nil
}

// applyTeamMembers reconciles the team's membership to exactly the desired members.
// Members assigned via external groups (SAML/SCIM) are never modified. Members whose
// membership_type changed are re-PUT to update in place.
func (r *TeamResource) applyTeamMembers(ctx context.Context, orgID, teamID string, desired map[string]desiredTeamMember) error {
	actual, err := r.membersClient.ListTeamMembers(ctx, orgID, teamID)
	if err != nil {
		return fmt.Errorf("could not list current team members: %w", err)
	}

	// Build the set of real org-user IDs. The team-members endpoint also returns
	// non-user identities: once a team has a child team, that child surfaces here as
	// a member entry (and the platform may inject other group identities). Those are
	// NOT org users, so they can never appear in `desired` (which is built by
	// resolving usernames). They must NOT be removed — deleting a child team via the
	// members endpoint fails with 405 "Cannot delete a child team using this
	// endpoint". Mirroring the read path (reconcileTeamMembersIntoState), we only
	// manage members that resolve to an org user; every other identity is left
	// untouched, exactly like external-group memberships.
	users, err := r.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		return fmt.Errorf("could not list organization users: %w", err)
	}
	isOrgUser := make(map[string]bool, len(users))
	for _, u := range users {
		isOrgUser[u.ID] = true
	}

	actualByID := make(map[string]accessmanagement.TeamMember, len(actual))
	for _, m := range actual {
		if m.IsAssignedViaExternalGroups {
			continue // never touch externally-managed memberships
		}
		if !isOrgUser[m.ID] {
			continue // non-user identity (e.g. a child team) — not manageable here
		}
		actualByID[m.ID] = m
	}

	// Add missing members, or update those whose membership_type changed.
	for id, d := range desired {
		cur, present := actualByID[id]
		if present && cur.MembershipType == d.membershipType {
			continue
		}
		if err := r.membersClient.AddTeamMember(ctx, orgID, teamID, id, d.membershipType); err != nil {
			return fmt.Errorf("failed to add/update member %q: %w", d.typedUsername, err)
		}
	}

	// Remove members that are no longer desired.
	for id := range actualByID {
		if _, ok := desired[id]; ok {
			continue
		}
		if err := r.membersClient.RemoveTeamMember(ctx, orgID, teamID, id); err != nil {
			return fmt.Errorf("failed to remove member (user_id %s): %w", id, err)
		}
	}

	return nil
}

// reconcileTeamMembersIntoState reads actual team members and builds a members set,
// preserving the user's typed username casing and (possibly null) membership_type
// for matched members. External-group memberships and non-user identities that
// cannot be mapped to an org username are excluded.
func (r *TeamResource) reconcileTeamMembersIntoState(ctx context.Context, orgID, teamID string, typedSource types.Set) (types.Set, error) {
	actual, err := r.membersClient.ListTeamMembers(ctx, orgID, teamID)
	if err != nil {
		return types.SetNull(teamMemberObjectType), fmt.Errorf("could not list team members: %w", err)
	}

	users, err := r.usersClient.ListOrgUsers(ctx, orgID)
	if err != nil {
		return types.SetNull(teamMemberObjectType), fmt.Errorf("could not list organization users: %w", err)
	}
	userIDToUsername := make(map[string]string, len(users))
	for _, u := range users {
		userIDToUsername[u.ID] = u.Username
	}

	// Index the user's typed members by lowercased username so we can preserve their
	// exact casing and membership_type form for matched members.
	typedByLower := map[string]typedTeamMember{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			obj := el.(types.Object)
			attrs := obj.Attributes()
			uname := attrs["username"].(types.String)
			mt, _ := attrs["membership_type"].(types.String)
			typedByLower[normalizeName(uname.ValueString())] = typedTeamMember{username: uname, membershipType: mt}
		}
	}

	objs := make([]attr.Value, 0, len(actual))
	for _, m := range actual {
		if m.IsAssignedViaExternalGroups {
			continue
		}
		username, ok := userIDToUsername[m.ID]
		if !ok {
			// Not an org user (e.g. a group identity) — cannot represent by username.
			continue
		}

		var usernameVal types.String
		var mtVal types.String
		if tm, matched := typedByLower[normalizeName(username)]; matched {
			usernameVal = tm.username
			// Preserve the typed membership_type verbatim (including null when the
			// user omitted it) so state matches config and avoids perpetual diffs.
			mtVal = tm.membershipType
		} else {
			usernameVal = types.StringValue(username)
			if m.MembershipType != "" {
				mtVal = types.StringValue(m.MembershipType)
			} else {
				mtVal = types.StringValue("member")
			}
		}

		obj, diags := types.ObjectValue(teamMemberObjectType.AttrTypes, map[string]attr.Value{
			"username":        usernameVal,
			"membership_type": mtVal,
		})
		if diags.HasError() {
			return types.SetNull(teamMemberObjectType), fmt.Errorf("failed to build team member object")
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(teamMemberObjectType, objs)
	if diags.HasError() {
		return types.SetNull(teamMemberObjectType), fmt.Errorf("failed to build team members set")
	}
	return set, nil
}
