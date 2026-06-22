package apimanagement

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ datasource.DataSource              = &APIInstanceSingleDataSource{}
	_ datasource.DataSourceWithConfigure = &APIInstanceSingleDataSource{}
)

// APIInstanceSingleDataSource fetches a single API instance by ID.
type APIInstanceSingleDataSource struct {
	client *apimanagement.APIInstanceClient
}

type APIInstanceSingleDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	OrganizationID   types.String `tfsdk:"organization_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Technology       types.String `tfsdk:"technology"`
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

func NewAPIInstanceSingleDataSource() datasource.DataSource {
	return &APIInstanceSingleDataSource{}
}

func (d *APIInstanceSingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance"
}

func (d *APIInstanceSingleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the full details of a single API instance by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The numeric API instance ID (as a string).",
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID. Defaults to the provider credentials organization.",
			},
			"environment_id": schema.StringAttribute{
				Required:    true,
				Description: "The environment ID where the API instance is deployed.",
			},
			"technology": schema.StringAttribute{
				Computed:    true,
				Description: "The gateway technology (e.g. 'omniGateway', 'flexGateway', 'mule3', 'mule4').",
			},
			"instance_label": schema.StringAttribute{
				Computed:    true,
				Description: "A human-readable label for this API instance.",
			},
			"approval_method": schema.StringAttribute{
				Computed:    true,
				Description: "Client approval method (e.g. 'manual'). Empty when no approval is required.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The current status of the API instance (e.g. 'active', 'pending').",
			},
			"asset_id": schema.StringAttribute{
				Computed:    true,
				Description: "The Exchange asset ID backing this API instance.",
			},
			"asset_version": schema.StringAttribute{
				Computed:    true,
				Description: "The Exchange asset version.",
			},
			"product_version": schema.StringAttribute{
				Computed:    true,
				Description: "The product version of the API instance.",
			},
			"consumer_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Consumer-facing endpoint URI (the public URL clients use to reach the API).",
			},
			"spec": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The Exchange asset specification backing this API instance.",
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
				Description: "Endpoint / proxy configuration for the API instance.",
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Computed:    true,
						Description: "Deployment type (e.g. 'HY' for hybrid, 'CH' for CloudHub, 'RF' for Runtime Fabric).",
					},
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "Endpoint protocol type (e.g. 'http', 'rest', 'raml').",
					},
					"base_path": schema.StringAttribute{
						Computed:    true,
						Description: "API base path for the Omni Gateway proxy listener (extracted from proxyUri).",
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
						Description: "Deployment type (e.g. 'HY', 'CH', 'RF').",
					},
					"expected_status": schema.StringAttribute{
						Computed:    true,
						Description: "Expected deployment status (e.g. 'deployed', 'undeployed').",
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
										Description: "TLS context for upstream connections in format 'secretGroupId/tlsContextId'.",
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

func (d *APIInstanceSingleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData))
		return
	}
	apiClient, err := apimanagement.NewAPIInstanceClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create API Instance Client", err.Error())
		return
	}
	d.client = apiClient
}

func (d *APIInstanceSingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIInstanceSingleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	apiID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID",
			fmt.Sprintf("Could not parse API instance ID as integer: %s", data.ID.ValueString()))
		return
	}

	instance, err := d.client.GetAPIInstance(ctx, orgID, envID, apiID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("API Instance Not Found",
				fmt.Sprintf("API instance with ID %d was not found in environment %s. It may have been deleted.", apiID, envID))
		} else {
			resp.Diagnostics.AddError("Error reading API instance",
				fmt.Sprintf("Could not read API instance ID %d: %s", apiID, err.Error()))
		}
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.Status = types.StringValue(instance.Status)
	data.AssetID = types.StringValue(instance.AssetID)
	data.AssetVersion = types.StringValue(instance.AssetVersion)
	data.ProductVersion = types.StringValue(instance.ProductVersion)
	data.Technology = types.StringValue(technologyFromAPI(instance.Technology))
	data.InstanceLabel = types.StringValue(instance.InstanceLabel)
	data.ApprovalMethod = types.StringValue(instance.ApprovalMethod)

	if instance.EndpointURI != "" {
		data.ConsumerEndpoint = types.StringValue(instance.EndpointURI)
	} else {
		data.ConsumerEndpoint = types.StringNull()
	}

	// spec
	if instance.Spec != nil {
		specObj, _ := types.ObjectValue(dsSpecAttrTypes, map[string]attr.Value{
			"asset_id": types.StringValue(instance.Spec.AssetID),
			"group_id": types.StringValue(instance.Spec.GroupID),
			"version":  types.StringValue(instance.Spec.Version),
		})
		data.Spec = specObj
	} else if instance.AssetID != "" {
		// Fallback: construct spec from top-level fields
		specObj, _ := types.ObjectValue(dsSpecAttrTypes, map[string]attr.Value{
			"asset_id": types.StringValue(instance.AssetID),
			"group_id": types.StringValue(orgID),
			"version":  types.StringValue(instance.AssetVersion),
		})
		data.Spec = specObj
	} else {
		data.Spec = types.ObjectNull(dsSpecAttrTypes)
	}

	// endpoint
	if instance.Endpoint != nil {
		basePath := types.StringNull()
		if instance.Endpoint.ProxyURI != nil && *instance.Endpoint.ProxyURI != "" {
			basePath = types.StringValue(extractBasePath(*instance.Endpoint.ProxyURI))
		}
		responseTimeout := types.Int64Null()
		if instance.Endpoint.ResponseTimeout != nil {
			responseTimeout = types.Int64Value(int64(*instance.Endpoint.ResponseTimeout))
		}
		endpointObj, _ := types.ObjectValue(dsEndpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue(instance.Endpoint.DeploymentType),
			"type":             types.StringValue(instance.Endpoint.Type),
			"base_path":        basePath,
			"response_timeout": responseTimeout,
		})
		data.Endpoint = endpointObj
	} else {
		data.Endpoint = types.ObjectNull(dsEndpointAttrTypes)
	}

	// deployment
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

	// routing
	if len(instance.Routing) > 0 {
		routeElems := make([]attr.Value, 0, len(instance.Routing))
		for _, route := range instance.Routing {
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
				rulesObj, _ := types.ObjectValue(dsRulesAttrTypes, map[string]attr.Value{
					"methods": methods,
					"path":    pathVal,
					"host":    hostVal,
					"headers": headersVal,
				})
				rulesVal = rulesObj
			} else {
				rulesVal = types.ObjectNull(dsRulesAttrTypes)
			}

			usElems := make([]attr.Value, 0, len(route.Upstreams))
			for _, us := range route.Upstreams {
				uriVal := types.StringNull()
				if us.URI != "" {
					uriVal = types.StringValue(us.URI)
				}
				labelVal := types.StringNull()
				if us.Label != "" {
					labelVal = types.StringValue(us.Label)
				}
				var tlsCtxVal attr.Value
				if us.TLSContext != nil && us.TLSContext.SecretGroupID != "" && us.TLSContext.TLSContextID != "" {
					tlsCtxVal = types.StringValue(fmt.Sprintf("%s/%s", us.TLSContext.SecretGroupID, us.TLSContext.TLSContextID))
				} else {
					tlsCtxVal = types.StringNull()
				}
				usObj, _ := types.ObjectValue(dsUpstreamAttrTypes, map[string]attr.Value{
					"weight":         types.Int64Value(int64(us.Weight)),
					"uri":            uriVal,
					"label":          labelVal,
					"tls_context_id": tlsCtxVal,
				})
				usElems = append(usElems, usObj)
			}

			routeObj, _ := types.ObjectValue(dsRouteAttrTypes, map[string]attr.Value{
				"label":     types.StringValue(route.Label),
				"rules":     rulesVal,
				"upstreams": types.ListValueMust(types.ObjectType{AttrTypes: dsUpstreamAttrTypes}, usElems),
			})
			routeElems = append(routeElems, routeObj)
		}
		data.Routing = types.ListValueMust(types.ObjectType{AttrTypes: dsRouteAttrTypes}, routeElems)
	} else {
		data.Routing = types.ListNull(types.ObjectType{AttrTypes: dsRouteAttrTypes})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// extractBasePath removes the "http://0.0.0.0:8081/" prefix from the proxyUri
func extractBasePath(proxyURI string) string {
	const prefix = "http://0.0.0.0:8081/"
	if len(proxyURI) > len(prefix) {
		return proxyURI[len(prefix):]
	}
	return ""
}

// technologyFromAPI converts API technology names to user-facing names
func technologyFromAPI(t string) string {
	if t == "flexGateway" {
		return "omniGateway"
	}
	return t
}
