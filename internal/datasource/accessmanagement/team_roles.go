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
	_ datasource.DataSource              = &TeamRolesDataSource{}
	_ datasource.DataSourceWithConfigure = &TeamRolesDataSource{}
)

// TeamRolesDataSource is the data source implementation.
type TeamRolesDataSource struct {
	client *accessmanagement.TeamRolesClient
}

// TeamRolesDataSourceModel describes the data source data model.
type TeamRolesDataSourceModel struct {
	TeamID         types.String `tfsdk:"team_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Roles          types.List   `tfsdk:"roles"`
	Total          types.Int64  `tfsdk:"total"`
}

func NewTeamRolesDataSource() datasource.DataSource {
	return &TeamRolesDataSource{}
}

// Metadata returns the data source type name.
func (d *TeamRolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_roles"
}

// Schema defines the schema for the data source.
func (d *TeamRolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all roles (permissions) assigned to a specific team.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Description: "The ID of the team to list roles for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "List of roles (permissions) assigned to the team.",
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
							Description: "The name of the role.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The description of the role.",
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
			"total": schema.Int64Attribute{
				Description: "Total number of roles assigned to the team.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TeamRolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	rolesClient, err := accessmanagement.NewTeamRolesClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Team Roles Client",
			"An unexpected error occurred when creating the Team Roles client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = rolesClient
}

// Read refreshes the Terraform state with the latest data.
func (d *TeamRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamRolesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// List all roles for the team
	roles, err := d.client.ListTeamRoles(ctx, orgID, data.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading team roles",
			"Could not list roles for team "+data.TeamID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Build the list of role objects
	roleAttrTypes := map[string]attr.Type{
		"id":             types.StringType,
		"role_id":        types.StringType,
		"name":           types.StringType,
		"description":    types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
		"created_at":     types.StringType,
	}
	roleObjType := types.ObjectType{AttrTypes: roleAttrTypes}

	roleValues := make([]attr.Value, len(roles))
	for i, r := range roles {
		// Convert context_params to types.Map
		cpElements := make(map[string]attr.Value)
		for k, v := range r.ContextParams {
			cpElements[k] = types.StringValue(v)
		}
		cpMap, diags := types.MapValue(types.StringType, cpElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		obj, diags := types.ObjectValue(roleAttrTypes, map[string]attr.Value{
			"id":             types.StringValue(r.RoleGroupAssignmentID),
			"role_id":        types.StringValue(r.RoleID),
			"name":           types.StringValue(r.Name),
			"description":    types.StringValue(r.Description),
			"context_params": cpMap,
			"created_at":     types.StringValue(r.CreatedAt),
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
	data.Total = types.Int64Value(int64(len(roles)))
	data.OrganizationID = types.StringValue(orgID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
