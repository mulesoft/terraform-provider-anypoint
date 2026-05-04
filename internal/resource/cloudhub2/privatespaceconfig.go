package cloudhub2

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

var (
	_ resource.Resource                     = &PrivateSpaceConfigResource{}
	_ resource.ResourceWithConfigure        = &PrivateSpaceConfigResource{}
	_ resource.ResourceWithImportState      = &PrivateSpaceConfigResource{}
	_ resource.ResourceWithConfigValidators = &PrivateSpaceConfigResource{}
)

type PrivateSpaceConfigResource struct {
	spaceClient    *cloudhub2.PrivateSpacesClient
	networkClient  *cloudhub2.PrivateNetworkClient
	firewallClient *cloudhub2.FirewallRulesClient
}

type PrivateSpaceConfigResourceModel struct {
	ID                      types.String        `tfsdk:"id"`
	Name                    types.String        `tfsdk:"name"`
	EnableIAMRole           types.Bool          `tfsdk:"enable_iam_role"`
	EnableEgress            types.Bool          `tfsdk:"enable_egress"`
	OrganizationID          types.String        `tfsdk:"organization_id"`
	RootOrganizationID      types.String        `tfsdk:"root_organization_id"`
	Status                  types.String        `tfsdk:"status"`
	MuleAppDeploymentCount  types.Int64         `tfsdk:"mule_app_deployment_count"`
	DaysLeftForRelaxedQuota types.Int64         `tfsdk:"days_left_for_relaxed_quota"`
	VPCMigrationInProgress  types.Bool          `tfsdk:"vpc_migration_in_progress"`
	ManagedFirewallRules    types.List          `tfsdk:"managed_firewall_rules"`
	GlobalSpaceStatus       types.Map           `tfsdk:"global_space_status"`
	Network                 *NetworkConfigModel `tfsdk:"network"`
	FirewallRules           []FirewallRuleModel `tfsdk:"firewall_rules"`
}

type NetworkConfigModel struct {
	Region                   types.String `tfsdk:"region"`
	CidrBlock                types.String `tfsdk:"cidr_block"`
	ReservedCIDRs            types.List   `tfsdk:"reserved_cidrs"`
	InboundStaticIPs         types.List   `tfsdk:"inbound_static_ips"`
	InboundInternalStaticIPs types.List   `tfsdk:"inbound_internal_static_ips"`
	OutboundStaticIPs        types.List   `tfsdk:"outbound_static_ips"`
	DNSTarget                types.String `tfsdk:"dns_target"`
}

func NewPrivateSpaceConfigResource() resource.Resource {
	return &PrivateSpaceConfigResource{}
}

func (r *PrivateSpaceConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_config"
}

func (r *PrivateSpaceConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Private Space together with its network configuration and firewall rules as a single resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private space.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the private space.",
				Required:    true,
			},
			"enable_iam_role": schema.BoolAttribute{
				Description: "Whether to enable IAM role for the private space.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_egress": schema.BoolAttribute{
				Description: "Whether to enable egress for the private space.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space will be created. Defaults to the provider organization.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"root_organization_id": schema.StringAttribute{
				Description: "The root organization ID of the private space.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the private space.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mule_app_deployment_count": schema.Int64Attribute{
				Description: "The number of Mule apps deployed in the private space.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"days_left_for_relaxed_quota": schema.Int64Attribute{
				Description: "The number of days left for relaxed quota.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"vpc_migration_in_progress": schema.BoolAttribute{
				Description: "Whether a VPC migration is in progress.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"managed_firewall_rules": schema.ListAttribute{
				Description: "The managed firewall rules for the private space.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"global_space_status": schema.MapAttribute{
				Description: "The global space status for the private space.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_rules": schema.ListNestedAttribute{
				Description: "Firewall rules for the private space.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr_block": schema.StringAttribute{
							Description: "The CIDR block for the firewall rule.",
							Required:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for the firewall rule (tcp, udp, icmp).",
							Required:    true,
						},
						"from_port": schema.Int64Attribute{
							Description: "The starting port for the firewall rule.",
							Required:    true,
						},
						"to_port": schema.Int64Attribute{
							Description: "The ending port for the firewall rule.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the firewall rule (inbound, outbound).",
							Required:    true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"network": schema.SingleNestedBlock{
				Description: "Network configuration for the private space. Omit to create the space without a network.",
				Attributes: map[string]schema.Attribute{
					"region": schema.StringAttribute{
						Description: "The AWS region for the private network.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"cidr_block": schema.StringAttribute{
						Description: "The CIDR block for the private network.",
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"reserved_cidrs": schema.ListAttribute{
						Description: "Reserved CIDR blocks for the private network.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"inbound_static_ips": schema.ListAttribute{
						Description: "Inbound static IPs assigned to the private network.",
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"inbound_internal_static_ips": schema.ListAttribute{
						Description: "Inbound internal static IPs assigned to the private network.",
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"outbound_static_ips": schema.ListAttribute{
						Description: "Outbound static IPs assigned to the private network.",
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"dns_target": schema.StringAttribute{
						Description: "The DNS target for the private network.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *PrivateSpaceConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	spaceClient, err := cloudhub2.NewPrivateSpacesClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Private Space Client", err.Error())
		return
	}

	networkClient, err := cloudhub2.NewPrivateNetworkClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Private Network Client", err.Error())
		return
	}

	firewallClient, err := cloudhub2.NewFirewallRulesClient(config)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Firewall Rules Client", err.Error())
		return
	}

	r.spaceClient = spaceClient
	r.networkClient = networkClient
	r.firewallClient = firewallClient
}

// ConfigValidators enforces that when the network block is present, region and cidr_block are set.
func (r *PrivateSpaceConfigResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&networkBlockValidator{},
	}
}

type networkBlockValidator struct{}

func (v *networkBlockValidator) Description(_ context.Context) string {
	return "When the network block is present, region and cidr_block are required."
}

func (v *networkBlockValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *networkBlockValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PrivateSpaceConfigResourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Network == nil {
		return
	}

	var d diag.Diagnostics
	if !data.Network.Region.IsUnknown() && (data.Network.Region.IsNull() || data.Network.Region.ValueString() == "") {
		d.AddAttributeError(path.Root("network").AtName("region"), "Missing required attribute", "region is required when the network block is present.")
	}
	if !data.Network.CidrBlock.IsUnknown() && (data.Network.CidrBlock.IsNull() || data.Network.CidrBlock.ValueString() == "") {
		d.AddAttributeError(path.Root("network").AtName("cidr_block"), "Missing required attribute", "cidr_block is required when the network block is present.")
	}
	resp.Diagnostics.Append(d...)
}

func (r *PrivateSpaceConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateSpaceConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.spaceClient.OrgID
	}
	// Always store the resolved org ID so an empty string config value doesn't
	// produce a plan→state inconsistency.
	data.OrganizationID = types.StringValue(orgID)

	// Step 1: Create the private space.
	enableIAMRole := data.EnableIAMRole.ValueBool()
	enableEgress := data.EnableEgress.ValueBool()
	spaceReq := &cloudhub2.CreatePrivateSpaceRequest{
		Name:          data.Name.ValueString(),
		EnableIAMRole: &enableIAMRole,
		EnableEgress:  &enableEgress,
	}
	space, err := r.spaceClient.CreatePrivateSpace(ctx, orgID, spaceReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating private space", err.Error())
		return
	}
	spaceID := space.ID

	// Step 2: Configure the network (optional).
	var spaceAfterNetwork *cloudhub2.PrivateSpace
	if data.Network != nil {
		var reservedCIDRs []string
		if !data.Network.ReservedCIDRs.IsNull() && !data.Network.ReservedCIDRs.IsUnknown() {
			data.Network.ReservedCIDRs.ElementsAs(ctx, &reservedCIDRs, false)
		}
		networkReq := &cloudhub2.CreatePrivateNetworkRequest{
			Network: cloudhub2.NetworkConfiguration{
				Region:        data.Network.Region.ValueString(),
				CidrBlock:     data.Network.CidrBlock.ValueString(),
				ReservedCIDRs: reservedCIDRs,
			},
		}
		spaceAfterNetwork, err = r.networkClient.CreatePrivateNetwork(ctx, orgID, spaceID, networkReq)
		if err != nil {
			resp.Diagnostics.AddError("Error configuring private network", err.Error())
			return
		}
	}

	// Step 3: Set firewall rules (if provided).
	var spaceAfterFirewall *cloudhub2.PrivateSpace
	if len(data.FirewallRules) > 0 {
		firewallReq := &cloudhub2.UpdateFirewallRulesRequest{
			ManagedFirewallRules: mapFirewallRulesToAPI(data.FirewallRules),
		}
		spaceAfterFirewall, err = r.firewallClient.UpdateFirewallRules(ctx, orgID, spaceID, firewallReq)
		if err != nil {
			resp.Diagnostics.AddError("Error setting firewall rules", err.Error())
			return
		}
	}

	tflog.Trace(ctx, "created private space config")

	mapSpaceConfigToModel(ctx, &data, spaceID, orgID, space, spaceAfterNetwork, spaceAfterFirewall)
	// Preserve plan values for fields the API may not reflect immediately.
	data.EnableIAMRole = types.BoolValue(enableIAMRole)
	data.EnableEgress = types.BoolValue(enableEgress)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateSpaceConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateSpaceConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ImportStatePassthroughID only sets id — everything else is null.
	// We use this to detect an import read so we can populate all fields.
	isImport := data.OrganizationID.IsNull()

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.spaceClient.OrgID
	}

	space, err := r.spaceClient.GetPrivateSpace(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading private space config", err.Error())
		return
	}

	// Preserve state values for enable_egress and enable_iam_role (API may lag).
	stateEnableEgress := data.EnableEgress
	stateEnableIAMRole := data.EnableIAMRole
	// Remember whether the user was managing firewall rules before this read.
	prevFirewallRules := data.FirewallRules

	mapSpaceConfigToModel(ctx, &data, space.ID, orgID, space, space, space)
	data.EnableEgress = stateEnableEgress
	data.EnableIAMRole = stateEnableIAMRole

	// On import, always capture firewall rules from the API. On a normal read,
	// only keep them when the user explicitly manages them (prevFirewallRules != nil).
	if isImport {
		data.FirewallRules = mapFirewallRulesFromAPI(space.ManagedFirewallRules)
		// Seed defaults for bool flags so the plan doesn't trigger a spurious update.
		if data.EnableEgress.IsNull() || data.EnableEgress.IsUnknown() {
			data.EnableEgress = types.BoolValue(false)
		}
		if data.EnableIAMRole.IsNull() || data.EnableIAMRole.IsUnknown() {
			data.EnableIAMRole = types.BoolValue(false)
		}
	} else if prevFirewallRules == nil {
		data.FirewallRules = nil
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PrivateSpaceConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PrivateSpaceConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.spaceClient.OrgID
	}
	spaceID := state.ID.ValueString()

	var latestSpace *cloudhub2.PrivateSpace

	// Update space name/flags if changed.
	spaceUpdateReq := &cloudhub2.UpdatePrivateSpaceRequest{}
	hasSpaceChanges := false
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		spaceUpdateReq.Name = &name
		hasSpaceChanges = true
	}
	if !plan.EnableIAMRole.Equal(state.EnableIAMRole) {
		v := plan.EnableIAMRole.ValueBool()
		spaceUpdateReq.EnableIAMRole = &v
		hasSpaceChanges = true
	}
	if !plan.EnableEgress.Equal(state.EnableEgress) {
		v := plan.EnableEgress.ValueBool()
		spaceUpdateReq.EnableEgress = &v
		hasSpaceChanges = true
	}
	if hasSpaceChanges {
		updated, err := r.spaceClient.UpdatePrivateSpace(ctx, orgID, spaceID, spaceUpdateReq)
		if err != nil {
			resp.Diagnostics.AddError("Error updating private space", err.Error())
			return
		}
		latestSpace = updated
	}

	// Update network if reserved_cidrs changed (region/cidr_block require replace).
	if plan.Network != nil && state.Network != nil && !reservedCIDRsEqual(ctx, plan.Network.ReservedCIDRs, state.Network.ReservedCIDRs) {
		var reservedCIDRs []string
		if !plan.Network.ReservedCIDRs.IsNull() && !plan.Network.ReservedCIDRs.IsUnknown() {
			plan.Network.ReservedCIDRs.ElementsAs(ctx, &reservedCIDRs, false)
		}
		networkReq := &cloudhub2.UpdatePrivateNetworkRequest{
			Network: cloudhub2.NetworkConfiguration{
				// Only send reserved_cidrs — region/cidr_block cannot be updated once set.
				ReservedCIDRs: reservedCIDRs,
			},
		}
		updated, err := r.networkClient.UpdatePrivateNetwork(ctx, orgID, spaceID, networkReq)
		if err != nil {
			resp.Diagnostics.AddError("Error updating private network", err.Error())
			return
		}
		latestSpace = updated
	}

	// Update firewall rules if changed.
	if !firewallRulesEqual(plan.FirewallRules, state.FirewallRules) {
		firewallReq := &cloudhub2.UpdateFirewallRulesRequest{
			ManagedFirewallRules: mapFirewallRulesToAPI(plan.FirewallRules),
		}
		updated, err := r.firewallClient.UpdateFirewallRules(ctx, orgID, spaceID, firewallReq)
		if err != nil {
			resp.Diagnostics.AddError("Error updating firewall rules", err.Error())
			return
		}
		latestSpace = updated
	}

	// If nothing changed, re-use state for computed fields.
	if latestSpace == nil {
		plan.RootOrganizationID = state.RootOrganizationID
		plan.Status = state.Status
		plan.MuleAppDeploymentCount = state.MuleAppDeploymentCount
		plan.DaysLeftForRelaxedQuota = state.DaysLeftForRelaxedQuota
		plan.VPCMigrationInProgress = state.VPCMigrationInProgress
		plan.ManagedFirewallRules = state.ManagedFirewallRules
		plan.GlobalSpaceStatus = state.GlobalSpaceStatus
		if plan.Network != nil && state.Network != nil {
			plan.Network.InboundStaticIPs = state.Network.InboundStaticIPs
			plan.Network.InboundInternalStaticIPs = state.Network.InboundInternalStaticIPs
			plan.Network.OutboundStaticIPs = state.Network.OutboundStaticIPs
			plan.Network.DNSTarget = state.Network.DNSTarget
		}
	} else {
		mapSpaceConfigToModel(ctx, &plan, spaceID, orgID, latestSpace, latestSpace, latestSpace)
	}

	plan.OrganizationID = types.StringValue(orgID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PrivateSpaceConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateSpaceConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.spaceClient.OrgID
	}

	if err := r.spaceClient.DeletePrivateSpace(ctx, orgID, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting private space config", err.Error())
	}
}

func (r *PrivateSpaceConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapSpaceConfigToModel populates model fields from API responses.
// spaceBase carries space-level fields, networkSrc carries network fields,
// firewallSrc carries firewall fields. They can be the same pointer.
func mapSpaceConfigToModel(
	ctx context.Context,
	data *PrivateSpaceConfigResourceModel,
	spaceID, orgID string,
	spaceBase *cloudhub2.PrivateSpace,
	networkSrc *cloudhub2.PrivateSpace,
	firewallSrc *cloudhub2.PrivateSpace,
) {
	data.ID = types.StringValue(spaceID)
	data.OrganizationID = types.StringValue(orgID)

	if spaceBase != nil {
		data.Name = types.StringValue(spaceBase.Name)
		data.Status = types.StringValue(spaceBase.Status)
		data.RootOrganizationID = types.StringValue(spaceBase.RootOrganizationID)
		data.MuleAppDeploymentCount = types.Int64Value(int64(spaceBase.MuleAppDeploymentCount))
		data.DaysLeftForRelaxedQuota = types.Int64Value(int64(spaceBase.DaysLeftForRelaxedQuota))
		data.VPCMigrationInProgress = types.BoolValue(spaceBase.VPCMigrationInProgress)
	}

	data.ManagedFirewallRules = types.ListNull(types.StringType)
	data.GlobalSpaceStatus = types.MapNull(types.StringType)

	if networkSrc != nil {
		n := networkSrc.Network
		// Auto-allocate the network model when the API reports a configured region
		// (handles import where data.Network starts nil).
		if data.Network == nil && n.Region != "" {
			data.Network = &NetworkConfigModel{}
		}
		if data.Network != nil {
			data.Network.Region = types.StringValue(n.Region)
			data.Network.CidrBlock = types.StringValue(n.CidrBlock)
			data.Network.DNSTarget = types.StringValue(n.DNSTarget)

			inbound, _ := types.ListValueFrom(ctx, types.StringType, n.InboundStaticIPs)
			data.Network.InboundStaticIPs = inbound

			inboundInternal, _ := types.ListValueFrom(ctx, types.StringType, n.InboundInternalStaticIPs)
			data.Network.InboundInternalStaticIPs = inboundInternal

			outbound, _ := types.ListValueFrom(ctx, types.StringType, n.OutboundStaticIPs)
			data.Network.OutboundStaticIPs = outbound

			// Only overwrite reserved_cidrs when the API returns a non-empty list.
			// Preserving the existing state value (null or []) avoids null↔[]
			// drift when the user left the field unset. Always ensure the field
			// has a correctly-typed value (never a zero-value types.List{}).
			if len(n.ReservedCIDRs) > 0 {
				reserved, _ := types.ListValueFrom(ctx, types.StringType, n.ReservedCIDRs)
				data.Network.ReservedCIDRs = reserved
			} else if data.Network.ReservedCIDRs.IsNull() && data.Network.ReservedCIDRs.ElementType(ctx) == nil {
				// Freshly allocated NetworkConfigModel (e.g. during import) — initialize
				// to a typed null so the framework doesn't see a missing element type.
				data.Network.ReservedCIDRs = types.ListNull(types.StringType)
			}
		}
	}

	// Only sync firewall rules from the API when the user explicitly manages them
	// (i.e. data.FirewallRules was non-nil coming in from state or plan). If the
	// user left firewall_rules unset (nil), the API may still assign default rules —
	// we intentionally ignore those to avoid drift on resources that don't manage
	// firewall rules.
	if data.FirewallRules != nil && firewallSrc != nil {
		data.FirewallRules = mapFirewallRulesFromAPI(firewallSrc.ManagedFirewallRules)
	}
}

// reservedCIDRsEqual treats null and empty list as equivalent to avoid spurious
// updates when generated config uses [] but state has null.
func reservedCIDRsEqual(ctx context.Context, a, b types.List) bool {
	var aVals, bVals []string
	if !a.IsNull() && !a.IsUnknown() {
		a.ElementsAs(ctx, &aVals, false)
	}
	if !b.IsNull() && !b.IsUnknown() {
		b.ElementsAs(ctx, &bVals, false)
	}
	if len(aVals) != len(bVals) {
		return false
	}
	for i := range aVals {
		if aVals[i] != bVals[i] {
			return false
		}
	}
	return true
}

// FirewallRuleModel is the Terraform model for a single firewall rule.
type FirewallRuleModel struct {
	CidrBlock types.String `tfsdk:"cidr_block"`
	Protocol  types.String `tfsdk:"protocol"`
	FromPort  types.Int64  `tfsdk:"from_port"`
	ToPort    types.Int64  `tfsdk:"to_port"`
	Type      types.String `tfsdk:"type"`
}

func mapFirewallRulesToAPI(rules []FirewallRuleModel) []cloudhub2.FirewallRule {
	out := make([]cloudhub2.FirewallRule, len(rules))
	for i, r := range rules {
		out[i] = cloudhub2.FirewallRule{
			CidrBlock: r.CidrBlock.ValueString(),
			Protocol:  r.Protocol.ValueString(),
			FromPort:  int(r.FromPort.ValueInt64()),
			ToPort:    int(r.ToPort.ValueInt64()),
			Type:      r.Type.ValueString(),
		}
	}
	return out
}

func mapFirewallRulesFromAPI(rules []cloudhub2.FirewallRule) []FirewallRuleModel {
	out := make([]FirewallRuleModel, len(rules))
	for i, r := range rules {
		out[i] = FirewallRuleModel{
			CidrBlock: types.StringValue(r.CidrBlock),
			Protocol:  types.StringValue(r.Protocol),
			FromPort:  types.Int64Value(int64(r.FromPort)),
			ToPort:    types.Int64Value(int64(r.ToPort)),
			Type:      types.StringValue(r.Type),
		}
	}
	// Sort into a canonical order so Platform's internal ordering never causes
	// perpetual plan drift. State always reflects this sorted order; users must
	// write their HCL rules in the same order (type → protocol → from_port →
	// to_port → cidr_block) to keep plans clean.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Type != b.Type {
			return a.Type.ValueString() < b.Type.ValueString()
		}
		if a.Protocol != b.Protocol {
			return a.Protocol.ValueString() < b.Protocol.ValueString()
		}
		if a.FromPort != b.FromPort {
			return a.FromPort.ValueInt64() < b.FromPort.ValueInt64()
		}
		if a.ToPort != b.ToPort {
			return a.ToPort.ValueInt64() < b.ToPort.ValueInt64()
		}
		return a.CidrBlock.ValueString() < b.CidrBlock.ValueString()
	})
	return out
}

// firewallRulesEqual compares two slices of FirewallRuleModel for equality.
func firewallRulesEqual(a, b []FirewallRuleModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].CidrBlock.Equal(b[i].CidrBlock) ||
			!a[i].Protocol.Equal(b[i].Protocol) ||
			!a[i].FromPort.Equal(b[i].FromPort) ||
			!a[i].ToPort.Equal(b[i].ToPort) ||
			!a[i].Type.Equal(b[i].Type) {
			return false
		}
	}
	return true
}
