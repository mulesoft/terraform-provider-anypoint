package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &PrivateSpaceAssociationDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpaceAssociationDataSource{}
)

// PrivateSpaceAssociationDataSource is the data source implementation.
type PrivateSpaceAssociationDataSource struct {
	client *cloudhub2.PrivateSpaceAssociationClient
}

// PrivateSpaceAssociationDataSourceModel describes the data source data model.
type PrivateSpaceAssociationDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	PrivateSpaceID types.String `tfsdk:"private_space_id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Associations   types.List   `tfsdk:"associations"`
}

func NewPrivateSpaceAssociationDataSource() datasource.DataSource {
	return &PrivateSpaceAssociationDataSource{}
}

// Metadata returns the data source type name.
func (d *PrivateSpaceAssociationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_associations"
}

// Schema defines the schema for the data source.
func (d *PrivateSpaceAssociationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads all private space associations for a given private space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for the data source.",
				Computed:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space to fetch associations for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not provided, the provider's default organization will be used.",
				Optional:    true,
				Computed:    true,
			},
			"associations": schema.ListNestedAttribute{
				Description: "List of associations for the private space.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the association.",
							Computed:    true,
						},
						"organization_id": schema.StringAttribute{
							Description: "The organization ID of the association.",
							Computed:    true,
						},
						"environment": schema.StringAttribute{
							Description: "The environment of the association.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateSpaceAssociationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	privateSpaceAssociationClient, err := cloudhub2.NewPrivateSpaceAssociationClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Association Client",
			"An unexpected error occurred when creating the Private Space Association client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = privateSpaceAssociationClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateSpaceAssociationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateSpaceAssociationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// Get associations from API
	allAssociations, err := d.client.GetPrivateSpaceAssociations(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Private Space Associations",
			"Could not read Private Space Associations for private space ID "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set the data source ID
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString() + "-associations")
	data.PrivateSpaceID = types.StringValue(data.PrivateSpaceID.ValueString())
	// Map associations to the schema
	var associations []attr.Value
	for _, assoc := range allAssociations {
		assocObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"id":              types.StringType,
				"organization_id": types.StringType,
				"environment":     types.StringType,
			},
			map[string]attr.Value{
				"id":              types.StringValue(assoc.ID),
				"organization_id": types.StringValue(assoc.OrganizationID),
				"environment":     types.StringValue(assoc.EnvironmentID),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		associations = append(associations, assocObj)
	}

	associationsList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"organization_id": types.StringType,
				"environment":     types.StringType,
			},
		},
		associations,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Associations = associationsList
	data.OrganizationID = types.StringValue(orgID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
