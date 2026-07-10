package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &ConnectedAppsDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectedAppsDataSource{}
)

// ConnectedAppsDataSource is the data source implementation.
type ConnectedAppsDataSource struct {
	client *accessmanagement.ConnectedAppClient
}

// ConnectedAppsDataSourceModel describes the data source data model.
type ConnectedAppsDataSourceModel struct {
	OrganizationID types.String             `tfsdk:"organization_id"`
	Apps           []ConnectedAppItemModel  `tfsdk:"apps"`
}

// ConnectedAppItemModel represents a single connected app in the list.
type ConnectedAppItemModel struct {
	ClientID    types.String `tfsdk:"client_id"`
	Name        types.String `tfsdk:"name"`
	GrantTypes  types.List   `tfsdk:"grant_types"`
	Audience    types.String `tfsdk:"audience"`
	ClientURI   types.String `tfsdk:"client_uri"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	OwnerUserID types.String `tfsdk:"owner_user_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewConnectedAppsDataSource() datasource.DataSource {
	return &ConnectedAppsDataSource{}
}

// Metadata returns the data source type name.
func (d *ConnectedAppsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_apps"
}

// Schema defines the schema for the data source.
func (d *ConnectedAppsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all connected apps in the organization.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. Defaults to the provider's org.",
				Optional:    true,
				Computed:    true,
			},
			"apps": schema.ListNestedAttribute{
				Description: "List of connected apps.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"client_id": schema.StringAttribute{
							Description: "The unique client ID of the app.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the app.",
							Computed:    true,
						},
						"grant_types": schema.ListAttribute{
							Description: "The OAuth grant types.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"audience": schema.StringAttribute{
							Description: "Who can use this application.",
							Computed:    true,
						},
						"client_uri": schema.StringAttribute{
							Description: "Website URL for the app.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the app is enabled.",
							Computed:    true,
						},
						"owner_user_id": schema.StringAttribute{
							Description: "The user who owns this app.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "When the app was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "When the app was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ConnectedAppsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
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
			"Unable to Create Connected Apps Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	d.client = appClient
}

// Read fetches the list of connected apps.
func (d *ConnectedAppsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ConnectedAppsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	apps, err := d.client.ListConnectedApps(ctx, orgID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing connected apps",
			"Could not list connected apps: "+err.Error(),
		)
		return
	}

	state.OrganizationID = types.StringValue(orgID)
	state.Apps = make([]ConnectedAppItemModel, len(apps))

	for i, app := range apps {
		grantTypesList, diags := types.ListValueFrom(ctx, types.StringType, app.GrantTypes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		state.Apps[i] = ConnectedAppItemModel{
			ClientID:    types.StringValue(app.ClientID),
			Name:        types.StringValue(app.ClientName),
			GrantTypes:  grantTypesList,
			Audience:    types.StringValue(app.Audience),
			ClientURI:   types.StringValue(app.ClientURI),
			Enabled:     types.BoolValue(app.Enabled),
			OwnerUserID: types.StringValue(app.OwnerUserID),
			CreatedAt:   types.StringValue(app.CreatedAt),
			UpdatedAt:   types.StringValue(app.UpdatedAt),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
