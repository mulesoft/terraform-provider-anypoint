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
	_ datasource.DataSource              = &UsersDataSource{}
	_ datasource.DataSourceWithConfigure = &UsersDataSource{}
)

// UsersDataSource is the data source implementation.
type UsersDataSource struct {
	client *accessmanagement.RoleUsersClient
}

// UsersDataSourceModel describes the data source data model.
type UsersDataSourceModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	NameFilter     types.String `tfsdk:"name_filter"`
	Users          types.List   `tfsdk:"users"`
}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

// Metadata returns the data source type name.
func (d *UsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *UsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists users in the organization. Use this to look up user IDs by username or email " +
			"instead of hardcoding UUIDs in anypoint_role_users resources.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider's org.",
				Optional:    true,
				Computed:    true,
			},
			"name_filter": schema.StringAttribute{
				Description: "Optional filter to match users by username, email, first name, or last name " +
					"(case-insensitive substring match).",
				Optional: true,
			},
			"users": schema.ListNestedAttribute{
				Description: "List of users in the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique ID of the user. Use this as the user_id in anypoint_role_users.",
							Computed:    true,
						},
						"username": schema.StringAttribute{
							Description: "The username.",
							Computed:    true,
						},
						"first_name": schema.StringAttribute{
							Description: "The first name.",
							Computed:    true,
						},
						"last_name": schema.StringAttribute{
							Description: "The last name.",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "The email address.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the user is enabled.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *UsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"Unable to Create Users Client",
			"An unexpected error occurred when creating the client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = usersClient
}

// Read refreshes the Terraform state with the latest data.
func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// Fetch all users in the org
	users, err := d.client.ListOrgUsers(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading organization users",
			"Could not list users: "+err.Error(),
		)
		return
	}

	// Apply name filter if provided
	nameFilter := data.NameFilter.ValueString()
	var filtered []accessmanagement.OrgUser
	if nameFilter != "" {
		for _, u := range users {
			if containsIgnoreCase(u.Username, nameFilter) ||
				containsIgnoreCase(u.Email, nameFilter) ||
				containsIgnoreCase(u.FirstName, nameFilter) ||
				containsIgnoreCase(u.LastName, nameFilter) {
				filtered = append(filtered, u)
			}
		}
	} else {
		filtered = users
	}

	// Build the list of user objects
	userAttrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"username":   types.StringType,
		"first_name": types.StringType,
		"last_name":  types.StringType,
		"email":      types.StringType,
		"enabled":    types.BoolType,
	}
	userObjType := types.ObjectType{AttrTypes: userAttrTypes}

	userValues := make([]attr.Value, len(filtered))
	for i, u := range filtered {
		obj, diags := types.ObjectValue(userAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(u.ID),
			"username":   types.StringValue(u.Username),
			"first_name": types.StringValue(u.FirstName),
			"last_name":  types.StringValue(u.LastName),
			"email":      types.StringValue(u.Email),
			"enabled":    types.BoolValue(u.Enabled),
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

	data.OrganizationID = types.StringValue(orgID)
	data.Users = usersList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
