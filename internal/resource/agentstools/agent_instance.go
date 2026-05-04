package agentstools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

var (
	_ resource.Resource                   = &AgentInstanceResource{}
	_ resource.ResourceWithConfigure      = &AgentInstanceResource{}
	_ resource.ResourceWithImportState    = &AgentInstanceResource{}
	_ resource.ResourceWithValidateConfig = &AgentInstanceResource{}
)

type AgentInstanceResource struct {
	client *agentstools.AgentInstanceClient
}

// --- Terraform State Models ---

type AgentInstanceResourceModel struct {
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
	UpstreamURI      types.String `tfsdk:"upstream_uri"`

	GatewayID types.String `tfsdk:"gateway_id"`

	Spec       *SpecModel   `tfsdk:"spec"`
	Endpoint   types.Object `tfsdk:"endpoint"`
	Deployment types.Object `tfsdk:"deployment"`
	Routing    types.List   `tfsdk:"routing"`
}

func NewAgentInstanceResource() resource.Resource {
	return &AgentInstanceResource{}
}

func (r *AgentInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_instance"
}

func (r *AgentInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Agent instance in Anypoint API Manager. An Agent instance represents " +
			"an Agent specification deployed to a Flex Gateway target with routing rules and upstream backends.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric identifier of the Agent instance (stored as string for Terraform compatibility).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "The environment ID where the Agent instance will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"technology": schema.StringAttribute{
				Description: "The gateway technology. Valid values: 'flexGateway', 'mule4', 'serviceMesh'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("flexGateway"),
				Validators: []validator.String{
					stringvalidator.OneOf("flexGateway", "mule4", "serviceMesh"),
				},
			},
			"provider_id": schema.StringAttribute{
				Description: "The identity provider ID for the Agent.",
				Optional:    true,
			},
			"instance_label": schema.StringAttribute{
				Description: "A human-readable label for this Agent instance.",
				Optional:    true,
			},
			"approval_method": schema.StringAttribute{
				Description: "Client approval method. Valid values: 'manual', 'automatic'. Defaults to null (no approval required).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("manual", "automatic"),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the Agent instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "The Exchange asset ID (computed from Agent response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_version": schema.StringAttribute{
				Description: "The Exchange asset version (computed from Agent response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_version": schema.StringAttribute{
				Description: "The product version (computed from Agent response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"spec": schema.SingleNestedAttribute{
				Description: "The Exchange asset specification backing this Agent instance.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"asset_id": schema.StringAttribute{
						Description: "The Exchange asset ID.",
						Required:    true,
					},
					"group_id": schema.StringAttribute{
						Description: "The Exchange group (organization) ID.",
						Required:    true,
					},
					"version": schema.StringAttribute{
						Description: "The asset version.",
						Required:    true,
					},
				},
			},
			"endpoint": schema.SingleNestedAttribute{
				Description: "Endpoint / proxy configuration for the Agent instance.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"deployment_type": schema.StringAttribute{
						Description: "Deployment type. Valid values: 'HY' (hybrid), 'CH' (CloudHub), 'RF' (Runtime Fabric).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("HY"),
						Validators: []validator.String{
							stringvalidator.OneOf("HY", "CH", "CH2", "RF"),
						},
					},
					"type": schema.StringAttribute{
						Description: "Endpoint protocol type. For agent instances, this is 'a2a' (Agent-to-Agent).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("a2a"),
						Validators: []validator.String{
							stringvalidator.OneOf("a2a"),
						},
					},
					"base_path": schema.StringAttribute{
						Description: "Agent base path for FlexGateway (e.g. 'my-agent'). The provider constructs the full proxy URI as http://0.0.0.0:8081/<base_path>. Required when technology='flexGateway'. Mutually exclusive with 'uri'.",
						Optional:    true,
					},
					"uri": schema.StringAttribute{
						Description: "Direct implementation URI for Mule4 or other technologies (e.g. 'http://www.google.com'). Required when technology='mule4'. Mutually exclusive with 'base_path'.",
						Optional:    true,
					},
					"response_timeout": schema.Int64Attribute{
						Description: "Response timeout in milliseconds.",
						Optional:    true,
					},
				},
			},
			"consumer_endpoint": schema.StringAttribute{
				Description: "Consumer-facing endpoint URI (the public URL clients use to reach the Agent). Maps to top-level endpointUri in the Agent.",
				Optional:    true,
			},
			"upstream_uri": schema.StringAttribute{
				Description: "Shorthand for a single-upstream routing configuration. " +
					"When set, the provider constructs routing as [{upstreams: [{weight: 100, uri: <value>}]}]. " +
					"Mutually exclusive with the 'routing' block.",
				Optional: true,
			},
			"gateway_id": schema.StringAttribute{
				Description: "The Flex Gateway UUID. When provided, the deployment block is auto-populated " +
					"by fetching gateway details (target_id, target_name, gateway_version) from the Gateway Manager Agent. " +
					"Mutually exclusive with specifying a full deployment block.",
				Optional: true,
			},
			"deployment": schema.SingleNestedAttribute{
				Description: "Deployment target configuration. Auto-populated when gateway_id is set.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"environment_id": schema.StringAttribute{
						Description: "The environment ID for deployment (usually matches the top-level environment_id).",
						Optional:    true,
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Deployment type. Valid values: 'HY', 'CH', 'RF'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("HY"),
					},
					"expected_status": schema.StringAttribute{
						Description: "Expected deployment status. Valid values: 'deployed', 'undeployed'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("deployed"),
					},
					"overwrite": schema.BoolAttribute{
						Description: "Whether to overwrite an existing deployment.",
						Optional:    true,
					},
					"target_id": schema.StringAttribute{
						Description: "The target gateway ID to deploy to.",
						Optional:    true,
						Computed:    true,
					},
					"target_name": schema.StringAttribute{
						Description: "The target gateway name.",
						Optional:    true,
						Computed:    true,
					},
					"gateway_version": schema.StringAttribute{
						Description: "The Flex Gateway runtime version.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
			"routing": schema.ListNestedAttribute{
				Description: "Routing rules with weighted upstream backends.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Description: "A label for this route.",
							Optional:    true,
						},
						"rules": schema.SingleNestedAttribute{
							Description: "Match conditions for this route (methods, path, headers).",
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"methods": schema.StringAttribute{
									Description: "Pipe-separated HTTP methods (e.g. 'GET', 'POST|PUT').",
									Optional:    true,
								},
								"path": schema.StringAttribute{
									Description: "URL path pattern to match (e.g. '/api/*').",
									Optional:    true,
								},
								"host": schema.StringAttribute{
									Description: "Host header value to match.",
									Optional:    true,
								},
								"headers": schema.MapAttribute{
									Description: "Header key-value pairs to match.",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
						"upstreams": schema.ListNestedAttribute{
							Description: "Weighted upstream backends for this route.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"weight": schema.Int64Attribute{
										Description: "Traffic weight percentage (0-100). Weights across upstreams should sum to 100. Defaults to 100.",
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(100),
									},
									"uri": schema.StringAttribute{
										Description: "The upstream backend URI.",
										Required:    true,
									},
									"label": schema.StringAttribute{
										Description: "A label for this upstream.",
										Optional:    true,
									},
									"tls_context_id": schema.StringAttribute{
										Description: "TLS context for upstream connections. Format: 'secretGroupId/tlsContextId'.",
										Optional:    true,
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

func (r *AgentInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	agentClient, err := agentstools.NewAgentInstanceClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Agent Instance Client",
			"An unexpected error occurred when creating the Agent Instance client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = agentClient
}

func (r *AgentInstanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AgentInstanceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasUpstreamURI := !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown()
	hasRouting := !data.Routing.IsNull() && !data.Routing.IsUnknown()

	if hasUpstreamURI && hasRouting {
		resp.Diagnostics.AddError(
			"Conflicting configuration",
			"'upstream_uri' and 'routing' are mutually exclusive. Use 'upstream_uri' for simple single-upstream routing, or 'routing' for advanced multi-route/multi-upstream configuration.",
		)
		return
	}

	if !hasRouting {
		return
	}

	var routes []RouteModel
	resp.Diagnostics.Append(data.Routing.ElementsAs(ctx, &routes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, route := range routes {
		if route.Upstreams.IsNull() || route.Upstreams.IsUnknown() {
			continue
		}

		var upstreams []UpstreamModel
		resp.Diagnostics.Append(route.Upstreams.ElementsAs(ctx, &upstreams, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Agent instances only support a single upstream
		if len(upstreams) > 1 {
			routeLabel := fmt.Sprintf("routing[%d]", i)
			if !route.Label.IsNull() && !route.Label.IsUnknown() && route.Label.ValueString() != "" {
				routeLabel = fmt.Sprintf("routing[%d] (label=%q)", i, route.Label.ValueString())
			}
			resp.Diagnostics.AddError(
				"Multiple upstreams not supported",
				fmt.Sprintf("%s: agent_instance only supports a single upstream per route (got %d upstreams). "+
					"Multi-upstream routing with weighted traffic distribution is only available for api_instance resources. "+
					"For agent instances, use 'upstream_uri' or a single upstream in the 'routing' block.",
					routeLabel, len(upstreams)),
			)
		}
	}
}

// --- CRUD ---

func (r *AgentInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AgentInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	if !data.GatewayID.IsNull() && !data.GatewayID.IsUnknown() && (data.Deployment.IsNull() || data.Deployment.IsUnknown()) {
		dep, err := r.resolveGatewayDeployment(ctx, orgID, envID, data.GatewayID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error resolving gateway_id", err.Error())
			return
		}
		data.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(dep.EnvironmentID),
			Type:           types.StringValue(dep.Type),
			ExpectedStatus: types.StringValue(dep.ExpectedStatus),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue(dep.TargetID),
			TargetName:     types.StringValue(dep.TargetName),
			GatewayVersion: types.StringValue(dep.GatewayVersion),
		})
	}

	createReq := r.expandCreateRequest(ctx, data)

	instance, err := r.client.CreateAgentInstance(ctx, orgID, envID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Agent instance", "Could not create Agent instance: "+err.Error())
		return
	}

	gatewayID := data.GatewayID
	plannedRouting := data.Routing
	plannedEndpoint := data.Endpoint
	plannedDeployment := data.Deployment
	plannedConsumerEndpoint := data.ConsumerEndpoint
	plannedUpstreamURI := data.UpstreamURI
	r.flattenInstance(ctx, instance, &data, orgID, envID)
	data.GatewayID = gatewayID
	if !plannedUpstreamURI.IsNull() && !plannedUpstreamURI.IsUnknown() {
		data.UpstreamURI = plannedUpstreamURI
		data.Routing = types.ListNull(data.Routing.ElementType(ctx))
	} else if !plannedRouting.IsNull() {
		data.Routing = plannedRouting
	}
	if !plannedEndpoint.IsNull() && !plannedEndpoint.IsUnknown() {
		data.Endpoint = plannedEndpoint
	}
	data.Deployment = mergeDeploymentObjects(data.Deployment, plannedDeployment)
	if !plannedConsumerEndpoint.IsNull() && !plannedConsumerEndpoint.IsUnknown() {
		data.ConsumerEndpoint = plannedConsumerEndpoint
	}
	tflog.Trace(ctx, "created Agent instance", map[string]interface{}{"id": instance.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	agentID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Agent Instance ID", "Could not parse Agent instance ID as integer: "+data.ID.ValueString())
		return
	}

	instance, err := r.client.GetAgentInstance(ctx, orgID, envID, agentID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Agent instance", "Could not read Agent instance ID "+data.ID.ValueString()+": "+err.Error())
		return
	}

	gatewayID := data.GatewayID
	existingRouting := data.Routing
	existingEndpoint := data.Endpoint
	existingDeployment := data.Deployment

	r.flattenInstance(ctx, instance, &data, orgID, envID)

	data.GatewayID = gatewayID
	if !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown() {
		data.Routing = types.ListNull(data.Routing.ElementType(ctx))
	} else if !existingRouting.IsNull() && !existingRouting.IsUnknown() {
		data.Routing = existingRouting
	}
	if !existingEndpoint.IsNull() && !existingEndpoint.IsUnknown() {
		data.Endpoint = existingEndpoint
	}
	data.Deployment = mergeDeploymentObjects(data.Deployment, existingDeployment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AgentInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AgentInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := state.EnvironmentID.ValueString()

	agentID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Agent Instance ID", "Could not parse Agent instance ID: "+state.ID.ValueString())
		return
	}

	if !plan.GatewayID.IsNull() && !plan.GatewayID.IsUnknown() && (plan.Deployment.IsNull() || plan.Deployment.IsUnknown()) {
		var dep *agentstools.AgentInstanceDeployment
		dep, err = r.resolveGatewayDeployment(ctx, orgID, envID, plan.GatewayID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error resolving gateway_id", err.Error())
			return
		}
		plan.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(dep.EnvironmentID),
			Type:           types.StringValue(dep.Type),
			ExpectedStatus: types.StringValue(dep.ExpectedStatus),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue(dep.TargetID),
			TargetName:     types.StringValue(dep.TargetName),
			GatewayVersion: types.StringValue(dep.GatewayVersion),
		})
	}

	updateReq := r.expandUpdateRequest(ctx, plan)

	if len(updateReq.Routing) > 0 {
		var current *agentstools.AgentInstance
		current, err = r.client.GetAgentInstance(ctx, orgID, envID, agentID)
		if err == nil && len(current.Routing) > 0 {
			r.mergeUpstreamIDs(current.Routing, updateReq.Routing)
		}
	}

	var instance *agentstools.AgentInstance
	instance, err = r.client.UpdateAgentInstance(ctx, orgID, envID, agentID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Agent instance", "Could not update Agent instance: "+err.Error())
		return
	}

	gatewayID := plan.GatewayID
	plannedRouting := plan.Routing
	plannedEndpoint := plan.Endpoint
	plannedUpstreamURI := plan.UpstreamURI
	r.flattenInstance(ctx, instance, &plan, orgID, envID)
	plan.GatewayID = gatewayID
	if !plannedUpstreamURI.IsNull() && !plannedUpstreamURI.IsUnknown() {
		plan.UpstreamURI = plannedUpstreamURI
		plan.Routing = types.ListNull(plan.Routing.ElementType(ctx))
	} else if !plannedRouting.IsNull() {
		plan.Routing = plannedRouting
	}
	if !plannedEndpoint.IsNull() && !plannedEndpoint.IsUnknown() {
		plan.Endpoint = plannedEndpoint
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	agentID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Agent Instance ID", "Could not parse Agent instance ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteAgentInstance(ctx, orgID, envID, agentID); err != nil {
		resp.Diagnostics.AddError("Error deleting Agent instance", "Could not delete Agent instance: "+err.Error())
	}
}

func (r *AgentInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Helpers ---

// resolveGatewayDeployment fetches gateway details and builds a deployment payload.
func (r *AgentInstanceResource) resolveGatewayDeployment(ctx context.Context, orgID, envID, gatewayID string) (*agentstools.AgentInstanceDeployment, error) {
	gw, err := r.client.GetGatewayInfo(ctx, orgID, envID, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gateway_id %s: %w", gatewayID, err)
	}
	return &agentstools.AgentInstanceDeployment{
		EnvironmentID:  envID,
		Type:           "HY",
		ExpectedStatus: "deployed",
		TargetID:       gw.ID,
		TargetName:     gw.Name,
		GatewayVersion: gw.RuntimeVersion,
	}, nil
}

func (r *AgentInstanceResource) expandCreateRequest(ctx context.Context, data AgentInstanceResourceModel) *agentstools.CreateAgentInstanceRequest {
	req := &agentstools.CreateAgentInstanceRequest{
		Technology: data.Technology.ValueString(),
	}

	if !data.ProviderID.IsNull() && !data.ProviderID.IsUnknown() {
		pid := data.ProviderID.ValueString()
		req.ProviderID = &pid
	}
	if !data.InstanceLabel.IsNull() && !data.InstanceLabel.IsUnknown() {
		req.InstanceLabel = data.InstanceLabel.ValueString()
	}
	if !data.ApprovalMethod.IsNull() && !data.ApprovalMethod.IsUnknown() {
		am := data.ApprovalMethod.ValueString()
		req.ApprovalMethod = &am
	}

	if data.Spec != nil {
		req.Spec = &agentstools.AgentInstanceSpec{
			AssetID: data.Spec.AssetID.ValueString(),
			GroupID: data.Spec.GroupID.ValueString(),
			Version: data.Spec.Version.ValueString(),
		}
	}

	if ep := endpointFromObject(data.Endpoint); ep != nil {
		req.Endpoint = &agentstools.AgentInstanceEndpoint{
			DeploymentType: ep.DeploymentType.ValueString(),
			Type:           "a2a",
		}

		technology := data.Technology.ValueString()
		if technology == "flexGateway" || technology == "" {
			if !ep.BasePath.IsNull() && !ep.BasePath.IsUnknown() {
				basePath := strings.TrimPrefix(ep.BasePath.ValueString(), "/")
				proxyURI := "http://0.0.0.0:8081/" + basePath
				req.Endpoint.ProxyURI = &proxyURI
			} else {
				proxyURI := "http://0.0.0.0:8081/"
				req.Endpoint.ProxyURI = &proxyURI
			}

			req.Endpoint.TLSContexts = &agentstools.AgentInstanceTLSContexts{}
		} else if technology == "mule4" {
			mule4 := true
			req.Endpoint.MuleVersion4OrAbove = &mule4
			req.Endpoint.ProxyURI = nil

			if !ep.URI.IsNull() && !ep.URI.IsUnknown() {
				uri := ep.URI.ValueString()
				req.Endpoint.URI = &uri
			}

			req.Endpoint.IsCloudHub = nil
			req.Endpoint.ReferencesUserDomain = nil
		}

		if !ep.ResponseTimeout.IsNull() && !ep.ResponseTimeout.IsUnknown() {
			rt := int(ep.ResponseTimeout.ValueInt64())
			req.Endpoint.ResponseTimeout = &rt
		}
	}

	if !data.ConsumerEndpoint.IsNull() && !data.ConsumerEndpoint.IsUnknown() {
		ce := data.ConsumerEndpoint.ValueString()
		req.EndpointURI = &ce
	}

	if dep := deploymentFromObject(data.Deployment); dep != nil {
		req.Deployment = &agentstools.AgentInstanceDeployment{
			EnvironmentID:  dep.EnvironmentID.ValueString(),
			Type:           dep.Type.ValueString(),
			ExpectedStatus: dep.ExpectedStatus.ValueString(),
			Overwrite:      dep.Overwrite.ValueBool(),
			TargetID:       dep.TargetID.ValueString(),
			TargetName:     dep.TargetName.ValueString(),
			GatewayVersion: dep.GatewayVersion.ValueString(),
		}
	}

	if !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown() {
		req.Routing = []agentstools.AgentInstanceRoute{
			{
				Upstreams: []agentstools.AgentInstanceUpstream{
					{Weight: 100, URI: data.UpstreamURI.ValueString()},
				},
			},
		}
	} else {
		req.Routing = r.expandRouting(ctx, data.Routing)
	}

	return req
}

func (r *AgentInstanceResource) expandUpdateRequest(ctx context.Context, data AgentInstanceResourceModel) *agentstools.UpdateAgentInstanceRequest {
	req := &agentstools.UpdateAgentInstanceRequest{}

	tech := data.Technology.ValueString()
	req.Technology = &tech

	if !data.InstanceLabel.IsNull() && !data.InstanceLabel.IsUnknown() {
		il := data.InstanceLabel.ValueString()
		req.InstanceLabel = &il
	}

	if ep := endpointFromObject(data.Endpoint); ep != nil {
		req.Endpoint = &agentstools.AgentInstanceEndpoint{
			DeploymentType: ep.DeploymentType.ValueString(),
			Type:           "a2a",
		}

		technology := data.Technology.ValueString()
		if technology == "flexGateway" || technology == "" {
			if !ep.BasePath.IsNull() && !ep.BasePath.IsUnknown() {
				basePath := strings.TrimPrefix(ep.BasePath.ValueString(), "/")
				proxyURI := "http://0.0.0.0:8081/" + basePath
				req.Endpoint.ProxyURI = &proxyURI
			} else {
				proxyURI := "http://0.0.0.0:8081/"
				req.Endpoint.ProxyURI = &proxyURI
			}

			req.Endpoint.TLSContexts = &agentstools.AgentInstanceTLSContexts{}
		} else if technology == "mule4" {
			mule4 := true
			req.Endpoint.MuleVersion4OrAbove = &mule4
			req.Endpoint.ProxyURI = nil

			if !ep.URI.IsNull() && !ep.URI.IsUnknown() {
				uri := ep.URI.ValueString()
				req.Endpoint.ProxyURI = &uri
			}

			req.Endpoint.IsCloudHub = nil
			req.Endpoint.ReferencesUserDomain = nil
		}

		if !ep.ResponseTimeout.IsNull() && !ep.ResponseTimeout.IsUnknown() {
			rt := int(ep.ResponseTimeout.ValueInt64())
			req.Endpoint.ResponseTimeout = &rt
		}
	}

	if !data.ConsumerEndpoint.IsNull() && !data.ConsumerEndpoint.IsUnknown() {
		ce := data.ConsumerEndpoint.ValueString()
		req.EndpointURI = &ce
	}

	if data.Spec != nil {
		req.Spec = &agentstools.AgentInstanceSpec{
			AssetID: data.Spec.AssetID.ValueString(),
			GroupID: data.Spec.GroupID.ValueString(),
			Version: data.Spec.Version.ValueString(),
		}
	}

	if dep := deploymentFromObject(data.Deployment); dep != nil {
		req.Deployment = &agentstools.AgentInstanceDeployment{
			EnvironmentID:  dep.EnvironmentID.ValueString(),
			Type:           dep.Type.ValueString(),
			ExpectedStatus: dep.ExpectedStatus.ValueString(),
			Overwrite:      dep.Overwrite.ValueBool(),
			TargetID:       dep.TargetID.ValueString(),
			TargetName:     dep.TargetName.ValueString(),
			GatewayVersion: dep.GatewayVersion.ValueString(),
		}
	}

	if !data.UpstreamURI.IsNull() && !data.UpstreamURI.IsUnknown() {
		req.Routing = []agentstools.AgentInstanceRoute{
			{
				Upstreams: []agentstools.AgentInstanceUpstream{
					{Weight: 100, URI: data.UpstreamURI.ValueString()},
				},
			},
		}
	} else {
		req.Routing = r.expandRouting(ctx, data.Routing)
	}

	return req
}

// mergeUpstreamIDs copies server-assigned upstream IDs from the current
// instance into the update payload. Matches by route index and upstream URI.
func (r *AgentInstanceResource) mergeUpstreamIDs(current, update []agentstools.AgentInstanceRoute) {
	for i := range update {
		if i >= len(current) {
			break
		}
		// Try URI-based matching first.
		currentByURI := make(map[string]string)
		for _, us := range current[i].Upstreams {
			if us.ID != "" && us.URI != "" {
				currentByURI[us.URI] = us.ID
			}
		}
		for j := range update[i].Upstreams {
			if id, ok := currentByURI[update[i].Upstreams[j].URI]; ok {
				update[i].Upstreams[j].ID = id
				update[i].Upstreams[j].URI = ""
			}
		}
		// Fallback: match by position for any upstreams that still lack an ID.
		for j := range update[i].Upstreams {
			if update[i].Upstreams[j].ID == "" && j < len(current[i].Upstreams) && current[i].Upstreams[j].ID != "" {
				update[i].Upstreams[j].ID = current[i].Upstreams[j].ID
				update[i].Upstreams[j].URI = ""
			}
		}
	}
}

func (r *AgentInstanceResource) expandRouting(ctx context.Context, routingList types.List) []agentstools.AgentInstanceRoute {
	if routingList.IsNull() || routingList.IsUnknown() {
		return nil
	}

	var routes []RouteModel
	routingList.ElementsAs(ctx, &routes, false)

	agentRoutes := make([]agentstools.AgentInstanceRoute, 0, len(routes))
	for _, route := range routes {
		agentRoute := agentstools.AgentInstanceRoute{
			Label: route.Label.ValueString(),
		}

		if route.Rules != nil {
			rules := &agentstools.AgentInstanceRules{}
			if !route.Rules.Methods.IsNull() && !route.Rules.Methods.IsUnknown() {
				rules.Methods = route.Rules.Methods.ValueString()
			}
			if !route.Rules.Path.IsNull() && !route.Rules.Path.IsUnknown() {
				rules.Path = route.Rules.Path.ValueString()
			}
			if !route.Rules.Host.IsNull() && !route.Rules.Host.IsUnknown() {
				rules.Host = route.Rules.Host.ValueString()
			}
			if !route.Rules.Headers.IsNull() && !route.Rules.Headers.IsUnknown() {
				hdrs := make(map[string]string)
				route.Rules.Headers.ElementsAs(ctx, &hdrs, false)
				rules.Headers = hdrs
			}
			agentRoute.Rules = rules
		}

		var upstreams []UpstreamModel
		route.Upstreams.ElementsAs(ctx, &upstreams, false)

		for _, us := range upstreams {
			upstream := agentstools.AgentInstanceUpstream{
				Weight: int(us.Weight.ValueInt64()),
				URI:    us.URI.ValueString(),
				Label:  us.Label.ValueString(),
			}
			if !us.TLSContextID.IsNull() && !us.TLSContextID.IsUnknown() {
				parts := strings.Split(us.TLSContextID.ValueString(), "/")
				if len(parts) == 2 {
					upstream.TLSContext = &agentstools.AgentInstanceUpstreamTLS{
						SecretGroupID: parts[0],
						TLSContextID:  parts[1],
					}
				}
			}
			agentRoute.Upstreams = append(agentRoute.Upstreams, upstream)
		}

		agentRoutes = append(agentRoutes, agentRoute)
	}

	return agentRoutes
}

func (r *AgentInstanceResource) flattenInstance(_ context.Context, inst *agentstools.AgentInstance, data *AgentInstanceResourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(inst.ID))
	if inst.Status != "" {
		data.Status = types.StringValue(inst.Status)
	}
	if inst.AssetID != "" {
		data.AssetID = types.StringValue(inst.AssetID)
	}
	if inst.AssetVersion != "" {
		data.AssetVersion = types.StringValue(inst.AssetVersion)
	}
	if inst.ProductVersion != "" {
		data.ProductVersion = types.StringValue(inst.ProductVersion)
	}
	if inst.Technology != "" {
		data.Technology = types.StringValue(inst.Technology)
	}

	if data.OrganizationID.IsNull() || data.OrganizationID.IsUnknown() || data.OrganizationID.ValueString() == "" {
		data.OrganizationID = types.StringValue(orgID)
	}
	data.EnvironmentID = types.StringValue(envID)

	if inst.ProviderID != nil {
		data.ProviderID = types.StringValue(*inst.ProviderID)
	}
	if inst.InstanceLabel != "" {
		data.InstanceLabel = types.StringValue(inst.InstanceLabel)
	}
	if inst.ApprovalMethod != "" {
		data.ApprovalMethod = types.StringValue(inst.ApprovalMethod)
	}

	if inst.Spec != nil {
		data.Spec = &SpecModel{
			AssetID: types.StringValue(inst.Spec.AssetID),
			GroupID: types.StringValue(inst.Spec.GroupID),
			Version: types.StringValue(inst.Spec.Version),
		}
	}

	if inst.Endpoint != nil {
		ep := &EndpointModel{
			DeploymentType: types.StringValue(inst.Endpoint.DeploymentType),
			Type:           types.StringValue(inst.Endpoint.Type),
		}

		technology := inst.Technology
		if technology == "flexGateway" || technology == "" {
			if inst.Endpoint.ProxyURI != nil && *inst.Endpoint.ProxyURI != "" {
				ep.BasePath = types.StringValue(strings.TrimPrefix(*inst.Endpoint.ProxyURI, "http://0.0.0.0:8081/"))
			} else {
				ep.BasePath = types.StringNull()
			}
			ep.URI = types.StringNull()
		} else if technology == "mule4" {
			if inst.Endpoint.URI != nil && *inst.Endpoint.URI != "" {
				ep.URI = types.StringValue(*inst.Endpoint.URI)
			} else {
				ep.URI = types.StringNull()
			}
			ep.BasePath = types.StringNull()
		} else {
			ep.BasePath = types.StringNull()
			ep.URI = types.StringNull()
		}

		if inst.Endpoint.ResponseTimeout != nil {
			ep.ResponseTimeout = types.Int64Value(int64(*inst.Endpoint.ResponseTimeout))
		} else {
			ep.ResponseTimeout = types.Int64Null()
		}
		data.Endpoint = endpointToObject(ep)
	} else {
		data.Endpoint = types.ObjectNull(endpointAttrTypes)
	}

	if inst.EndpointURI != "" {
		data.ConsumerEndpoint = types.StringValue(inst.EndpointURI)
	} else {
		data.ConsumerEndpoint = types.StringNull()
	}

	if inst.Deployment != nil {
		data.Deployment = deploymentToObject(&DeploymentModel{
			EnvironmentID:  types.StringValue(inst.Deployment.EnvironmentID),
			Type:           types.StringValue(inst.Deployment.Type),
			ExpectedStatus: types.StringValue(inst.Deployment.ExpectedStatus),
			Overwrite:      types.BoolValue(inst.Deployment.Overwrite),
			TargetID:       types.StringValue(inst.Deployment.TargetID),
			TargetName:     types.StringValue(inst.Deployment.TargetName),
			GatewayVersion: types.StringValue(inst.Deployment.GatewayVersion),
		})
	} else {
		data.Deployment = types.ObjectNull(deploymentAttrTypes)
	}

	// Flatten routing into types.List
	rulesObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"methods": types.StringType,
			"path":    types.StringType,
			"host":    types.StringType,
			"headers": types.MapType{ElemType: types.StringType},
		},
	}
	upstreamObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"weight":         types.Int64Type,
			"uri":            types.StringType,
			"label":          types.StringType,
			"tls_context_id": types.StringType,
		},
	}
	routeObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"label":     types.StringType,
			"rules":     rulesObjType,
			"upstreams": types.ListType{ElemType: upstreamObjType},
		},
	}

	if len(inst.Routing) > 0 {
		routeElems := make([]attr.Value, 0, len(inst.Routing))
		for _, route := range inst.Routing {
			usElems := make([]attr.Value, 0, len(route.Upstreams))
			for _, us := range route.Upstreams {
				var tlsCtxVal attr.Value
				if us.TLSContext != nil && us.TLSContext.SecretGroupID != "" && us.TLSContext.TLSContextID != "" {
					tlsCtxVal = types.StringValue(fmt.Sprintf("%s/%s", us.TLSContext.SecretGroupID, us.TLSContext.TLSContextID))
				} else {
					tlsCtxVal = types.StringNull()
				}
				usObj, usDiags := types.ObjectValue(
					upstreamObjType.AttrTypes,
					map[string]attr.Value{
						"weight":         types.Int64Value(int64(us.Weight)),
						"uri":            types.StringValue(us.URI),
						"label":          types.StringValue(us.Label),
						"tls_context_id": tlsCtxVal,
					},
				)
				if usDiags.HasError() {
					continue
				}
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
				rulesObj, rulesDiags := types.ObjectValue(
					rulesObjType.AttrTypes,
					map[string]attr.Value{
						"methods": methods,
						"path":    pathVal,
						"host":    hostVal,
						"headers": headersVal,
					},
				)
				if rulesDiags.HasError() {
					rulesVal = types.ObjectNull(rulesObjType.AttrTypes)
				} else {
					rulesVal = rulesObj
				}
			} else {
				rulesVal = types.ObjectNull(rulesObjType.AttrTypes)
			}

			routeObj, routeDiags := types.ObjectValue(
				routeObjType.AttrTypes,
				map[string]attr.Value{
					"label":     types.StringValue(route.Label),
					"rules":     rulesVal,
					"upstreams": types.ListValueMust(upstreamObjType, usElems),
				},
			)
			if !routeDiags.HasError() {
				routeElems = append(routeElems, routeObj)
			}
		}

		data.Routing = types.ListValueMust(routeObjType, routeElems)
	} else {
		data.Routing = types.ListNull(routeObjType)
	}
}
