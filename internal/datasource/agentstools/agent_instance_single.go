package agentstools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ datasource.DataSource              = &AgentInstanceSingleDataSource{}
	_ datasource.DataSourceWithConfigure = &AgentInstanceSingleDataSource{}
)

// AgentInstanceSingleDataSource fetches a single agent instance by ID.
type AgentInstanceSingleDataSource struct {
	client *agentstools.AgentInstanceClient
}

type AgentInstanceSingleDataSourceModel struct {
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
	Spec             types.Object `tfsdk:"spec"`
	Endpoint         types.Object `tfsdk:"endpoint"`
	Deployment       types.Object `tfsdk:"deployment"`
	Routing          types.List   `tfsdk:"routing"`
}

var (
	dsSpecAttrTypes = map[string]attr.Type{
		"asset_id": types.StringType,
		"group_id": types.StringType,
		"version":  types.StringType,
	}
	dsEndpointAttrTypes = map[string]attr.Type{
		"deployment_type":  types.StringType,
		"type":             types.StringType,
		"base_path":        types.StringType,
		"uri":              types.StringType,
		"response_timeout": types.Int64Type,
	}
	dsDeploymentAttrTypes = map[string]attr.Type{
		"environment_id":  types.StringType,
		"type":            types.StringType,
		"expected_status": types.StringType,
		"overwrite":       types.BoolType,
		"target_id":       types.StringType,
		"target_name":     types.StringType,
		"gateway_version": types.StringType,
	}
	dsRulesAttrTypes = map[string]attr.Type{
		"methods": types.StringType,
		"path":    types.StringType,
		"host":    types.StringType,
		"headers": types.MapType{ElemType: types.StringType},
	}
	dsUpstreamAttrTypes = map[string]attr.Type{
		"weight":         types.Int64Type,
		"uri":            types.StringType,
		"label":          types.StringType,
		"tls_context_id": types.StringType,
	}
	dsRouteAttrTypes = map[string]attr.Type{
		"label":     types.StringType,
		"rules":     types.ObjectType{AttrTypes: dsRulesAttrTypes},
		"upstreams": types.ListType{ElemType: types.ObjectType{AttrTypes: dsUpstreamAttrTypes}},
	}
)

func NewAgentInstanceSingleDataSource() datasource.DataSource {
	return &AgentInstanceSingleDataSource{}
}

func (d *AgentInstanceSingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_instance"
}

func (d *AgentInstanceSingleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the full details of a single agent instance by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The numeric identifier of the agent instance (as a string).",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider credentials organization.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the agent instance is registered.",
			},
			"technology": schema.StringAttribute{
				Computed:    true,
				Description: "The gateway technology (e.g., omniGateway).",
			},
			"provider_id": schema.StringAttribute{
				Computed:    true,
				Description: "The identity provider ID for the agent.",
			},
			"instance_label": schema.StringAttribute{
				Computed:    true,
				Description: "A human-readable label for this agent instance.",
			},
			"approval_method": schema.StringAttribute{
				Computed:    true,
				Description: "Client approval method (e.g., manual).",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the agent instance.",
			},
			"asset_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Exchange asset ID.",
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
				Description: "Consumer-facing endpoint URI (the public URL clients use to reach the agent).",
			},
			"spec": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The Exchange asset specification backing this agent instance.",
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
				Description: "Endpoint / proxy configuration for the agent instance.",
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Computed:    true,
						Description: "Deployment type (HY, CH, RF).",
					},
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "Endpoint protocol type (e.g., a2a for Agent-to-Agent).",
					},
					"base_path": schema.StringAttribute{
						Computed:    true,
						Description: "Agent base path for Omni Gateway (e.g., my-agent).",
					},
					"uri": schema.StringAttribute{
						Computed:    true,
						Description: "Direct implementation URI.",
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
						Description: "Deployment type (HY, CH, RF).",
					},
					"expected_status": schema.StringAttribute{
						Computed:    true,
						Description: "Expected deployment status (deployed, undeployed).",
					},
					"overwrite": schema.BoolAttribute{
						Computed:    true,
						Description: "Whether to overwrite an existing deployment.",
					},
					"target_id": schema.StringAttribute{
						Computed:    true,
						Description: "The target gateway ID.",
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
									Description: "Pipe-separated HTTP methods (e.g., GET, POST|PUT).",
								},
								"path": schema.StringAttribute{
									Computed:    true,
									Description: "URL path pattern to match (e.g., /api/*).",
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
							Description: "Weighted upstream backends for this route.",
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
										Description: "TLS context for upstream connections (format: secretGroupId/tlsContextId).",
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

func (d *AgentInstanceSingleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	agentClient, err := agentstools.NewAgentInstanceClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Agent Instance Client", err.Error())
		return
	}
	d.client = agentClient
}

func (d *AgentInstanceSingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentInstanceSingleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	agentIDStr := data.ID.ValueString()

	agentID, err := strconv.Atoi(agentIDStr)
	if err != nil {
		resp.Diagnostics.AddError("Invalid agent instance ID",
			fmt.Sprintf("Expected numeric ID, got: %s", agentIDStr))
		return
	}

	instance, err := d.client.GetAgentInstance(ctx, orgID, envID, agentID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Agent instance not found",
				fmt.Sprintf("Agent instance %d not found in environment %s. It may have been deleted.", agentID, envID))
		} else {
			resp.Diagnostics.AddError("Error reading agent instance", err.Error())
		}
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.ID = types.StringValue(strconv.Itoa(instance.ID))

	if instance.Technology != "" {
		// Convert flexGateway back to omniGateway for user-facing schema
		tech := instance.Technology
		if tech == "flexGateway" {
			tech = "omniGateway"
		}
		data.Technology = types.StringValue(tech)
	} else {
		data.Technology = types.StringNull()
	}

	if instance.ProviderID != nil {
		data.ProviderID = types.StringValue(*instance.ProviderID)
	} else {
		data.ProviderID = types.StringNull()
	}

	if instance.InstanceLabel != "" {
		data.InstanceLabel = types.StringValue(instance.InstanceLabel)
	} else {
		data.InstanceLabel = types.StringNull()
	}

	if instance.ApprovalMethod != "" {
		data.ApprovalMethod = types.StringValue(instance.ApprovalMethod)
	} else {
		data.ApprovalMethod = types.StringNull()
	}

	if instance.Status != "" {
		data.Status = types.StringValue(instance.Status)
	} else {
		data.Status = types.StringNull()
	}

	if instance.AssetID != "" {
		data.AssetID = types.StringValue(instance.AssetID)
	} else {
		data.AssetID = types.StringNull()
	}

	if instance.AssetVersion != "" {
		data.AssetVersion = types.StringValue(instance.AssetVersion)
	} else {
		data.AssetVersion = types.StringNull()
	}

	if instance.ProductVersion != "" {
		data.ProductVersion = types.StringValue(instance.ProductVersion)
	} else {
		data.ProductVersion = types.StringNull()
	}

	if instance.EndpointURI != "" {
		data.ConsumerEndpoint = types.StringValue(instance.EndpointURI)
	} else {
		data.ConsumerEndpoint = types.StringNull()
	}

	// Spec
	if instance.Spec != nil {
		specObj, _ := types.ObjectValue(dsSpecAttrTypes, map[string]attr.Value{
			"asset_id": types.StringValue(instance.Spec.AssetID),
			"group_id": types.StringValue(instance.Spec.GroupID),
			"version":  types.StringValue(instance.Spec.Version),
		})
		data.Spec = specObj
	} else {
		data.Spec = types.ObjectNull(dsSpecAttrTypes)
	}

	// Endpoint
	if instance.Endpoint != nil {
		var basePath types.String
		if instance.Endpoint.ProxyURI != nil && *instance.Endpoint.ProxyURI != "" {
			basePath = types.StringValue(*instance.Endpoint.ProxyURI)
		} else {
			basePath = types.StringNull()
		}

		var uri types.String
		if instance.Endpoint.URI != nil {
			uri = types.StringValue(*instance.Endpoint.URI)
		} else {
			uri = types.StringNull()
		}

		var respTimeout types.Int64
		if instance.Endpoint.ResponseTimeout != nil {
			respTimeout = types.Int64Value(int64(*instance.Endpoint.ResponseTimeout))
		} else {
			respTimeout = types.Int64Null()
		}

		endpointObj, _ := types.ObjectValue(dsEndpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue(instance.Endpoint.DeploymentType),
			"type":             types.StringValue(instance.Endpoint.Type),
			"base_path":        basePath,
			"uri":              uri,
			"response_timeout": respTimeout,
		})
		data.Endpoint = endpointObj
	} else {
		data.Endpoint = types.ObjectNull(dsEndpointAttrTypes)
	}

	// Deployment
	if instance.Deployment != nil {
		deploymentObj, _ := types.ObjectValue(dsDeploymentAttrTypes, map[string]attr.Value{
			"environment_id":  types.StringValue(instance.Deployment.EnvironmentID),
			"type":            types.StringValue(instance.Deployment.Type),
			"expected_status": types.StringValue(instance.Deployment.ExpectedStatus),
			"overwrite":       types.BoolValue(instance.Deployment.Overwrite),
			"target_id":       types.StringValue(instance.Deployment.TargetID),
			"target_name":     types.StringValue(instance.Deployment.TargetName),
			"gateway_version": types.StringValue(instance.Deployment.GatewayVersion),
		})
		data.Deployment = deploymentObj
	} else {
		data.Deployment = types.ObjectNull(dsDeploymentAttrTypes)
	}

	// Routing
	if len(instance.Routing) > 0 {
		routeElems := make([]attr.Value, 0, len(instance.Routing))
		for _, route := range instance.Routing {
			usElems := make([]attr.Value, 0, len(route.Upstreams))
			for _, us := range route.Upstreams {
				var tlsCtxVal attr.Value
				if us.TLSContext != nil && us.TLSContext.SecretGroupID != "" && us.TLSContext.TLSContextID != "" {
					tlsCtxVal = types.StringValue(fmt.Sprintf("%s/%s", us.TLSContext.SecretGroupID, us.TLSContext.TLSContextID))
				} else {
					tlsCtxVal = types.StringNull()
				}
				usObj, _ := types.ObjectValue(
					dsUpstreamAttrTypes,
					map[string]attr.Value{
						"weight":         types.Int64Value(int64(us.Weight)),
						"uri":            types.StringValue(us.URI),
						"label":          types.StringValue(us.Label),
						"tls_context_id": tlsCtxVal,
					},
				)
				usElems = append(usElems, usObj)
			}

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
				rulesObj, _ := types.ObjectValue(
					dsRulesAttrTypes,
					map[string]attr.Value{
						"methods": methods,
						"path":    pathVal,
						"host":    hostVal,
						"headers": headersVal,
					},
				)
				rulesVal = rulesObj
			} else {
				rulesVal = types.ObjectNull(dsRulesAttrTypes)
			}

			routeObj, _ := types.ObjectValue(
				dsRouteAttrTypes,
				map[string]attr.Value{
					"label":     types.StringValue(route.Label),
					"rules":     rulesVal,
					"upstreams": types.ListValueMust(types.ObjectType{AttrTypes: dsUpstreamAttrTypes}, usElems),
				},
			)
			routeElems = append(routeElems, routeObj)
		}

		data.Routing = types.ListValueMust(types.ObjectType{AttrTypes: dsRouteAttrTypes}, routeElems)
	} else {
		data.Routing = types.ListNull(types.ObjectType{AttrTypes: dsRouteAttrTypes})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
