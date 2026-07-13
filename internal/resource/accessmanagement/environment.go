package accessmanagement

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &EnvironmentResource{}
	_ resource.ResourceWithConfigure   = &EnvironmentResource{}
	_ resource.ResourceWithImportState = &EnvironmentResource{}
	_ resource.ResourceWithModifyPlan  = &EnvironmentResource{}
)

// EnvironmentResource is the resource implementation.
type EnvironmentResource struct {
	client *accessmanagement.EnvironmentClient
}

// EnvironmentResourceModel describes the resource data model.
type EnvironmentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	IsProduction   types.Bool   `tfsdk:"is_production"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ClientID       types.String `tfsdk:"client_id"`
	ArcNamespace   types.String `tfsdk:"arc_namespace"`
}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

// Metadata returns the resource type name.
func (r *EnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

// Schema defines the schema for the resource.
func (r *EnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the environment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the environment (e.g., 'design', 'sandbox', 'production').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("sandbox"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("design", "sandbox", "production"),
				},
			},
			"is_production": schema.BoolAttribute{
				Description: "Whether this is a production environment. " +
					"Derived automatically from type: true when type='production', false otherwise. " +
					"Set explicitly only when type is not specified.",
				Optional: true,
				Computed: true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the environment will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID associated with the environment.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"arc_namespace": schema.StringAttribute{
				Description: "The ARC namespace for the environment.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *EnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	// Convert ClientConfig to UserClientConfig for environment operations
	userConfig := &client.UserClientConfig{
		BaseURL:      config.BaseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
		Timeout:      config.Timeout,
	}

	// Create the environment client
	environmentClient, err := accessmanagement.NewEnvironmentClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anypoint Environment API Client",
			"An unexpected error occurred when creating the Anypoint Environment API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = environmentClient
}

// ModifyPlan derives is_production from type so the planned value always matches
// what Platform will return — preventing plan/apply inconsistency and tainting.
// Platform sets isProduction=true for type="production" and false for all others,
// regardless of what the caller sends.
func (r *EnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to do on destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only derive when type is known. If type is still unknown (e.g. references
	// another resource not yet created) leave is_production unknown too.
	if plan.Type.IsUnknown() {
		return
	}

	// If the user set is_production explicitly and type is not set, honour their
	// value. When type IS known, Platform's rule always wins so we override.
	if !plan.Type.IsNull() {
		derived := plan.Type.ValueString() == "production"
		plan.IsProduction = types.BoolValue(derived)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Create the environment
	// Handle type field - use default if empty or null
	envType := data.Type.ValueString()
	if envType == "" || data.Type.IsNull() || data.Type.IsUnknown() {
		envType = "sandbox" // Use the default value
	}

	environmentRequest := &accessmanagement.CreateEnvironmentRequest{
		Name:         data.Name.ValueString(),
		Type:         envType,
		IsProduction: data.IsProduction.ValueBool(),
	}

	environment, err := r.client.CreateEnvironment(ctx, orgID, environmentRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating environment",
			"Could not create environment, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	data.ID = types.StringValue(environment.ID)
	// Use the name from our plan to ensure plan/apply consistency
	data.Name = types.StringValue(data.Name.ValueString())
	// Use the type we sent in the request to ensure plan/apply consistency
	data.Type = types.StringValue(envType)
	data.IsProduction = types.BoolValue(environment.IsProduction)
	// Only set organization_id if it was explicitly provided in the plan
	// If it was empty/null in plan, leave it empty to maintain plan/apply consistency
	if !data.OrganizationID.IsNull() && !data.OrganizationID.IsUnknown() && data.OrganizationID.ValueString() != "" {
		// Keep the originally provided organization_id value
		data.OrganizationID = types.StringValue(data.OrganizationID.ValueString())
	} else if data.OrganizationID.IsNull() || data.OrganizationID.IsUnknown() {
		// Only set if it was null/unknown in the plan (computed field scenario)
		data.OrganizationID = types.StringValue(orgID)
	}
	// If organization_id was empty string in plan, leave it as empty string
	data.ClientID = types.StringValue(environment.ClientID)

	// Handle nullable ArcNamespace field
	if environment.ArcNamespace != nil {
		data.ArcNamespace = types.StringValue(*environment.ArcNamespace)
	} else {
		data.ArcNamespace = types.StringNull()
	}

	// Write logs using the tflog package
	tflog.Trace(ctx, "created an environment")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	priorName := data.Name.ValueString()
	priorType := data.Type.ValueString()
	priorIsProduction := data.IsProduction.ValueBool()

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Get the environment from the API
	environment, err := r.client.GetEnvironment(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading environment",
			"Could not read environment ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "anypoint_environment refreshed from API", map[string]interface{}{
		"id":                  environment.ID,
		"prior_state_name":    priorName,
		"api_response_name":   environment.Name,
		"prior_state_type":    priorType,
		"api_response_type":   environment.Type,
		"prior_is_production": priorIsProduction,
		"api_is_production":   environment.IsProduction,
	})

	// Map response body to schema and populate Computed attribute values.
	// IMPORTANT: always overwrite Required/Optional attributes from the API response
	// so that backend (UI) changes surface as Terraform drift.
	data.ID = types.StringValue(environment.ID)
	data.Name = types.StringValue(environment.Name)
	data.Type = types.StringValue(environment.Type)
	data.IsProduction = types.BoolValue(environment.IsProduction)
	// Only update organization_id if it was explicitly set in state
	// If it was empty/null in state, keep it that way to maintain consistency
	if !data.OrganizationID.IsNull() && !data.OrganizationID.IsUnknown() && data.OrganizationID.ValueString() != "" {
		// Keep the originally configured organization_id value
		data.OrganizationID = types.StringValue(data.OrganizationID.ValueString())
	} else if data.OrganizationID.IsNull() || data.OrganizationID.IsUnknown() {
		// Only set if it was null/unknown in state (computed field scenario)
		data.OrganizationID = types.StringValue(environment.OrganizationID)
	}
	// If organization_id was empty string in state, leave it as empty string
	data.ClientID = types.StringValue(environment.ClientID)

	// Handle nullable ArcNamespace field
	if environment.ArcNamespace != nil {
		data.ArcNamespace = types.StringValue(*environment.ArcNamespace)
	} else {
		data.ArcNamespace = types.StringNull()
	}

	if priorName != environment.Name {
		tflog.Info(ctx, "anypoint_environment drift detected: name changed on backend", map[string]interface{}{
			"id":       environment.ID,
			"old_name": priorName,
			"new_name": environment.Name,
		})
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request with only changed fields
	environmentRequest := &accessmanagement.UpdateEnvironmentRequest{}
	hasChanges := false

	// Check if name has changed
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		environmentRequest.Name = &name
		hasChanges = true
	}

	// Check if type has changed
	if !plan.Type.Equal(state.Type) {
		envType := plan.Type.ValueString()
		environmentRequest.Type = &envType
		hasChanges = true
	}

	// Check if is_production has changed
	if !plan.IsProduction.Equal(state.IsProduction) {
		isProduction := plan.IsProduction.ValueBool()
		environmentRequest.IsProduction = &isProduction
		hasChanges = true
	}

	// If no changes, just update the computed fields from the current state and return
	if !hasChanges {
		// Just update the computed fields from the current state and return
		plan.ID = state.ID
		plan.OrganizationID = state.OrganizationID
		plan.ClientID = state.ClientID
		plan.ArcNamespace = state.ArcNamespace
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	environment, err := r.client.UpdateEnvironment(ctx, orgID, plan.ID.ValueString(), environmentRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating environment",
			"Could not update environment, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(environment.ID)
	plan.Name = types.StringValue(environment.Name)
	plan.Type = types.StringValue(environment.Type)
	plan.IsProduction = types.BoolValue(environment.IsProduction)
	// Preserve organization_id consistency - don't change it if it was empty in plan
	if !plan.OrganizationID.IsNull() && !plan.OrganizationID.IsUnknown() && plan.OrganizationID.ValueString() != "" {
		// Keep the originally planned organization_id value
		plan.OrganizationID = types.StringValue(plan.OrganizationID.ValueString())
	} else if plan.OrganizationID.IsNull() || plan.OrganizationID.IsUnknown() {
		// Only set if it was null/unknown in plan (computed field scenario)
		plan.OrganizationID = types.StringValue(environment.OrganizationID)
	}
	// If organization_id was empty string in plan, leave it as empty string

	// ClientID and ArcNamespace may not be returned in PUT response
	// Preserve existing values if not returned
	if environment.ClientID != "" {
		plan.ClientID = types.StringValue(environment.ClientID)
	} else {
		plan.ClientID = state.ClientID
	}

	// Handle nullable ArcNamespace field
	if environment.ArcNamespace != nil {
		plan.ArcNamespace = types.StringValue(*environment.ArcNamespace)
	} else {
		// Preserve existing ArcNamespace value if not returned in update response
		plan.ArcNamespace = state.ArcNamespace
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Delete the environment
	err := r.client.DeleteEnvironment(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting environment",
			"Could not delete environment, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
