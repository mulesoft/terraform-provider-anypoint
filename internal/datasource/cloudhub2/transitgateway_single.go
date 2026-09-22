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
	_ datasource.DataSource              = &TransitGatewaySingleDataSource{}
	_ datasource.DataSourceWithConfigure = &TransitGatewaySingleDataSource{}
)

// TransitGatewaySingleDataSource is the SINGULAR transit gateway connection data
// source. It is the read-only twin of the anypoint_transit_gateway_connection
// resource: it looks up ONE connection by id and surfaces the full attribute set,
// including fields the plural anypoint_transit_gateway_connections data source does
// NOT expose — aws_transit_gateway_id, resource_share_id/account, region, and
// attachment. Use it to reference an existing connection (e.g. to read its live
// routes or the AWS TGW id the platform discovered) from other configuration.
type TransitGatewaySingleDataSource struct {
	client *cloudhub2.TransitGatewayClient
}

// TransitGatewaySingleDataSourceModel describes the singular data source data model.
// It mirrors the resource's attribute set (routes as a plain list of CIDR strings)
// and adds region + attachment, which are computed on the platform side.
type TransitGatewaySingleDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	PrivateSpaceID       types.String `tfsdk:"private_space_id"`
	Name                 types.String `tfsdk:"name"`
	AwsTransitGatewayID  types.String `tfsdk:"aws_transit_gateway_id"`
	AwsConsoleURL        types.String `tfsdk:"aws_console_url"`
	ResourceShareID      types.String `tfsdk:"resource_share_id"`
	ResourceShareAccount types.String `tfsdk:"resource_share_account"`
	Region               types.String `tfsdk:"region"`
	Status               types.String `tfsdk:"status"`
	Attachment           types.String `tfsdk:"attachment"`
	Routes               types.List   `tfsdk:"routes"`
}

func NewTransitGatewaySingleDataSource() datasource.DataSource {
	return &TransitGatewaySingleDataSource{}
}

func (d *TransitGatewaySingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transit_gateway_connection"
}

func (d *TransitGatewaySingleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single Transit Gateway connection (attachment) in a CloudHub 2.0 " +
			"Private Space by its ID. Surfaces the full attribute set — including the AWS Transit " +
			"Gateway ID, resource share, region, attachment state, and routes — which the plural " +
			"anypoint_transit_gateway_connections data source does not expose.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the transit gateway connection to fetch.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID.",
				Required:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the Private Space the transit gateway connection belongs to.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the transit gateway connection.",
				Computed:    true,
			},
			"aws_transit_gateway_id": schema.StringAttribute{
				Description: "The AWS Transit Gateway ID the platform discovered from the resource share, " +
					"as a bare `tgw-...` identifier suitable for passing to the AWS provider.",
				Computed: true,
			},
			"aws_console_url": schema.StringAttribute{
				Description: "Deep link to this transit gateway in the AWS console, as shown by the " +
					"Anypoint UI's \"View on AWS\" link. Empty when the platform does not supply one. " +
					"Use `aws_transit_gateway_id` for the identifier itself.",
				Computed: true,
			},
			"resource_share_id": schema.StringAttribute{
				Description: "The AWS RAM resource share ID (UUID format).",
				Computed:    true,
			},
			"resource_share_account": schema.StringAttribute{
				Description: "The AWS account ID that owns the Transit Gateway.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The AWS region of the transit gateway connection.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current gateway status (e.g. 'Pending', 'Available').",
				Computed:    true,
			},
			"attachment": schema.StringAttribute{
				Description: "The current attachment status of the transit gateway connection.",
				Computed:    true,
			},
			"routes": schema.ListAttribute{
				Description: "The CIDR routes configured on this transit gateway connection.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *TransitGatewaySingleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TransitGatewaySingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TransitGatewaySingleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	psID := state.PrivateSpaceID.ValueString()
	tgwID := state.ID.ValueString()

	tgw, err := d.client.GetTransitGateway(ctx, orgID, psID, tgwID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				"Transit gateway connection not found",
				fmt.Sprintf("No transit gateway connection with id %q exists in Private Space %q.", tgwID, psID),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading transit gateway connection",
			"Could not read transit gateway connection: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(tgw.Name)
	state.AwsTransitGatewayID = types.StringValue(tgw.Status.AWSTransitGatewayID())
	state.AwsConsoleURL = types.StringValue(tgw.Status.AWSConsoleURL())
	state.ResourceShareID = types.StringValue(tgw.Spec.ResourceShare.ID)
	state.ResourceShareAccount = types.StringValue(tgw.Spec.ResourceShare.Account)
	state.Region = types.StringValue(tgw.Spec.Region)
	state.Status = types.StringValue(tgw.Status.Gateway)
	state.Attachment = types.StringValue(tgw.Status.Attachment)

	routes := tgw.Status.Routes
	if routes == nil {
		routes = []string{}
	}
	routesList, diags := types.ListValueFrom(ctx, types.StringType, routes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Routes = routesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
