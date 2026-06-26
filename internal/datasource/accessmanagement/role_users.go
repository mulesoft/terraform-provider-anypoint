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
	_ datasource.DataSource              = &RoleUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &RoleUsersDataSource{}
)

// RoleUsersDataSource is the data source implementation.
type RoleUsersDataSource struct {
	client *accessmanagement.RoleUsersClient
}

// RoleUsersDataSourceModel describes the data source data model.
type RoleUsersDataSourceModel struct {
	RoleGroupID    types.String `tfsdk:"role_group_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Users          types.List   `tfsdk:"users"`
	Total          types.Int64  `tfsdk:"total"`
}

func NewRoleUsersDataSource() datasource.DataSource {
	return &RoleUsersDataSource{}
}

// Metadata returns the data source type name.
func (d *RoleUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_users"
}

// Schema defines the schema for the data source.
func (d *RoleUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all users assigned to a specific role group.",
		Attributes: map[string]schema.Attribute{
			"role_group_id": schema.StringAttribute{
				Description: "The ID of the role group to list users for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"users": schema.ListNestedAttribute{
				Description: "List of users assigned to the role group.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The user ID.",
							Computed:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username.",
							Computed:    true,
						},
						"first_name": schema.StringAttribute{
							Description: "The user's first name.",
							Computed:    true,
						},
						"last_name": schema.StringAttribute{
							Description: "The user's last name.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The user's email address.",
							Computed:    true,
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				Description: "Total number of users assigned to the role group.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *RoleUsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	usersClient, err := accessmanagement.NewRoleUsersClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Role Users Client",
			"An unexpected error occurred when creating the Role Users client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = usersClient
}

// Read refreshes the Terraform state with the latest data.
func (d *RoleUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleUsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// List all users for the role group
	users, err := d.client.ListRoleGroupUsers(ctx, orgID, data.RoleGroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading role group users",
			"Could not list users for role group "+data.RoleGroupID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Build the list of user objects
	userAttrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"username":   types.StringType,
		"first_name": types.StringType,
		"last_name":  types.StringType,
		"email":      types.StringType,
	}
	userObjType := types.ObjectType{AttrTypes: userAttrTypes}

	userValues := make([]attr.Value, len(users))
	for i, u := range users {
		obj, diags := types.ObjectValue(userAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(u.ID),
			"username":   types.StringValue(u.Username),
			"first_name": types.StringValue(u.FirstName),
			"last_name":  types.StringValue(u.LastName),
			"email":      types.StringValue(u.Email),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		userValues[i] = obj
	}

	usersList, diags := types.ListValue(userObjType, userValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Users = usersList
	data.Total = types.Int64Value(int64(len(users)))
	data.OrganizationID = types.StringValue(orgID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
