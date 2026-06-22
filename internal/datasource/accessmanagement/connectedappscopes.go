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
	_ datasource.DataSource              = &ConnectedAppScopesDataSource{}
	_ datasource.DataSourceWithConfigure = &ConnectedAppScopesDataSource{}
)

// ConnectedAppScopesDataSource is the data source implementation.
type ConnectedAppScopesDataSource struct {
	client *accessmanagement.ConnectedAppScopesClient
}

// ConnectedAppScopesDataSourceModel describes the data source data model.
type ConnectedAppScopesDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ConnectedAppID types.String `tfsdk:"connected_app_id"`
	Scopes         types.Set    `tfsdk:"scopes"`
}

// ScopeDataSourceModel represents a single scope within the set
type ScopeDataSourceModel struct {
	Scope         types.String `tfsdk:"scope"`
	ContextParams types.Map    `tfsdk:"context_params"`
}

func NewConnectedAppScopesDataSource() datasource.DataSource {
	return &ConnectedAppScopesDataSource{}
}

// Metadata returns the data source type name.
func (d *ConnectedAppScopesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app_scopes"
}

// Schema defines the schema for the data source.
func (d *ConnectedAppScopesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches scopes information for an Anypoint Connected Application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the connected app scopes (same as connected_app_id).",
				Computed:    true,
			},
			"connected_app_id": schema.StringAttribute{
				Description: "The ID of the connected application to read scopes for.",
				Required:    true,
			},
			"scopes": schema.SetNestedAttribute{
				Description: "The set of scopes assigned to the connected application.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scope": schema.StringAttribute{
							Description: "The scope name (e.g., 'admin:cloudhub', 'read:applications').",
							Computed:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the scope (e.g., organization ID).",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *ConnectedAppScopesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Extract the client configuration
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Create user client config - ConnectedAppScopesClient requires UserClientConfig
	userConfig := &client.UserClientConfig{
		BaseURL:      config.BaseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
		Timeout:      config.Timeout,
	}

	// Create the connected app scopes client
	scopesClient, err := accessmanagement.NewConnectedAppScopesClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anypoint Connected App Scopes API Client",
			"An unexpected error occurred when creating the Anypoint Connected App Scopes API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	d.client = scopesClient
}

// Read refreshes the Terraform state with the latest data.
func (d *ConnectedAppScopesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConnectedAppScopesDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the connected app scopes from the API
	connectedAppID := data.ConnectedAppID.ValueString()
	scopes, err := d.client.GetConnectedAppScopes(ctx, connectedAppID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading connected app scopes",
			"Could not read connected app scopes for ID "+connectedAppID+": "+err.Error(),
		)
		return
	}

	// Set the ID to the connected app ID
	data.ID = types.StringValue(connectedAppID)

	// Convert API response to Terraform state
	if err := d.updateStateFromAPI(ctx, &data, scopes); err != nil {
		resp.Diagnostics.AddError(
			"Error updating state",
			"Could not update state from API response: "+err.Error(),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// updateStateFromAPI is a helper function to convert API response to Terraform state
func (d *ConnectedAppScopesDataSource) updateStateFromAPI(_ context.Context, data *ConnectedAppScopesDataSourceModel, apiScopes *accessmanagement.ConnectedAppScopes) error {
	// Convert API scopes to Terraform attribute values
	var scopeObjects []attr.Value

	for _, apiScope := range apiScopes.Scopes {
		// Handle context parameters - convert map[string]interface{} to map[string]string
		contextParams := types.MapNull(types.StringType)
		if len(apiScope.ContextParams) > 0 {
			contextParamsMap := make(map[string]attr.Value)
			for k, v := range apiScope.ContextParams {
				// Convert interface{} value to string
				var strVal string
				switch val := v.(type) {
				case string:
					strVal = val
				case fmt.Stringer:
					strVal = val.String()
				default:
					strVal = fmt.Sprintf("%v", val)
				}
				contextParamsMap[k] = types.StringValue(strVal)
			}
			var diags diag.Diagnostics
			contextParams, diags = types.MapValue(types.StringType, contextParamsMap)
			if diags.HasError() {
				return fmt.Errorf("failed to create context params map")
			}
		}

		// Create scope object
		scopeAttrs := map[string]attr.Value{
			"scope":          types.StringValue(apiScope.Scope),
			"context_params": contextParams,
		}

		scopeObject, diags := types.ObjectValue(map[string]attr.Type{
			"scope":          types.StringType,
			"context_params": types.MapType{ElemType: types.StringType},
		}, scopeAttrs)

		if diags.HasError() {
			return fmt.Errorf("failed to create scope object")
		}

		scopeObjects = append(scopeObjects, scopeObject)
	}

	// Convert to set
	scopesSet, diags := types.SetValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"scope":          types.StringType,
			"context_params": types.MapType{ElemType: types.StringType},
		},
	}, scopeObjects)

	if diags.HasError() {
		return fmt.Errorf("failed to create scopes set")
	}

	data.Scopes = scopesSet
	return nil
}
