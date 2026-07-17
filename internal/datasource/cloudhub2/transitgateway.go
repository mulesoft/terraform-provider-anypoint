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
	_ datasource.DataSource              = &TransitGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &TransitGatewayDataSource{}
)

// TransitGatewayDataSource is the data source implementation.
type TransitGatewayDataSource struct {
	client *cloudhub2.TransitGatewayClient
}

// TransitGatewayDataSourceModel describes the data source data model.
type TransitGatewayDataSourceModel struct {
	PrivateSpaceID            types.String              `tfsdk:"private_space_id"`
	OrganizationID            types.String              `tfsdk:"organization_id"`
	TransitGatewayConnections []TransitGatewayListModel `tfsdk:"transit_gateway_connections"`
}

// TransitGatewayListModel represents a single transit gateway entry.
type TransitGatewayListModel struct {
	ID     types.String               `tfsdk:"id"`
	Name   types.String               `tfsdk:"name"`
	Status types.String               `tfsdk:"status"`
	Routes []TransitGatewayRouteModel `tfsdk:"routes"`
}

// TransitGatewayRouteModel represents a route on a transit gateway.
type TransitGatewayRouteModel struct {
	CIDR types.String `tfsdk:"cidr"`
}

func NewTransitGatewayDataSource() datasource.DataSource {
	return &TransitGatewayDataSource{}
}

func (d *TransitGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transit_gateway_connections"
}

func (d *TransitGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all transit gateway connections (attachments) in a CloudHub 2.0 Private Space, including their routes.",
		Attributes: map[string]schema.Attribute{
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the Private Space to list transit gateway connections for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
			},
			"transit_gateway_connections": schema.ListNestedAttribute{
				Description: "The list of transit gateway connections (attachments).",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the transit gateway.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the transit gateway attachment.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status (e.g. 'Pending', 'Available').",
							Computed:    true,
						},
						"routes": schema.ListNestedAttribute{
							Description: "The static routes configured on this transit gateway.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"cidr": schema.StringAttribute{
										Description: "The CIDR block of the route.",
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *TransitGatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	tgwClient, err := cloudhub2.NewTransitGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Transit Gateway Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	d.client = tgwClient
}

func (d *TransitGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TransitGatewayDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	psID := state.PrivateSpaceID.ValueString()

	tgws, err := d.client.ListTransitGateways(ctx, orgID, psID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading transit gateways",
			"Could not list transit gateways: "+err.Error(),
		)
		return
	}

	state.TransitGatewayConnections = []TransitGatewayListModel{}
	for _, tgw := range tgws {
		tgwModel := TransitGatewayListModel{
			ID:     types.StringValue(tgw.ID),
			Name:   types.StringValue(tgw.Name),
			Status: types.StringValue(tgw.Status.Gateway),
			Routes: []TransitGatewayRouteModel{},
		}

		// Routes are already in the list response's status.routes field
		for _, route := range tgw.Status.Routes {
			tgwModel.Routes = append(tgwModel.Routes, TransitGatewayRouteModel{
				CIDR: types.StringValue(route),
			})
		}

		state.TransitGatewayConnections = append(state.TransitGatewayConnections, tgwModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
