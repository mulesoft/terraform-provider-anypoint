package agentstools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ datasource.DataSource              = &MCPServerDataSource{}
	_ datasource.DataSourceWithConfigure = &MCPServerDataSource{}
)

// MCPServerDataSource lists all MCP servers in an environment.
type MCPServerDataSource struct {
	client *agentstools.MCPServerClient
}

type MCPServerDataSourceModel struct {
	ID             types.String         `tfsdk:"id"`
	OrganizationID types.String         `tfsdk:"organization_id"`
	EnvironmentID  types.String         `tfsdk:"environment_id"`
	Servers        []MCPServerItemModel `tfsdk:"servers"`
}

type MCPServerItemModel struct {
	ID                        types.String `tfsdk:"id"`
	AssetID                   types.String `tfsdk:"asset_id"`
	AssetVersion              types.String `tfsdk:"asset_version"`
	ProductVersion            types.String `tfsdk:"product_version"`
	GroupID                   types.String `tfsdk:"group_id"`
	Technology                types.String `tfsdk:"technology"`
	InstanceLabel             types.String `tfsdk:"instance_label"`
	Status                    types.String `tfsdk:"status"`
	EndpointURI               types.String `tfsdk:"endpoint_uri"`
	ProxyURI                  types.String `tfsdk:"proxy_uri"`
	AutodiscoveryInstanceName types.String `tfsdk:"autodiscovery_instance_name"`
}

func NewMCPServerDataSource() datasource.DataSource {
	return &MCPServerDataSource{}
}

func (d *MCPServerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_servers"
}

func (d *MCPServerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all MCP servers registered in API Manager for the given environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier: <organization_id>/<environment_id>.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider credentials organization.",
				Optional:    true,
				Computed:    true,
			},
			"environment_id": schema.StringAttribute{
				Description: "The environment ID to list MCP servers from.",
				Required:    true,
			},
			"servers": schema.ListNestedAttribute{
				Description: "List of MCP servers.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The numeric ID of the MCP server.",
							Computed:    true,
						},
						"asset_id": schema.StringAttribute{
							Description: "The Exchange asset ID.",
							Computed:    true,
						},
						"asset_version": schema.StringAttribute{
							Description: "The Exchange asset version.",
							Computed:    true,
						},
						"product_version": schema.StringAttribute{
							Description: "The product version.",
							Computed:    true,
						},
						"group_id": schema.StringAttribute{
							Description: "The Exchange group (organization) ID.",
							Computed:    true,
						},
						"technology": schema.StringAttribute{
							Description: "The gateway technology (typically flexGateway for MCP).",
							Computed:    true,
						},
						"instance_label": schema.StringAttribute{
							Description: "The label of the MCP server.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the MCP server.",
							Computed:    true,
						},
						"endpoint_uri": schema.StringAttribute{
							Description: "The endpoint URI for the MCP server.",
							Computed:    true,
						},
						"proxy_uri": schema.StringAttribute{
							Description: "The MCP proxy URI (e.g., http://0.0.0.0:8081/mcp1).",
							Computed:    true,
						},
						"autodiscovery_instance_name": schema.StringAttribute{
							Description: "The autodiscovery instance name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *MCPServerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	mcpClient, err := agentstools.NewMCPServerClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create MCP Server Client",
			"An unexpected error occurred when creating the MCP Server client.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = mcpClient
}

func (d *MCPServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MCPServerDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	servers, err := d.client.ListMCPServers(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing MCP servers",
			"Could not list MCP servers for environment "+envID+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(orgID + "/" + envID)
	data.OrganizationID = types.StringValue(orgID)
	data.Servers = make([]MCPServerItemModel, 0, len(servers))

	for _, server := range servers {
		data.Servers = append(data.Servers, mapMCPServerToItemModel(server))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapMCPServerToItemModel converts a client MCPServer to the datasource item model.
func mapMCPServerToItemModel(server agentstools.MCPServer) MCPServerItemModel {
	endpointURI := types.StringNull()
	if server.EndpointURI != "" {
		endpointURI = types.StringValue(server.EndpointURI)
	}

	proxyURI := types.StringNull()
	if server.Endpoint != nil && server.Endpoint.ProxyURI != nil && *server.Endpoint.ProxyURI != "" {
		proxyURI = types.StringValue(*server.Endpoint.ProxyURI)
	}

	autodiscovery := types.StringNull()
	if server.AutodiscoveryInstanceName != "" {
		autodiscovery = types.StringValue(server.AutodiscoveryInstanceName)
	}

	instanceLabel := types.StringNull()
	if server.InstanceLabel != "" {
		instanceLabel = types.StringValue(server.InstanceLabel)
	}

	return MCPServerItemModel{
		ID:                        types.StringValue(strconv.Itoa(server.ID)),
		AssetID:                   types.StringValue(server.AssetID),
		AssetVersion:              types.StringValue(server.AssetVersion),
		ProductVersion:            types.StringValue(server.ProductVersion),
		GroupID:                   types.StringValue(server.GroupID),
		Technology:                types.StringValue(server.Technology),
		InstanceLabel:             instanceLabel,
		Status:                    types.StringValue(server.Status),
		EndpointURI:               endpointURI,
		ProxyURI:                  proxyURI,
		AutodiscoveryInstanceName: autodiscovery,
	}
}
