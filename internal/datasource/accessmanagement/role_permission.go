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
	_ datasource.DataSource              = &RolePermissionsDataSource{}
	_ datasource.DataSourceWithConfigure = &RolePermissionsDataSource{}
)

// RolePermissionsDataSource is the data source implementation.
type RolePermissionsDataSource struct {
	client *accessmanagement.RolePermissionClient
}

// RolePermissionsDataSourceModel describes the data source data model.
type RolePermissionsDataSourceModel struct {
	RoleGroupID    types.String `tfsdk:"role_group_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Permissions    types.List   `tfsdk:"permissions"`
}

func NewRolePermissionsDataSource() datasource.DataSource {
	return &RolePermissionsDataSource{}
}

// Metadata returns the data source type name.
func (d *RolePermissionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_permissions"
}

// Schema defines the schema for the data source.
func (d *RolePermissionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all permissions (roles) assigned to a specific role group.",
		Attributes: map[string]schema.Attribute{
			"role_group_id": schema.StringAttribute{
				Description: "The ID of the role group to list permissions for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"permissions": schema.ListNestedAttribute{
				Description: "List of permissions (roles) assigned to the role group.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The role_group_assignment_id.",
							Computed:    true,
						},
						"role_id": schema.StringAttribute{
							Description: "The role (permission) ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the role (permission).",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the role (permission).",
							Computed:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters (org, envId).",
							Computed:    true,
							ElementType: types.StringType,
						},
						"created_at": schema.StringAttribute{
							Description: "When the assignment was created.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *RolePermissionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = permClient
}

// Read refreshes the Terraform state with the latest data.
func (d *RolePermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolePermissionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// List all role assignments for the role group
	assignments, err := d.client.ListRoleAssignments(ctx, orgID, data.RoleGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading role permissions",
			"Could not list role permissions for role group "+data.RoleGroupID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Build the list of permission objects
	permissionAttrTypes := map[string]attr.Type{
		"id":             types.StringType,
		"role_id":        types.StringType,
		"name":           types.StringType,
		"description":    types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
		"created_at":     types.StringType,
	}
	permissionObjType := types.ObjectType{AttrTypes: permissionAttrTypes}

	permissionValues := make([]attr.Value, len(assignments))
	for i, a := range assignments {
		// Convert context_params to types.Map
		cpElements := make(map[string]attr.Value)
		for k, v := range a.ContextParams {
			cpElements[k] = types.StringValue(v)
		}
		cpMap, diags := types.MapValue(types.StringType, cpElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		obj, diags := types.ObjectValue(permissionAttrTypes, map[string]attr.Value{
			"id":             types.StringValue(a.RoleGroupAssignmentID),
			"role_id":        types.StringValue(a.RoleID),
			"name":           types.StringValue(a.Name),
			"description":    types.StringValue(a.Description),
			"context_params": cpMap,
			"created_at":     types.StringValue(a.CreatedAt),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		permissionValues[i] = obj
	}

	permissionsList, diags := types.ListValue(permissionObjType, permissionValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Permissions = permissionsList
	data.OrganizationID = types.StringValue(orgID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
