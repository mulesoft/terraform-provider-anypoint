package accessmanagement

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
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
	_ resource.Resource                = &OrganizationResource{}
	_ resource.ResourceWithConfigure   = &OrganizationResource{}
	_ resource.ResourceWithImportState = &OrganizationResource{}
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

// EntitlementsModel represents the entitlements model exposed by the
// provider. `static_ips` and `vpns` are intentionally omitted: the Anypoint
// Access Management API treats them as server-managed attributes that are
// never settable via the organizations create/update payload. Including them
// here would (a) force users to declare them in HCL just to satisfy object
// validation and (b) mismatch the actual API contract.
type EntitlementsModel struct {
	CreateSubOrgs         types.Bool   `tfsdk:"create_sub_orgs"`
	CreateEnvironments    types.Bool   `tfsdk:"create_environments"`
	GlobalDeployment      types.Bool   `tfsdk:"global_deployment"`
	Hybrid                types.Object `tfsdk:"hybrid"`
	VCoresProduction      types.Object `tfsdk:"vcores_production"`
	VCoresSandbox         types.Object `tfsdk:"vcores_sandbox"`
	VCoresDesign          types.Object `tfsdk:"vcores_design"`
	Vpcs                  types.Object `tfsdk:"vpcs"`
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
	IDProviderID                    types.String `tfsdk:"idprovider_id"`
	IsFederated                     types.Bool   `tfsdk:"is_federated"`
	ParentOrganizationIDs           types.List   `tfsdk:"parent_organization_ids"`
	SubOrganizationIDs              types.List   `tfsdk:"sub_organization_ids"`
	TenantOrganizationIDs           types.List   `tfsdk:"tenant_organization_ids"`
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
				Description: "The name of the organization. Can be updated in place via PUT /accounts/api/organizations/{id}.",
				Required:    true,
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
				Description: "Entitlements for the organization. Optional — any omitted sub-attribute defaults to its zero value (false for booleans, 0 for quotas). Omitting the whole block is equivalent to declaring `entitlements = {}`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"create_sub_orgs": schema.BoolAttribute{
						Description: "Whether sub-organizations can be created. Defaults to false.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"create_environments": schema.BoolAttribute{
						Description: "Whether environments can be created. Defaults to false.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"global_deployment": schema.BoolAttribute{
						Description: "Whether global deployment is enabled. Defaults to false.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"vcores_production": schema.SingleNestedAttribute{
						Description: "Production vCore entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getVCoreEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroVCore()),
					},
					"vcores_sandbox": schema.SingleNestedAttribute{
						Description: "Sandbox vCore entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getVCoreEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroVCore()),
					},
					"vcores_design": schema.SingleNestedAttribute{
						Description: "Design vCore entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getVCoreEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroVCore()),
					},
					// NOTE: `static_ips` and `vpns` are deliberately absent.
					// The Anypoint Access Management API does not accept them
					// in the organizations create/update payload, so they are
					// not user-settable. Exposing them would have forced users
					// to populate placeholder blocks just to satisfy object
					// validation (e.g. "attributes static_ips and vpns are
					// required").
					"vpcs": schema.SingleNestedAttribute{
						Description: "VPC entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getVCoreEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroVCore()),
					},
					"network_connections": schema.SingleNestedAttribute{
						Description: "Network connections entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getVCoreEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroVCore()),
					},
					// hybrid / runtime_fabric / flex_gateway / service_mesh /
					// worker_logging_override / design_center are
					// MASTER-ORG-ONLY entitlements: on a sub-org their values
					// are inherited from the master and the Anypoint API
					// rewrites whatever the provider sends. Giving these a
					// schema-level Default would pin the plan to a concrete
					// `false` value while the API responds with the inherited
					// `true`, producing the "Provider produced inconsistent
					// result after apply" diagnostic on Create. Instead we
					// leave them as plain Optional+Computed and rely on
					// UseStateForUnknown so that once Read populates the
					// field from the API, subsequent plans don't show
					// perpetual `(known after apply)`.
					"hybrid": schema.SingleNestedAttribute{
						Description: "Hybrid entitlement. Inherited from the master organization on sub-orgs; declaring it explicitly on a business group has no effect.",
						Optional:    true,
						Computed:    true,
						Attributes:  getEnabledEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"runtime_fabric": schema.BoolAttribute{
						Description: "Whether Runtime Fabric is enabled. Inherited from the master organization on sub-orgs.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"flex_gateway": schema.SingleNestedAttribute{
						Description: "Omni Gateway entitlement. Inherited from the master organization on sub-orgs.",
						Optional:    true,
						Computed:    true,
						Attributes:  getEnabledEntitlementSchema(),
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"worker_logging_override": schema.SingleNestedAttribute{
						Description: "Worker logging override entitlement. Inherited from the master organization on sub-orgs.",
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
						Default:     objectdefault.StaticValue(zeroMq()),
					},
					"mq_requests": schema.SingleNestedAttribute{
						Description: "MQ requests entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getMqEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroMq()),
					},
					"gateways": schema.SingleNestedAttribute{
						Description: "Gateways entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getAssignedEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroAssigned()),
					},
					"design_center": schema.SingleNestedAttribute{
						Description: "Design Center entitlement. Inherited from the master organization on sub-orgs.",
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
						Default:     objectdefault.StaticValue(zeroAssigned()),
					},
					"service_mesh": schema.SingleNestedAttribute{
						Description: "Service Mesh entitlement. Inherited from the master organization on sub-orgs.",
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
						Default:     objectdefault.StaticValue(zeroAssigned()),
					},
					"managed_gateway_large": schema.SingleNestedAttribute{
						Description: "Managed Gateway (large) entitlement.",
						Optional:    true,
						Computed:    true,
						Attributes:  getAssignedEntitlementSchema(),
						Default:     objectdefault.StaticValue(zeroAssigned()),
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
					// Carry the prior state value through into the plan so a
					// no-op refresh doesn't flag the resource for in-place
					// update with a single `updated_at -> (known after apply)`
					// diff. The Update handler must NOT write the fresh
					// server-side timestamp back into state; doing so would
					// produce "inconsistent result after apply" because the
					// plan inherited the previous (older) value via this
					// modifier. The next refresh's Read() picks up the new
					// timestamp from the API.
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
			Description: "The number of assigned units. Defaults to 0 if not provided.",
			Optional:    true,
			Computed:    true,
			Default:     float64default.StaticFloat64(0),
		},
		"reassigned": schema.Float64Attribute{
			Description: "The number of reassigned units. Defaults to 0 if not provided.",
			Optional:    true,
			Computed:    true,
			Default:     float64default.StaticFloat64(0),
		},
	}
}

func getMqEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base": schema.Int64Attribute{
			Description: "The base number of MQ units. Defaults to 0 if not provided.",
			Optional:    true,
			Computed:    true,
			Default:     int64default.StaticInt64(0),
		},
		"add_on": schema.Int64Attribute{
			Description: "The add-on number of MQ units. Defaults to 0 if not provided.",
			Optional:    true,
			Computed:    true,
			Default:     int64default.StaticInt64(0),
		},
	}
}

func getEntitlementsAttributeTypes() map[string]attr.Type {
	vcoreType := types.ObjectType{AttrTypes: getVCoreEntitlementAttributeTypes()}
	enabledType := types.ObjectType{AttrTypes: map[string]attr.Type{"enabled": types.BoolType}}
	assignedType := types.ObjectType{AttrTypes: map[string]attr.Type{"assigned": types.Int64Type}}
	mqType := types.ObjectType{AttrTypes: getMqEntitlementAttributeTypes()}
	return map[string]attr.Type{
		"create_sub_orgs":     types.BoolType,
		"create_environments": types.BoolType,
		"global_deployment":   types.BoolType,
		"vcores_production":   vcoreType,
		"vcores_sandbox":      vcoreType,
		"vcores_design":       vcoreType,
		// static_ips and vpns intentionally omitted — not settable via the API.
		"vpcs":                    vcoreType,
		"network_connections":     vcoreType,
		"hybrid":                  enabledType,
		"runtime_fabric":          types.BoolType,
		"flex_gateway":            enabledType,
		"worker_logging_override": enabledType,
		"mq_messages":             mqType,
		"mq_requests":             mqType,
		"gateways":                assignedType,
		"design_center":           types.ObjectType{AttrTypes: map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType}},
		"load_balancer":           assignedType,
		"service_mesh":            enabledType,
		"managed_gateway_small":   assignedType,
		"managed_gateway_large":   assignedType,
	}
}

func getEnabledEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			Description: "Whether this feature is enabled.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
	}
}

func getAssignedEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"assigned": schema.Int64Attribute{
			Description: "The number of assigned units.",
			Optional:    true,
			Computed:    true,
			Default:     int64default.StaticInt64(0),
		},
	}
}

func getDesignCenterEntitlementSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api": schema.BoolAttribute{
			Description: "Whether API Designer is enabled.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
		},
		"mozart": schema.BoolAttribute{
			Description: "Whether Flow Designer (Mozart) is enabled.",
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
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

// stripMasterOrgOnlyEntitlements zeroes out the Entitlements fields that the
// Anypoint Access Management API only accepts for master organizations.
//
// Background: when the body of a POST or PUT for a business group mentions any
// of these fields — even with value `false`/`nil`/zero — the server responds
// with `403 Can not enable entitlement on a business group. It can only be
// set for a master organization`. The values for these entitlements on a
// sub-org are entirely server-managed (inherited from the master), so dropping
// them from the wire payload is always safe.
//
// This helper is called on Create (every Create through this resource is a
// sub-org because parent_organization_id is Required) and on Update for
// non-master orgs.
func stripMasterOrgOnlyEntitlements(e *accessmanagement.Entitlements) {
	if e == nil {
		return
	}
	e.Hybrid = nil
	e.FlexGateway = nil
	e.ServiceMesh = nil
	e.WorkerLoggingOverride = nil
	e.RuntimeFabric = nil
	e.DesignCenter = nil
}

// expandEntitlements converts a Terraform types.Object into the client Entitlements struct.
//
// A null/unknown input is treated as "user omitted the entitlements block" —
// we return a zero-valued Entitlements so the downstream JSON payload is the
// minimal `{create_sub_orgs:false, create_environments:false, global_deployment:false}`
// shape. Critically, no master-org-only fields (runtime_fabric, hybrid, etc.)
// are emitted, which is what prevents the Access Management endpoint from
// responding with 403 on business-group creates.
func expandEntitlements(ctx context.Context, obj types.Object) (accessmanagement.Entitlements, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return accessmanagement.Entitlements{}, diags
	}
	var model EntitlementsModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return accessmanagement.Entitlements{}, diags
	}

	ent := accessmanagement.Entitlements{
		CreateSubOrgs:      model.CreateSubOrgs.ValueBool(),
		CreateEnvironments: model.CreateEnvironments.ValueBool(),
		GlobalDeployment:   model.GlobalDeployment.ValueBool(),
	}
	// runtime_fabric is master-org-only on the backend: sending it (even as
	// `false`) against a business group produces a 403. Only emit it when the
	// user actually declared the attribute in HCL — null/unknown means "no
	// opinion", so we leave the field nil and let omitempty drop it.
	if !model.RuntimeFabric.IsNull() && !model.RuntimeFabric.IsUnknown() {
		v := model.RuntimeFabric.ValueBool()
		ent.RuntimeFabric = &v
	}

	ent.VCoresProduction = vCoreEntitlementFromModel(model.VCoresProduction)
	ent.VCoresSandbox = vCoreEntitlementFromModel(model.VCoresSandbox)
	ent.VCoresDesign = vCoreEntitlementFromModel(model.VCoresDesign)
	// StaticIps and Vpns are intentionally left nil here — the provider does
	// not model them (not settable via the organizations endpoint). Their
	// `omitempty` JSON tags keep them out of the outgoing payload.
	ent.Vpcs = vCoreEntitlementFromModel(model.Vpcs)
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

// zeroEnabled returns a concrete {enabled = false} Object.
//
// NOTE: flatten no longer synthesises this when the server omits an
// enabled-style entitlement (hybrid / flexGateway / worker_logging_override /
// service_mesh). On sub-orgs those flags can be inherited-true at the master
// level and a hardcoded false desyncs from what the server echoes in PUT
// responses, causing "inconsistent result after apply". Enabled-style
// entitlements now flatten to null when omitted; the helper is retained for
// tests that still construct canonical zero-valued plan objects.
func zeroEnabled() types.Object {
	return types.ObjectValueMust(
		map[string]attr.Type{"enabled": types.BoolType},
		map[string]attr.Value{"enabled": types.BoolValue(false)},
	)
}

// zeroAssigned returns a concrete {assigned = 0} Object. Same rationale as
// zeroEnabled — keeping state concrete prevents perpetual drift against
// `{ assigned = 0 }` configs (e.g. the managed_gateway_large bug).
func zeroAssigned() types.Object {
	return types.ObjectValueMust(
		map[string]attr.Type{"assigned": types.Int64Type},
		map[string]attr.Value{"assigned": types.Int64Value(0)},
	)
}

// zeroVCore returns a concrete {assigned = 0, reassigned = 0} Object.
func zeroVCore() types.Object {
	return types.ObjectValueMust(
		getVCoreEntitlementAttributeTypes(),
		map[string]attr.Value{
			"assigned":   types.Float64Value(0),
			"reassigned": types.Float64Value(0),
		},
	)
}


// zeroMq returns a concrete {base = 0, add_on = 0} Object.
func zeroMq() types.Object {
	return types.ObjectValueMust(
		getMqEntitlementAttributeTypes(),
		map[string]attr.Value{
			"base":   types.Int64Value(0),
			"add_on": types.Int64Value(0),
		},
	)
}

// flattenEntitlements converts the client Entitlements struct into a Terraform types.Object.
//
// Every Optional+Computed entitlement flattens to a concrete zero-valued Object
// when the server omits it. This is required so that a HCL value such as
// `managed_gateway_large = { assigned = 0 }` matches the refreshed state and
// does not trigger a perpetual in-place update plan.
func flattenEntitlements(ctx context.Context, ent accessmanagement.Entitlements) (types.Object, diag.Diagnostics) {
	enabledNull := types.ObjectNull(map[string]attr.Type{"enabled": types.BoolType})
	var hybrid, flexGateway, workerLogging, serviceMesh types.Object
	if ent.Hybrid != nil {
		hybrid = hybridEntitlementToModel(ent.Hybrid)
	} else {
		hybrid = enabledNull
	}
	if ent.FlexGateway != nil {
		flexGateway = enabledEntitlementToModel(ent.FlexGateway)
	} else {
		flexGateway = enabledNull
	}
	if ent.WorkerLoggingOverride != nil {
		workerLogging = workerLoggingOverrideEntitlementToModel(ent.WorkerLoggingOverride)
	} else {
		workerLogging = enabledNull
	}
	if ent.ServiceMesh != nil {
		serviceMesh = enabledEntitlementToModel(ent.ServiceMesh)
	} else {
		serviceMesh = enabledNull
	}

	var mqMessages, mqRequests types.Object
	if ent.MqMessages != nil {
		mqMessages = mqEntitlementToModel(ent.MqMessages)
	} else {
		mqMessages = zeroMq()
	}
	if ent.MqRequests != nil {
		mqRequests = mqEntitlementToModel(ent.MqRequests)
	} else {
		mqRequests = zeroMq()
	}

	var gateways, loadBalancer, gwSmall, gwLarge types.Object
	if ent.Gateways != nil {
		gateways = assignedEntitlementToModel(ent.Gateways)
	} else {
		gateways = zeroAssigned()
	}
	if ent.LoadBalancer != nil {
		loadBalancer = assignedEntitlementToModel(ent.LoadBalancer)
	} else {
		loadBalancer = zeroAssigned()
	}
	if ent.ManagedGatewaySmall != nil {
		gwSmall = assignedEntitlementToModel(ent.ManagedGatewaySmall)
	} else {
		gwSmall = zeroAssigned()
	}
	if ent.ManagedGatewayLarge != nil {
		gwLarge = assignedEntitlementToModel(ent.ManagedGatewayLarge)
	} else {
		gwLarge = zeroAssigned()
	}

	var designCenter types.Object
	if ent.DesignCenter != nil {
		designCenter = designCenterEntitlementToModel(ent.DesignCenter)
	} else {
		designCenter = types.ObjectNull(map[string]attr.Type{"api": types.BoolType, "mozart": types.BoolType})
	}

	runtimeFabric := types.BoolValue(false)
	if ent.RuntimeFabric != nil {
		runtimeFabric = types.BoolValue(*ent.RuntimeFabric)
	}

	model := EntitlementsModel{
		CreateSubOrgs:      types.BoolValue(ent.CreateSubOrgs),
		CreateEnvironments: types.BoolValue(ent.CreateEnvironments),
		GlobalDeployment:   types.BoolValue(ent.GlobalDeployment),
		RuntimeFabric:      runtimeFabric,
		VCoresProduction:   vcoreOrZero(ent.VCoresProduction),
		VCoresSandbox:      vcoreOrZero(ent.VCoresSandbox),
		VCoresDesign:       vcoreOrZero(ent.VCoresDesign),
		// StaticIps/Vpns intentionally not surfaced.
		Vpcs:                  vcoreOrZero(ent.Vpcs),
		NetworkConnections:    vcoreOrZero(ent.NetworkConnections),
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

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
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

// vcoreOrZero returns a Terraform Object for a VCoreEntitlement, falling back
// to {assigned = 0, reassigned = 0} when the server omits the entitlement. A
// null fallback here would trigger perpetual drift against HCL that declares
// `{ assigned = 0 }`, mirroring the managed_gateway_large bug.
func vcoreOrZero(vcore *accessmanagement.VCoreEntitlement) types.Object {
	if vcore != nil {
		return vCoreEntitlementToModel(vcore)
	}
	return zeroVCore()
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

func environmentToModel(env *accessmanagement.OrgEnvironment) types.Object {
	if env == nil {
		return types.ObjectNull(getEnvironmentsAttributeTypes())
	}
	var arcNamespace types.String
	if env.ArcNamespace != nil {
		arcNamespace = types.StringValue(*env.ArcNamespace)
	} else {
		arcNamespace = types.StringNull()
	}
	obj, diags := types.ObjectValue(getEnvironmentsAttributeTypes(), map[string]attr.Value{
		"id":              types.StringValue(env.ID),
		"name":            types.StringValue(env.Name),
		"organization_id": types.StringValue(env.OrganizationID),
		"is_production":   types.BoolValue(env.IsProduction),
		"type":            types.StringValue(env.Type),
		"client_id":       types.StringValue(env.ClientID),
		"arc_namespace":   arcNamespace,
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

	// For entitlements we read from Config (not Plan) so the POST only mentions
	// the ones the user actually declared in HCL. Shipping
	// UseStateForUnknown-filled defaults (e.g. hybrid:{enabled:false}) would
	// cause the Access Management endpoint to 403 on business groups with
	// "Can not enable entitlement on a business group".
	var config OrganizationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entitlements, entDiags := expandEntitlements(ctx, config.Entitlements)
	resp.Diagnostics.Append(entDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// parent_organization_id is Required, so every Create through this
	// provider produces a sub-org / business group. Master-org-only
	// entitlements (hybrid, flexGateway, serviceMesh, workerLoggingOverride,
	// runtimeFabric, designCenter) are inherited from the master and the
	// API 403s on a POST/PUT body that even mentions them. If the user
	// declared one of these in HCL we drop it here so the create succeeds;
	// the post-create flatten/merge step will surface the inherited value
	// the server returned.
	stripMasterOrgOnlyEntitlements(&entitlements)

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
	data.IDProviderID = types.StringValue(organization.IDProviderID)
	data.IsFederated = types.BoolValue(organization.IsFederated)
	// Set organization ID lists - always ensure known values
	if organization.ParentOrganizationIDs != nil {
		data.ParentOrganizationIDs, _ = types.ListValueFrom(ctx, types.StringType, organization.ParentOrganizationIDs)
	} else {
		data.ParentOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.SubOrganizationIDs != nil {
		data.SubOrganizationIDs, _ = types.ListValueFrom(ctx, types.StringType, organization.SubOrganizationIDs)
	} else {
		data.SubOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.TenantOrganizationIDs != nil {
		data.TenantOrganizationIDs, _ = types.ListValueFrom(ctx, types.StringType, organization.TenantOrganizationIDs)
	} else {
		data.TenantOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
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

	// Flatten entitlements from the API response and merge with the plan so
	// any fields the server omitted from the POST response (common for
	// inherited entitlements on sub-orgs) fall back to whatever the user
	// declared in HCL rather than getting clobbered to null. This keeps the
	// post-apply state consistent with the plan Terraform computed.
	entObj, entDiags := flattenEntitlements(ctx, organization.Entitlements)
	resp.Diagnostics.Append(entDiags...)
	if !resp.Diagnostics.HasError() {
		data.Entitlements = mergeEntitlementsPreservingPlan(ctx, data.Entitlements, entObj, false, &resp.Diagnostics)
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

	// parent_organization_id is not returned as a scalar by the Access Management
	// GET endpoint — the server only echoes the full ancestor chain via
	// `parentOrganizationIds`. For normal refreshes after Create we keep whatever
	// value state already carries (the user's HCL value). For the
	// `terraform import` path that value is null on first Read, so we derive the
	// immediate parent from the tail of the chain — otherwise the subsequent
	// plan would show `null -> "<user value>"` and, because this attribute has
	// RequiresReplace, Terraform would destroy+recreate the resource the user
	// just imported.
	if data.ParentOrganizationID.IsNull() || data.ParentOrganizationID.IsUnknown() || data.ParentOrganizationID.ValueString() == "" {
		if n := len(organization.ParentOrganizationIDs); n > 0 {
			data.ParentOrganizationID = types.StringValue(organization.ParentOrganizationIDs[n-1])
		}
	}
	data.ClientID = types.StringValue(organization.ClientID)
	data.IDProviderID = types.StringValue(organization.IDProviderID)
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
	if organization.ParentOrganizationIDs != nil {
		parentOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.ParentOrganizationIDs)
		resp.Diagnostics.Append(diags...)
		data.ParentOrganizationIDs = parentOrgIDs
	} else {
		data.ParentOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.SubOrganizationIDs != nil {
		subOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.SubOrganizationIDs)
		resp.Diagnostics.Append(diags...)
		data.SubOrganizationIDs = subOrgIDs
	} else {
		data.SubOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.TenantOrganizationIDs != nil {
		tenantOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.TenantOrganizationIDs)
		resp.Diagnostics.Append(diags...)
		data.TenantOrganizationIDs = tenantOrgIDs
	} else {
		data.TenantOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
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
//
// Mutable fields: name, entitlements (including quota entitlements). owner_id and
// parent_organization_id are marked RequiresReplace in the schema, so any change to
// those triggers a recreate and never reaches this handler.
//
// Properties (e.g. flow_designer) are not exposed on the resource but must be round
// tripped in the PUT body or the server drops them; we GET the current org and feed
// its Properties into the request.
func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Config (what the user literally declared in HCL) is the authoritative
	// source for which entitlements the PUT body should even mention. The plan
	// carries UseStateForUnknown-filled values for Optional+Computed
	// entitlements the user didn't set, and shipping those (e.g.
	// hybrid:{enabled:false} on a sub-org) makes the Access Management API
	// return 403 "Can not enable entitlement on a business group".
	var config OrganizationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID := state.ID.ValueString()
	if organizationID == "" {
		resp.Diagnostics.AddError(
			"Missing organization ID in state",
			"Cannot update an organization whose ID is unknown; the resource state appears to be corrupted.",
		)
		return
	}

	// Pull the current server-side view so we can round-trip fields the provider
	// does not model (Properties) and fall back to a known owner_id.
	current, err := r.client.GetOrganization(ctx, organizationID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddWarning(
				"Organization no longer exists",
				fmt.Sprintf("Organization %s was not found on the server; removing it from state.", organizationID),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading organization before update",
			fmt.Sprintf("Could not read organization %s: %s", organizationID, err.Error()),
		)
		return
	}

	entitlements, entDiags := expandEntitlements(ctx, config.Entitlements)
	resp.Diagnostics.Append(entDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Anypoint API 403s when a PUT for a business group includes any
	// master-org-only entitlements (hybrid, flexGateway, serviceMesh,
	// workerLoggingOverride, runtimeFabric, designCenter) even when set to
	// false/nil/zero. After `terraform import`, generated.tf often captures
	// these as concrete values from state (e.g. design_center.api = true
	// inherited from the master), so they end up in expandEntitlements
	// output and cause the 403 on the next apply. Strip them unconditionally
	// for non-master orgs.
	if !current.IsMaster {
		stripMasterOrgOnlyEntitlements(&entitlements)
	}

	// owner_id is RequiresReplace, so state and plan always agree; we prefer state
	// (it is the authoritative server value).
	ownerID := state.OwnerID.ValueString()
	if ownerID == "" {
		ownerID = current.OwnerID
	}

	updateReq := accessmanagement.UpdateOrganizationRequest{
		ID:           organizationID,
		Name:         plan.Name.ValueString(),
		OwnerID:      ownerID,
		Properties:   current.Properties,
		Entitlements: entitlements,
	}

	tflog.Info(ctx, "Updating organization", map[string]interface{}{
		"id":   organizationID,
		"name": updateReq.Name,
	})

	organization, err := r.client.UpdateOrganization(ctx, organizationID, &updateReq)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddWarning(
				"Organization no longer exists",
				fmt.Sprintf("Organization %s was not found on the server during update; removing it from state.", organizationID),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating organization",
			fmt.Sprintf("Could not update organization %s: %s", organizationID, err.Error()),
		)
		return
	}

	// Map server response back into the plan-shaped model so Terraform sees a fully
	// known, post-apply state.
	plan.ID = types.StringValue(organization.ID)
	plan.Name = types.StringValue(organization.Name)
	plan.OwnerID = types.StringValue(organization.OwnerID)
	plan.CreatedAt = types.StringValue(organization.CreatedAt)
	// NOTE: deliberately do NOT overwrite plan.UpdatedAt with the server's
	// fresh timestamp. The schema marks updated_at as Computed with
	// UseStateForUnknown, meaning Terraform planned this field as the prior
	// state value (so a no-op refresh doesn't show a noisy
	// `updated_at -> (known after apply)` diff). Writing the API value
	// here would violate the post-apply state == plan contract and produce
	// "inconsistent result after apply". The next refresh's Read() reads
	// the fresh timestamp from the API and brings state up to date.
	plan.ClientID = types.StringValue(organization.ClientID)
	plan.IDProviderID = types.StringValue(organization.IDProviderID)
	plan.IsFederated = types.BoolValue(organization.IsFederated)
	plan.IsAutomaticAdminPromotionExempt = types.BoolValue(organization.IsAutomaticAdminPromotionExempt)
	plan.OrgType = types.StringValue(organization.OrgType)
	plan.IsRoot = types.BoolValue(organization.IsRoot)
	plan.IsMaster = types.BoolValue(organization.IsMaster)
	plan.SessionTimeout = types.Int64Value(int64(organization.SessionTimeout))

	if organization.MfaRequired != "" {
		plan.MfaRequired = types.StringValue(organization.MfaRequired)
	} else {
		plan.MfaRequired = types.StringNull()
	}
	if organization.GdotID != nil {
		plan.GdotID = types.StringValue(*organization.GdotID)
	} else {
		plan.GdotID = types.StringNull()
	}
	if organization.DeletedAt != nil {
		plan.DeletedAt = types.StringValue(*organization.DeletedAt)
	} else {
		plan.DeletedAt = types.StringNull()
	}
	if organization.Domain != nil {
		plan.Domain = types.StringValue(*organization.Domain)
	} else {
		plan.Domain = types.StringNull()
	}

	if organization.ParentOrganizationIDs != nil {
		parentOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.ParentOrganizationIDs)
		resp.Diagnostics.Append(diags...)
		plan.ParentOrganizationIDs = parentOrgIDs
	} else {
		plan.ParentOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.SubOrganizationIDs != nil {
		subOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.SubOrganizationIDs)
		resp.Diagnostics.Append(diags...)
		plan.SubOrganizationIDs = subOrgIDs
	} else {
		plan.SubOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if organization.TenantOrganizationIDs != nil {
		tenantOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.TenantOrganizationIDs)
		resp.Diagnostics.Append(diags...)
		plan.TenantOrganizationIDs = tenantOrgIDs
	} else {
		plan.TenantOrganizationIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// The PUT /organizations/{id} response doesn't include the org's
	// environments for business groups — they're managed by the separate
	// anypoint_environment resource. If the response carries an env list, use
	// it; otherwise preserve whatever plan/state already has (UseStateForUnknown
	// on the `environments` attribute ensures that value is the prior state).
	// Overwriting with an empty list here would cause Terraform's "element has
	// vanished" inconsistent-result diagnostic.
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
		plan.Environments = envs
	}
	// else: leave plan.Environments untouched (it already holds the
	// UseStateForUnknown-propagated prior-state value).

	if !reflect.ValueOf(organization.Subscription).IsZero() {
		subscription, diags := types.ObjectValueFrom(ctx, getSubscriptionAttributeTypes(), organization.Subscription)
		resp.Diagnostics.Append(diags...)
		plan.Subscription = subscription
	} else {
		plan.Subscription = types.ObjectNull(getSubscriptionAttributeTypes())
	}

	// Merge the PUT response's entitlements with the plan. planIsAuthoritative=true
	// ensures state == plan for all user-declared fields, even when the API echoes
	// a stale/zero value for fields it accepted but didn't reflect (e.g.
	// managedGatewayLarge.assigned = 1 written but response still shows 0).
	entObj, entFlattenDiags := flattenEntitlements(ctx, organization.Entitlements)
	resp.Diagnostics.Append(entFlattenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Entitlements = mergeEntitlementsPreservingPlan(ctx, plan.Entitlements, entObj, true, &resp.Diagnostics)

	tflog.Trace(ctx, "updated organization")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// mergeEntitlementsPreservingPlan reconciles the entitlements blob returned by
// the API (apiEnt) with what Terraform planned (planEnt) so the post-apply
// state respects Terraform's contract: post-apply MUST equal plan, otherwise
// the framework emits "Provider produced inconsistent result after apply".
//
// planIsAuthoritative should be true on Update and false on Create:
//   - Update: the plan is what we sent to the API; Terraform requires post-apply
//     state == plan, so user-declared concrete values always win over whatever
//     the server echoed back (the API often returns stale/zero values for fields
//     it accepted but didn't echo, e.g. managedGatewayLarge.assigned).
//   - Create: we haven't written the value yet from the user's perspective, so
//     the API response is the canonical initial state for unknown plan fields.
//
// Per-attribute precedence (highest to lowest):
//
//  1. Plan unknown          → API value wins.
//  2. Plan null             → preserve null.
//  3. Plan concrete, API null → preserve plan.
//  4. Plan concrete, API concrete, planIsAuthoritative=true  → plan wins.
//  5. Plan concrete, API concrete, planIsAuthoritative=false → API wins.
func mergeEntitlementsPreservingPlan(ctx context.Context, planEnt types.Object, apiEnt types.Object, planIsAuthoritative bool, diagsOut *diag.Diagnostics) types.Object {
	if planEnt.IsUnknown() {
		return apiEnt
	}
	if planEnt.IsNull() {
		return planEnt
	}
	if apiEnt.IsNull() || apiEnt.IsUnknown() {
		return planEnt
	}

	apiAttrs := apiEnt.Attributes()
	planAttrs := planEnt.Attributes()
	merged := make(map[string]attr.Value, len(apiAttrs))
	for name, apiVal := range apiAttrs {
		planVal, hasPlan := planAttrs[name]
		switch {
		case !hasPlan:
			merged[name] = apiVal
		case planVal.IsUnknown():
			// Rule 1 — API wins; plan said "known after apply".
			merged[name] = apiVal
		case planVal.IsNull():
			// Rule 2 — preserve plan null.
			merged[name] = planVal
		case apiVal.IsNull():
			// Rule 3 — server omitted; keep the user-declared value.
			merged[name] = planVal
		case planIsAuthoritative:
			// Rule 4 — Update: plan is what we sent; state must equal plan.
			merged[name] = planVal
		default:
			// Rule 5 — Create: API is the canonical initial state.
			merged[name] = apiVal
		}
	}
	obj, diags := types.ObjectValue(getEntitlementsAttributeTypes(), merged)
	diagsOut.Append(diags...)
	if diags.HasError() {
		return apiEnt
	}
	return obj
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

// ImportState supports `terraform import anypoint_organization.<name> <org-id>`.
//
// The framework seeds state with the provided ID; the subsequent Read() call
// hydrates every other attribute (including deriving parent_organization_id
// from the ancestor chain the server returns). The user's HCL must declare
// `name`, `parent_organization_id`, and `owner_id` — these are Required and
// cannot be computed, but they all show up correctly in state on the first
// refresh.
func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
