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
	_ datasource.DataSource              = &RolesDataSource{}
	_ datasource.DataSourceWithConfigure = &RolesDataSource{}
)

// RolesDataSource is the data source implementation for listing all role groups.
type RolesDataSource struct {
	client *accessmanagement.RoleClient
}

// RolesDataSourceModel describes the data source data model.
type RolesDataSourceModel struct {
	OrganizationID types.String `tfsdk:"organization_id"`
	Roles          types.List   `tfsdk:"roles"`
}

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

// Metadata returns the data source type name.
func (d *RolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

// Schema defines the schema for the data source.
func (d *RolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all role groups in an Anypoint Platform organization.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "The list of role groups.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier for the role group.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the role group.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of the role group.",
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
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the role group was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the role group was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *RolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = roleClient
}

// Read refreshes the Terraform state with the latest data.
func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel

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

	// List all role groups from the API
	roleGroups, err := d.client.ListRoleGroups(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing role groups",
			"Could not list role groups: "+err.Error(),
		)
		return
	}

	// Define the object type for role entries
	roleObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":             types.StringType,
			"name":           types.StringType,
			"description":    types.StringType,
			"editable":       types.BoolType,
			"external_names": types.ListType{ElemType: types.StringType},
			"created_at":     types.StringType,
			"updated_at":     types.StringType,
		},
	}

	// Map role groups to state
	roleElements := make([]attr.Value, len(roleGroups))
	for i, rg := range roleGroups {
		// Build external_names list
		var externalNamesList types.List
		if len(rg.ExternalNames) > 0 {
			elems := make([]attr.Value, len(rg.ExternalNames))
			for j, name := range rg.ExternalNames {
				elems[j] = types.StringValue(name)
			}
			externalNamesList = types.ListValueMust(types.StringType, elems)
		} else {
			externalNamesList = types.ListValueMust(types.StringType, []attr.Value{})
		}

		roleElements[i], _ = types.ObjectValue(
			roleObjectType.AttrTypes,
			map[string]attr.Value{
				"id":             types.StringValue(rg.ID),
				"name":           types.StringValue(rg.Name),
				"description":    types.StringValue(rg.Description),
				"editable":       types.BoolValue(rg.Editable),
				"external_names": externalNamesList,
				"created_at":     types.StringValue(rg.CreatedAt),
				"updated_at":     types.StringValue(rg.UpdatedAt),
			},
		)
	}

	data.OrganizationID = types.StringValue(orgID)
	data.Roles = types.ListValueMust(roleObjectType, roleElements)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
