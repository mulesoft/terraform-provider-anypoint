package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &RoleResource{}
	_ resource.ResourceWithConfigure   = &RoleResource{}
	_ resource.ResourceWithImportState = &RoleResource{}
)

// RoleResource is the resource implementation.
type RoleResource struct {
	client      *accessmanagement.RoleClient
	permClient  *accessmanagement.RolePermissionClient
	usersClient *accessmanagement.RoleUsersClient
}

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Editable       types.Bool   `tfsdk:"editable"`
	ExternalNames  types.List   `tfsdk:"external_names"`
	Permissions    types.Set    `tfsdk:"permissions"`
	Members        types.Set    `tfsdk:"members"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// Metadata returns the resource type name.
func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource.
func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform role group (custom or default). Requires Organization Administrator privileges.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the role group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the role group.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the role group.",
				Optional:    true,
				Computed:    true,
				// Optional+Computed: when the user omits `description`, the server
				// resolves it (to ""). Without this modifier, any OTHER change (e.g. a
				// name edit) re-marks description as "(known after apply)", which then
				// resolves to "" during Update and WIPES the server-side description.
				// UseStateForUnknown pins the known prior value so an omitted
				// description is never spuriously cleared. (Class B invariant.)
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the role group will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"editable": schema.BoolAttribute{
				Description: "Whether the role group can be edited. Default (system) role groups are not editable.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"external_names": schema.ListAttribute{
				Description: "External group names mapped to this role group (for SSO/SAML integration). Read-only.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.SetNestedAttribute{
				Description: "The set of permissions granted by this role group. When set, this list is " +
					"authoritative: permissions not listed here are removed on apply. Omit the attribute " +
					"entirely to leave permissions unmanaged. System (internal) assignments are never modified.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The permission's display name as shown in the Anypoint UI (e.g., 'Exchange Viewer'). " +
								"Case-insensitive. Use the anypoint_available_roles data source to discover valid names.",
							Required: true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the permission. Typically includes 'org' (organization ID) " +
								"and, for environment-scoped permissions, 'envId'.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"members": schema.SetAttribute{
				Description: "The set of usernames that are members of this role group. When set, this list is " +
					"authoritative: members not listed here are removed on apply. Omit the attribute entirely to " +
					"leave membership unmanaged. Usernames are case-insensitive; use the anypoint_users data source " +
					"to discover usernames.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the role group was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the role group was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	roleClient, err := accessmanagement.NewRoleClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Group Client",
			"An unexpected error occurred when creating the Role Group client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	// Sub-clients for managing the role group's permissions and members inline.
	// They share the same cached token via userConfig, so no extra authentication.
	permClient, err := accessmanagement.NewRolePermissionClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Permission Client",
			"An unexpected error occurred when creating the Role Permission client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	usersClient, err := accessmanagement.NewRoleUsersClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Users Client",
			"An unexpected error occurred when creating the Role Users client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = roleClient
	r.permClient = permClient
	r.usersClient = usersClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel

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

	managePerms := !data.Permissions.IsNull() && !data.Permissions.IsUnknown()
	manageMembers := !data.Members.IsNull() && !data.Members.IsUnknown()

	// Resolve (and thereby validate) permission names and usernames up front so we
	// fail before creating anything if a name is invalid — no orphaned role group.
	desiredPerms, err := r.resolvePermissions(ctx, data.Permissions)
	if err != nil {
		resp.Diagnostics.AddError("Invalid permission", err.Error())
		return
	}
	desiredMembers, err := r.resolveMembers(ctx, orgID, data.Members)
	if err != nil {
		resp.Diagnostics.AddError("Invalid member", err.Error())
		return
	}

	// Create the role group
	roleGroupRequest := &accessmanagement.CreateRoleGroupRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	roleGroup, err := r.client.CreateRoleGroup(ctx, orgID, roleGroupRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating role group",
			"Could not create role group: "+err.Error(),
		)
		return
	}

	// Map response body to schema (leaves Permissions/Members untouched).
	r.mapRoleGroupToState(roleGroup, &data, orgID)

	// Persist partial state now so a later failure doesn't orphan the role group.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reconcile permissions.
	if managePerms {
		if err := r.applyPermissions(ctx, orgID, roleGroup.ID, desiredPerms); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &data, orgID, roleGroup.ID, managePerms, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("Error applying permissions", err.Error())
			return
		}
	}

	// Reconcile members.
	if manageMembers {
		if err := r.applyMembers(ctx, orgID, roleGroup.ID, desiredMembers); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &data, orgID, roleGroup.ID, managePerms, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("Error applying members", err.Error())
			return
		}
	}

	tflog.Trace(ctx, "created role group")

	// On success, actual == desired == plan, so keep the plan's typed values verbatim
	// (guaranteeing state == config for these Optional attributes).
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel

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

	// Get the role group from the API
	roleGroup, err := r.client.GetRoleGroup(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Role group was deleted outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role group",
			"Could not read role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema (leaves Permissions/Members as they were in state).
	r.mapRoleGroupToState(roleGroup, &data, orgID)

	// Only refresh permissions/members if they are being managed (non-null in state).
	// Omitted (null) attributes stay null so the resource leaves them unmanaged.
	if !data.Permissions.IsNull() {
		perms, err := r.reconcilePermissionsIntoState(ctx, orgID, data.ID.ValueString(), data.Permissions)
		if err != nil {
			resp.Diagnostics.AddError("Error reading permissions", err.Error())
			return
		}
		data.Permissions = perms
	}
	if !data.Members.IsNull() {
		members, err := r.reconcileMembersIntoState(ctx, orgID, data.ID.ValueString(), data.Members)
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
func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleResourceModel

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

	roleGroupID := state.ID.ValueString()

	managePerms := !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown()
	manageMembers := !plan.Members.IsNull() && !plan.Members.IsUnknown()

	// Resolve (validate) desired permissions/members before mutating anything.
	desiredPerms, err := r.resolvePermissions(ctx, plan.Permissions)
	if err != nil {
		resp.Diagnostics.AddError("Invalid permission", err.Error())
		return
	}
	desiredMembers, err := r.resolveMembers(ctx, orgID, plan.Members)
	if err != nil {
		resp.Diagnostics.AddError("Invalid member", err.Error())
		return
	}

	// Build update request — always send both name and description
	updateRequest := &accessmanagement.UpdateRoleGroupRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	roleGroup, err := r.client.UpdateRoleGroup(ctx, orgID, roleGroupID, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating role group",
			"Could not update role group: "+err.Error(),
		)
		return
	}

	// Map response body to schema (preserves plan's Permissions/Members typed values).
	r.mapRoleGroupToState(roleGroup, &plan, orgID)

	// Reconcile permissions if managed. On failure, refresh actual state so a partial
	// apply is recorded accurately for the next run.
	if managePerms {
		if err := r.applyPermissions(ctx, orgID, roleGroupID, desiredPerms); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &plan, orgID, roleGroupID, managePerms, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddError("Error applying permissions", err.Error())
			return
		}
	}

	if manageMembers {
		if err := r.applyMembers(ctx, orgID, roleGroupID, desiredMembers); err != nil {
			r.refreshManagedIntoStateBestEffort(ctx, &plan, orgID, roleGroupID, managePerms, manageMembers)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddError("Error applying members", err.Error())
			return
		}
	}

	tflog.Trace(ctx, "updated role group")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel

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

	err := r.client.DeleteRoleGroup(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted, nothing to do
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting role group",
			"Could not delete role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted role group")
}

// ImportState imports the resource by ID.
func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// refreshManagedIntoStateBestEffort re-reads the actual permissions/members from the
// API into `data` after a partial-failure so the recorded state matches reality (the
// next apply then reconciles the remainder). Errors here are ignored — this is a
// best-effort cleanup path invoked only when an apply has already failed.
func (r *RoleResource) refreshManagedIntoStateBestEffort(ctx context.Context, data *RoleResourceModel, orgID, roleGroupID string, managePerms, manageMembers bool) {
	if managePerms {
		if perms, err := r.reconcilePermissionsIntoState(ctx, orgID, roleGroupID, data.Permissions); err == nil {
			data.Permissions = perms
		}
	}
	if manageMembers {
		if members, err := r.reconcileMembersIntoState(ctx, orgID, roleGroupID, data.Members); err == nil {
			data.Members = members
		}
	}
}

// mapRoleGroupToState maps an API RoleGroup response to the Terraform state model.
func (r *RoleResource) mapRoleGroupToState(roleGroup *accessmanagement.RoleGroup, data *RoleResourceModel, orgID string) {
	data.ID = types.StringValue(roleGroup.ID)
	data.Name = types.StringValue(roleGroup.Name)
	data.Description = types.StringValue(roleGroup.Description)
	data.OrganizationID = types.StringValue(orgID)
	data.Editable = types.BoolValue(roleGroup.Editable)
	data.CreatedAt = types.StringValue(roleGroup.CreatedAt)
	data.UpdatedAt = types.StringValue(roleGroup.UpdatedAt)

	// Map external_names list
	if len(roleGroup.ExternalNames) > 0 {
		elems := make([]attr.Value, len(roleGroup.ExternalNames))
		for i, name := range roleGroup.ExternalNames {
			elems[i] = types.StringValue(name)
		}
		data.ExternalNames = types.ListValueMust(types.StringType, elems)
	} else {
		data.ExternalNames = types.ListValueMust(types.StringType, []attr.Value{})
	}
}
