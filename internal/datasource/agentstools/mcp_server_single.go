package agentstools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ datasource.DataSource              = &MCPServerSingleDataSource{}
	_ datasource.DataSourceWithConfigure = &MCPServerSingleDataSource{}
)

// MCPServerSingleDataSource fetches a single MCP server by ID.
type MCPServerSingleDataSource struct {
	client *agentstools.MCPServerClient
}

type MCPServerSingleDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Technology       types.String `tfsdk:"technology"`
	ProviderID       types.String `tfsdk:"provider_id"`
	InstanceLabel    types.String `tfsdk:"instance_label"`
	ApprovalMethod   types.String `tfsdk:"approval_method"`
	Status           types.String `tfsdk:"status"`
	AssetID          types.String `tfsdk:"asset_id"`
	AssetVersion     types.String `tfsdk:"asset_version"`
	ProductVersion   types.String `tfsdk:"product_version"`
	ConsumerEndpoint types.String `tfsdk:"consumer_endpoint"`
	UpstreamID       types.String `tfsdk:"upstream_id"`
	Spec             types.Object `tfsdk:"spec"`
	Endpoint         types.Object `tfsdk:"endpoint"`
	Deployment       types.Object `tfsdk:"deployment"`
	Routing          types.List   `tfsdk:"routing"`
}

var (
	dsSingleSpecAttrTypes = map[string]attr.Type{
		"asset_id": types.StringType,
		"group_id": types.StringType,
		"version":  types.StringType,
	}
	dsSingleEndpointAttrTypes = map[string]attr.Type{
		"deployment_type":  types.StringType,
		"type":             types.StringType,
		"base_path":        types.StringType,
		"uri":              types.StringType,
		"response_timeout": types.Int64Type,
	}
	dsSingleDeploymentAttrTypes = map[string]attr.Type{
		"environment_id":  types.StringType,
		"type":            types.StringType,
		"expected_status": types.StringType,
		"overwrite":       types.BoolType,
		"target_id":       types.StringType,
		"target_name":     types.StringType,
		"gateway_version": types.StringType,
	}
	dsSingleRulesAttrTypes = map[string]attr.Type{
		"methods": types.StringType,
		"path":    types.StringType,
		"host":    types.StringType,
		"headers": types.MapType{ElemType: types.StringType},
	}
	dsSingleUpstreamAttrTypes = map[string]attr.Type{
		"weight":         types.Int64Type,
		"uri":            types.StringType,
		"label":          types.StringType,
		"tls_context_id": types.StringType,
	}
	dsSingleRouteAttrTypes = map[string]attr.Type{
		"label":     types.StringType,
		"rules":     types.ObjectType{AttrTypes: dsSingleRulesAttrTypes},
		"upstreams": types.ListType{ElemType: types.ObjectType{AttrTypes: dsSingleUpstreamAttrTypes}},
	}
)

func NewMCPServerSingleDataSource() datasource.DataSource {
	return &MCPServerSingleDataSource{}
}

func (d *MCPServerSingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mcp_server"
}

func (d *MCPServerSingleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the full details of a single MCP server by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The numeric identifier of the MCP server.",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider credentials organization.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the MCP server is deployed.",
			},
			"technology": schema.StringAttribute{
				Computed:    true,
				Description: "The gateway technology (typically omniGateway for MCP servers).",
			},
			"provider_id": schema.StringAttribute{
				Computed:    true,
				Description: "The identity provider ID for the MCP server.",
			},
			"instance_label": schema.StringAttribute{
				Computed:    true,
				Description: "A human-readable label for this MCP server.",
			},
			"approval_method": schema.StringAttribute{
				Computed:    true,
				Description: "Client approval method (e.g. manual, or null if no approval required).",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the MCP server.",
			},
			"asset_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Exchange asset ID backing this MCP server.",
			},
			"asset_version": schema.StringAttribute{
				Computed:    true,
				Description: "The Exchange asset version.",
			},
			"product_version": schema.StringAttribute{
				Computed:    true,
				Description: "The product version.",
			},
			"consumer_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Consumer-facing endpoint URI (the public URL clients use to reach the MCP server).",
			},
			"upstream_id": schema.StringAttribute{
				Computed:    true,
				Description: "The server-assigned upstream ID for the first upstream backend.",
			},
			"spec": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The Exchange asset specification backing this MCP server.",
				Attributes: map[string]schema.Attribute{
					"asset_id": schema.StringAttribute{
						Computed:    true,
						Description: "The Exchange asset ID.",
					},
					"group_id": schema.StringAttribute{
						Computed:    true,
						Description: "The Exchange group (organization) ID.",
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "The asset version.",
					},
				},
			},
			"endpoint": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Endpoint / proxy configuration for the MCP server.",
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Computed:    true,
						Description: "Deployment type (e.g. HY for hybrid, CH for CloudHub, RF for Runtime Fabric).",
					},
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "Endpoint protocol type (for MCP servers, this is 'mcp').",
					},
					"base_path": schema.StringAttribute{
						Computed:    true,
						Description: "MCP server base path for Omni Gateway (e.g. 'my-mcp-server').",
					},
					"uri": schema.StringAttribute{
						Computed:    true,
						Description: "Direct implementation URI (if configured instead of base_path).",
					},
					"response_timeout": schema.Int64Attribute{
						Computed:    true,
						Description: "Response timeout in milliseconds.",
					},
				},
			},
			"deployment": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Deployment target configuration.",
				Attributes: map[string]schema.Attribute{
					"environment_id": schema.StringAttribute{
						Computed:    true,
						Description: "The environment ID for deployment.",
					},
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "Deployment type (e.g. HY, CH, RF).",
					},
					"expected_status": schema.StringAttribute{
						Computed:    true,
						Description: "Expected deployment status (e.g. deployed, undeployed).",
					},
					"overwrite": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether to overwrite an existing deployment.",
					},
					"target_id": schema.StringAttribute{
						Computed:    true,
						Description: "The target gateway ID to deploy to.",
					},
					"target_name": schema.StringAttribute{
						Computed:    true,
						Description: "The target gateway name.",
					},
					"gateway_version": schema.StringAttribute{
						Computed:    true,
						Description: "The Omni Gateway runtime version.",
					},
				},
			},
			"routing": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Routing rules with weighted upstream backends.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Computed:    true,
							Description: "A label for this route.",
						},
						"rules": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "Match conditions for this route (methods, path, headers).",
							Attributes: map[string]schema.Attribute{
								"methods": schema.StringAttribute{
									Computed:    true,
									Description: "Pipe-separated HTTP methods (e.g. 'GET', 'POST|PUT').",
								},
								"path": schema.StringAttribute{
									Computed:    true,
									Description: "URL path pattern to match (e.g. '/api/*').",
								},
								"host": schema.StringAttribute{
									Computed:    true,
									Description: "Host header value to match.",
								},
								"headers": schema.MapAttribute{
									Computed:    true,
									ElementType: types.StringType,
									Description: "Header key-value pairs to match.",
								},
							},
						},
						"upstreams": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Weighted upstream backends for this route (actual MCP server implementation endpoints).",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"weight": schema.Int64Attribute{
										Computed:    true,
										Description: "Traffic weight percentage (0-100).",
									},
									"uri": schema.StringAttribute{
										Computed:    true,
										Description: "The upstream backend URI.",
									},
									"label": schema.StringAttribute{
										Computed:    true,
										Description: "A label for this upstream.",
									},
									"tls_context_id": schema.StringAttribute{
										Computed:    true,
										Description: "TLS context for upstream connections (format: 'secretGroupId/tlsContextId').",
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

func (d *MCPServerSingleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	mcpClient, err := agentstools.NewMCPServerClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create MCP Server Client", err.Error())
		return
	}
	d.client = mcpClient
}

func (d *MCPServerSingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MCPServerSingleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	serverIDStr := data.ID.ValueString()

	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		resp.Diagnostics.AddError("Invalid MCP Server ID",
			fmt.Sprintf("Could not parse MCP server ID as integer: %s", serverIDStr))
		return
	}

	mcpServer, err := d.client.GetMCPServer(ctx, orgID, envID, serverID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("MCP Server Not Found",
				fmt.Sprintf("MCP server with ID %d not found in environment %s", serverID, envID))
		} else {
			resp.Diagnostics.AddError("Error reading MCP server",
				fmt.Sprintf("Could not read MCP server ID %d: %s", serverID, err.Error()))
		}
		return
	}

	// Flatten the response into the data model
	data.OrganizationID = types.StringValue(orgID)
	if mcpServer.Technology != "" {
		data.Technology = types.StringValue(technologyFromAPI(mcpServer.Technology))
	} else {
		data.Technology = types.StringNull()
	}
	if mcpServer.ProviderID != nil {
		data.ProviderID = types.StringValue(*mcpServer.ProviderID)
	} else {
		data.ProviderID = types.StringNull()
	}
	data.InstanceLabel = types.StringValue(mcpServer.InstanceLabel)
	data.ApprovalMethod = types.StringValue(mcpServer.ApprovalMethod)
	data.Status = types.StringValue(mcpServer.Status)
	data.AssetID = types.StringValue(mcpServer.AssetID)
	data.AssetVersion = types.StringValue(mcpServer.AssetVersion)
	data.ProductVersion = types.StringValue(mcpServer.ProductVersion)
	data.ConsumerEndpoint = types.StringValue(mcpServer.EndpointURI)

	// Spec
	if mcpServer.Spec != nil {
		specObj, _ := types.ObjectValue(dsSingleSpecAttrTypes, map[string]attr.Value{
			"asset_id": types.StringValue(mcpServer.Spec.AssetID),
			"group_id": types.StringValue(mcpServer.Spec.GroupID),
			"version":  types.StringValue(mcpServer.Spec.Version),
		})
		data.Spec = specObj
	} else {
		data.Spec = types.ObjectNull(dsSingleSpecAttrTypes)
	}

	// Endpoint
	if mcpServer.Endpoint != nil {
		basePath := types.StringNull()
		if mcpServer.Endpoint.ProxyURI != nil && *mcpServer.Endpoint.ProxyURI != "" {
			basePath = types.StringValue(strings.TrimPrefix(*mcpServer.Endpoint.ProxyURI, "http://0.0.0.0:8081/"))
		}
		respTimeout := types.Int64Null()
		if mcpServer.Endpoint.ResponseTimeout != nil {
			respTimeout = types.Int64Value(int64(*mcpServer.Endpoint.ResponseTimeout))
		}
		endpointObj, _ := types.ObjectValue(dsSingleEndpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue(mcpServer.Endpoint.DeploymentType),
			"type":             types.StringValue(mcpServer.Endpoint.Type),
			"base_path":        basePath,
			"uri":              types.StringNull(),
			"response_timeout": respTimeout,
		})
		data.Endpoint = endpointObj
	} else {
		data.Endpoint = types.ObjectNull(dsSingleEndpointAttrTypes)
	}

	// Deployment
	if mcpServer.Deployment != nil {
		deploymentObj, _ := types.ObjectValue(dsSingleDeploymentAttrTypes, map[string]attr.Value{
			"environment_id":  types.StringValue(mcpServer.Deployment.EnvironmentID),
			"type":            types.StringValue(mcpServer.Deployment.Type),
			"expected_status": types.StringValue(mcpServer.Deployment.ExpectedStatus),
			"overwrite":       types.BoolValue(mcpServer.Deployment.Overwrite),
			"target_id":       types.StringValue(mcpServer.Deployment.TargetID),
			"target_name":     types.StringValue(mcpServer.Deployment.TargetName),
			"gateway_version": types.StringValue(mcpServer.Deployment.GatewayVersion),
		})
		data.Deployment = deploymentObj
	} else {
		data.Deployment = types.ObjectNull(dsSingleDeploymentAttrTypes)
	}

	// Routing
	data.UpstreamID = types.StringNull()
	if len(mcpServer.Routing) > 0 && len(mcpServer.Routing[0].Upstreams) > 0 && mcpServer.Routing[0].Upstreams[0].ID != "" {
		data.UpstreamID = types.StringValue(mcpServer.Routing[0].Upstreams[0].ID)
	}

	if len(mcpServer.Routing) > 0 {
		routeElems := make([]attr.Value, 0, len(mcpServer.Routing))
		for _, route := range mcpServer.Routing {
			// Upstreams
			usElems := make([]attr.Value, 0, len(route.Upstreams))
			for _, us := range route.Upstreams {
				var tlsCtxVal attr.Value
				if us.TLSContext != nil && us.TLSContext.SecretGroupID != "" && us.TLSContext.TLSContextID != "" {
					tlsCtxVal = types.StringValue(fmt.Sprintf("%s/%s", us.TLSContext.SecretGroupID, us.TLSContext.TLSContextID))
				} else {
					tlsCtxVal = types.StringNull()
				}
				usObj, _ := types.ObjectValue(dsSingleUpstreamAttrTypes, map[string]attr.Value{
					"weight":         types.Int64Value(int64(us.Weight)),
					"uri":            types.StringValue(us.URI),
					"label":          types.StringValue(us.Label),
					"tls_context_id": tlsCtxVal,
				})
				usElems = append(usElems, usObj)
			}

			// Rules
			var rulesVal attr.Value
			if route.Rules != nil {
				methods := types.StringNull()
				if route.Rules.Methods != "" {
					methods = types.StringValue(route.Rules.Methods)
				}
				pathVal := types.StringNull()
				if route.Rules.Path != "" {
					pathVal = types.StringValue(route.Rules.Path)
				}
				hostVal := types.StringNull()
				if route.Rules.Host != "" {
					hostVal = types.StringValue(route.Rules.Host)
				}
				var headersVal attr.Value
				if len(route.Rules.Headers) > 0 {
					hdrElems := make(map[string]attr.Value, len(route.Rules.Headers))
					for k, v := range route.Rules.Headers {
						hdrElems[k] = types.StringValue(v)
					}
					headersVal = types.MapValueMust(types.StringType, hdrElems)
				} else {
					headersVal = types.MapNull(types.StringType)
				}
				rulesObj, _ := types.ObjectValue(dsSingleRulesAttrTypes, map[string]attr.Value{
					"methods": methods,
					"path":    pathVal,
					"host":    hostVal,
					"headers": headersVal,
				})
				rulesVal = rulesObj
			} else {
				rulesVal = types.ObjectNull(dsSingleRulesAttrTypes)
			}

			routeObj, _ := types.ObjectValue(dsSingleRouteAttrTypes, map[string]attr.Value{
				"label":     types.StringValue(route.Label),
				"rules":     rulesVal,
				"upstreams": types.ListValueMust(types.ObjectType{AttrTypes: dsSingleUpstreamAttrTypes}, usElems),
			})
			routeElems = append(routeElems, routeObj)
		}

		data.Routing = types.ListValueMust(types.ObjectType{AttrTypes: dsSingleRouteAttrTypes}, routeElems)
	} else {
		data.Routing = types.ListNull(types.ObjectType{AttrTypes: dsSingleRouteAttrTypes})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// technologyFromAPI converts API technology values to Terraform schema values.
func technologyFromAPI(tech string) string {
	// The API may return "flexGateway" or other variants; resource normalizes to "omniGateway"
	if tech == "flexGateway" || tech == "omniGateway" {
		return "omniGateway"
	}
	return tech
}
