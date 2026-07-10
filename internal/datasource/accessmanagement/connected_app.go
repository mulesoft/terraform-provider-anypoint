package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &ConnectedAppDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectedAppDataSource{}
)

// ConnectedAppDataSource is the data source implementation for reading a single connected app.
type ConnectedAppDataSource struct {
	appClient    *accessmanagement.ConnectedAppClient
	scopesClient *accessmanagement.ConnectedAppScopesClient
}

// ConnectedAppDataSourceModel describes the data source data model.
type ConnectedAppDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationID types.String `tfsdk:"organization_id"`
	GrantTypes     types.List   `tfsdk:"grant_types"`
	RedirectURIs   types.List   `tfsdk:"redirect_uris"`
	Audience       types.String `tfsdk:"audience"`
	ClientURI      types.String `tfsdk:"client_uri"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	OwnerUserID    types.String `tfsdk:"owner_user_id"`
	Scopes         types.Set    `tfsdk:"scopes"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// connectedAppDSScopeObjectType is the object type for a scope entry.
var connectedAppDSScopeObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"scope":          types.StringType,
		"context_params": types.MapType{ElemType: types.StringType},
	},
}

func NewConnectedAppDataSource() datasource.DataSource {
	return &ConnectedAppDataSource{}
}

// Metadata returns the data source type name.
func (d *ConnectedAppDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

// Schema defines the schema for the data source.
func (d *ConnectedAppDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a single Anypoint Connected Application by its client ID, including its assigned scopes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The client ID of the connected application to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The display name of the connected application.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the connected app resides. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"grant_types": schema.ListAttribute{
				Description: "The OAuth grant types configured for this application (e.g., client_credentials, authorization_code).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"redirect_uris": schema.ListAttribute{
				Description: "The registered redirect URIs for this application (used with authorization_code grant type).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"audience": schema.StringAttribute{
				Description: "Who can use this application. 'internal' means only members of this organization can authorize; 'everyone' means all Anypoint Platform users can authorize.",
				Computed:    true,
			},
			"client_uri": schema.StringAttribute{
				Description: "The website URL for this application.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the application is currently enabled.",
				Computed:    true,
			},
			"owner_user_id": schema.StringAttribute{
				Description: "The user ID of the application owner.",
				Computed:    true,
			},
			"scopes": schema.SetNestedAttribute{
				Description: "The scopes assigned to this connected application. For client_credentials apps, these are context-aware scopes from the /scopes subresource. For user-behalf apps (authorization_code, password, jwt-bearer), these are flat scopes from the app body.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scope": schema.StringAttribute{
							Description: "The scope identifier (e.g., 'read:applications', 'create:generations').",
							Computed:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the scope (e.g., org, envId). Only populated for client_credentials grant type scopes.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the application was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the application was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ConnectedAppDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	userConfig := config.ToUserClientConfig()

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

	d.appClient = appClient
	d.scopesClient = scopesClient
}

// Read refreshes the Terraform state with the latest data.
func (d *ConnectedAppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectedAppDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.appClient.OrgID
	}

	clientID := data.ID.ValueString()

	// Get the connected app
	app, err := d.appClient.GetConnectedApp(ctx, orgID, clientID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading connected app",
			"Could not read connected app "+clientID+": "+err.Error(),
		)
		return
	}

	// Populate scalar fields
	data.ID = types.StringValue(app.ClientID)
	data.Name = types.StringValue(app.ClientName)
	data.OrganizationID = types.StringValue(app.OrgID)
	data.Audience = types.StringValue(app.Audience)
	data.ClientURI = types.StringValue(app.ClientURI)
	data.Enabled = types.BoolValue(app.Enabled)
	data.OwnerUserID = types.StringValue(app.OwnerUserID)
	data.CreatedAt = types.StringValue(app.CreatedAt)
	data.UpdatedAt = types.StringValue(app.UpdatedAt)

	// Populate grant_types list
	grantTypesList, diags := types.ListValueFrom(ctx, types.StringType, app.GrantTypes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.GrantTypes = grantTypesList

	// Populate redirect_uris list
	redirectURIs := app.RedirectURIs
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	redirectURIsList, diags := types.ListValueFrom(ctx, types.StringType, redirectURIs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RedirectURIs = redirectURIsList

	// Populate scopes - approach depends on grant type
	scopeSet, scopeDiags := d.readScopes(ctx, app)
	resp.Diagnostics.Append(scopeDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Scopes = scopeSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// readScopes fetches scopes for the connected app.
// For client_credentials apps, scopes come from the context-aware /scopes subresource.
// For user-behalf apps, scopes are flat strings from the app body.
func (d *ConnectedAppDataSource) readScopes(ctx context.Context, app *accessmanagement.ConnectedApp) (types.Set, diag.Diagnostics) {
	var allDiags diag.Diagnostics

	isClientCredentials := false
	for _, gt := range app.GrantTypes {
		if gt == "client_credentials" {
			isClientCredentials = true
			break
		}
	}

	var scopeObjects []attr.Value

	if isClientCredentials {
		// Fetch scopes from the subresource API
		scopesResp, err := d.scopesClient.GetConnectedAppScopes(ctx, app.ClientID)
		if err != nil {
			allDiags.AddError("Error reading connected app scopes",
				"Could not read scopes for connected app "+app.ClientID+": "+err.Error())
			return types.SetNull(connectedAppDSScopeObjectType), allDiags
		}

		for _, s := range scopesResp.Scopes {
			contextParams := types.MapNull(types.StringType)
			if len(s.ContextParams) > 0 {
				cpMap := make(map[string]attr.Value, len(s.ContextParams))
				for k, v := range s.ContextParams {
					var strVal string
					switch val := v.(type) {
					case string:
						strVal = val
					default:
						strVal = fmt.Sprintf("%v", val)
					}
					cpMap[k] = types.StringValue(strVal)
				}
				var mapDiags diag.Diagnostics
				contextParams, mapDiags = types.MapValue(types.StringType, cpMap)
				allDiags.Append(mapDiags...)
				if allDiags.HasError() {
					return types.SetNull(connectedAppDSScopeObjectType), allDiags
				}
			}

			obj, objDiags := types.ObjectValue(connectedAppDSScopeObjectType.AttrTypes, map[string]attr.Value{
				"scope":          types.StringValue(s.Scope),
				"context_params": contextParams,
			})
			allDiags.Append(objDiags...)
			if allDiags.HasError() {
				return types.SetNull(connectedAppDSScopeObjectType), allDiags
			}
			scopeObjects = append(scopeObjects, obj)
		}
	} else {
		// User-behalf apps: scopes are flat strings in the app body (no context_params)
		for _, s := range app.Scopes {
			obj, objDiags := types.ObjectValue(connectedAppDSScopeObjectType.AttrTypes, map[string]attr.Value{
				"scope":          types.StringValue(s),
				"context_params": types.MapNull(types.StringType),
			})
			allDiags.Append(objDiags...)
			if allDiags.HasError() {
				return types.SetNull(connectedAppDSScopeObjectType), allDiags
			}
			scopeObjects = append(scopeObjects, obj)
		}
	}

	if scopeObjects == nil {
		scopeObjects = []attr.Value{}
	}

	scopeSet, setDiags := types.SetValue(connectedAppDSScopeObjectType, scopeObjects)
	allDiags.Append(setDiags...)
	return scopeSet, allDiags
}
