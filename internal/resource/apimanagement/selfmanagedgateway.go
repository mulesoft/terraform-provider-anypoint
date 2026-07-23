package apimanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &SelfManagedGatewayResource{}
	_ resource.ResourceWithConfigure   = &SelfManagedGatewayResource{}
	_ resource.ResourceWithImportState = &SelfManagedGatewayResource{}
)

// SelfManagedGatewayResource manages a self-managed (connected-mode) Flex/Omni Gateway.
//
// Unlike the *managed* Omni Gateway, the platform does not provision a runtime for a
// self-managed gateway — the customer runs the Flex runtime on their own infrastructure.
// There is therefore NO create-gateway endpoint. This resource models the genuine
// platform-side primitives of the connected-mode flow:
//
//   - Create : mints a registration token (scoped to org/env). The token is the value the
//     operator feeds to the Flex runtime (e.g. `flexctl registration create <name>
//     --connected --token=<token> ...`). The runtime then self-registers and the gateway
//     object appears on the platform.
//   - Read   : best-effort resolves the registered gateway by name and surfaces its
//     platform id / status / version. It does NOT drop the resource from state when the
//     gateway has not registered yet (that is the normal state right after apply).
//   - Delete : removes the platform-side gateway object if it has registered.
//
// The private key / CSR NEVER pass through the provider (they are generated on the runtime
// host by flexctl and never leave it), so no key material lands in tfstate. This mirrors
// the idiomatic connected-mode pattern and the old provider's dropped
// `anypoint_omnigateway_registration_token` concept.
type SelfManagedGatewayResource struct {
	client *apimanagement.SelfManagedGatewayClient
}

// SelfManagedGatewayResourceModel is the Terraform state model.
type SelfManagedGatewayResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	OrganizationID    types.String `tfsdk:"organization_id"`
	EnvironmentID     types.String `tfsdk:"environment_id"`
	RegistrationToken types.String `tfsdk:"registration_token"`
	GatewayID         types.String `tfsdk:"gateway_id"`
	Status            types.String `tfsdk:"status"`
	LastUpdate        types.String `tfsdk:"last_update"`
}

func NewSelfManagedGatewayResource() resource.Resource {
	return &SelfManagedGatewayResource{}
}

func (r *SelfManagedGatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_self_managed_gateway"
}

func (r *SelfManagedGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a self-managed (connected-mode) Flex/Omni Gateway in Anypoint Platform. " +
			"The platform does not provision the runtime — you run the Flex runtime on your own " +
			"infrastructure and register it with the `registration_token` this resource mints. The " +
			"gateway object appears on the platform once the runtime self-registers.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier: <organization_id>/<environment_id>/<name>. Stable " +
					"across the gateway's lifecycle even before the runtime registers.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the self-managed gateway. Must be unique within the " +
					"organization/environment. Changing it forces a new registration.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				Description: "The environment ID the gateway registers into.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"registration_token": schema.StringAttribute{
				Description: "The registration token minted for this gateway. Feed it to the Flex " +
					"runtime (e.g. `flexctl registration create <name> --mode=connected --token=<token>`) " +
					"so the runtime can self-register. This is a short-lived, one-shot enrollment secret " +
					"and cannot be recovered on import.",
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway_id": schema.StringAttribute{
				Description: "The platform-assigned gateway ID, resolved once the runtime registers. " +
					"Empty until the gateway appears on the platform.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status reported by the registered gateway (e.g. CONNECTED, " +
					"DISCONNECTED). Empty until the runtime registers.",
				Computed: true,
			},
			"last_update": schema.StringAttribute{
				Description: "Timestamp of the gateway's last status update, as reported by the " +
					"platform (RFC 3339). Empty until the runtime registers.",
				Computed: true,
			},
		},
	}
}

func (r *SelfManagedGatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	gwClient, err := apimanagement.NewSelfManagedGatewayClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Self-Managed Gateway API Client",
			"An unexpected error occurred when creating the Self-Managed Gateway API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = gwClient
}

// --- CRUD ---

func (r *SelfManagedGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SelfManagedGatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	name := data.Name.ValueString()

	// Step 1 of the connected-mode flow: mint the registration token. The token is the
	// deliverable the operator feeds to the Flex runtime; Steps 2-3 (keypair + CSR + cert
	// signing) happen on the runtime host via flexctl, so no key material touches tfstate.
	tokenResp, err := r.client.MintRegistrationToken(ctx, orgID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error minting self-managed gateway registration token",
			"Could not mint a registration token for gateway "+name+": "+err.Error(),
		)
		return
	}

	data.OrganizationID = types.StringValue(orgID)
	data.RegistrationToken = types.StringValue(tokenResp.RegistrationToken)
	data.ID = types.StringValue(buildSelfManagedID(orgID, envID, name))

	// The gateway object does not exist yet — it materializes when the runtime registers.
	// Try to resolve it (in case a runtime with this name already registered), but treat
	// absence as normal.
	r.resolveGateway(ctx, &data, orgID, envID, name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SelfManagedGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SelfManagedGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()
	name := data.Name.ValueString()

	// Best-effort resolve. If the gateway has not registered yet, we keep the resource in
	// state (the minted token is still valid / pending) rather than removing it — removing
	// it would force a token re-mint on every apply.
	r.resolveGateway(ctx, &data, orgID, envID, name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update handles in-place changes. name and environment_id are RequiresReplace, and every
// other non-computed field is derived, so there is no mutable server-side state to push.
// We simply re-resolve the registered gateway and carry the minted token forward.
func (r *SelfManagedGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SelfManagedGatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SelfManagedGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := plan.EnvironmentID.ValueString()
	name := plan.Name.ValueString()

	// Carry forward the values that only Create can produce.
	plan.OrganizationID = types.StringValue(orgID)
	plan.ID = types.StringValue(buildSelfManagedID(orgID, envID, name))
	plan.RegistrationToken = state.RegistrationToken

	r.resolveGateway(ctx, &plan, orgID, envID, name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SelfManagedGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SelfManagedGatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	// Nothing to delete on the platform if the runtime never registered (a minted token
	// cannot be "un-minted"; it simply expires). Only delete when we have a real gateway id.
	gatewayID := data.GatewayID.ValueString()
	if gatewayID == "" {
		tflog.Info(ctx, "Self-managed gateway has no registered platform object; nothing to delete",
			map[string]interface{}{"name": data.Name.ValueString()})
		return
	}

	if err := r.client.DeleteSelfManagedGateway(ctx, orgID, envID, gatewayID); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting self-managed gateway",
			"Could not delete self-managed gateway "+gatewayID+": "+err.Error(),
		)
		return
	}
}

// ImportState supports two import ID formats:
//   - "<environmentID>/<name>"                   — falls back to the provider-credentials org
//   - "<organizationID>/<environmentID>/<name>"  — required when the gateway lives in a sub-org
//
// Note: registration_token is a one-shot secret and cannot be recovered on import; it will
// be null after importing.
func (r *SelfManagedGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	case 3:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[2])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: environment_id/name or organization_id/environment_id/name")
	}
}

// --- Helpers ---

// buildSelfManagedID builds the synthetic composite id for the resource.
func buildSelfManagedID(orgID, envID, name string) string {
	return orgID + "/" + envID + "/" + name
}

// resolveGateway looks up the registered gateway by name and, if found, populates the
// computed platform fields (gateway_id / status / last_update). When the gateway has not
// registered yet it leaves those fields as empty strings (known, not unknown) so the
// framework is satisfied after apply and no spurious drift is reported. Lookup failures
// are logged and swallowed — a transient list error must not fail the whole apply, since
// the durable state (the minted token) is already captured.
//
// IMPORTANT: DELETE is a soft-delete — the platform never removes the gateway object; it
// lingers in the list forever with status "DELETED" (a tombstone). We therefore SKIP any
// gateway whose status is DELETED when matching by name. Without this, a `terraform destroy`
// followed by a refresh would re-bind gateway_id/status to the dead object (spurious drift),
// and a re-registered gateway that reuses the name could resolve to the tombstone instead of
// the live runtime. Skipping DELETED also means that after a destroy the fields fall through
// to known+empty, which is the correct "not registered" representation.
func (r *SelfManagedGatewayResource) resolveGateway(ctx context.Context, data *SelfManagedGatewayResourceModel, orgID, envID, name string) {
	gateways, err := r.client.ListSelfManagedGateways(ctx, orgID, envID)
	if err != nil {
		tflog.Warn(ctx, "Could not list self-managed gateways to resolve registration status",
			map[string]interface{}{"error": err.Error(), "name": name})
		// Ensure computed fields are known (empty) rather than unknown.
		setEmptyIfUnknown(data)
		return
	}

	for i := range gateways {
		if gateways[i].Name != name {
			continue
		}
		// Skip soft-deleted tombstones so a destroyed/renamed gateway does not re-bind.
		if strings.EqualFold(gateways[i].Status, apimanagement.SelfManagedGatewayStatusDeleted) {
			tflog.Debug(ctx, "Skipping soft-deleted (tombstone) self-managed gateway during name resolution",
				map[string]interface{}{"name": name, "gateway_id": gateways[i].ID})
			continue
		}
		data.GatewayID = types.StringValue(gateways[i].ID)
		data.Status = types.StringValue(gateways[i].Status)
		data.LastUpdate = types.StringValue(gateways[i].LastUpdate)
		return
	}

	// No LIVE gateway with this name — either it never registered, or only a DELETED
	// tombstone remains (Bug B). This is a DEFINITIVE "not currently registered" answer, so
	// the computed fields must be CLEARED unconditionally. We must NOT merely fill-if-unknown
	// here: on the Read/refresh path the fields arrive as prior known state (e.g. a gateway
	// that was deleted out-of-band still carries gateway_id/status="CONNECTED"), and leaving
	// those stale values in place would hide the deletion (no drift reported) and defeat the
	// tombstone skip above. Clearing to empty is the correct "not registered" representation.
	clearGatewayFields(data)
}

// clearGatewayFields resets the resolved/computed gateway fields to known-empty. Used when
// the gateway is definitively not currently registered (no live match / only a tombstone),
// overriding any stale prior-state values so an out-of-band delete surfaces correctly.
func clearGatewayFields(data *SelfManagedGatewayResourceModel) {
	data.GatewayID = types.StringValue("")
	data.Status = types.StringValue("")
	data.LastUpdate = types.StringValue("")
}

// setEmptyIfUnknown fills any still-unknown computed field with an empty string so the
// framework never sees an unknown value after apply. Unlike clearGatewayFields it PRESERVES
// existing known values — used on the list-error path, where a transient lookup failure must
// not wipe good state; we only satisfy the framework's "no unknowns after apply" invariant.
func setEmptyIfUnknown(data *SelfManagedGatewayResourceModel) {
	if data.GatewayID.IsUnknown() || data.GatewayID.IsNull() {
		data.GatewayID = types.StringValue("")
	}
	if data.Status.IsUnknown() || data.Status.IsNull() {
		data.Status = types.StringValue("")
	}
	if data.LastUpdate.IsUnknown() || data.LastUpdate.IsNull() {
		data.LastUpdate = types.StringValue("")
	}
}
