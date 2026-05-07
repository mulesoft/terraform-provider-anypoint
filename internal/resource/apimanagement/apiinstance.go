package apimanagement

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
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                   = &APIInstanceResource{}
	_ resource.ResourceWithConfigure      = &APIInstanceResource{}
	_ resource.ResourceWithImportState    = &APIInstanceResource{}
	_ resource.ResourceWithValidateConfig = &APIInstanceResource{}
)

type APIInstanceResource struct {
	client *apimanagement.APIInstanceClient
}

// --- Terraform State Models ---

type APIInstanceResourceModel struct {
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

type SpecModel struct {
	AssetID types.String `tfsdk:"asset_id"`
	GroupID types.String `tfsdk:"group_id"`
	Version types.String `tfsdk:"version"`
}

type EndpointModel struct {
	DeploymentType  types.String `tfsdk:"deployment_type"`
	Type            types.String `tfsdk:"type"`
	BasePath        types.String `tfsdk:"base_path"`
	URI             types.String `tfsdk:"uri"`
	ResponseTimeout types.Int64  `tfsdk:"response_timeout"`
}

type DeploymentModel struct {
	EnvironmentID  types.String `tfsdk:"environment_id"`
	Type           types.String `tfsdk:"type"`
	ExpectedStatus types.String `tfsdk:"expected_status"`
	Overwrite      types.Bool   `tfsdk:"overwrite"`
	TargetID       types.String `tfsdk:"target_id"`
	TargetName     types.String `tfsdk:"target_name"`
	GatewayVersion types.String `tfsdk:"gateway_version"`
}

// endpointAttrTypes maps EndpointModel field names to their Terraform types.
// endpoint is Optional+Computed; when omitted from config with no prior state the
// framework marks the planned value as unknown. types.Object can hold an unknown
// value — *EndpointModel cannot, which causes a Value Conversion Error at apply.
var endpointAttrTypes = map[string]attr.Type{
	"deployment_type":  types.StringType,
	"type":             types.StringType,
	"base_path":        types.StringType,
	"uri":              types.StringType,
	"response_timeout": types.Int64Type,
}

// endpointFromObject extracts an EndpointModel from a types.Object.
// Returns nil when the object is null or unknown.
func endpointFromObject(obj types.Object) *EndpointModel {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	attrs := obj.Attributes()
	return &EndpointModel{
		DeploymentType:  attrs["deployment_type"].(types.String),
		Type:            attrs["type"].(types.String),
		BasePath:        attrs["base_path"].(types.String),
		URI:             attrs["uri"].(types.String),
		ResponseTimeout: attrs["response_timeout"].(types.Int64),
	}
}

// endpointToObject converts an EndpointModel into a types.Object for state storage.
// Returns a null object when ep is nil.
func endpointToObject(ep *EndpointModel) types.Object {
	if ep == nil {
		return types.ObjectNull(endpointAttrTypes)
	}
	obj, diags := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
		"deployment_type":  ep.DeploymentType,
		"type":             ep.Type,
		"base_path":        ep.BasePath,
		"uri":              ep.URI,
		"response_timeout": ep.ResponseTimeout,
	})
	if diags.HasError() {
		return types.ObjectNull(endpointAttrTypes)
	}
	return obj
}

// deploymentAttrTypes maps DeploymentModel field names to their Terraform types.
// This is required to represent deployment as types.Object (which handles unknown/null
// during plan for Computed attributes — *DeploymentModel cannot).
var deploymentAttrTypes = map[string]attr.Type{
	"environment_id":  types.StringType,
	"type":            types.StringType,
	"expected_status": types.StringType,
	"overwrite":       types.BoolType,
	"target_id":       types.StringType,
	"target_name":     types.StringType,
	"gateway_version": types.StringType,
}

// deploymentFromObject extracts a DeploymentModel from a types.Object.
// Returns nil when the object is null or unknown.
func deploymentFromObject(obj types.Object) *DeploymentModel {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	attrs := obj.Attributes()
	return &DeploymentModel{
		EnvironmentID:  attrs["environment_id"].(types.String),
		Type:           attrs["type"].(types.String),
		ExpectedStatus: attrs["expected_status"].(types.String),
		Overwrite:      attrs["overwrite"].(types.Bool),
		TargetID:       attrs["target_id"].(types.String),
		TargetName:     attrs["target_name"].(types.String),
		GatewayVersion: attrs["gateway_version"].(types.String),
	}
}

// deploymentToObject converts a DeploymentModel into a types.Object for state storage.
// Returns a null object when dep is nil.
func deploymentToObject(dep *DeploymentModel) types.Object {
	if dep == nil {
		return types.ObjectNull(deploymentAttrTypes)
	}
	obj, diags := types.ObjectValue(deploymentAttrTypes, map[string]attr.Value{
		"environment_id":  dep.EnvironmentID,
		"type":            dep.Type,
		"expected_status": dep.ExpectedStatus,
		"overwrite":       dep.Overwrite,
		"target_id":       dep.TargetID,
		"target_name":     dep.TargetName,
		"gateway_version": dep.GatewayVersion,
	})
	if diags.HasError() {
		return types.ObjectNull(deploymentAttrTypes)
	}
	return obj
}

// mergeDeploymentObjects returns apiDep with any known (non-null, non-unknown)
// attributes from plannedDep applied on top. This prevents Computed sub-attributes
// that were unknown in the plan (e.g. environment_id) from surviving in state as
// unknown after apply — they are filled from the API response instead.
func mergeDeploymentObjects(apiDep, plannedDep types.Object) types.Object {
	if plannedDep.IsNull() || plannedDep.IsUnknown() {
		return apiDep
	}
	if apiDep.IsNull() || apiDep.IsUnknown() {
		return plannedDep
	}
	planned := deploymentFromObject(plannedDep)
	api := deploymentFromObject(apiDep)
	if planned == nil {
		return apiDep
	}
	if api == nil {
		return plannedDep
	}
	merged := *api
	if !planned.EnvironmentID.IsNull() && !planned.EnvironmentID.IsUnknown() {
		merged.EnvironmentID = planned.EnvironmentID
	}
	if !planned.Type.IsNull() && !planned.Type.IsUnknown() {
		merged.Type = planned.Type
	}
	if !planned.ExpectedStatus.IsNull() && !planned.ExpectedStatus.IsUnknown() {
		merged.ExpectedStatus = planned.ExpectedStatus
	}
	if !planned.Overwrite.IsNull() && !planned.Overwrite.IsUnknown() {
		merged.Overwrite = planned.Overwrite
	}
	if !planned.TargetID.IsNull() && !planned.TargetID.IsUnknown() {
		merged.TargetID = planned.TargetID
	}
	if !planned.TargetName.IsNull() && !planned.TargetName.IsUnknown() {
		merged.TargetName = planned.TargetName
	}
	if !planned.GatewayVersion.IsNull() && !planned.GatewayVersion.IsUnknown() {
		merged.GatewayVersion = planned.GatewayVersion
	}
	return deploymentToObject(&merged)
}

type RouteModel struct {
	Label     types.String `tfsdk:"label"`
	Rules     *RulesModel  `tfsdk:"rules"`
	Upstreams types.List   `tfsdk:"upstreams"`
}

type RulesModel struct {
	Methods types.String `tfsdk:"methods"`
	Path    types.String `tfsdk:"path"`
	Host    types.String `tfsdk:"host"`
	Headers types.Map    `tfsdk:"headers"`
}

type UpstreamModel struct {
	Weight       types.Int64  `tfsdk:"weight"`
	URI          types.String `tfsdk:"uri"`
	Label        types.String `tfsdk:"label"`
	TLSContextID types.String `tfsdk:"tls_context_id"`
}

func NewAPIInstanceResource() resource.Resource {
	return &APIInstanceResource{}
}

func (r *APIInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance"
}

func (r *APIInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API instance in Anypoint API Manager. An API instance represents " +
			"an API specification deployed to a Omni Gateway target with routing rules and upstream backends.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric identifier of the API instance (stored as string for Terraform compatibility).",
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
				Description: "The environment ID where the API instance will be created.",
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
				Description: "The identity provider ID for the API.",
				Optional:    true,
			},
			"instance_label": schema.StringAttribute{
				Description: "A human-readable label for this API instance.",
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
				Description: "The current status of the API instance.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "The Exchange asset ID (computed from API response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"asset_version": schema.StringAttribute{
				Description: "The Exchange asset version (computed from API response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"product_version": schema.StringAttribute{
				Description: "The product version (computed from API response).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"spec": schema.SingleNestedAttribute{
				Description: "The Exchange asset specification backing this API instance.",
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
				Description: "Endpoint / proxy configuration for the API instance.",
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
						Description: "Endpoint protocol type. Valid values: 'http', 'rest', 'raml'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("http"),
						Validators: []validator.String{
							stringvalidator.OneOf("http", "rest", "raml"),
						},
					},
					"base_path": schema.StringAttribute{
						Description: "API base path for OmniGateway (e.g. 'my-api'). The provider constructs the full proxy URI as http://0.0.0.0:8081/<base_path>. Required when technology='omniGateway'. Mutually exclusive with 'uri'.",
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
				Description: "Consumer-facing endpoint URI (the public URL clients use to reach the API). Maps to top-level endpointUri in the API.",
				Optional:    true,
			},
			"upstream_uri": schema.StringAttribute{
				Description: "Shorthand for a single-upstream routing configuration. " +
					"When set, the provider constructs routing as [{upstreams: [{weight: 100, uri: <value>}]}]. " +
					"Mutually exclusive with the 'routing' block.",
				Optional: true,
			},
			"gateway_id": schema.StringAttribute{
				Description: "The Omni Gateway UUID. When provided, the deployment block is auto-populated " +
					"by fetching gateway details (target_id, target_name, gateway_version) from the Gateway Manager API. " +
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
						Description: "The Omni Gateway runtime version.",
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

func (r *APIInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	apiClient, err := apimanagement.NewAPIInstanceClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Instance Client",
			"An unexpected error occurred when creating the API Instance client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = apiClient
}

func (r *APIInstanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data APIInstanceResourceModel
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

		if len(upstreams) <= 1 {
			continue
		}

		var totalWeight int64
		allKnown := true
		for _, us := range upstreams {
			if us.Weight.IsUnknown() {
				allKnown = false
				break
			}
			totalWeight += us.Weight.ValueInt64()
		}

		if allKnown && totalWeight != 100 {
			routeLabel := fmt.Sprintf("routing[%d]", i)
			if !route.Label.IsNull() && !route.Label.IsUnknown() && route.Label.ValueString() != "" {
				routeLabel = fmt.Sprintf("routing[%d] (label=%q)", i, route.Label.ValueString())
			}
			resp.Diagnostics.AddError(
				"Invalid upstream weights",
				fmt.Sprintf("%s: upstream weights must sum to 100 when there are multiple upstreams, got %d.", routeLabel, totalWeight),
			)
		}
	}
}

// --- CRUD ---

func (r *APIInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIInstanceResourceModel
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

	instance, err := r.client.CreateAPIInstance(ctx, orgID, envID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating API instance", "Could not create API instance: "+err.Error())
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
	tflog.Trace(ctx, "created API instance", map[string]interface{}{"id": instance.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	apiID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID", "Could not parse API instance ID as integer: "+data.ID.ValueString())
		return
	}

	instance, err := r.client.GetAPIInstance(ctx, orgID, envID, apiID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API instance", "Could not read API instance ID "+data.ID.ValueString()+": "+err.Error())
		return
	}

	// Snapshot user-managed fields before overwriting with API response.
	// The Anypoint Platform may silently modify routing upstreams, endpoint
	// details, or deployment fields after creation. If we let those API-side
	// changes propagate into state, Terraform will generate a spurious diff
	// and attempt an update that often fails (e.g. "upstream uri is mandatory").
	// User-managed fields are owned exclusively by Terraform config/state.
	gatewayID := data.GatewayID
	existingRouting := data.Routing
	existingEndpoint := data.Endpoint
	existingDeployment := data.Deployment

	r.flattenInstance(ctx, instance, &data, orgID, envID)

	// Restore user-managed fields from state.
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

func (r *APIInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state APIInstanceResourceModel
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

	apiID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID", "Could not parse API instance ID: "+state.ID.ValueString())
		return
	}

	if !plan.GatewayID.IsNull() && !plan.GatewayID.IsUnknown() && (plan.Deployment.IsNull() || plan.Deployment.IsUnknown()) {
		var dep *apimanagement.APIInstanceDeployment
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
		var current *apimanagement.APIInstance
		current, err = r.client.GetAPIInstance(ctx, orgID, envID, apiID)
		if err == nil && len(current.Routing) > 0 {
			r.mergeUpstreamIDs(current.Routing, updateReq.Routing)
		}
	}

	var instance *apimanagement.APIInstance
	instance, err = r.client.UpdateAPIInstance(ctx, orgID, envID, apiID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating API instance", "Could not update API instance: "+err.Error())
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

func (r *APIInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	apiID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API Instance ID", "Could not parse API instance ID: "+data.ID.ValueString())
		return
	}

	if err := r.client.DeleteAPIInstance(ctx, orgID, envID, apiID); err != nil {
		resp.Diagnostics.AddError("Error deleting API instance", "Could not delete API instance: "+err.Error())
	}
}

func (r *APIInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Helpers ---

// resolveGatewayDeployment fetches gateway details and builds a deployment payload.
func (r *APIInstanceResource) resolveGatewayDeployment(ctx context.Context, orgID, envID, gatewayID string) (*apimanagement.APIInstanceDeployment, error) {
	gw, err := r.client.GetGatewayInfo(ctx, orgID, envID, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve gateway_id %s: %w", gatewayID, err)
	}
	return &apimanagement.APIInstanceDeployment{
		EnvironmentID:  envID,
		Type:           "HY",
		ExpectedStatus: "deployed",
		TargetID:       gw.ID,
		TargetName:     gw.Name,
		GatewayVersion: gw.RuntimeVersion,
	}, nil
}

func (r *APIInstanceResource) expandCreateRequest(ctx context.Context, data APIInstanceResourceModel) *apimanagement.CreateAPIInstanceRequest {
	req := &apimanagement.CreateAPIInstanceRequest{
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
		req.Spec = &apimanagement.APIInstanceSpec{
			AssetID: data.Spec.AssetID.ValueString(),
			GroupID: data.Spec.GroupID.ValueString(),
			Version: data.Spec.Version.ValueString(),
		}
	}

	if ep := endpointFromObject(data.Endpoint); ep != nil {
		req.Endpoint = &apimanagement.APIInstanceEndpoint{
			DeploymentType: ep.DeploymentType.ValueString(),
			Type:           ep.Type.ValueString(),
		}

		technology := data.Technology.ValueString()
		switch technology {
		case "flexGateway", "":
			if !ep.BasePath.IsNull() && !ep.BasePath.IsUnknown() {
				basePath := strings.TrimPrefix(ep.BasePath.ValueString(), "/")
				proxyURI := "http://0.0.0.0:8081/" + basePath
				req.Endpoint.ProxyURI = &proxyURI
			} else {
				proxyURI := "http://0.0.0.0:8081/"
				req.Endpoint.ProxyURI = &proxyURI
			}

			req.Endpoint.TLSContexts = &apimanagement.APIInstanceTLSContexts{}
		case "mule4":
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
		req.Deployment = &apimanagement.APIInstanceDeployment{
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
		req.Routing = []apimanagement.APIInstanceRoute{
			{
				Upstreams: []apimanagement.APIInstanceUpstream{
					{Weight: 100, URI: data.UpstreamURI.ValueString()},
				},
			},
		}
	} else {
		req.Routing = r.expandRouting(ctx, data.Routing)
	}

	return req
}

func (r *APIInstanceResource) expandUpdateRequest(ctx context.Context, data APIInstanceResourceModel) *apimanagement.UpdateAPIInstanceRequest {
	req := &apimanagement.UpdateAPIInstanceRequest{}

	tech := data.Technology.ValueString()
	req.Technology = &tech

	if !data.InstanceLabel.IsNull() && !data.InstanceLabel.IsUnknown() {
		il := data.InstanceLabel.ValueString()
		req.InstanceLabel = &il
	}

	if ep := endpointFromObject(data.Endpoint); ep != nil {
		req.Endpoint = &apimanagement.APIInstanceEndpoint{
			DeploymentType: ep.DeploymentType.ValueString(),
			Type:           ep.Type.ValueString(),
		}

		technology := data.Technology.ValueString()
		switch technology {
		case "flexGateway", "":
			if !ep.BasePath.IsNull() && !ep.BasePath.IsUnknown() {
				basePath := strings.TrimPrefix(ep.BasePath.ValueString(), "/")
				proxyURI := "http://0.0.0.0:8081/" + basePath
				req.Endpoint.ProxyURI = &proxyURI
			} else {
				proxyURI := "http://0.0.0.0:8081/"
				req.Endpoint.ProxyURI = &proxyURI
			}

			req.Endpoint.TLSContexts = &apimanagement.APIInstanceTLSContexts{}
		case "mule4":
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

	if data.Spec != nil {
		req.Spec = &apimanagement.APIInstanceSpec{
			AssetID: data.Spec.AssetID.ValueString(),
			GroupID: data.Spec.GroupID.ValueString(),
			Version: data.Spec.Version.ValueString(),
		}
	}

	if dep := deploymentFromObject(data.Deployment); dep != nil {
		req.Deployment = &apimanagement.APIInstanceDeployment{
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
		req.Routing = []apimanagement.APIInstanceRoute{
			{
				Upstreams: []apimanagement.APIInstanceUpstream{
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
func (r *APIInstanceResource) mergeUpstreamIDs(current, update []apimanagement.APIInstanceRoute) {
	for i := range update {
		if i >= len(current) {
			break
		}
		currentByURI := make(map[string]string)
		for _, us := range current[i].Upstreams {
			if us.ID != "" {
				currentByURI[us.URI] = us.ID
			}
		}
		for j := range update[i].Upstreams {
			if id, ok := currentByURI[update[i].Upstreams[j].URI]; ok {
				update[i].Upstreams[j].ID = id
			}
		}
	}
}

func (r *APIInstanceResource) expandRouting(ctx context.Context, routingList types.List) []apimanagement.APIInstanceRoute {
	if routingList.IsNull() || routingList.IsUnknown() {
		return nil
	}

	var routes []RouteModel
	routingList.ElementsAs(ctx, &routes, false)

	apiRoutes := make([]apimanagement.APIInstanceRoute, 0, len(routes))
	for _, route := range routes {
		apiRoute := apimanagement.APIInstanceRoute{
			Label: route.Label.ValueString(),
		}

		if route.Rules != nil {
			rules := &apimanagement.APIInstanceRules{}
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
			apiRoute.Rules = rules
		}

		var upstreams []UpstreamModel
		route.Upstreams.ElementsAs(ctx, &upstreams, false)

		for _, us := range upstreams {
			upstream := apimanagement.APIInstanceUpstream{
				Weight: int(us.Weight.ValueInt64()),
				URI:    us.URI.ValueString(),
				Label:  us.Label.ValueString(),
			}
			if !us.TLSContextID.IsNull() && !us.TLSContextID.IsUnknown() {
				parts := strings.Split(us.TLSContextID.ValueString(), "/")
				if len(parts) == 2 {
					upstream.TLSContext = &apimanagement.APIInstanceUpstreamTLS{
						SecretGroupID: parts[0],
						TLSContextID:  parts[1],
					}
				}
			}
			apiRoute.Upstreams = append(apiRoute.Upstreams, upstream)
		}

		apiRoutes = append(apiRoutes, apiRoute)
	}

	return apiRoutes
}

func (r *APIInstanceResource) flattenInstance(_ context.Context, inst *apimanagement.APIInstance, data *APIInstanceResourceModel, orgID, envID string) {
	data.ID = types.StringValue(strconv.Itoa(inst.ID))
	data.Status = types.StringValue(inst.Status)
	data.AssetID = types.StringValue(inst.AssetID)
	data.AssetVersion = types.StringValue(inst.AssetVersion)
	data.ProductVersion = types.StringValue(inst.ProductVersion)
	data.Technology = types.StringValue(inst.Technology)

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
		switch technology {
		case "flexGateway", "":
			if inst.Endpoint.ProxyURI != nil && *inst.Endpoint.ProxyURI != "" {
				ep.BasePath = types.StringValue(strings.TrimPrefix(*inst.Endpoint.ProxyURI, "http://0.0.0.0:8081/"))
			} else {
				ep.BasePath = types.StringNull()
			}
			ep.URI = types.StringNull()
		case "mule4":
			if inst.Endpoint.URI != nil && *inst.Endpoint.URI != "" {
				ep.URI = types.StringValue(*inst.Endpoint.URI)
			} else {
				ep.URI = types.StringNull()
			}
			ep.BasePath = types.StringNull()
		default:
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
