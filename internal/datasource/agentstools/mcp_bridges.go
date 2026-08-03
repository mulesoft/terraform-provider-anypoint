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
	_ datasource.DataSource              = &MCPBridgesDataSource{}
	_ datasource.DataSourceWithConfigure = &MCPBridgesDataSource{}
)

// MCPBridgesDataSource lists all MCP bridges in an environment. A bridge is an API
// Manager instance tagged metadata.generatedBy=mcp_bridge (the client filters these out
// of the shared instances endpoint), so this only returns bridges — never plain MCP
// servers or other APIs.
type MCPBridgesDataSource struct {
	client *agentstools.MCPBridgeClient
}

type MCPBridgesDataSourceModel struct {
	ID             types.String         `tfsdk:"id"`
	OrganizationID types.String         `tfsdk:"organization_id"`
	EnvironmentID  types.String         `tfsdk:"environment_id"`
	Bridges        []MCPBridgeItemModel `tfsdk:"bridges"`
}

type MCPBridgeItemModel struct {
	ID             types.String `tfsdk:"id"`
	AssetID        types.String `tfsdk:"asset_id"`
	AssetVersion   types.String `tfsdk:"asset_version"`
	ProductVersion types.String `tfsdk:"product_version"`
	GroupID        types.String `tfsdk:"group_id"`
	Technology     types.String `tfsdk:"technology"`
	InstanceLabel  types.String `tfsdk:"instance_label"`
	Status         types.String `tfsdk:"status"`
	EndpointURI    types.String `tfsdk:"endpoint_uri"`
	ProxyURI       types.String `tfsdk:"proxy_uri"`
}

func NewMCPBridgesDataSource() datasource.DataSource {
	return &MCPBridgesDataSource{}
}

func (d *MCPBridgesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_bridges"
}

func (d *MCPBridgesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all MCP bridges (metadata.generatedBy=mcp_bridge) registered in API Manager for the given environment.",
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
				Description: "The environment ID to list MCP bridges from.",
				Required:    true,
			},
			"bridges": schema.ListNestedAttribute{
				Description: "List of MCP bridges.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The numeric ID of the MCP bridge (API Manager instance ID).",
							Computed:    true,
						},
						"asset_id": schema.StringAttribute{
							Description: "The generated Exchange asset ID backing the bridge.",
							Computed:    true,
						},
						"asset_version": schema.StringAttribute{
							Description: "The generated Exchange asset version.",
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
							Description: "The gateway technology (flexGateway for MCP bridges).",
							Computed:    true,
						},
						"instance_label": schema.StringAttribute{
							Description: "The label of the MCP bridge.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The current status of the MCP bridge.",
							Computed:    true,
						},
						"endpoint_uri": schema.StringAttribute{
							Description: "The consumer-facing endpoint URI (endpointUri); may be null for self-managed (flexGateway) bridges — use proxy_uri instead.",
							Computed:    true,
						},
						"proxy_uri": schema.StringAttribute{
							Description: "The gateway proxy URI where the bridge listens (http://0.0.0.0:<port>/<base_path>). The list endpoint omits it, so it is fetched per bridge; null if the per-bridge fetch fails.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *MCPBridgesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	bridgeClient, err := agentstools.NewMCPBridgeClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create MCP Bridge Client",
			"An unexpected error occurred when creating the MCP Bridge client.\n\nClient Error: "+err.Error(),
		)
		return
	}

	d.client = bridgeClient
}

func (d *MCPBridgesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MCPBridgesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	bridges, err := d.client.ListBridges(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing MCP bridges",
			"Could not list MCP bridges for environment "+envID+": "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(orgID + "/" + envID)
	data.OrganizationID = types.StringValue(orgID)
	data.Bridges = make([]MCPBridgeItemModel, 0, len(bridges))
	for _, b := range bridges {
		item := mapMCPBridgeToItemModel(b)
		// The list endpoint returns an empty endpoint block, so proxy_uri (the bridge's
		// reachable proxy URI) is only available from the single GET. Enrich best-effort:
		// on any error we leave proxy_uri null rather than failing the whole list.
		if full, err := d.client.GetBridge(ctx, orgID, envID, b.ID); err == nil &&
			full.Endpoint != nil && full.Endpoint.ProxyURI != nil {
			item.ProxyURI = stringOrNull(*full.Endpoint.ProxyURI)
		}
		data.Bridges = append(data.Bridges, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapMCPBridgeToItemModel(b agentstools.MCPBridge) MCPBridgeItemModel {
	technology := "flexGateway"
	if b.Technology != "" {
		technology = b.Technology
	}
	return MCPBridgeItemModel{
		ID:             types.StringValue(strconv.Itoa(b.ID)),
		AssetID:        stringOrNull(b.AssetID),
		AssetVersion:   stringOrNull(b.AssetVersion),
		ProductVersion: stringOrNull(b.ProductVersion),
		GroupID:        stringOrNull(b.GroupID),
		Technology:     types.StringValue(technology),
		InstanceLabel:  stringOrNull(b.InstanceLabel),
		Status:         stringOrNull(b.Status),
		EndpointURI:    stringOrNull(b.EndpointURI),
		ProxyURI:       types.StringNull(),
	}
}

// stringOrNull returns a null string for empty values so optional/omitted fields don't
// surface as empty strings in the data source output.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
