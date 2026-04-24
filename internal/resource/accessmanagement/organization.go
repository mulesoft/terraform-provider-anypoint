package accessmanagement

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource              = &OrganizationResource{}
	_ resource.ResourceWithConfigure = &OrganizationResource{}
)

// OrganizationResource is the resource implementation.
type OrganizationResource struct {
	client *accessmanagement.OrganizationClient
}

type VCoreEntitlementModel struct {
	Assigned   types.Float64 `tfsdk:"assigned"`
	Reassigned types.Float64 `tfsdk:"reassigned"`
}

type AssignedEntitlementModel struct {
	Assigned types.Int64 `tfsdk:"assigned"`
}

type EnabledEntitlementModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type HybridEntitlementModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type WorkerLoggingOverrideEntitlementModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type MqEntitlementModel struct {
	Base  types.Int64 `tfsdk:"base"`
	AddOn types.Int64 `tfsdk:"add_on"`
}

type DesignCenterEntitlementModel struct {
	API    types.Bool `tfsdk:"api"`
	Mozart types.Bool `tfsdk:"mozart"`
}

type MonitoringCenterEntitlementModel struct {
	ProductSKU           types.Int64 `tfsdk:"product_sku"`
	RawStorageOverrideGB types.Int64 `tfsdk:"raw_storage_override_gb"`
}

// EntitlementsModel represents the entitlements model
type EntitlementsModel struct {
	CreateSubOrgs         types.Bool   `tfsdk:"create_sub_orgs"`
	CreateEnvironments    types.Bool   `tfsdk:"create_environments"`
	GlobalDeployment      types.Bool   `tfsdk:"global_deployment"`
	Hybrid                types.Object `tfsdk:"hybrid"`
	VCoresProduction      types.Object `tfsdk:"vcores_production"`
	VCoresSandbox         types.Object `tfsdk:"vcores_sandbox"`
	VCoresDesign          types.Object `tfsdk:"vcores_design"`
	StaticIps             types.Object `tfsdk:"static_ips"`
	Vpcs                  types.Object `tfsdk:"vpcs"`
	Vpns                  types.Object `tfsdk:"vpns"`
	NetworkConnections    types.Object `tfsdk:"network_connections"`
	WorkerLoggingOverride types.Object `tfsdk:"worker_logging_override"`
	MqMessages            types.Object `tfsdk:"mq_messages"`
	MqRequests            types.Object `tfsdk:"mq_requests"`
	Gateways              types.Object `tfsdk:"gateways"`
	DesignCenter          types.Object `tfsdk:"design_center"`
	LoadBalancer          types.Object `tfsdk:"load_balancer"`
	RuntimeFabric         types.Bool   `tfsdk:"runtime_fabric"`
	ServiceMesh           types.Object `tfsdk:"service_mesh"`
	FlexGateway           types.Object `tfsdk:"flex_gateway"`
	ManagedGatewaySmall   types.Object `tfsdk:"managed_gateway_small"`
	ManagedGatewayLarge   types.Object `tfsdk:"managed_gateway_large"`
}

// SubscriptionModel represents the subscription model
type SubscriptionModel struct {
	Category   types.String `tfsdk:"category"`
	Type       types.String `tfsdk:"type"`
	Expiration types.String `tfsdk:"expiration"`
}

// OrgEnvironmentModel represents the environment model
type OrgEnvironmentModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	IsProduction   types.Bool   `tfsdk:"is_production"`
	Type           types.String `tfsdk:"type"`
	ClientID       types.String `tfsdk:"client_id"`
}

// OrganizationResourceModel describes the resource data model.
type OrganizationResourceModel struct {
	ID                              types.String `tfsdk:"id"`
	Name                            types.String `tfsdk:"name"`
	ParentOrganizationID            types.String `tfsdk:"parent_organization_id"`
	OwnerID                         types.String `tfsdk:"owner_id"`
	Entitlements                    types.Object `tfsdk:"entitlements"`
	CreatedAt                       types.String `tfsdk:"created_at"`
	UpdatedAt                       types.String `tfsdk:"updated_at"`
	ClientID                        types.String `tfsdk:"client_id"`
	IdProviderID                    types.String `tfsdk:"idprovider_id"`
	IsFederated                     types.Bool   `tfsdk:"is_federated"`
	ParentOrganizationIds           types.List   `tfsdk:"parent_organization_ids"`
	SubOrganizationIds              types.List   `tfsdk:"sub_organization_ids"`
	TenantOrganizationIds           types.List   `tfsdk:"tenant_organization_ids"`
	MfaRequired                     types.String `tfsdk:"mfa_required"`
	IsAutomaticAdminPromotionExempt types.Bool   `tfsdk:"is_automatic_admin_promotion_exempt"`
	OrgType                         types.String `tfsdk:"org_type"`
	GdotID                          types.String `tfsdk:"gdot_id"`
	DeletedAt                       types.String `tfsdk:"deleted_at"`
	Domain                          types.String `tfsdk:"domain"`
	IsRoot                          types.Bool   `tfsdk:"is_root"`
	IsMaster                        types.Bool   `tfsdk:"is_master"`
	Subscription                    types.Object `tfsdk:"subscription"`
	Environments                    types.List   `tfsdk:"environments"`
	SessionTimeout                  types.Int64  `tfsdk:"session_timeout"`
}

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

// Metadata returns the resource type name.
func (r *OrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the resource.
func (r *OrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Anypoint Platform organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_organization_id": schema.StringAttribute{
				Description: "The ID of the parent organization.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_id": schema.StringAttribute{
				Description: "The ID of the organization owner.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"session_timeout": schema.Int64Attribute{
				Description: "The session timeout for the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"subscription": schema.SingleNestedAttribute{
				Description: "The subscription details for the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"category": schema.StringAttribute{
						Description: "The subscription category.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "The subscription type.",
						Computed:    true,
					},
					"expiration": schema.StringAttribute{
						Description: "The subscription expiration date.",
						Computed:    true,
					},
					"justification": schema.StringAttribute{
						Description: "The subscription justification.",
						Computed:    true,
						Optional:    true,
					},
				},
			},
			"environments": schema.ListNestedAttribute{
				Description: "The environments within the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The environment ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The environment name.",
							Computed:    true,
						},
						"organization_id": schema.StringAttribute{
							Description: "The organization ID.",
							Computed:    true,
						},
						"is_production": schema.BoolAttribute{
							Description: "Whether the environment is a production environment.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The environment type.",
							Computed:    true,
						},
						"client_id": schema.StringAttribute{
							Description: "The environment client ID.",
							Computed:    true,
						},
						"arc_namespace": schema.StringAttribute{
							Description: "The ARC namespace of the environment.",
							Computed:    true,
							Optional:    true,
						},
					},
				},
			},
			"entitlements": schema.SingleNestedAttribute{
				Description: "Entitlements for the organization.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"create_sub_orgs": schema.BoolAttribute{
						Description: "Whether sub-organizations can be created.",
						Required:    true,
					},
					"create_environments": schema.BoolAttribute{
						Description: "Whether environments can be created.",
						Required:    true,
					},
					"global_deployment": schema.BoolAttribute{
						Description: "Whether global deployment is enabled.",
						Required:    true,
					},
				"vcores_production": schema.SingleNestedAttribute{
					Description: "Production vCore entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
				"vcores_sandbox": schema.SingleNestedAttribute{
					Description: "Sandbox vCore entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
				"vcores_design": schema.SingleNestedAttribute{
					Description: "Design vCore entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
				"static_ips": schema.SingleNestedAttribute{
					Description: "Static IP entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
				"vpcs": schema.SingleNestedAttribute{
					Description: "VPC entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
				"vpns": schema.SingleNestedAttribute{
					Description: "VPN entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
				"network_connections": schema.SingleNestedAttribute{
					Description: "Network connections entitlement.",
					Optional:    true,
					Computed:    true,
					Attributes:  getVCoreEntitlementSchema(),
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
				},
					"hybrid": schema.SingleNestedAttribute{
						Description: "Hybrid entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getEnabledEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"runtime_fabric": schema.BoolAttribute{
						Description: "Whether Runtime Fabric is enabled.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"flex_gateway": schema.SingleNestedAttribute{
						Description: "Flex Gateway entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getEnabledEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"worker_logging_override": schema.SingleNestedAttribute{
						Description: "Worker logging override entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getEnabledEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"mq_messages": schema.SingleNestedAttribute{
						Description: "MQ messages entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getMqEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"mq_requests": schema.SingleNestedAttribute{
						Description: "MQ requests entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getMqEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"gateways": schema.SingleNestedAttribute{
						Description: "Gateways entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getAssignedEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"design_center": schema.SingleNestedAttribute{
						Description: "Design Center entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getDesignCenterEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"load_balancer": schema.SingleNestedAttribute{
						Description: "Load balancer entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getAssignedEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"service_mesh": schema.SingleNestedAttribute{
						Description: "Service Mesh entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getEnabledEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"managed_gateway_small": schema.SingleNestedAttribute{
						Description: "Managed Gateway (small) entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getAssignedEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"managed_gateway_large": schema.SingleNestedAttribute{
						Description: "Managed Gateway (large) entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getAssignedEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			// Computed attributes from API response
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID associated with the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"idprovider_id": schema.StringAttribute{
				Description: "The ID provider ID for the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_federated": schema.BoolAttribute{
				Description: "Whether the organization is federated.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_organization_ids": schema.ListAttribute{
				Description: "List of parent organization IDs.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"sub_organization_ids": schema.ListAttribute{
				Description: "List of sub-organization IDs.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_organization_ids": schema.ListAttribute{
				Description: "List of tenant organization IDs.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"mfa_required": schema.StringAttribute{
				Description: "Whether MFA is required for the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_automatic_admin_promotion_exempt": schema.BoolAttribute{
				Description: "Whether the organization is exempt from automatic admin promotion.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"org_type": schema.StringAttribute{
				Description: "The type of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gdot_id": schema.StringAttribute{
				Description: "The GDOT ID of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deleted_at": schema.StringAttribute{
				Description: "The deletion timestamp of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "The domain of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_root": schema.BoolAttribute{
				Description: "Whether the organization is a root organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_master": schema.BoolAttribute{
				Description: "Whether the organization is a master organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func getVCoreEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"assigned": schema.Float64Attribute{
			Description: "The number of assigned units.",
			Required:    true,
		},
		"reassigned": schema.Float64Attribute{
			Description: "The number of reassigned units. Defaults to 0 if not provided.",
			Optional:    true,
			Computed:    true,
		},
	}
}

func getMqEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base": schema.Int64Attribute{
			Description: "The base number of MQ units.",
			Required:    true,
		},
		"add_on": schema.Int64Attribute{
			Description: "The add-on number of MQ units.",
			Required:    true,
		},
	}
}

func getEntitlementsAttributeTypes() map[string]attr.Type {
	vcoreType := types.ObjectType{AttrTypes: getVCoreEntitlementAttributeTypes()}
	enabledType := types.ObjectType{AttrTypes: map[string]attr.Type{"enabled": types.BoolType}}
	assignedType := types.ObjectType{AttrTypes: map[string]attr.Type{"assigned": types.Int64Type}}
	mqType := types.ObjectType{AttrTypes: getMqEntitlementAttributeTypes()}
	return map[string]attr.Type{
		"create_sub_orgs":          types.BoolType,
		"create_environments":      types.BoolType,
		"global_deployment":        types.BoolType,
		"vcores_production":        vcoreType,
		"vcores_sandbox":           vcoreType,
		"vcores_design":            vcoreType,
		"static_ips":               vcoreType,
		"vpcs":                     vcoreType,
		"vpns":                     vcoreType,
		"network_connections":      vcoreType,
		"hybrid":                   enabledType,
		"runtime_fabric":           types.BoolType,
		"flex_gateway":             enabledType,
		"worker_logging_override":  enabledType,
		"mq_messages":              mqType,
		"mq_requests":              mqType,
		"gateways":                 assignedType,
		"design_center":            types.ObjectType{AttrTypes: map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType}},
		"load_balancer":            assignedType,
		"service_mesh":             enabledType,
		"managed_gateway_small":    assignedType,
		"managed_gateway_large":    assignedType,
	}
}

func getEnabledEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Description: "Whether this feature is enabled.",
			Optional:    true,
			Computed:    true,
		},
	}
}

func getAssignedEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"assigned": schema.Int64Attribute{
			Description: "The number of assigned units.",
			Optional:    true,
			Computed:    true,
		},
	}
}

func getDesignCenterEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api": schema.BoolAttribute{
			Description: "Whether API Designer is enabled.",
			Optional:    true,
			Computed:    true,
		},
		"mozart": schema.BoolAttribute{
			Description: "Whether Flow Designer (Mozart) is enabled.",
			Optional:    true,
			Computed:    true,
		},
	}
}

func getVCoreEntitlementAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"assigned":   types.Float64Type,
		"reassigned": types.Float64Type,
	}
}

func getMqEntitlementAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"base":   types.Int64Type,
		"add_on": types.Int64Type,
	}
}

func getSubscriptionAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"category":      types.StringType,
		"type":          types.StringType,
		"expiration":    types.StringType,
		"justification": types.StringType,
	}
}

func getEnvironmentsAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":              types.StringType,
		"name":            types.StringType,
		"organization_id": types.StringType,
		"is_production":   types.BoolType,
		"type":            types.StringType,
		"client_id":       types.StringType,
		"arc_namespace":   types.StringType,
	}
}

func getMonitoringCenterEntitlementAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"product_sku":             types.Int64Type,
		"raw_storage_override_gb": types.Int64Type,
	}
}

// expandEntitlements converts a Terraform types.Object into the client Entitlements struct.
func expandEntitlements(ctx context.Context, obj types.Object) (accessmanagement.Entitlements, diag.Diagnostics) {
	var diags diag.Diagnostics
	var model EntitlementsModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return accessmanagement.Entitlements{}, diags
	}

	ent := accessmanagement.Entitlements{
		CreateSubOrgs:      model.CreateSubOrgs.ValueBool(),
		CreateEnvironments: model.CreateEnvironments.ValueBool(),
		GlobalDeployment:   model.GlobalDeployment.ValueBool(),
		RuntimeFabric:      model.RuntimeFabric.ValueBool(),
	}

	ent.VCoresProduction = vCoreEntitlementFromModel(model.VCoresProduction)
	ent.VCoresSandbox = vCoreEntitlementFromModel(model.VCoresSandbox)
	ent.VCoresDesign = vCoreEntitlementFromModel(model.VCoresDesign)
	ent.StaticIps = vCoreEntitlementFromModel(model.StaticIps)
	ent.Vpcs = vCoreEntitlementFromModel(model.Vpcs)
	ent.Vpns = vCoreEntitlementFromModel(model.Vpns)
	ent.NetworkConnections = vCoreEntitlementFromModel(model.NetworkConnections)

	ent.Hybrid = hybridEntitlementFromModel(model.Hybrid)
	ent.FlexGateway = enabledEntitlementFromModel(model.FlexGateway)
	ent.WorkerLoggingOverride = workerLoggingOverrideEntitlementFromModel(model.WorkerLoggingOverride)
	ent.MqMessages = mqEntitlementFromModel(model.MqMessages)
	ent.MqRequests = mqEntitlementFromModel(model.MqRequests)
	ent.Gateways = assignedEntitlementFromModel(model.Gateways)
	ent.DesignCenter = designCenterEntitlementFromModel(model.DesignCenter)
	ent.LoadBalancer = assignedEntitlementFromModel(model.LoadBalancer)
	ent.ServiceMesh = enabledEntitlementFromModel(model.ServiceMesh)
	ent.ManagedGatewaySmall = assignedEntitlementFromModel(model.ManagedGatewaySmall)
	ent.ManagedGatewayLarge = assignedEntitlementFromModel(model.ManagedGatewayLarge)

	return ent, diags
}

// nullEnabled returns a null types.Object for an {enabled bool} attribute type.
func nullEnabled() types.Object {
	return types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType})
}

// nullAssigned returns a null types.Object for an {assigned int64} attribute type.
func nullAssigned() types.Object {
	return types.ObjectNull(map[string]attr.Type{"assigned": types.Int64Type})
}

// flattenEntitlements converts the client Entitlements struct into a Terraform types.Object.
func flattenEntitlements(ctx context.Context, ent accessmanagement.Entitlements) (types.Object, diag.Diagnostics) {
	var hybrid, flexGateway, workerLogging, serviceMesh types.Object
	if ent.Hybrid != nil {
		hybrid = hybridEntitlementToModel(ent.Hybrid)
	} else {
		hybrid = nullEnabled()
	}
	if ent.FlexGateway != nil {
		flexGateway = enabledEntitlementToModel(ent.FlexGateway)
	} else {
		flexGateway = nullEnabled()
	}
	if ent.WorkerLoggingOverride != nil {
		workerLogging = workerLoggingOverrideEntitlementToModel(ent.WorkerLoggingOverride)
	} else {
		workerLogging = nullEnabled()
	}
	if ent.ServiceMesh != nil {
		serviceMesh = enabledEntitlementToModel(ent.ServiceMesh)
	} else {
		serviceMesh = nullEnabled()
	}

	mqNull := types.ObjectNull(getMqEntitlementAttributeTypes())
	var mqMessages, mqRequests types.Object
	if ent.MqMessages != nil {
		mqMessages = mqEntitlementToModel(ent.MqMessages)
	} else {
		mqMessages = mqNull
	}
	if ent.MqRequests != nil {
		mqRequests = mqEntitlementToModel(ent.MqRequests)
	} else {
		mqRequests = mqNull
	}

	var gateways, loadBalancer, gwSmall, gwLarge types.Object
	if ent.Gateways != nil {
		gateways = assignedEntitlementToModel(ent.Gateways)
	} else {
		gateways = nullAssigned()
	}
	if ent.LoadBalancer != nil {
		loadBalancer = assignedEntitlementToModel(ent.LoadBalancer)
	} else {
		loadBalancer = nullAssigned()
	}
	if ent.ManagedGatewaySmall != nil {
		gwSmall = assignedEntitlementToModel(ent.ManagedGatewaySmall)
	} else {
		gwSmall = nullAssigned()
	}
	if ent.ManagedGatewayLarge != nil {
		gwLarge = assignedEntitlementToModel(ent.ManagedGatewayLarge)
	} else {
		gwLarge = nullAssigned()
	}

	dcNull := types.ObjectNull(map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType})
	var designCenter types.Object
	if ent.DesignCenter != nil {
		designCenter = designCenterEntitlementToModel(ent.DesignCenter)
	} else {
		designCenter = dcNull
	}

	vcoreNull := types.ObjectNull(getVCoreEntitlementAttributeTypes())

	model := EntitlementsModel{
		CreateSubOrgs:         types.BoolValue(ent.CreateSubOrgs),
		CreateEnvironments:    types.BoolValue(ent.CreateEnvironments),
		GlobalDeployment:      types.BoolValue(ent.GlobalDeployment),
		RuntimeFabric:         types.BoolValue(ent.RuntimeFabric),
		VCoresProduction:      vcoreOrNull(ent.VCoresProduction, vcoreNull),
		VCoresSandbox:         vcoreOrNull(ent.VCoresSandbox, vcoreNull),
		VCoresDesign:          vcoreOrNull(ent.VCoresDesign, vcoreNull),
		StaticIps:             vcoreOrNull(ent.StaticIps, vcoreNull),
		Vpcs:                  vcoreOrNull(ent.Vpcs, vcoreNull),
		Vpns:                  vcoreOrNull(ent.Vpns, vcoreNull),
		NetworkConnections:    vcoreOrNull(ent.NetworkConnections, vcoreNull),
		Hybrid:                hybrid,
		FlexGateway:           flexGateway,
		WorkerLoggingOverride: workerLogging,
		MqMessages:            mqMessages,
		MqRequests:            mqRequests,
		Gateways:              gateways,
		DesignCenter:          designCenter,
		LoadBalancer:          loadBalancer,
		ServiceMesh:           serviceMesh,
		ManagedGatewaySmall:   gwSmall,
		ManagedGatewayLarge:   gwLarge,
	}

	return types.ObjectValueFrom(ctx, getEntitlementsAttributeTypes(), model)
}

func (r *OrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Create user client config for organization operations (requires user authentication)
	userConfig := &client.UserClientConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		BaseURL:      config.BaseURL,
		Timeout:      config.Timeout,
		Username:     config.Username,
		Password:     config.Password,
	}

	orgClient, err := accessmanagement.NewOrganizationClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create OrganizationClient",
			fmt.Sprintf("Could not create OrganizationClient: %s", err),
		)
		return
	}

	r.client = orgClient
}

// Helper function to convert VCoreEntitlementModel to accessmanagement.VCoreEntitlement
func vCoreEntitlementFromModel(model types.Object) *accessmanagement.VCoreEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel VCoreEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting vCore entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.VCoreEntitlement{
		Assigned:   entitlementModel.Assigned.ValueFloat64(),
		Reassigned: entitlementModel.Reassigned.ValueFloat64(),
	}
}

func vCoreEntitlementToModel(vcore *accessmanagement.VCoreEntitlement) types.Object {
	if vcore == nil {
		return types.ObjectNull(getVCoreEntitlementAttributeTypes())
	}

	attrs := map[string]attr.Value{
		"assigned":   types.Float64Value(vcore.Assigned),
		"reassigned": types.Float64Value(vcore.Reassigned),
	}

	objVal, _ := types.ObjectValue(getVCoreEntitlementAttributeTypes(), attrs)
	return objVal
}

func vcoreOrNull(vcore *accessmanagement.VCoreEntitlement, nullVal types.Object) types.Object {
	if vcore != nil {
		return vCoreEntitlementToModel(vcore)
	}
	return nullVal
}

func assignedEntitlementFromModel(model types.Object) *accessmanagement.AssignedEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel AssignedEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting assigned entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.AssignedEntitlement{
		Assigned: int(entitlementModel.Assigned.ValueInt64()),
	}
}

func assignedEntitlementToModel(entitlement *accessmanagement.AssignedEntitlement) types.Object {
	if entitlement == nil {
		entitlement = &accessmanagement.AssignedEntitlement{Assigned: 0}
	}

	attrs := map[string]attr.Value{
		"assigned": types.Int64Value(int64(entitlement.Assigned)),
	}

	attrTypes := map[string]attr.Type{
		"assigned": types.Int64Type,
	}

	objVal, _ := types.ObjectValue(attrTypes, attrs)
	return objVal
}

func enabledEntitlementFromModel(model types.Object) *accessmanagement.EnabledEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel EnabledEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting enabled entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.EnabledEntitlement{
		Enabled: entitlementModel.Enabled.ValueBool(),
	}
}

func enabledEntitlementToModel(entitlement *accessmanagement.EnabledEntitlement) types.Object {
	if entitlement == nil {
		entitlement = &accessmanagement.EnabledEntitlement{Enabled: false}
	}

	attrs := map[string]attr.Value{
		"enabled": types.BoolValue(entitlement.Enabled),
	}

	attrTypes := map[string]attr.Type{
		"enabled": types.BoolType,
	}

	objVal, _ := types.ObjectValue(attrTypes, attrs)
	return objVal
}

func hybridEntitlementFromModel(model types.Object) *accessmanagement.HybridEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel HybridEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting hybrid entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.HybridEntitlement{
		Enabled: entitlementModel.Enabled.ValueBool(),
	}
}

func hybridEntitlementToModel(entitlement *accessmanagement.HybridEntitlement) types.Object {
	if entitlement == nil {
		entitlement = &accessmanagement.HybridEntitlement{Enabled: false}
	}

	attrs := map[string]attr.Value{
		"enabled": types.BoolValue(entitlement.Enabled),
	}

	attrTypes := map[string]attr.Type{
		"enabled": types.BoolType,
	}

	objVal, _ := types.ObjectValue(attrTypes, attrs)
	return objVal
}

func workerLoggingOverrideEntitlementFromModel(model types.Object) *accessmanagement.WorkerLoggingOverrideEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel WorkerLoggingOverrideEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting worker logging override entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.WorkerLoggingOverrideEntitlement{
		Enabled: entitlementModel.Enabled.ValueBool(),
	}
}

func workerLoggingOverrideEntitlementToModel(entitlement *accessmanagement.WorkerLoggingOverrideEntitlement) types.Object {
	if entitlement == nil {
		entitlement = &accessmanagement.WorkerLoggingOverrideEntitlement{Enabled: false}
	}

	attrs := map[string]attr.Value{
		"enabled": types.BoolValue(entitlement.Enabled),
	}

	attrTypes := map[string]attr.Type{
		"enabled": types.BoolType,
	}

	objVal, _ := types.ObjectValue(attrTypes, attrs)
	return objVal
}

func mqEntitlementFromModel(model types.Object) *accessmanagement.MqEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel MqEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting mq entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.MqEntitlement{
		Base:  int(entitlementModel.Base.ValueInt64()),
		AddOn: int(entitlementModel.AddOn.ValueInt64()),
	}
}

func mqEntitlementToModel(entitlement *accessmanagement.MqEntitlement) types.Object {
	if entitlement == nil {
		entitlement = &accessmanagement.MqEntitlement{Base: 0, AddOn: 0}
	}

	attrs := map[string]attr.Value{
		"base":   types.Int64Value(int64(entitlement.Base)),
		"add_on": types.Int64Value(int64(entitlement.AddOn)),
	}

	attrTypes := map[string]attr.Type{
		"base":   types.Int64Type,
		"add_on": types.Int64Type,
	}

	objVal, _ := types.ObjectValue(attrTypes, attrs)
	return objVal
}

func designCenterEntitlementFromModel(model types.Object) *accessmanagement.DesignCenterEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel DesignCenterEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting design center entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.DesignCenterEntitlement{
		API:    entitlementModel.API.ValueBool(),
		Mozart: entitlementModel.Mozart.ValueBool(),
	}
}

func designCenterEntitlementToModel(entitlement *accessmanagement.DesignCenterEntitlement) types.Object {
	if entitlement == nil {
		entitlement = &accessmanagement.DesignCenterEntitlement{API: false, Mozart: false}
	}

	attrs := map[string]attr.Value{
		"api":    types.BoolValue(entitlement.API),
		"mozart": types.BoolValue(entitlement.Mozart),
	}

	attrTypes := map[string]attr.Type{
		"api":    types.BoolType,
		"mozart": types.BoolType,
	}

	objVal, _ := types.ObjectValue(attrTypes, attrs)
	return objVal
}

func monitoringCenterEntitlementFromModel(model types.Object) *accessmanagement.MonitoringCenterEntitlement {
	if model.IsNull() || model.IsUnknown() {
		return nil
	}
	var entitlementModel MonitoringCenterEntitlementModel
	diags := model.As(context.Background(), &entitlementModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		tflog.Error(context.Background(), "Error converting monitoring center entitlement from model", map[string]interface{}{"diagnostics": diags})
		return nil
	}
	return &accessmanagement.MonitoringCenterEntitlement{
		ProductSKU:           int(entitlementModel.ProductSKU.ValueInt64()),
		RawStorageOverrideGB: int(entitlementModel.RawStorageOverrideGB.ValueInt64()),
	}
}

func monitoringCenterEntitlementToModel(entitlement *accessmanagement.MonitoringCenterEntitlement) types.Object {
	if entitlement == nil {
		return types.ObjectNull(getMonitoringCenterEntitlementAttributeTypes())
	}

	attributeTypes := getMonitoringCenterEntitlementAttributeTypes()
	attributeValues := map[string]attr.Value{
		"product_sku":             types.Int64Value(int64(entitlement.ProductSKU)),
		"raw_storage_override_gb": types.Int64Value(int64(entitlement.RawStorageOverrideGB)),
	}

	obj, diags := types.ObjectValue(attributeTypes, attributeValues)
	if diags.HasError() {
		// Handle error, maybe return a null object or log the diagnostics
		return types.ObjectNull(attributeTypes)
	}
	return obj
}

func subscriptionToModel(sub *accessmanagement.Subscription) types.Object {
	if sub == nil {
		return types.ObjectNull(getSubscriptionAttributeTypes())
	}
	obj, diags := types.ObjectValue(getSubscriptionAttributeTypes(), map[string]attr.Value{
		"category":   types.StringValue(sub.Category),
		"type":       types.StringValue(sub.Type),
		"expiration": types.StringValue(sub.Expiration),
	})
	if diags.HasError() {
		return types.ObjectNull(getSubscriptionAttributeTypes())
	}
	return obj
}

func environmentToModel(env *accessmanagement.OrgEnvironment) types.Object {
	if env == nil {
		return types.ObjectNull(getEnvironmentsAttributeTypes())
	}
	obj, diags := types.ObjectValue(getEnvironmentsAttributeTypes(), map[string]attr.Value{
		"id":              types.StringValue(env.ID),
		"name":            types.StringValue(env.Name),
		"organization_id": types.StringValue(env.OrganizationID),
		"is_production":   types.BoolValue(env.IsProduction),
		"type":            types.StringValue(env.Type),
		"client_id":       types.StringValue(env.ClientID),
	})
	if diags.HasError() {
		return types.ObjectNull(getEnvironmentsAttributeTypes())
	}
	return obj
}

// Create creates the resource and sets the initial Terraform state.
func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrganizationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare the request
	entitlements, entDiags := expandEntitlements(ctx, data.Entitlements)
	resp.Diagnostics.Append(entDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := accessmanagement.CreateOrganizationRequest{
		Name:                 data.Name.ValueString(),
		ParentOrganizationID: data.ParentOrganizationID.ValueString(),
		OwnerID:              data.OwnerID.ValueString(),
		Entitlements:         entitlements,
	}

	// Create the organization
	organization, err := r.client.CreateOrganization(ctx, &createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Anypoint Organization", fmt.Sprintf("Could not create organization, unexpected error: %s", err))
		return
	}

	// Set the ID
	data.ID = types.StringValue(organization.ID)

	// Set other attributes
	data.Name = types.StringValue(organization.Name)
	data.OwnerID = types.StringValue(organization.OwnerID)
	data.CreatedAt = types.StringValue(organization.CreatedAt)
	data.UpdatedAt = types.StringValue(organization.UpdatedAt)
	data.ClientID = types.StringValue(organization.ClientID)
	data.IdProviderID = types.StringValue(organization.IdProviderID)
	data.IsFederated = types.BoolValue(organization.IsFederated)
	// Set organization ID lists - always ensure known values
	if organization.ParentOrganizationIds != nil {
		data.ParentOrganizationIds, _ = types.ListValueFrom(ctx, types.StringType, organization.ParentOrganizationIds)
	} else {
		data.ParentOrganizationIds = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.SubOrganizationIds != nil {
		data.SubOrganizationIds, _ = types.ListValueFrom(ctx, types.StringType, organization.SubOrganizationIds)
	} else {
		data.SubOrganizationIds = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.TenantOrganizationIds != nil {
		data.TenantOrganizationIds, _ = types.ListValueFrom(ctx, types.StringType, organization.TenantOrganizationIds)
	} else {
		data.TenantOrganizationIds = types.ListValueMust(types.StringType, []attr.Value{})
	}
	if organization.MfaRequired != "" {
		data.MfaRequired = types.StringValue(organization.MfaRequired)
	} else {
		data.MfaRequired = types.StringNull()
	}
	data.IsAutomaticAdminPromotionExempt = types.BoolValue(organization.IsAutomaticAdminPromotionExempt)
	data.OrgType = types.StringValue(organization.OrgType)
	if organization.GdotID != nil {
		data.GdotID = types.StringValue(*organization.GdotID)
	} else {
		data.GdotID = types.StringNull()
	}
	if organization.DeletedAt != nil {
		data.DeletedAt = types.StringValue(*organization.DeletedAt)
	} else {
		data.DeletedAt = types.StringNull()
	}
	if organization.Domain != nil {
		data.Domain = types.StringValue(*organization.Domain)
	} else {
		data.Domain = types.StringNull()
	}
	data.IsRoot = types.BoolValue(organization.IsRoot)
	data.IsMaster = types.BoolValue(organization.IsMaster)

	// Set subscription - always set to a known value
	if !reflect.ValueOf(organization.Subscription).IsZero() {
		subscription, diags := types.ObjectValueFrom(ctx, getSubscriptionAttributeTypes(), organization.Subscription)
		resp.Diagnostics.Append(diags...)
		data.Subscription = subscription
	} else {
		// Set to null object if no subscription
		data.Subscription = types.ObjectNull(getSubscriptionAttributeTypes())
	}

	// Set environments - always set to a known value
	if len(organization.Environments) > 0 {
		environments := make([]attr.Value, len(organization.Environments))
		for i, env := range organization.Environments {
			environments[i] = environmentToModel(&env)
		}
		data.Environments = types.ListValueMust(types.ObjectType{AttrTypes: getEnvironmentsAttributeTypes()}, environments)
	} else {
		// Set to empty list if no environments
		data.Environments = types.ListValueMust(types.ObjectType{AttrTypes: getEnvironmentsAttributeTypes()}, []attr.Value{})
	}

	// Set session timeout
	data.SessionTimeout = types.Int64Value(int64(organization.SessionTimeout))

	// Flatten entitlements from the API response so all Optional+Computed fields
	// are resolved to known values (null or concrete) rather than staying unknown.
	entObj, entDiags := flattenEntitlements(ctx, organization.Entitlements)
	resp.Diagnostics.Append(entDiags...)
	if !resp.Diagnostics.HasError() {
		data.Entitlements = entObj
	}

	tflog.Trace(ctx, "created an organization")

	// Save the new state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrganizationResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the organization from the API
	organization, err := r.client.GetOrganization(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading organization",
			"Could not read organization ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(organization.ID)
	data.Name = types.StringValue(organization.Name)
	data.CreatedAt = types.StringValue(organization.CreatedAt)
	data.UpdatedAt = types.StringValue(organization.UpdatedAt)
	data.OwnerID = types.StringValue(organization.OwnerID)
	data.ClientID = types.StringValue(organization.ClientID)
	data.IdProviderID = types.StringValue(organization.IdProviderID)
	data.IsFederated = types.BoolValue(organization.IsFederated)
	data.IsAutomaticAdminPromotionExempt = types.BoolValue(organization.IsAutomaticAdminPromotionExempt)
	data.OrgType = types.StringValue(organization.OrgType)
	data.IsRoot = types.BoolValue(organization.IsRoot)
	data.IsMaster = types.BoolValue(organization.IsMaster)

	// Handle nullable fields
	if organization.MfaRequired != "" {
		data.MfaRequired = types.StringValue(organization.MfaRequired)
	} else {
		data.MfaRequired = types.StringNull()
	}

	if organization.GdotID != nil {
		data.GdotID = types.StringValue(*organization.GdotID)
	} else {
		data.GdotID = types.StringNull()
	}

	if organization.DeletedAt != nil {
		data.DeletedAt = types.StringValue(*organization.DeletedAt)
	} else {
		data.DeletedAt = types.StringNull()
	}

	if organization.Domain != nil {
		data.Domain = types.StringValue(*organization.Domain)
	} else {
		data.Domain = types.StringNull()
	}

	// Convert string slices to Terraform lists - always ensure known values
	if organization.ParentOrganizationIds != nil {
		parentOrgIds, diags := types.ListValueFrom(ctx, types.StringType, organization.ParentOrganizationIds)
		resp.Diagnostics.Append(diags...)
		data.ParentOrganizationIds = parentOrgIds
	} else {
		data.ParentOrganizationIds = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.SubOrganizationIds != nil {
		subOrgIds, diags := types.ListValueFrom(ctx, types.StringType, organization.SubOrganizationIds)
		resp.Diagnostics.Append(diags...)
		data.SubOrganizationIds = subOrgIds
	} else {
		data.SubOrganizationIds = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.TenantOrganizationIds != nil {
		tenantOrgIds, diags := types.ListValueFrom(ctx, types.StringType, organization.TenantOrganizationIds)
		resp.Diagnostics.Append(diags...)
		data.TenantOrganizationIds = tenantOrgIds
	} else {
		data.TenantOrganizationIds = types.ListValueMust(types.StringType, []attr.Value{})
	}

	data.SessionTimeout = types.Int64Value(int64(organization.SessionTimeout))

	// Map environments - always set to a known value
	if len(organization.Environments) > 0 {
		environments := make([]attr.Value, len(organization.Environments))
		for i, env := range organization.Environments {
			envObj, diags := types.ObjectValueFrom(ctx, getEnvironmentsAttributeTypes(), env)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			environments[i] = envObj
		}
		envs, diags := types.ListValue(types.ObjectType{AttrTypes: getEnvironmentsAttributeTypes()}, environments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Environments = envs
	} else {
		// Set to empty list if no environments
		data.Environments = types.ListValueMust(types.ObjectType{AttrTypes: getEnvironmentsAttributeTypes()}, []attr.Value{})
	}

	// Map subscription - always set to a known value
	if !reflect.ValueOf(organization.Subscription).IsZero() {
		subscription, diags := types.ObjectValueFrom(ctx, getSubscriptionAttributeTypes(), organization.Subscription)
		resp.Diagnostics.Append(diags...)
		data.Subscription = subscription
	} else {
		// Set to null object if no subscription
		data.Subscription = types.ObjectNull(getSubscriptionAttributeTypes())
	}

	// Map entitlements
	entObj, entDiags := flattenEntitlements(ctx, organization.Entitlements)
	resp.Diagnostics.Append(entDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Entitlements = entObj

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrganizationResourceModel

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Organizations cannot be updated, so we just keep the current state
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"Updating an organization is not supported by this resource. No changes were made.",
	)

	// Set the state to the current state to prevent automatic refresh
	// This avoids JSON key ordering issues with entitlements
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OrganizationResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := data.ID.ValueString()

	tflog.Info(ctx, "Deleting organization", map[string]interface{}{"id": organizationID})

	// Delete the organization
	err := r.client.DeleteOrganization(ctx, organizationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Organization",
			fmt.Sprintf("Could not delete organization ID %s: %s", organizationID, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Organization delete request sent, waiting for deletion to complete")

	// Wait for the organization to be fully deleted to prevent "name already used" errors
	// when recreating an organization with the same name
	// 15 retries (30 seconds) should be sufficient as deletion typically completes within 15 seconds
	err = r.client.WaitForOrganizationDeletion(ctx, organizationID, 15, 2*time.Second)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Organization Deletion Timeout",
			fmt.Sprintf("Organization delete request was sent, but could not verify deletion completed: %s. The organization may still be deleting.", err.Error()),
		)
		// Don't return error here - the delete request was sent successfully
		// The warning informs the user that there might be a delay
	}

	tflog.Info(ctx, "Organization deleted successfully", map[string]interface{}{"id": organizationID})
}
