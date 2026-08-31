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
	_ datasource.DataSource              = &RoleDataSource{}
	_ datasource.DataSourceWithConfigure = &RoleDataSource{}
)

// RoleDataSource is the data source implementation.
type RoleDataSource struct {
	client      *accessmanagement.RoleClient
	permClient  *accessmanagement.RolePermissionClient
	usersClient *accessmanagement.RoleUsersClient
}

// RoleDataSourceModel describes the data source data model.
type RoleDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Editable       types.Bool   `tfsdk:"editable"`
	ExternalNames  types.List   `tfsdk:"external_names"`
	Permissions    types.List   `tfsdk:"permissions"`
	Members        types.List   `tfsdk:"members"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// roleDSPermissionObjectType is the object type for a permission entry in the data source output.
var roleDSPermissionObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":           types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

// Metadata returns the data source type name.
func (d *RoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the data source.
func (d *RoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an Anypoint Platform role group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the role group.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the role group.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the role group.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the role group is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"editable": schema.BoolAttribute{
				Description: "Whether the role group can be edited.",
				Computed:    true,
			},
			"external_names": schema.ListAttribute{
				Description: "External group names mapped to this role group.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"permissions": schema.ListNestedAttribute{
				Description: "The permissions (role assignments) granted by this role group. Excludes system/internal assignments.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The permission's display name.",
							Computed:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the permission (e.g., org, envId).",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"members": schema.ListAttribute{
				Description: "The usernames of members in this role group.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the role group was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the role group was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *RoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	roleClient, err := accessmanagement.NewRoleClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anypoint Role Group API Client",
			"An unexpected error occurred when creating the Anypoint Role Group API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	permClient, err := accessmanagement.NewRolePermissionClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Permission Client",
			"An unexpected error occurred when creating the Role Permission client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	usersClient, err := accessmanagement.NewRoleUsersClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Users Client",
			"An unexpected error occurred when creating the Role Users client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = roleClient
	d.permClient = permClient
	d.usersClient = usersClient
}

// Read refreshes the Terraform state with the latest data.
func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// Get the role group from the API
	roleGroup, err := d.client.GetRoleGroup(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading role group",
			"Could not read role group ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to state
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

	// Populate permissions (excluding internal/system assignments), labeled by display name.
	assignments, err := d.permClient.ListRoleAssignments(ctx, orgID, roleGroup.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading role permissions", "Could not list role assignments: "+err.Error())
		return
	}
	roles, err := d.permClient.ListAvailableRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading available roles", "Could not list available roles: "+err.Error())
		return
	}
	roleIDToName := make(map[string]string, len(roles))
	for _, role := range roles {
		roleIDToName[role.RoleID] = role.Name
	}
	permValues := make([]attr.Value, 0, len(assignments))
	for _, a := range assignments {
		if a.Internal {
			continue
		}
		// Skip platform-injected side-effect grants whose role_id is not in the
		// assignable catalog (e.g. the org-scoped "Business Group Viewer" the platform
		// auto-adds alongside an env-scoped role). Mirroring the anypoint_role
		// resource's reconcile, the data source must not surface them — otherwise they
		// appear with an empty name and produce a perpetual output diff. (Class A.)
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
		obj, diags := types.ObjectValue(roleDSPermissionObjectType.AttrTypes, map[string]attr.Value{
			"name":           types.StringValue(roleIDToName[a.RoleID]),
			"context_params": cp,
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		permValues = append(permValues, obj)
	}
	data.Permissions = types.ListValueMust(roleDSPermissionObjectType, permValues)

	// Populate members (usernames).
	members, err := d.usersClient.ListRoleGroupUsers(ctx, orgID, roleGroup.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading role members", "Could not list role group members: "+err.Error())
		return
	}
	memberValues := make([]attr.Value, 0, len(members))
	for _, u := range members {
		memberValues = append(memberValues, types.StringValue(u.Username))
	}
	data.Members = types.ListValueMust(types.StringType, memberValues)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
