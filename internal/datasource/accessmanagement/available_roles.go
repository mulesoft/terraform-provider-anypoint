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
	_ datasource.DataSource              = &AvailableRolesDataSource{}
	_ datasource.DataSourceWithConfigure = &AvailableRolesDataSource{}
)

// AvailableRolesDataSource is the data source implementation.
type AvailableRolesDataSource struct {
	client *accessmanagement.RolePermissionClient
}

// AvailableRolesDataSourceModel describes the data source data model.
type AvailableRolesDataSourceModel struct {
	NameFilter types.String `tfsdk:"name_filter"`
	Roles      types.List   `tfsdk:"roles"`
}

func NewAvailableRolesDataSource() datasource.DataSource {
	return &AvailableRolesDataSource{}
}

// Metadata returns the data source type name.
func (d *AvailableRolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_roles"
}

// Schema defines the schema for the data source.
func (d *AvailableRolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all available roles (permissions) that can be assigned to role groups. " +
			"Use this to look up role IDs by name instead of hardcoding UUIDs.",
		Attributes: map[string]schema.Attribute{
			"name_filter": schema.StringAttribute{
				Description: "Optional filter to match roles by name (case-insensitive substring match). " +
					"For example, 'Read Applications' returns only roles with that name.",
				Optional: true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "List of available roles (permissions).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_id": schema.StringAttribute{
							Description: "The unique ID of the role. Permissions are referenced by name in the permissions block of anypoint_role, so this ID is informational.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The human-readable name of the role.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of what the role grants.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *AvailableRolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"An unexpected error occurred when creating the client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = permClient
}

// Read refreshes the Terraform state with the latest data.
func (d *AvailableRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AvailableRolesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all available roles
	roles, err := d.client.ListAvailableRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading available roles",
			"Could not list available roles: "+err.Error(),
		)
		return
	}

	// Apply name filter if provided
	nameFilter := data.NameFilter.ValueString()
	var filtered []accessmanagement.AvailableRole
	if nameFilter != "" {
		for _, r := range roles {
			if containsIgnoreCase(r.Name, nameFilter) {
				filtered = append(filtered, r)
			}
		}
	} else {
		filtered = roles
	}

	// Build the list of role objects
	roleAttrTypes := map[string]attr.Type{
		"role_id":     types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
	}
	roleObjType := types.ObjectType{AttrTypes: roleAttrTypes}

	roleValues := make([]attr.Value, len(filtered))
	for i, r := range filtered {
		obj, diags := types.ObjectValue(roleAttrTypes, map[string]attr.Value{
			"role_id":     types.StringValue(r.RoleID),
			"name":        types.StringValue(r.Name),
			"description": types.StringValue(r.Description),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		roleValues[i] = obj
	}

	rolesList, diags := types.ListValue(roleObjType, roleValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Roles = rolesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
