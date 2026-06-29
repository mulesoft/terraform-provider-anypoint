package accessmanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &ConnectedAppResource{}
	_ resource.ResourceWithConfigure   = &ConnectedAppResource{}
	_ resource.ResourceWithImportState = &ConnectedAppResource{}
)

// ConnectedAppResource is the resource implementation.
type ConnectedAppResource struct {
	client *accessmanagement.ConnectedAppClient
}

// ConnectedAppResourceModel describes the resource data model.
type ConnectedAppResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ClientName     types.String `tfsdk:"client_name"`
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
			"client_name": schema.StringAttribute{
				Description: "The name of the connected app.",
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
					"'everyone' = all Anypoint Platform users.",
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

	r.client = appClient
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
		ClientName:   plan.ClientName.ValueString(),
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

	tflog.Trace(ctx, "created connected app", map[string]interface{}{
		"client_id":   app.ClientID,
		"client_name": app.ClientName,
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
	state.ClientName = types.StringValue(app.ClientName)
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

	if !plan.ClientName.Equal(state.ClientName) {
		name := plan.ClientName.ValueString()
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
