package accessmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &RolePermissionResource{}
	_ resource.ResourceWithConfigure   = &RolePermissionResource{}
	_ resource.ResourceWithImportState = &RolePermissionResource{}
)

// RolePermissionResource is the resource implementation.
type RolePermissionResource struct {
	client *accessmanagement.RolePermissionClient
}

// RolePermissionResourceModel describes the resource data model.
type RolePermissionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	RoleGroupID    types.String `tfsdk:"role_group_id"`
	RoleID         types.String `tfsdk:"role_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ContextParams  types.Map    `tfsdk:"context_params"`
	RoleName       types.String `tfsdk:"role_name"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func NewRolePermissionResource() resource.Resource {
	return &RolePermissionResource{}
}

// Metadata returns the resource type name.
func (r *RolePermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_permission"
}

// Schema defines the schema for the resource.
func (r *RolePermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a permission (role) to an Anypoint role group. " +
			"Each instance represents a single role assignment. " +
			"Changing any identifier attribute forces recreation (destroy + create). " +
			"Requires Organization Administrator privileges.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique role_group_assignment_id for this assignment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_group_id": schema.StringAttribute{
				Description: "The ID of the role group to assign the permission to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Description: "The ID of the role (permission) to assign. Use data.anypoint_roles_available to discover available role IDs.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"context_params": schema.MapAttribute{
				Description: "Context parameters for the assignment. Must include 'org' (organization ID). " +
					"For environment-scoped permissions, also include 'envId'.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				Description: "The name of the assigned role (permission). Populated from the API on read.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the assignment was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *RolePermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	permClient, err := accessmanagement.NewRolePermissionClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Permission Client",
			"An unexpected error occurred when creating the Role Permission client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = permClient
}

// Create assigns a role to a role group.
func (r *RolePermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RolePermissionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Extract context_params from the map
	contextParams := r.extractContextParams(ctx, data.ContextParams)

	// Build the request
	assignReq := &accessmanagement.AssignRoleRequest{
		RoleID:        data.RoleID.ValueString(),
		ContextParams: contextParams,
	}

	assignment, err := r.client.AssignRole(ctx, orgID, data.RoleGroupID.ValueString(), assignReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning role to role group",
			"Could not assign role: "+err.Error(),
		)
		return
	}

	// Set state from response
	data.ID = types.StringValue(assignment.RoleGroupAssignmentID)
	data.OrganizationID = types.StringValue(orgID)

	// Do a read to get the full assignment details (name, created_at)
	fullAssignment, err := r.client.GetRoleAssignmentByID(ctx, orgID, data.RoleGroupID.ValueString(), assignment.RoleGroupAssignmentID)
	if err != nil {
		// Non-fatal: we have the ID, just set what we can
		tflog.Warn(ctx, "Could not read back assignment details after create", map[string]interface{}{
			"error": err.Error(),
		})
		data.RoleName = types.StringValue("")
		data.CreatedAt = types.StringValue("")
	} else {
		data.RoleName = types.StringValue(fullAssignment.Name)
		data.CreatedAt = types.StringValue(fullAssignment.CreatedAt)
	}

	tflog.Trace(ctx, "assigned role to role group", map[string]interface{}{
		"role_group_id":    data.RoleGroupID.ValueString(),
		"role_id":          data.RoleID.ValueString(),
		"assignment_id":    assignment.RoleGroupAssignmentID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *RolePermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RolePermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	var assignment *accessmanagement.RoleAssignment
	var err error

	// If role_id is set, look up by role_id + context_params (normal case).
	// If role_id is empty (import case), look up by assignment_id.
	if data.RoleID.ValueString() != "" && !data.ContextParams.IsNull() && !data.ContextParams.IsUnknown() {
		contextParams := r.extractContextParams(ctx, data.ContextParams)
		assignment, err = r.client.GetRoleAssignment(ctx, orgID, data.RoleGroupID.ValueString(), data.RoleID.ValueString(), contextParams)
	} else {
		// Import path: we only have role_group_id + assignment_id
		assignment, err = r.client.GetRoleAssignmentByID(ctx, orgID, data.RoleGroupID.ValueString(), data.ID.ValueString())
	}

	if err != nil {
		if client.IsNotFound(err) {
			// Assignment was removed outside of Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading role assignment",
			"Could not read role assignment: "+err.Error(),
		)
		return
	}

	// Update state with API response
	data.ID = types.StringValue(assignment.RoleGroupAssignmentID)
	data.RoleID = types.StringValue(assignment.RoleID)
	data.RoleName = types.StringValue(assignment.Name)
	data.CreatedAt = types.StringValue(assignment.CreatedAt)
	data.OrganizationID = types.StringValue(orgID)

	// Populate context_params from the API response
	if len(assignment.ContextParams) > 0 {
		elements := make(map[string]types.String, len(assignment.ContextParams))
		for k, v := range assignment.ContextParams {
			elements[k] = types.StringValue(v)
		}
		// Convert map[string]types.String to map[string]attr.Value for MapValueFrom
		attrElements := make(map[string]attr.Value, len(elements))
		for k, v := range elements {
			attrElements[k] = v
		}
		data.ContextParams = types.MapValueMust(types.StringType, attrElements)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is a no-op because all attributes force recreation.
func (r *RolePermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes have RequiresReplace, so this should never be called.
	// If it is called, just re-read the state.
	var data RolePermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete unassigns the role from the role group.
func (r *RolePermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RolePermissionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Build the unassign request
	contextParams := r.extractContextParams(ctx, data.ContextParams)
	unassignReq := &accessmanagement.AssignRoleRequest{
		RoleID:        data.RoleID.ValueString(),
		ContextParams: contextParams,
	}

	err := r.client.UnassignRole(ctx, orgID, data.RoleGroupID.ValueString(), unassignReq)
	if err != nil {
		if client.IsNotFound(err) {
			// Already removed, nothing to do
			return
		}
		resp.Diagnostics.AddError(
			"Error unassigning role from role group",
			"Could not unassign role: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "unassigned role from role group")
}

// ImportState imports a role permission assignment by composite ID.
// Format: {role_group_id}:{role_group_assignment_id}
//
// After import, Terraform will call Read which looks up the assignment by
// role_id + context_params. Since we only have the assignment_id at import time,
// we set it as the resource ID and let Read populate the full state.
func (r *RolePermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: {role_group_id}:{role_group_assignment_id}, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_group_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// extractContextParams converts a types.Map to map[string]string
func (r *RolePermissionResource) extractContextParams(ctx context.Context, m types.Map) map[string]string {
	result := make(map[string]string)
	if m.IsNull() || m.IsUnknown() {
		return result
	}
	elements := m.Elements()
	for k, v := range elements {
		if strVal, ok := v.(types.String); ok {
			result[k] = strVal.ValueString()
		}
	}
	return result
}
