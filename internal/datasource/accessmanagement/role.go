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
	client *accessmanagement.RoleClient
}

// RoleDataSourceModel describes the data source data model.
type RoleDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Editable       types.Bool   `tfsdk:"editable"`
	ExternalNames  types.List   `tfsdk:"external_names"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
