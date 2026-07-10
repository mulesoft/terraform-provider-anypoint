package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
								"Case-insensitive. Use the anypoint_available_permissions data source to discover valid names.",
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

	// Seed permissions and members as empty sets so the subsequent Read populates
	// them from the API (null would cause Read to skip them as "unmanaged").
	emptyPerms, diags := types.SetValue(permissionObjectType, []attr.Value{})
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permissions"), emptyPerms)...)

	emptyMembers, diags := types.SetValue(types.StringType, []attr.Value{})
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("members"), emptyMembers)...)
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

// permissionObjectType is the Terraform object type for a single permission entry
// in the anypoint_role resource's `permissions` set.
var permissionObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

// desiredPermission is a resolved permission ready to be sent to the API.
type desiredPermission struct {
	roleID        string
	typedName     string            // exactly what the user wrote (preserved in state)
	contextParams map[string]string // resolved context params
	contextKey    string            // canonical JSON of contextParams for matching
}

// typedPerm holds the user's original (typed) representation of a permission so
// Read can preserve casing / context_params form and avoid perpetual diffs.
type typedPerm struct {
	name types.String
	cp   types.Map
}

// canonicalContextParams returns a stable JSON encoding of a context params map.
// A nil map and an empty map both canonicalize to "{}" so they compare equal.
// json.Marshal already emits map keys in sorted order, giving us a canonical form.
func canonicalContextParams(m map[string]string) string {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// mapToStringMap converts a types.Map (possibly null/unknown) to a plain map.
func mapToStringMap(m types.Map) map[string]string {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	for k, v := range m.Elements() {
		if s, ok := v.(types.String); ok {
			out[k] = s.ValueString()
		}
	}
	return out
}

// stringMapToTypesMap converts a plain map to a types.Map. An empty map becomes
// a null map so it matches a user who omitted context_params.
func stringMapToTypesMap(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}

// normalizeName lowercases and trims a display name / username for case-insensitive matching.
func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// resolvePermissions resolves each permission's display name to a role_id using the
// available-roles catalog (case-insensitive). Returns an error listing the offending
// name if it is unknown or ambiguous. Returns (nil, nil) when the set is unmanaged.
func (r *RoleResource) resolvePermissions(ctx context.Context, permSet types.Set) ([]desiredPermission, error) {
	if permSet.IsNull() || permSet.IsUnknown() {
		return nil, nil
	}

	roles, err := r.permClient.ListAvailableRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list available roles for name resolution: %w", err)
	}

	// Build a case-insensitive name -> []role_id map so we can detect ambiguity.
	nameToIDs := make(map[string][]string, len(roles))
	for _, role := range roles {
		key := normalizeName(role.Name)
		nameToIDs[key] = append(nameToIDs[key], role.RoleID)
	}

	out := make([]desiredPermission, 0, len(permSet.Elements()))
	for _, el := range permSet.Elements() {
		obj := el.(types.Object)
		attrs := obj.Attributes()
		typedName := attrs["name"].(types.String).ValueString()
		cpMap := mapToStringMap(attrs["context_params"].(types.Map))

		ids := nameToIDs[normalizeName(typedName)]
		if len(ids) == 0 {
			return nil, fmt.Errorf(
				"permission %q is not a valid role name; use the anypoint_available_permissions data source to discover valid permission names",
				typedName,
			)
		}
		if len(ids) > 1 {
			return nil, fmt.Errorf(
				"permission name %q is ambiguous (matches %d roles); the platform has multiple roles with this name so it cannot be resolved by name alone",
				typedName, len(ids),
			)
		}

		out = append(out, desiredPermission{
			roleID:        ids[0],
			typedName:     typedName,
			contextParams: cpMap,
			contextKey:    canonicalContextParams(cpMap),
		})
	}
	return out, nil
}

// applyPermissions reconciles the role group's assignments to exactly `desired`.
// It adds missing assignments and removes extras, skipping internal (system) ones.
func (r *RoleResource) applyPermissions(ctx context.Context, orgID, roleGroupID string, desired []desiredPermission) error {
	assignments, err := r.permClient.ListRoleAssignments(ctx, orgID, roleGroupID)
	if err != nil {
		return fmt.Errorf("could not list current role assignments: %w", err)
	}

	// Build the set of catalog (assignable) role IDs. Any assignment whose role_id
	// is not in the catalog is a platform-injected side-effect grant (e.g. the
	// org-scoped "Business Group Viewer" the platform auto-adds alongside an
	// env-scoped role) and must never be removed — the user can't express it in
	// config, so removing it (in the remove loop below) would fail or fight the
	// platform on every apply. Mirror applyTeamRoles. (Class A invariant.)
	catalog, err := r.permClient.ListAvailableRoles(ctx)
	if err != nil {
		return fmt.Errorf("could not list available roles for assignment reconciliation: %w", err)
	}
	catalogIDs := make(map[string]struct{}, len(catalog))
	for _, role := range catalog {
		catalogIDs[role.RoleID] = struct{}{}
	}

	actual := make(map[string]accessmanagement.RoleAssignment)
	for _, a := range assignments {
		if a.Internal {
			continue // never touch system-managed assignments
		}
		if _, inCatalog := catalogIDs[a.RoleID]; !inCatalog {
			continue // never touch platform-injected side-effect grants
		}
		actual[a.RoleID+"|"+canonicalContextParams(a.ContextParams)] = a
	}

	desiredByKey := make(map[string]desiredPermission, len(desired))
	for _, d := range desired {
		desiredByKey[d.roleID+"|"+d.contextKey] = d
	}

	// Add anything desired that isn't already present.
	for key, d := range desiredByKey {
		if _, ok := actual[key]; ok {
			continue
		}
		if _, err := r.permClient.AssignRole(ctx, orgID, roleGroupID, &accessmanagement.AssignRoleRequest{
			RoleID:        d.roleID,
			ContextParams: d.contextParams,
		}); err != nil {
			return fmt.Errorf("failed to assign permission %q: %w", d.typedName, err)
		}
	}

	// Remove anything present that is no longer desired.
	for key, a := range actual {
		if _, ok := desiredByKey[key]; ok {
			continue
		}
		if err := r.permClient.UnassignRole(ctx, orgID, roleGroupID, &accessmanagement.AssignRoleRequest{
			RoleID:        a.RoleID,
			ContextParams: a.ContextParams,
		}); err != nil {
			return fmt.Errorf("failed to unassign permission %q: %w", a.Name, err)
		}
	}

	return nil
}

// reconcilePermissionsIntoState reads the actual (non-internal) assignments from the
// API and builds a permissions set. For each assignment that matches an entry in
// typedSource, the user's original name/context_params are preserved (so casing
// differences don't cause perpetual diffs). Unmatched assignments (drift or import)
// are labeled with the canonical role name from the catalog.
func (r *RoleResource) reconcilePermissionsIntoState(ctx context.Context, orgID, roleGroupID string, typedSource types.Set) (types.Set, error) {
	assignments, err := r.permClient.ListRoleAssignments(ctx, orgID, roleGroupID)
	if err != nil {
		return types.SetNull(permissionObjectType), fmt.Errorf("could not list role assignments: %w", err)
	}

	roles, err := r.permClient.ListAvailableRoles(ctx)
	if err != nil {
		return types.SetNull(permissionObjectType), fmt.Errorf("could not list available roles: %w", err)
	}
	roleIDToName := make(map[string]string, len(roles))
	nameToID := make(map[string]string, len(roles))
	for _, role := range roles {
		roleIDToName[role.RoleID] = role.Name
		nameToID[normalizeName(role.Name)] = role.RoleID
	}

	// Index the user's typed permissions by (role_id | context) so we can preserve
	// their exact representation for matched assignments.
	typedByKey := map[string]typedPerm{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			obj := el.(types.Object)
			attrs := obj.Attributes()
			typedName := attrs["name"].(types.String)
			cpTypes := attrs["context_params"].(types.Map)
			rid := nameToID[normalizeName(typedName.ValueString())]
			key := rid + "|" + canonicalContextParams(mapToStringMap(cpTypes))
			typedByKey[key] = typedPerm{name: typedName, cp: cpTypes}
		}
	}

	objs := make([]attr.Value, 0, len(assignments))
	for _, a := range assignments {
		if a.Internal {
			continue
		}
		// Skip platform-injected side-effect grants that are not in the assignable
		// catalog (e.g. the org-scoped "Business Group Viewer" the platform auto-adds
		// alongside an env-scoped role). The user cannot express these in config —
		// resolvePermissions only accepts catalog names — so surfacing them would show
		// a phantom `name = ""` entry and a perpetual diff. Mirror the internal-skip
		// above and the identical filter in applyPermissions. (Class A invariant.)
		if _, inCatalog := roleIDToName[a.RoleID]; !inCatalog {
			continue
		}
		key := a.RoleID + "|" + canonicalContextParams(a.ContextParams)

		var nameVal types.String
		var cpVal types.Map
		if tp, ok := typedByKey[key]; ok {
			nameVal = tp.name
			cpVal = tp.cp
		} else {
			nameVal = types.StringValue(roleIDToName[a.RoleID])
			cpVal = stringMapToTypesMap(a.ContextParams)
		}

		obj, diags := types.ObjectValue(permissionObjectType.AttrTypes, map[string]attr.Value{
			"name":           nameVal,
			"context_params": cpVal,
		})
		if diags.HasError() {
			return types.SetNull(permissionObjectType), fmt.Errorf("failed to build permission object")
		}
		objs = append(objs, obj)
	}

	set, diags := types.SetValue(permissionObjectType, objs)
	if diags.HasError() {
		return types.SetNull(permissionObjectType), fmt.Errorf("failed to build permissions set")
	}
	return set, nil
}

// resolveMembers resolves each username to a user_id (case-insensitive) using the
// org user list. Returns a map of user_id -> typed username. Returns an error listing
// the offending username if it is unknown. Returns (nil, nil) when unmanaged.
func (r *RoleResource) resolveMembers(ctx context.Context, orgID string, memberSet types.Set) (map[string]string, error) {
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

	out := make(map[string]string)
	for _, el := range memberSet.Elements() {
		typed := el.(types.String).ValueString()
		id, ok := usernameToID[normalizeName(typed)]
		if !ok {
			return nil, fmt.Errorf(
				"member %q is not a valid username in organization %s; use the anypoint_users data source to discover usernames",
				typed, orgID,
			)
		}
		out[id] = typed
	}
	return out, nil
}

// applyMembers reconciles the role group's membership to exactly the desired user IDs.
func (r *RoleResource) applyMembers(ctx context.Context, orgID, roleGroupID string, desiredIDToUsername map[string]string) error {
	actual, err := r.usersClient.ListRoleGroupUsers(ctx, orgID, roleGroupID)
	if err != nil {
		return fmt.Errorf("could not list current role group members: %w", err)
	}
	actualIDs := make(map[string]bool, len(actual))
	for _, u := range actual {
		actualIDs[u.ID] = true
	}

	// Add missing members.
	for id, username := range desiredIDToUsername {
		if actualIDs[id] {
			continue
		}
		if err := r.usersClient.AddUserToRoleGroup(ctx, orgID, roleGroupID, id); err != nil {
			return fmt.Errorf("failed to add member %q: %w", username, err)
		}
	}

	// Remove members that are no longer desired.
	for id := range actualIDs {
		if _, ok := desiredIDToUsername[id]; ok {
			continue
		}
		if err := r.usersClient.RemoveUserFromRoleGroup(ctx, orgID, roleGroupID, id); err != nil {
			return fmt.Errorf("failed to remove member (user_id %s): %w", id, err)
		}
	}

	return nil
}

// reconcileMembersIntoState reads actual role group members and builds a members set,
// preserving the user's typed username casing for matched members.
func (r *RoleResource) reconcileMembersIntoState(ctx context.Context, orgID, roleGroupID string, typedSource types.Set) (types.Set, error) {
	actual, err := r.usersClient.ListRoleGroupUsers(ctx, orgID, roleGroupID)
	if err != nil {
		return types.SetNull(types.StringType), fmt.Errorf("could not list role group members: %w", err)
	}

	typedByLower := map[string]string{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			s := el.(types.String).ValueString()
			typedByLower[normalizeName(s)] = s
		}
	}

	vals := make([]attr.Value, 0, len(actual))
	for _, u := range actual {
		if typed, ok := typedByLower[normalizeName(u.Username)]; ok {
			vals = append(vals, types.StringValue(typed))
		} else {
			vals = append(vals, types.StringValue(u.Username))
		}
	}

	set, diags := types.SetValue(types.StringType, vals)
	if diags.HasError() {
		return types.SetNull(types.StringType), fmt.Errorf("failed to build members set")
	}
	return set, nil
}
