package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

var (
	_ resource.Resource                = &ConnectedAppResource{}
	_ resource.ResourceWithConfigure   = &ConnectedAppResource{}
	_ resource.ResourceWithImportState = &ConnectedAppResource{}
)

type ConnectedAppResource struct {
	client *accessmanagement.ConnectedAppClient
}

type ConnectedAppResourceModel struct {
	ClientID                     types.String `tfsdk:"client_id"`
	OwnerOrgID                   types.String `tfsdk:"owner_org_id"`
	ClientName                   types.String `tfsdk:"client_name"`
	ClientSecret                 types.String `tfsdk:"client_secret"`
	PublicKeys                   types.List   `tfsdk:"public_keys"`
	RedirectURIs                 types.List   `tfsdk:"redirect_uris"`
	GrantTypes                   types.List   `tfsdk:"grant_types"`
	Scopes                       types.List   `tfsdk:"scopes"`
	Enabled                      types.Bool   `tfsdk:"enabled"`
	Audience                     types.String `tfsdk:"audience"`
	GenerateIssClaimWithoutToken types.Bool   `tfsdk:"generate_iss_claim_without_token"`
}

func NewConnectedAppResource() resource.Resource {
	return &ConnectedAppResource{}
}

func (r *ConnectedAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

func (r *ConnectedAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Connected Applications in Anypoint Platform.",
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				Description: "The client ID of the connected application.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_org_id": schema.StringAttribute{
				Description: "The organization ID that owns the connected application.",
				Required:    true,
			},
			"client_name": schema.StringAttribute{
				Description: "The name of the connected application.",
				Required:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret of the connected application.",
				Required:    true,
				Sensitive:   true,
			},
			"public_keys": schema.ListAttribute{
				Description: "List of public keys for the connected application.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"redirect_uris": schema.ListAttribute{
				Description: "List of redirect URIs for the connected application.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"grant_types": schema.ListAttribute{
				Description: "List of grant types for the connected application.",
				Required:    true,
				ElementType: types.StringType,
			},
			"scopes": schema.ListAttribute{
				Description: "List of scopes for the connected application.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the connected application is enabled.",
				Computed:    true,
				Optional:    true,
			},
			"audience": schema.StringAttribute{
				Description: "The audience for the connected application.",
				Required:    true,
			},
			"generate_iss_claim_without_token": schema.BoolAttribute{
				Description: "Whether to generate iss claim without token.",
				Computed:    true,
				Optional:    true,
			},
		},
	}
}

func (r *ConnectedAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T.", req.ProviderData),
		)
		return
	}

	client, err := accessmanagement.NewConnectedAppClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Connected App Client",
			"An error occurred while creating the connected app client: "+err.Error(),
		)
		return
	}

	r.client = client
}

func (r *ConnectedAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConnectedAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert lists to string slices
	var publicKeys, redirectURIs, grantTypes, scopes []string

	if !plan.PublicKeys.IsNull() {
		diags = plan.PublicKeys.ElementsAs(ctx, &publicKeys, false)
		resp.Diagnostics.Append(diags...)
	}
	if !plan.RedirectURIs.IsNull() {
		diags = plan.RedirectURIs.ElementsAs(ctx, &redirectURIs, false)
		resp.Diagnostics.Append(diags...)
	}
	diags = plan.GrantTypes.ElementsAs(ctx, &grantTypes, false)
	resp.Diagnostics.Append(diags...)
	if !plan.Scopes.IsNull() {
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the connected app
	createReq := &accessmanagement.CreateConnectedAppRequest{
		ClientID:                     plan.ClientID.ValueString(),
		OwnerOrgID:                   plan.OwnerOrgID.ValueString(),
		ClientName:                   plan.ClientName.ValueString(),
		ClientSecret:                 plan.ClientSecret.ValueString(),
		PublicKeys:                   publicKeys,
		RedirectURIs:                 redirectURIs,
		GrantTypes:                   grantTypes,
		Scopes:                       scopes,
		Enabled:                      plan.Enabled.ValueBool(),
		Audience:                     plan.Audience.ValueString(),
		GenerateIssClaimWithoutToken: plan.GenerateIssClaimWithoutToken.ValueBool(),
	}

	connectedApp, err := r.client.CreateConnectedApp(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Connected App", err.Error())
		return
	}

	// Update plan with response values
	plan.ClientID = types.StringValue(connectedApp.ClientID)
	plan.OwnerOrgID = types.StringValue(connectedApp.OwnerOrgID)
	plan.ClientName = types.StringValue(connectedApp.ClientName)
	plan.ClientSecret = types.StringValue(connectedApp.ClientSecret)
	plan.Enabled = types.BoolValue(connectedApp.Enabled)
	plan.Audience = types.StringValue(connectedApp.Audience)
	plan.GenerateIssClaimWithoutToken = types.BoolValue(connectedApp.GenerateIssClaimWithoutToken)

	// Convert response arrays to lists
	plan.PublicKeys, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.PublicKeys)
	resp.Diagnostics.Append(diags...)
	plan.RedirectURIs, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.RedirectURIs)
	resp.Diagnostics.Append(diags...)
	plan.GrantTypes, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.GrantTypes)
	resp.Diagnostics.Append(diags...)
	plan.Scopes, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.Scopes)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created connected app resource")
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ConnectedAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConnectedAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectedApp, err := r.client.GetConnectedApp(ctx, state.ClientID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Connected App", err.Error())
		return
	}

	// Update state with API response
	state.ClientID = types.StringValue(connectedApp.ClientID)
	state.OwnerOrgID = types.StringValue(connectedApp.OwnerOrgID)
	state.ClientName = types.StringValue(connectedApp.ClientName)
	state.ClientSecret = types.StringValue(connectedApp.ClientSecret)
	state.Enabled = types.BoolValue(connectedApp.Enabled)
	state.Audience = types.StringValue(connectedApp.Audience)
	state.GenerateIssClaimWithoutToken = types.BoolValue(connectedApp.GenerateIssClaimWithoutToken)

	state.PublicKeys, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.PublicKeys)
	resp.Diagnostics.Append(diags...)
	state.RedirectURIs, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.RedirectURIs)
	resp.Diagnostics.Append(diags...)
	state.GrantTypes, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.GrantTypes)
	resp.Diagnostics.Append(diags...)
	state.Scopes, diags = types.ListValueFrom(ctx, types.StringType, connectedApp.Scopes)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ConnectedAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Connected App updates are not currently supported. Changes require replacement.",
	)
}

func (r *ConnectedAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConnectedAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteConnectedApp(ctx, state.ClientID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting Connected App", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted connected app resource")
}

func (r *ConnectedAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("client_id"), req, resp)
}
