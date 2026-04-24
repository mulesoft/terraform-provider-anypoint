package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/constants"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &ConnectedAppScopesResource{}
	_ resource.ResourceWithConfigure   = &ConnectedAppScopesResource{}
	_ resource.ResourceWithImportState = &ConnectedAppScopesResource{}
)

// ConnectedAppScopesResource is the resource implementation.
type ConnectedAppScopesResource struct {
	client *accessmanagement.ConnectedAppScopesClient
}

// ConnectedAppScopesResourceModel describes the resource data model.
type ConnectedAppScopesResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ConnectedAppID types.String `tfsdk:"connected_app_id"`
	Scopes         types.Set    `tfsdk:"scopes"`
}

// ScopeModel represents a single scope within the set
type ScopeModel struct {
	Scope         types.String `tfsdk:"scope"`
	ContextParams types.Map    `tfsdk:"context_params"`
}

func NewConnectedAppScopesResource() resource.Resource {
	return &ConnectedAppScopesResource{}
}

// Metadata returns the resource type name.
func (r *ConnectedAppScopesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app_scopes"
}

// Schema defines the schema for the resource.
func (r *ConnectedAppScopesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages scopes for an Anypoint Connected Application using user authentication.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the connected app scopes (same as connected_app_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connected_app_id": schema.StringAttribute{
				Description: "The ID of the connected application to manage scopes for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scopes": schema.SetNestedAttribute{
				Description: "The set of scopes to assign to the connected application.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scope": schema.StringAttribute{
							Description: "The scope name (e.g., 'admin:cloudhub', 'read:applications').",
							Required:    true,
						},
						"context_params": schema.MapAttribute{
							Description: "Context parameters for the scope (e.g., organization ID).",
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
func (r *ConnectedAppScopesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// This resource requires the provider to be configured with user authentication
	// For now, we'll create a simple UserClientConfig with placeholder values
	// In a real implementation, you would get these from environment variables or provider config
	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Create user client config - this is a simplified implementation
	// In practice, you'd need to get username/password from the provider configuration
	userConfig := &client.UserClientConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		BaseURL:      config.BaseURL,
		Timeout:      config.Timeout,
		Username:     config.Username,
		Password:     config.Password,
	}

	scopesClient, err := accessmanagement.NewConnectedAppScopesClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Connected App Scopes Client",
			"An unexpected error occurred when creating the Connected App Scopes client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = scopesClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *ConnectedAppScopesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConnectedAppScopesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate scopes before converting
	if diags := r.validateScopes(ctx, data.Scopes); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Convert Terraform scopes to API format
	apiScopes, err := r.convertScopesToAPI(ctx, data.Scopes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting scopes",
			"Could not convert scopes to API format: "+err.Error(),
		)
		return
	}

	// Create the update request
	updateRequest := &accessmanagement.UpdateConnectedAppScopesRequest{
		Scopes: apiScopes,
	}

	// Update connected app scopes
	connectedAppID := data.ConnectedAppID.ValueString()
	_, err = r.client.UpdateConnectedAppScopes(ctx, connectedAppID, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating connected app scopes",
			"Could not create connected app scopes: "+err.Error(),
		)
		return
	}

	// Set the ID
	data.ID = data.ConnectedAppID

	tflog.Trace(ctx, "created connected app scopes")

	// Save data into Terraform state - use planned values to ensure consistency
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ConnectedAppScopesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConnectedAppScopesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get connected app scopes from the API
	connectedAppID := data.ConnectedAppID.ValueString()
	scopes, err := r.client.GetConnectedAppScopes(ctx, connectedAppID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading connected app scopes",
			"Could not read connected app scopes for ID "+connectedAppID+": "+err.Error(),
		)
		return
	}

	// Only update scopes from API if the response actually contains scopes.
	// The GET endpoint may return an empty list even when scopes exist
	// (e.g., different response envelope). In that case, preserve the
	// existing state to avoid spurious plan diffs.
	if len(scopes.Scopes) > 0 {
		if err := r.updateStateFromAPI(ctx, &data, scopes); err != nil {
			resp.Diagnostics.AddError(
				"Error updating state",
				"Could not update state from API response: "+err.Error(),
			)
			return
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ConnectedAppScopesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConnectedAppScopesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate scopes before converting
	if diags := r.validateScopes(ctx, data.Scopes); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Convert Terraform scopes to API format
	apiScopes, err := r.convertScopesToAPI(ctx, data.Scopes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting scopes",
			"Could not convert scopes to API format: "+err.Error(),
		)
		return
	}

	// Create the update request
	updateRequest := &accessmanagement.UpdateConnectedAppScopesRequest{
		Scopes: apiScopes,
	}

	// Update connected app scopes
	connectedAppID := data.ConnectedAppID.ValueString()
	_, err = r.client.UpdateConnectedAppScopes(ctx, connectedAppID, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating connected app scopes",
			"Could not update connected app scopes: "+err.Error(),
		)
		return
	}

	// Save updated data into Terraform state - use planned values to ensure consistency
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ConnectedAppScopesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConnectedAppScopesResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete connected app scopes (set to empty)
	connectedAppID := data.ConnectedAppID.ValueString()
	err := r.client.DeleteConnectedAppScopes(ctx, connectedAppID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting connected app scopes",
			"Could not delete connected app scopes: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *ConnectedAppScopesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using the connected app ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connected_app_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// updateStateFromAPI is a helper function to convert API response to Terraform state
func (r *ConnectedAppScopesResource) updateStateFromAPI(ctx context.Context, data *ConnectedAppScopesResourceModel, apiScopes *accessmanagement.ConnectedAppScopes) error {
	// Convert API scopes to Terraform attribute values
	var scopeObjects []attr.Value

	for _, apiScope := range apiScopes.Scopes {
		// Handle context parameters
		contextParams := types.MapNull(types.StringType)
		if len(apiScope.ContextParams) > 0 {
			contextParamsMap := make(map[string]attr.Value)
			for k, v := range apiScope.ContextParams {
				if str, ok := v.(string); ok {
					contextParamsMap[k] = types.StringValue(str)
				}
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

// convertScopesToAPI is a helper function to convert Terraform scopes to API format
func (r *ConnectedAppScopesResource) convertScopesToAPI(ctx context.Context, scopesSet types.Set) ([]accessmanagement.Scope, error) {
	var apiScopes []accessmanagement.Scope

	// Get scope elements as objects
	scopeElements := scopesSet.Elements()
	for _, scopeElement := range scopeElements {
		scopeObj := scopeElement.(types.Object)
		scopeAttrs := scopeObj.Attributes()

		// Get scope name
		scopeName := scopeAttrs["scope"].(types.String).ValueString()

		// Get context params
		contextParams := make(map[string]interface{})
		contextParamsAttr := scopeAttrs["context_params"]
		if !contextParamsAttr.IsNull() && !contextParamsAttr.IsUnknown() {
			contextParamsMap := contextParamsAttr.(types.Map)
			for k, v := range contextParamsMap.Elements() {
				contextParams[k] = v.(types.String).ValueString()
			}
		}

		apiScopes = append(apiScopes, accessmanagement.Scope{
			Scope:         scopeName,
			ContextParams: contextParams,
		})
	}

	return apiScopes, nil
}

// validateScopes validates that all scope names are valid Anypoint Platform scopes
func (r *ConnectedAppScopesResource) validateScopes(ctx context.Context, scopesSet types.Set) diag.Diagnostics {
	var diags diag.Diagnostics

	// Get scope elements as objects
	scopeElements := scopesSet.Elements()
	for i, scopeElement := range scopeElements {
		scopeObj := scopeElement.(types.Object)
		scopeAttrs := scopeObj.Attributes()

		// Get scope name
		scopeName := scopeAttrs["scope"].(types.String).ValueString()

		// Validate scope name
		if !constants.IsValidScope(scopeName) {
			diags.AddError(
				"Invalid Scope Name",
				fmt.Sprintf("The scope '%s' at index %d is not a valid Anypoint Platform scope. "+
					"Please check the scope name for typos. Valid scopes include: "+
					"admin:cloudhub, manage:runtime_fabrics, create:environment, manage:private_spaces, "+
					"admin:api_manager, read:api_query, edit:api_query, manage:api_query, etc. "+
					"For a complete list of valid scopes, see the provider documentation.",
					scopeName, i),
			)
		}
	}

	return diags
}
