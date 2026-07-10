package accessmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/constants"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &ConnectedAppResource{}
	_ resource.ResourceWithConfigure   = &ConnectedAppResource{}
	_ resource.ResourceWithImportState = &ConnectedAppResource{}
)

// ConnectedAppResource is the resource implementation.
type ConnectedAppResource struct {
	client       *accessmanagement.ConnectedAppClient
	scopesClient *accessmanagement.ConnectedAppScopesClient
}

// ConnectedAppResourceModel describes the resource data model.
type ConnectedAppResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	GrantTypes     types.List   `tfsdk:"grant_types"`
	RedirectURIs   types.List   `tfsdk:"redirect_uris"`
	PublicKeys     types.List   `tfsdk:"public_keys"`
	Audience       types.String `tfsdk:"audience"`
	ClientURI      types.String `tfsdk:"client_uri"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	OrganizationID types.String `tfsdk:"organization_id"`
	OwnerUserID    types.String `tfsdk:"owner_user_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	Scopes         types.Set    `tfsdk:"scopes"`
}

// connectedAppScopeObjectType is the object type of a single element in the inline `scopes` set.
var connectedAppScopeObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"scope":          types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

// systemConnectedAppScopes are scopes the platform auto-injects on every connected app and that
// cannot be removed (verified on devx 2026-07-07). They are never user-manageable, so the
// reconcile filters them out of state and validation rejects them if listed explicitly. Keeping
// them out of state is what makes the authoritative `scopes` attribute idempotent — otherwise the
// injected "profile" scope would show up as a perpetual diff (same class of bug as the team
// "Business Group Viewer" side-effect).
var systemConnectedAppScopes = map[string]bool{
	"profile": true,
}

func NewConnectedAppResource() resource.Resource {
	return &ConnectedAppResource{}
}

// Metadata returns the resource type name.
func (r *ConnectedAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

// Schema defines the schema for the resource.
func (r *ConnectedAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Anypoint Connected Application. " +
			"Connected apps provide a framework for programmatic access to the Anypoint Platform APIs. " +
			"Two types are supported: 'App acts on behalf of a user' (authorization_code/password/jwt_bearer grants) " +
			"and 'App acts on its own behalf' (client_credentials grant).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The client_id of the connected app (unique identifier).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the connected app (shown as 'Name' in the UI).",
				Required:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret. Only returned on creation. Sensitive.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"grant_types": schema.ListAttribute{
				Description: "The OAuth grant types. Valid values: 'authorization_code', 'password', " +
					"'urn:ietf:params:oauth:grant-type:jwt-bearer' (for apps on behalf of a user), " +
					"or 'client_credentials' (for apps on their own behalf).",
				Required:    true,
				ElementType: types.StringType,
			},
			"redirect_uris": schema.ListAttribute{
				Description: "OAuth redirect URIs. Required for 'authorization_code' grant type.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"public_keys": schema.ListAttribute{
				Description: "Public keys for JWT Bearer grant type.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"audience": schema.StringAttribute{
				Description: "Who can use this application. 'internal' = members of this organization only, " +
					"'everyone' = all Anypoint Platform users. Note: client_credentials apps are always " +
					"'internal' (the platform enforces this).",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("internal"),
				Validators: []validator.String{
					stringvalidator.OneOf("internal", "everyone"),
				},
			},
			"client_uri": schema.StringAttribute{
				Description: "Website URL where users can learn more about the app.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the connected app is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider's org.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_user_id": schema.StringAttribute{
				Description: "The user ID of the app owner.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "When the connected app was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "When the connected app was last updated.",
				Computed:    true,
			},
			"scopes": schema.SetNestedAttribute{
				Description: "Context-aware scopes assigned to the connected application. AUTHORITATIVE when set: " +
					"the provider makes the app's scopes match this set exactly (extra scopes assigned out-of-band " +
					"are removed on the next apply). Omit the block entirely to leave scopes unmanaged; set it to an " +
					"empty list ([]) to remove all user-assigned scopes. The platform auto-assigns an undeletable " +
					"'profile' scope to every app — it is managed by the platform, never appears in this set, and " +
					"must not be listed here. Scopes are orthogonal to grant type (they apply to both " +
					"client_credentials and user-behalf apps). Prefer this over the separate " +
					"anypoint_connected_app_scopes resource, which is deprecated.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scope": schema.StringAttribute{
							Description: "The scope identifier (e.g., 'read:applications', 'admin:cloudhub') or display " +
								"name (e.g., 'Cloudhub Organization Admin'). Display names are resolved to identifiers automatically. " +
								"Use the anypoint_available_scopes data source to discover available scopes.",
							Required: true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the scope. Always include 'org'; add 'envId' for " +
								"environment-scoped scopes.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ConnectedAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	userConfig := &client.UserClientConfig{
		BaseURL:      config.BaseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
		Timeout:      config.Timeout,
	}

	appClient, err := accessmanagement.NewConnectedAppClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Connected App Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	scopesClient, err := accessmanagement.NewConnectedAppScopesClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Connected App Scopes Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	r.client = appClient
	r.scopesClient = scopesClient
}

// Create creates a connected app.
func (r *ConnectedAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectedAppResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Convert grant_types from Terraform list to []string
	var grantTypes []string
	resp.Diagnostics.Append(plan.GrantTypes.ElementsAs(ctx, &grantTypes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert redirect_uris
	var redirectURIs []string
	if !plan.RedirectURIs.IsNull() && !plan.RedirectURIs.IsUnknown() {
		resp.Diagnostics.Append(plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert public_keys
	var publicKeys []string
	if !plan.PublicKeys.IsNull() && !plan.PublicKeys.IsUnknown() {
		resp.Diagnostics.Append(plan.PublicKeys.ElementsAs(ctx, &publicKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	createReq := &accessmanagement.CreateConnectedAppRequest{
		ClientName:   plan.Name.ValueString(),
		GrantTypes:   grantTypes,
		RedirectURIs: redirectURIs,
		PublicKeys:   publicKeys,
		Audience:     plan.Audience.ValueString(),
	}

	// Only set client_uri if provided
	if !plan.ClientURI.IsNull() && !plan.ClientURI.IsUnknown() && plan.ClientURI.ValueString() != "" {
		createReq.ClientURI = plan.ClientURI.ValueString()
	}

	app, err := r.client.CreateConnectedApp(ctx, orgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating connected app",
			"Could not create connected app: "+err.Error(),
		)
		return
	}

	// Set state from response — client_secret is only available at creation
	plan.ID = types.StringValue(app.ClientID)
	plan.ClientSecret = types.StringValue(app.ClientSecret)
	plan.OrganizationID = types.StringValue(app.OrgID)
	plan.OwnerUserID = types.StringValue(app.OwnerUserID)
	plan.Enabled = types.BoolValue(app.Enabled)
	plan.CreatedAt = types.StringValue(app.CreatedAt)
	plan.UpdatedAt = types.StringValue(app.UpdatedAt)

	// Set lists from API response
	r.setListsFromAPI(ctx, &plan, app, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always set client_uri to avoid "unknown after apply" errors
	plan.ClientURI = types.StringValue(app.ClientURI)

	// Apply inline scopes if the block is set. Null => unmanaged.
	// Per RAML spec: context-aware /scopes endpoint is ONLY for client_credentials.
	// User-behalf apps use the flat body `scopes` field via PATCH.
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		if isClientCredentials(grantTypes) {
			// client_credentials: use context-aware PUT /scopes endpoint
			if diags := r.applyScopes(ctx, app.ClientID, plan.Scopes); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		} else {
			// user-behalf: resolve scopes and PATCH the body field
			resolved, diags := resolveUserBehalfScopes(plan.Scopes)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			if len(resolved) > 0 {
				orgIDForUpdate := app.OrgID
				if orgIDForUpdate == "" {
					orgIDForUpdate = r.client.OrgID
				}
				updateReq := &accessmanagement.UpdateConnectedAppRequest{Scopes: resolved}
				if _, err := r.client.UpdateConnectedApp(ctx, orgIDForUpdate, app.ClientID, updateReq); err != nil {
					resp.Diagnostics.AddError(
						"Error setting connected app scopes",
						"Could not set scopes via PATCH: "+err.Error(),
					)
					return
				}
			}
		}
	}

	tflog.Trace(ctx, "created connected app", map[string]interface{}{
		"client_id": app.ClientID,
		"name":      app.ClientName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ConnectedAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectedAppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	app, err := r.client.GetConnectedApp(ctx, orgID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading connected app",
			"Could not read connected app: "+err.Error(),
		)
		return
	}

	// Update state — but preserve client_secret (not returned on GET)
	state.Name = types.StringValue(app.ClientName)
	state.OrganizationID = types.StringValue(app.OrgID)
	state.OwnerUserID = types.StringValue(app.OwnerUserID)
	state.Enabled = types.BoolValue(app.Enabled)
	state.Audience = types.StringValue(app.Audience)
	state.ClientURI = types.StringValue(app.ClientURI)
	state.CreatedAt = types.StringValue(app.CreatedAt)
	state.UpdatedAt = types.StringValue(app.UpdatedAt)

	r.setListsFromAPI(ctx, &state, app, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reconcile scopes only when they are being managed (state.Scopes non-null).
	// Per RAML spec: context-aware /scopes is ONLY for client_credentials.
	// User-behalf apps read from the flat body scopes field.
	if !state.Scopes.IsNull() {
		if isClientCredentials(app.GrantTypes) {
			reconciled, diags := r.reconcileScopesIntoState(ctx, state.ID.ValueString(), state.Scopes)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.Scopes = reconciled
		} else {
			// User-behalf: read from the flat body scopes field (already in app.Scopes)
			reconciled, diags := reconcileUserBehalfScopesIntoState(app.Scopes, state.Scopes)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.Scopes = reconciled
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update modifies the connected app via PATCH.
func (r *ConnectedAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ConnectedAppResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Build PATCH request with only changed fields
	updateReq := &accessmanagement.UpdateConnectedAppRequest{}

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		updateReq.ClientName = &name
	}

	if !plan.Audience.Equal(state.Audience) {
		aud := plan.Audience.ValueString()
		updateReq.Audience = &aud
	}

	if !plan.ClientURI.Equal(state.ClientURI) {
		uri := plan.ClientURI.ValueString()
		updateReq.ClientURI = &uri
	}

	if !plan.Enabled.Equal(state.Enabled) {
		enabled := plan.Enabled.ValueBool()
		updateReq.Enabled = &enabled
	}

	if !plan.GrantTypes.Equal(state.GrantTypes) {
		var grantTypes []string
		resp.Diagnostics.Append(plan.GrantTypes.ElementsAs(ctx, &grantTypes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.GrantTypes = grantTypes
	}

	if !plan.RedirectURIs.Equal(state.RedirectURIs) {
		var redirectURIs []string
		if !plan.RedirectURIs.IsNull() && !plan.RedirectURIs.IsUnknown() {
			resp.Diagnostics.Append(plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		updateReq.RedirectURIs = redirectURIs
	}

	if !plan.PublicKeys.Equal(state.PublicKeys) {
		var publicKeys []string
		if !plan.PublicKeys.IsNull() && !plan.PublicKeys.IsUnknown() {
			resp.Diagnostics.Append(plan.PublicKeys.ElementsAs(ctx, &publicKeys, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		updateReq.PublicKeys = publicKeys
	}

	app, err := r.client.UpdateConnectedApp(ctx, orgID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating connected app",
			"Could not update connected app: "+err.Error(),
		)
		return
	}

	// Update state from response — preserve client_secret
	plan.ID = state.ID
	plan.ClientSecret = state.ClientSecret
	plan.OrganizationID = types.StringValue(app.OrgID)
	plan.OwnerUserID = types.StringValue(app.OwnerUserID)
	plan.Enabled = types.BoolValue(app.Enabled)
	plan.Audience = types.StringValue(app.Audience)
	plan.ClientURI = types.StringValue(app.ClientURI)
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = types.StringValue(app.UpdatedAt)

	r.setListsFromAPI(ctx, &plan, app, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reconcile scopes when the plan differs from state. Per RAML spec: context-aware /scopes
	// is ONLY for client_credentials. User-behalf apps use the flat body scopes field.
	if !plan.Scopes.Equal(state.Scopes) && !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		var planGrantTypes []string
		resp.Diagnostics.Append(plan.GrantTypes.ElementsAs(ctx, &planGrantTypes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if isClientCredentials(planGrantTypes) {
			if diags := r.applyScopes(ctx, state.ID.ValueString(), plan.Scopes); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		} else {
			// user-behalf: resolve and PATCH body
			resolved, diags := resolveUserBehalfScopes(plan.Scopes)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			scopeReq := &accessmanagement.UpdateConnectedAppRequest{Scopes: resolved}
			if _, err := r.client.UpdateConnectedApp(ctx, orgID, state.ID.ValueString(), scopeReq); err != nil {
				resp.Diagnostics.AddError(
					"Error updating connected app scopes",
					"Could not set scopes via PATCH: "+err.Error(),
				)
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the connected app.
func (r *ConnectedAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectedAppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	err := r.client.DeleteConnectedApp(ctx, orgID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting connected app",
			"Could not delete connected app: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted connected app")
}

// ImportState imports a connected app by client_id or org_id:client_id.
func (r *ConnectedAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Support both "client_id" and "org_id:client_id" formats
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) == 2 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	}

	// Note: client_secret is not available after import — it's only shown at creation.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_secret"), "")...)

	// Seed scopes as empty set so the subsequent Read populates them from the API.
	// (null would cause Read to skip them as "unmanaged".)
	emptyScopes, diags := types.SetValue(connectedAppScopeObjectType, []attr.Value{})
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scopes"), emptyScopes)...)
}

// setListsFromAPI converts API response arrays to Terraform list types.
func (r *ConnectedAppResource) setListsFromAPI(ctx context.Context, model *ConnectedAppResourceModel, app *accessmanagement.ConnectedApp, diags *diag.Diagnostics) {
	// Grant types
	grantTypesList, d := types.ListValueFrom(ctx, types.StringType, app.GrantTypes)
	diags.Append(d...)
	if !diags.HasError() {
		model.GrantTypes = grantTypesList
	}

	// Redirect URIs
	if app.RedirectURIs == nil {
		app.RedirectURIs = []string{}
	}
	redirectList, d := types.ListValueFrom(ctx, types.StringType, app.RedirectURIs)
	diags.Append(d...)
	if !diags.HasError() {
		model.RedirectURIs = redirectList
	}

	// Public keys
	if app.PublicKeys == nil {
		app.PublicKeys = []string{}
	}
	publicKeysList, d := types.ListValueFrom(ctx, types.StringType, app.PublicKeys)
	diags.Append(d...)
	if !diags.HasError() {
		model.PublicKeys = publicKeysList
	}
}

// isClientCredentials returns true if the grant types include client_credentials.
// Per RAML spec, the context-aware /scopes subresource is ONLY for client_credentials apps.
// User-behalf apps (authorization_code, password, jwt-bearer) use the flat body scopes field.
func isClientCredentials(grantTypes []string) bool {
	for _, gt := range grantTypes {
		if gt == "client_credentials" {
			return true
		}
	}
	return false
}

// resolveUserBehalfScopes resolves the typed scope set into a flat []string of identifiers
// for user-behalf apps. These get sent in the body `scopes` field via PATCH.
// context_params are ignored for user-behalf apps (the platform determines context from the token).
func resolveUserBehalfScopes(scopeSet types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if scopeSet.IsNull() || scopeSet.IsUnknown() {
		return nil, diags
	}

	out := make([]string, 0, len(scopeSet.Elements()))
	for i, el := range scopeSet.Elements() {
		attrs := el.(types.Object).Attributes()
		typed := attrs["scope"].(types.String).ValueString()

		resolved, ok := constants.ResolveScopeIdentifier(typed)
		if !ok {
			diags.AddError(
				"Invalid Scope Name",
				fmt.Sprintf("The scope %q at index %d is not a valid Anypoint Platform scope. Use either the scope "+
					"identifier (e.g. 'read:applications') or the display name (e.g. 'Cloudhub Organization Admin'). Use the "+
					"anypoint_available_scopes data source to discover valid scopes.", typed, i),
			)
			continue
		}
		out = append(out, resolved)
	}
	return out, diags
}

// reconcileUserBehalfScopesIntoState builds the state set from the flat scopes array in the app body.
// It preserves the user's typed representation (display name) for matched entries.
func reconcileUserBehalfScopesIntoState(apiScopes []string, typedSource types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build typed lookup: resolved identifier -> typed scope attrs
	typedByID := map[string]typedScope{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			attrs := el.(types.Object).Attributes()
			nameVal := attrs["scope"].(types.String)
			cpVal := attrs["context_params"].(types.Map)
			resolved, _ := constants.ResolveScopeIdentifier(nameVal.ValueString())
			typedByID[resolved] = typedScope{name: nameVal, cp: cpVal}
		}
	}

	objs := make([]attr.Value, 0, len(apiScopes))
	for _, scopeID := range apiScopes {
		// Skip the platform-injected "profile" scope
		if systemConnectedAppScopes[scopeID] {
			continue
		}

		var nameVal types.String
		var cpVal types.Map
		if ts, ok := typedByID[scopeID]; ok {
			// Preserve the user's typed form (display name/casing)
			nameVal = ts.name
			cpVal = ts.cp
		} else {
			// Out-of-band scope (e.g. after import): reverse-resolve to display name
			// so state matches what users write in config.
			scopeName := scopeID
			if dn, ok := constants.GetDisplayName(scopeID); ok {
				scopeName = dn
			}
			nameVal = types.StringValue(scopeName)
			cpVal = stringMapToTypesMap(nil)
		}

		obj, d := types.ObjectValue(connectedAppScopeObjectType.AttrTypes, map[string]attr.Value{
			"scope":          nameVal,
			"context_params": cpVal,
		})
		if d.HasError() {
			diags.Append(d...)
			return types.SetNull(connectedAppScopeObjectType), diags
		}
		objs = append(objs, obj)
	}

	set, d := types.SetValue(connectedAppScopeObjectType, objs)
	if d.HasError() {
		diags.Append(d...)
		return types.SetNull(connectedAppScopeObjectType), diags
	}
	return set, diags
}

// This file implements the inline, authoritative-when-set `scopes` attribute on
// anypoint_connected_app. It mirrors the team roles/members reconcile pattern:
//   - resolve the user's typed scopes (display name OR identifier) to canonical identifiers,
//   - apply them authoritatively (PUT replaces the whole list; empty => DELETE all non-system),
//   - reconcile the live list back into state while (a) skipping platform-injected system scopes
//     (e.g. the undeletable "profile") and (b) preserving the user's typed representation to
//     avoid perpetual diffs.

// typedScope holds the user's original (typed) representation of a scope so Read can preserve the
// exact form (display name vs identifier, and the context_params map) and avoid perpetual diffs.
type typedScope struct {
	name types.String
	cp   types.Map
}

// validateAndResolveScopes validates every scope name and returns the resolved API scopes.
// Accepts identifiers ("read:applications") and display names ("Cloudhub Organization Admin"). Rejects
// system scopes (e.g. "profile") — they are platform-managed and undeletable, so listing them
// would create a perpetual diff. Returns (nil, nil) when the set is null/unknown (unmanaged).
func validateAndResolveScopes(scopeSet types.Set) ([]accessmanagement.Scope, diag.Diagnostics) {
	var diags diag.Diagnostics
	if scopeSet.IsNull() || scopeSet.IsUnknown() {
		return nil, diags
	}

	out := make([]accessmanagement.Scope, 0, len(scopeSet.Elements()))
	for i, el := range scopeSet.Elements() {
		attrs := el.(types.Object).Attributes()
		typed := attrs["scope"].(types.String).ValueString()

		if systemConnectedAppScopes[typed] {
			diags.AddError(
				"System Scope Not Manageable",
				fmt.Sprintf("The scope %q at index %d is automatically assigned by the platform and cannot be "+
					"managed by Terraform. Remove it from the scopes list.", typed, i),
			)
			continue
		}

		resolved, ok := constants.ResolveScopeIdentifier(typed)
		if !ok {
			diags.AddError(
				"Invalid Scope Name",
				fmt.Sprintf("The scope %q at index %d is not a valid Anypoint Platform scope. Use either the scope "+
					"identifier (e.g. 'read:applications') or the display name (e.g. 'Cloudhub Organization Admin'). Use the "+
					"anypoint_available_scopes data source to discover valid scopes.", typed, i),
			)
			continue
		}
		if systemConnectedAppScopes[resolved] {
			diags.AddError(
				"System Scope Not Manageable",
				fmt.Sprintf("The scope %q (resolves to %q) at index %d is platform-managed and cannot be set.", typed, resolved, i),
			)
			continue
		}

		cp := map[string]interface{}{}
		for k, v := range mapToStringMap(attrs["context_params"].(types.Map)) {
			cp[k] = v
		}
		out = append(out, accessmanagement.Scope{Scope: resolved, ContextParams: cp})
	}
	return out, diags
}

// applyScopes reconciles the connected app's scopes to exactly `scopeSet` (authoritative).
//   - non-empty desired => PUT (replaces the whole list; the platform keeps its injected "profile").
//   - empty desired ([]) => DELETE every current non-system scope (PUT cannot send an empty list).
func (r *ConnectedAppResource) applyScopes(ctx context.Context, clientID string, scopeSet types.Set) diag.Diagnostics {
	desired, diags := validateAndResolveScopes(scopeSet)
	if diags.HasError() {
		return diags
	}

	if len(desired) > 0 {
		if _, err := r.scopesClient.ReplaceConnectedAppScopes(ctx, clientID, desired); err != nil {
			diags.AddError("Error setting connected app scopes", "Could not replace connected app scopes: "+err.Error())
		}
		return diags
	}

	// Authoritative-empty: remove every currently-assigned non-system scope.
	current, err := r.scopesClient.GetConnectedAppScopes(ctx, clientID)
	if err != nil {
		diags.AddError("Error reading connected app scopes", "Could not read current scopes before clearing: "+err.Error())
		return diags
	}
	toRemove := make([]accessmanagement.Scope, 0, len(current.Scopes))
	for _, s := range current.Scopes {
		if systemConnectedAppScopes[s.Scope] {
			continue
		}
		toRemove = append(toRemove, s)
	}
	if len(toRemove) > 0 {
		if err := r.scopesClient.RemoveConnectedAppScopes(ctx, clientID, toRemove); err != nil {
			diags.AddError("Error clearing connected app scopes", "Could not remove connected app scopes: "+err.Error())
		}
	}
	return diags
}

// reconcileScopesIntoState reads the live scopes and returns a set suitable for state, skipping
// platform-injected system scopes and preserving the user's typed representation for matched
// entries. `typedSource` is the prior/config scopes set (may be null => unmanaged).
func (r *ConnectedAppResource) reconcileScopesIntoState(ctx context.Context, clientID string, typedSource types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	scopes, err := r.scopesClient.GetConnectedAppScopes(ctx, clientID)
	if err != nil {
		diags.AddError("Error reading connected app scopes", "Could not list connected app scopes: "+err.Error())
		return types.SetNull(connectedAppScopeObjectType), diags
	}

	// Index the user's typed scopes by (resolved_id | context) so we can preserve their exact
	// representation (display name vs id, context_params form) for matched assignments.
	typedByKey := map[string]typedScope{}
	if !typedSource.IsNull() && !typedSource.IsUnknown() {
		for _, el := range typedSource.Elements() {
			attrs := el.(types.Object).Attributes()
			nameVal := attrs["scope"].(types.String)
			cpVal := attrs["context_params"].(types.Map)
			resolved, _ := constants.ResolveScopeIdentifier(nameVal.ValueString())
			key := resolved + "|" + canonicalContextParams(mapToStringMap(cpVal))
			typedByKey[key] = typedScope{name: nameVal, cp: cpVal}
		}
	}

	objs := make([]attr.Value, 0, len(scopes.Scopes))
	for _, s := range scopes.Scopes {
		// Skip platform-injected, undeletable system scopes (e.g. "profile"). The user cannot
		// express these in config, so treating them as managed would surface a phantom entry and a
		// perpetual diff — same class as the team "Business Group Viewer" side-effect.
		if systemConnectedAppScopes[s.Scope] {
			continue
		}

		key := s.Scope + "|" + canonicalContextParamsIface(s.ContextParams)

		var nameVal types.String
		var cpVal types.Map
		if ts, ok := typedByKey[key]; ok {
			nameVal = ts.name
			cpVal = ts.cp
		} else {
			// After import, typedSource is empty so we have no user-typed representation.
			// Reverse-resolve the API identifier to its display name so that state matches
			// what users write in config (display names, not raw IDs like "read:exchange").
			scopeName := s.Scope
			if dn, ok := constants.GetDisplayName(s.Scope); ok {
				scopeName = dn
			}
			nameVal = types.StringValue(scopeName)
			cpVal = stringMapToTypesMap(ifaceMapToStringMap(s.ContextParams))
		}

		obj, d := types.ObjectValue(connectedAppScopeObjectType.AttrTypes, map[string]attr.Value{
			"scope":          nameVal,
			"context_params": cpVal,
		})
		if d.HasError() {
			diags.Append(d...)
			return types.SetNull(connectedAppScopeObjectType), diags
		}
		objs = append(objs, obj)
	}

	set, d := types.SetValue(connectedAppScopeObjectType, objs)
	if d.HasError() {
		diags.Append(d...)
		return types.SetNull(connectedAppScopeObjectType), diags
	}
	return set, diags
}

// canonicalContextParamsIface is canonicalContextParams for an API scope's map[string]interface{}.
func canonicalContextParamsIface(m map[string]interface{}) string {
	return canonicalContextParams(ifaceMapToStringMap(m))
}

// ifaceMapToStringMap flattens the API's map[string]interface{} context_params to map[string]string.
func ifaceMapToStringMap(m map[string]interface{}) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		} else {
			out[k] = fmt.Sprintf("%v", v)
		}
	}
	return out
}
