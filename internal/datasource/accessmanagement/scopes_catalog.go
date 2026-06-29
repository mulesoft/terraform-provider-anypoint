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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &ScopesCatalogDataSource{}
	_ datasource.DataSourceWithConfigure = &ScopesCatalogDataSource{}
)

// ScopesCatalogDataSource is the data source implementation.
type ScopesCatalogDataSource struct {
	client *accessmanagement.ScopesCatalogClient
}

// ScopesCatalogDataSourceModel describes the data source data model.
type ScopesCatalogDataSourceModel struct {
	Scopes         []ScopeCatalogModel `tfsdk:"scopes"`
	IncludeInternal types.Bool          `tfsdk:"include_internal"`
}

// ScopeCatalogModel represents a single scope entry in the catalog.
type ScopeCatalogModel struct {
	Scope        types.String `tfsdk:"scope"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	ProductLabel types.String `tfsdk:"product_label"`
	Internal     types.Bool   `tfsdk:"internal"`
}

func NewScopesCatalogDataSource() datasource.DataSource {
	return &ScopesCatalogDataSource{}
}

// Metadata returns the data source type name.
func (d *ScopesCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scopes_catalog"
}

// Schema defines the schema for the data source.
func (d *ScopesCatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the catalog of all available scopes in the Anypoint Platform. " +
			"This helps users discover valid scope names, their display names, and which product they belong to " +
			"when assigning scopes to connected apps.",
		Attributes: map[string]schema.Attribute{
			"include_internal": schema.BoolAttribute{
				Description: "Whether to include internal/system scopes in the results. Defaults to false.",
				Optional:    true,
			},
			"scopes": schema.ListNestedAttribute{
				Description: "The list of available scopes.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scope": schema.StringAttribute{
							Description: "The scope identifier used in API calls (e.g., 'create:generations', 'read:exchange').",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "The human-readable display name shown in the UI (e.g., 'Mule Developer Generative AI User').",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of what the scope grants access to.",
							Computed:    true,
						},
						"product_label": schema.StringAttribute{
							Description: "The product or feature area this scope belongs to (e.g., 'Runtime Manager', 'Exchange').",
							Computed:    true,
						},
						"internal": schema.BoolAttribute{
							Description: "Whether this is an internal/system scope.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ScopesCatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	userConfig := &client.UserClientConfig{
		BaseURL:      config.BaseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
		Timeout:      config.Timeout,
	}

	catalogClient, err := accessmanagement.NewScopesCatalogClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Scopes Catalog Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	d.client = catalogClient
}

// Read refreshes the data source data.
func (d *ScopesCatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ScopesCatalogDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, err := d.client.ListScopesCatalog(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading scopes catalog",
			"Could not read scopes catalog: "+err.Error(),
		)
		return
	}

	// Filter internal scopes unless include_internal is true
	includeInternal := false
	if !state.IncludeInternal.IsNull() && !state.IncludeInternal.IsUnknown() {
		includeInternal = state.IncludeInternal.ValueBool()
	}

	state.Scopes = []ScopeCatalogModel{}
	for _, entry := range entries {
		if !includeInternal && entry.Internal {
			continue
		}
		state.Scopes = append(state.Scopes, ScopeCatalogModel{
			Scope:        types.StringValue(entry.Scope),
			DisplayName:  types.StringValue(entry.DisplayName),
			Description:  types.StringValue(entry.Description),
			ProductLabel: types.StringValue(entry.ProductLabel),
			Internal:     types.BoolValue(entry.Internal),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
