package agentstools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &MCPBridgeDataSource{}
	_ datasource.DataSourceWithConfigure = &MCPBridgeDataSource{}
)

// MCPBridgeDataSource fetches the full details of a single MCP bridge by ID, including
// its source APIs and the tools reconstructed from the live transcoding policies (the
// same rebuild the resource uses on import — shared via agentstools.ReconstructBridgeSources).
type MCPBridgeDataSource struct {
	client *agentstools.MCPBridgeClient
}

type MCPBridgeDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	GatewayID        types.String `tfsdk:"gateway_id"`
	Port             types.Int64  `tfsdk:"port"`
	BasePath         types.String `tfsdk:"base_path"`
	AssetID          types.String `tfsdk:"asset_id"`
	AssetVersion     types.String `tfsdk:"asset_version"`
	ProductVersion   types.String `tfsdk:"product_version"`
	GroupID          types.String `tfsdk:"group_id"`
	Technology       types.String `tfsdk:"technology"`
	InstanceLabel    types.String `tfsdk:"instance_label"`
	Status           types.String `tfsdk:"status"`
	ConsumerEndpoint types.String `tfsdk:"consumer_endpoint"`
	ProxyURI         types.String `tfsdk:"proxy_uri"`
	Deployment       types.Object `tfsdk:"deployment"`
	SourceAPIs       types.List   `tfsdk:"source_apis"`
}

var (
	dsBridgeToolAttrTypes = map[string]attr.Type{
		"name":          types.StringType,
		"description":   types.StringType,
		"method":        types.StringType,
		"path":          types.StringType,
		"query_params":  types.ListType{ElemType: types.StringType},
		"header_params": types.ListType{ElemType: types.StringType},
		"has_body":      types.BoolType,
	}
	dsBridgeSourceAttrTypes = map[string]attr.Type{
		"label":        types.StringType,
		"upstream_uri": types.StringType,
		"asset_id":     types.StringType,
		"group_id":     types.StringType,
		"version":      types.StringType,
		"tools":        types.ListType{ElemType: types.ObjectType{AttrTypes: dsBridgeToolAttrTypes}},
	}
	dsBridgeDeploymentAttrTypes = map[string]attr.Type{
		"environment_id":  types.StringType,
		"type":            types.StringType,
		"expected_status": types.StringType,
		"target_id":       types.StringType,
		"target_name":     types.StringType,
		"gateway_version": types.StringType,
	}
)

func NewMCPBridgeDataSource() datasource.DataSource {
	return &MCPBridgeDataSource{}
}

func (d *MCPBridgeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_bridge"
}

func (d *MCPBridgeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the full details of a single MCP bridge by ID, including its source APIs and reconstructed tools.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The numeric identifier of the MCP bridge (API Manager instance ID).",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider credentials organization.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the MCP bridge is deployed.",
			},
			"gateway_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Flex Gateway ID the bridge is deployed to.",
			},
			"port": schema.Int64Attribute{
				Computed:    true,
				Description: "The listener port derived from the bridge proxy URI.",
			},
			"base_path": schema.StringAttribute{
				Computed:    true,
				Description: "The base path derived from the bridge proxy URI.",
			},
			"asset_id": schema.StringAttribute{
				Computed:    true,
				Description: "The generated Exchange asset ID backing the bridge.",
			},
			"asset_version": schema.StringAttribute{
				Computed:    true,
				Description: "The generated Exchange asset version.",
			},
			"product_version": schema.StringAttribute{
				Computed:    true,
				Description: "The product version.",
			},
			"group_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Exchange group (organization) ID.",
			},
			"technology": schema.StringAttribute{
				Computed:    true,
				Description: "The gateway technology (flexGateway for MCP bridges).",
			},
			"instance_label": schema.StringAttribute{
				Computed:    true,
				Description: "The label of the MCP bridge.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the MCP bridge.",
			},
			"consumer_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "The consumer-facing MCP endpoint URI (UI: \"Consumer Endpoint\"). Populated from the platform's endpointUri; may be null for self-managed (flexGateway) bridges, in which case use proxy_uri.",
			},
			"proxy_uri": schema.StringAttribute{
				Computed:    true,
				Description: "The gateway proxy URI where the bridge listens (http://0.0.0.0:<port>/<base_path>), reconstructed from the instance endpoint.",
			},
			"deployment": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Deployment target configuration.",
				Attributes: map[string]schema.Attribute{
					"environment_id":  schema.StringAttribute{Computed: true, Description: "The environment ID for deployment."},
					"type":            schema.StringAttribute{Computed: true, Description: "Deployment type (e.g. HY)."},
					"expected_status": schema.StringAttribute{Computed: true, Description: "Expected deployment status."},
					"target_id":       schema.StringAttribute{Computed: true, Description: "The target gateway ID."},
					"target_name":     schema.StringAttribute{Computed: true, Description: "The target gateway name."},
					"gateway_version": schema.StringAttribute{Computed: true, Description: "The gateway runtime version."},
				},
			},
			"source_apis": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The source REST APIs bridged by this MCP server, with their reconstructed tools.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label":        schema.StringAttribute{Computed: true, Description: "The source API label (X-UPSTREAM-NAME)."},
						"upstream_uri": schema.StringAttribute{Computed: true, Description: "The backend URI of the source REST API."},
						"asset_id":     schema.StringAttribute{Computed: true, Description: "The source REST API's Exchange asset ID."},
						"group_id":     schema.StringAttribute{Computed: true, Description: "The source REST API's Exchange group ID."},
						"version":      schema.StringAttribute{Computed: true, Description: "The source REST API's Exchange asset version."},
						"tools": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The MCP tools exposed for this source API.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name":          schema.StringAttribute{Computed: true, Description: "The tool name."},
									"description":   schema.StringAttribute{Computed: true, Description: "The tool description (null; lives in the asset metadata)."},
									"method":        schema.StringAttribute{Computed: true, Description: "The HTTP method the tool maps to."},
									"path":          schema.StringAttribute{Computed: true, Description: "The REST path the tool maps to."},
									"query_params":  schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Query parameter names passed through."},
									"header_params": schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Header parameter names passed through."},
									"has_body":      schema.BoolAttribute{Computed: true, Description: "Whether the tool sends a request body."},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *MCPBridgeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	bridgeClient, err := agentstools.NewMCPBridgeClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create MCP Bridge Client", err.Error())
		return
	}
	d.client = bridgeClient
}

func (d *MCPBridgeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MCPBridgeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	bridgeID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Bridge ID",
			fmt.Sprintf("Could not parse MCP bridge ID as integer: %s", data.ID.ValueString()))
		return
	}

	inst, err := d.client.GetBridge(ctx, orgID, envID, bridgeID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("MCP Bridge Not Found",
				fmt.Sprintf("MCP bridge with ID %d not found in environment %s", bridgeID, envID))
		} else {
			resp.Diagnostics.AddError("Error reading MCP bridge",
				fmt.Sprintf("Could not read MCP bridge ID %d: %s", bridgeID, err.Error()))
		}
		return
	}

	ups, err := d.client.GetBridgeUpstreams(ctx, orgID, envID, bridgeID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MCP bridge upstreams", err.Error())
		return
	}
	policies, err := d.client.Policies.ListAPIPolicies(ctx, orgID, envID, bridgeID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MCP bridge policies", err.Error())
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.Technology = types.StringValue("flexGateway")
	data.AssetID = stringOrNull(inst.AssetID)
	data.AssetVersion = stringOrNull(inst.AssetVersion)
	data.ProductVersion = stringOrNull(inst.ProductVersion)
	data.GroupID = stringOrNull(inst.GroupID)
	data.InstanceLabel = stringOrNull(inst.InstanceLabel)
	data.Status = stringOrNull(inst.Status)
	data.ConsumerEndpoint = stringOrNull(inst.EndpointURI)

	data.GatewayID = types.StringNull()
	if inst.Deployment != nil && inst.Deployment.TargetID != "" {
		data.GatewayID = types.StringValue(inst.Deployment.TargetID)
	}

	data.Port = types.Int64Null()
	data.BasePath = types.StringNull()
	data.ProxyURI = types.StringNull()
	if inst.Endpoint != nil && inst.Endpoint.ProxyURI != nil {
		data.ProxyURI = stringOrNull(*inst.Endpoint.ProxyURI)
		if port, bp, ok := parseBridgeProxyURI(*inst.Endpoint.ProxyURI); ok {
			data.Port = types.Int64Value(port)
			data.BasePath = types.StringValue(bp)
		}
	}

	if inst.Deployment != nil {
		obj, _ := types.ObjectValue(dsBridgeDeploymentAttrTypes, map[string]attr.Value{
			"environment_id":  types.StringValue(inst.Deployment.EnvironmentID),
			"type":            types.StringValue(inst.Deployment.Type),
			"expected_status": types.StringValue(inst.Deployment.ExpectedStatus),
			"target_id":       types.StringValue(inst.Deployment.TargetID),
			"target_name":     types.StringValue(inst.Deployment.TargetName),
			"gateway_version": types.StringValue(inst.Deployment.GatewayVersion),
		})
		data.Deployment = obj
	} else {
		data.Deployment = types.ObjectNull(dsBridgeDeploymentAttrTypes)
	}

	data.SourceAPIs = flattenBridgeSourcesForDS(orgID, inst, ups, policies)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// flattenBridgeSourcesForDS reconstructs the bridge's source APIs (shared pure logic)
// and flattens them into the data source's Terraform types. Unlike the resource (which
// nulls a derived tool name to avoid post-import drift), the data source surfaces the
// effective tool name — the actual name a client sees — for observability.
func flattenBridgeSourcesForDS(orgID string, inst *agentstools.MCPBridge, ups []agentstools.MCPBridgeUpstreamDetail, policies []apimanagement.APIPolicy) types.List {
	srcObjType := types.ObjectType{AttrTypes: dsBridgeSourceAttrTypes}

	sources := agentstools.ReconstructBridgeSources(orgID, inst, ups, policies)
	srcElems := make([]attr.Value, 0, len(sources))
	for _, src := range sources {
		toolElems := make([]attr.Value, 0, len(src.Tools))
		for _, t := range src.Tools {
			name := t.Name
			if name == "" {
				name = agentstools.BridgeToolName(t.Method, t.Path)
			}
			qList, _ := types.ListValue(types.StringType, dsStringsToValues(t.QueryParams))
			hList, _ := types.ListValue(types.StringType, dsStringsToValues(t.HeaderParams))
			toolObj, diags := types.ObjectValue(dsBridgeToolAttrTypes, map[string]attr.Value{
				"name":          types.StringValue(name),
				"description":   types.StringNull(),
				"method":        types.StringValue(t.Method),
				"path":          types.StringValue(t.Path),
				"query_params":  qList,
				"header_params": hList,
				"has_body":      types.BoolValue(t.HasBody),
			})
			if diags.HasError() {
				continue
			}
			toolElems = append(toolElems, toolObj)
		}
		toolsList, _ := types.ListValue(types.ObjectType{AttrTypes: dsBridgeToolAttrTypes}, toolElems)
		srcObj, diags := types.ObjectValue(dsBridgeSourceAttrTypes, map[string]attr.Value{
			"label":        types.StringValue(src.Label),
			"upstream_uri": types.StringValue(src.UpstreamURI),
			"asset_id":     types.StringValue(src.AssetID),
			"group_id":     types.StringValue(src.GroupID),
			"version":      types.StringValue(src.Version),
			"tools":        toolsList,
		})
		if diags.HasError() {
			continue
		}
		srcElems = append(srcElems, srcObj)
	}
	list, _ := types.ListValue(srcObjType, srcElems)
	return list
}

func dsStringsToValues(in []string) []attr.Value {
	out := make([]attr.Value, 0, len(in))
	for _, s := range in {
		out = append(out, types.StringValue(s))
	}
	return out
}

// parseBridgeProxyURI extracts the listener port and base path from a bridge proxy URI
// of the form http://0.0.0.0:<port>/<base_path>.
func parseBridgeProxyURI(raw string) (port int64, basePath string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0, "", false
	}
	p, err := strconv.ParseInt(u.Port(), 10, 64)
	if err != nil {
		return 0, "", false
	}
	return p, strings.TrimPrefix(u.Path, "/"), true
}
