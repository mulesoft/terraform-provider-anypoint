package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

var (
	_ datasource.DataSource              = &PrivateSpacesDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpacesDataSource{}
)

// PrivateSpacesDataSource is the data source implementation.
type PrivateSpacesDataSource struct {
	client *cloudhub2.PrivateSpacesClient
}

// PrivateSpacesDataSourceModel describes the data source data model.
type PrivateSpacesDataSourceModel struct {
	OrganizationID types.String            `tfsdk:"organization_id"`
	PrivateSpaces  []PrivateSpaceListModel `tfsdk:"private_spaces"`
}

// PrivateSpaceListModel represents a single private space entry in the list.
type PrivateSpaceListModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Status             types.String `tfsdk:"status"`
	StatusMessage      types.String `tfsdk:"status_message"`
	Region             types.String `tfsdk:"region"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	RootOrganizationID types.String `tfsdk:"root_organization_id"`
}

func NewPrivateSpacesDataSource() datasource.DataSource {
	return &PrivateSpacesDataSource{}
}

func (d *PrivateSpacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_spaces"
}

func (d *PrivateSpacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Private Spaces in an organization. Use this to discover the " +
			"'id' (private_space_id) needed by resources such as anypoint_transit_gateway_connection " +
			"and anypoint_vpn_connection.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Description: "The organization ID to list Private Spaces for.",
				Required:    true,
			},
			"private_spaces": schema.ListNestedAttribute{
				Description: "The list of Private Spaces in the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the Private Space (use this as private_space_id).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the Private Space.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the Private Space.",
							Computed:    true,
						},
						"status_message": schema.StringAttribute{
							Description: "A human-readable message describing the current status.",
							Computed:    true,
						},
						"region": schema.StringAttribute{
							Description: "The region where the Private Space is provisioned.",
							Computed:    true,
						},
						"organization_id": schema.StringAttribute{
							Description: "The organization ID that owns the Private Space.",
							Computed:    true,
						},
						"root_organization_id": schema.StringAttribute{
							Description: "The root organization ID.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *PrivateSpacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	psClient, err := cloudhub2.NewPrivateSpacesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Spaces Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	d.client = psClient
}

func (d *PrivateSpacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state PrivateSpacesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()

	spaces, err := d.client.ListPrivateSpaces(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private spaces",
			"Could not list private spaces: "+err.Error(),
		)
		return
	}

	state.PrivateSpaces = []PrivateSpaceListModel{}
	for _, ps := range spaces {
		// The LIST endpoint returns region at the TOP LEVEL (per the
		// privateSpaceList data type). Single-space payloads instead nest it
		// under network.region, so fall back to the nested field when the
		// top-level one is empty — this handles both wire shapes.
		region := ps.Region
		if region == "" {
			region = ps.Network.Region
		}
		state.PrivateSpaces = append(state.PrivateSpaces, PrivateSpaceListModel{
			ID:                 types.StringValue(ps.ID),
			Name:               types.StringValue(ps.Name),
			Status:             types.StringValue(ps.Status),
			StatusMessage:      types.StringValue(ps.StatusMessage),
			Region:             types.StringValue(region),
			OrganizationID:     types.StringValue(ps.OrganizationID),
			RootOrganizationID: types.StringValue(ps.RootOrganizationID),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
