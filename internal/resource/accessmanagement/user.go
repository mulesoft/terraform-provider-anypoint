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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

// UserResource is the resource implementation.
type UserResource struct {
	client *accessmanagement.UserClient
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Username                types.String `tfsdk:"username"`
	FirstName               types.String `tfsdk:"first_name"`
	LastName                types.String `tfsdk:"last_name"`
	Email                   types.String `tfsdk:"email"`
	PhoneNumber             types.String `tfsdk:"phone_number"`
	Password                types.String `tfsdk:"password"`
	OrganizationID          types.String `tfsdk:"organization_id"`
	MfaVerificationExcluded types.Bool   `tfsdk:"mfa_verification_excluded"`
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

// Metadata returns the resource type name.
func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the resource.
func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Platform user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The username of the user.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"first_name": schema.StringAttribute{
				Description: "The first name of the user.",
				Required:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "The last name of the user.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email address of the user.",
				Required:    true,
			},
			"phone_number": schema.StringAttribute{
				Description: "The phone number of the user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Description: "The password for the user. This is only used during creation and updates.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the user will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mfa_verification_excluded": schema.BoolAttribute{
				Description: "Indicates whether the user is excluded from MFA verification.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Extract the client configuration
	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	userConfig := &client.UserClientConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		BaseURL:      config.BaseURL,
		Timeout:      config.Timeout,
		Username:     config.Username,
		Password:     config.Password,
	}

	userClient, err := accessmanagement.NewUserClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anypoint User API Client",
			"An unexpected error occurred when creating the Anypoint User API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = userClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserResourceModel

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

	// Create the user request
	userRequest := &accessmanagement.CreateUserRequest{
		Username:                data.Username.ValueString(),
		FirstName:               data.FirstName.ValueString(),
		LastName:                data.LastName.ValueString(),
		Email:                   data.Email.ValueString(),
		Password:                data.Password.ValueString(),
		MfaVerificationExcluded: data.MfaVerificationExcluded.ValueBool(),
	}

	// Add phone number if provided
	if !data.PhoneNumber.IsNull() && !data.PhoneNumber.IsUnknown() {
		userRequest.PhoneNumber = data.PhoneNumber.ValueString()
	}

	user, err := r.client.CreateUser(ctx, orgID, userRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			"Could not create user, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	data.ID = types.StringValue(user.ID)
	data.Username = types.StringValue(user.Username)
	data.FirstName = types.StringValue(user.FirstName)
	data.LastName = types.StringValue(user.LastName)
	data.Email = types.StringValue(user.Email)
	// Set the actual organization ID used
	data.OrganizationID = types.StringValue(orgID)
	data.MfaVerificationExcluded = types.BoolValue(user.MfaVerificationExcluded)

	// Handle optional phone number
	if user.PhoneNumber != "" {
		data.PhoneNumber = types.StringValue(user.PhoneNumber)
	} else if data.PhoneNumber.IsNull() {
		data.PhoneNumber = types.StringNull()
	}

	// Note: Password is not returned from API for security reasons,
	// so we keep the value from the request

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a user")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserResourceModel

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

	// Get the user from the API
	user, err := r.client.GetUser(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading user",
			"Could not read user ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate values
	data.ID = types.StringValue(user.ID)
	data.Username = types.StringValue(user.Username)
	data.FirstName = types.StringValue(user.FirstName)
	data.LastName = types.StringValue(user.LastName)
	data.Email = types.StringValue(user.Email)
	data.MfaVerificationExcluded = types.BoolValue(user.MfaVerificationExcluded)

	// Handle optional phone number
	if user.PhoneNumber != "" {
		data.PhoneNumber = types.StringValue(user.PhoneNumber)
	} else {
		data.PhoneNumber = types.StringNull()
	}

	// Note: Password is not returned from API for security reasons,
	// so we keep the existing value from state

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state UserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request with only changed fields
	userRequest := &accessmanagement.UpdateUserRequest{}
	hasChanges := false

	// Check if username has changed
	if !plan.Username.Equal(state.Username) {
		username := plan.Username.ValueString()
		userRequest.Username = &username
		hasChanges = true
	}

	// Check if first_name has changed
	if !plan.FirstName.Equal(state.FirstName) {
		firstName := plan.FirstName.ValueString()
		userRequest.FirstName = &firstName
		hasChanges = true
	}

	// Check if last_name has changed
	if !plan.LastName.Equal(state.LastName) {
		lastName := plan.LastName.ValueString()
		userRequest.LastName = &lastName
		hasChanges = true
	}

	// Check if email has changed
	if !plan.Email.Equal(state.Email) {
		email := plan.Email.ValueString()
		userRequest.Email = &email
		hasChanges = true
	}

	// Check if phone_number has changed
	if !plan.PhoneNumber.Equal(state.PhoneNumber) {
		if !plan.PhoneNumber.IsNull() && !plan.PhoneNumber.IsUnknown() {
			phoneNumber := plan.PhoneNumber.ValueString()
			userRequest.PhoneNumber = &phoneNumber
		} else {
			// Set to empty string to clear the phone number
			emptyPhone := ""
			userRequest.PhoneNumber = &emptyPhone
		}
		hasChanges = true
	}

	// Check if mfa_verification_excluded has changed
	if !plan.MfaVerificationExcluded.Equal(state.MfaVerificationExcluded) {
		mfaVerificationExcluded := plan.MfaVerificationExcluded.ValueBool()
		userRequest.MfaVerificationExcluded = &mfaVerificationExcluded
		hasChanges = true
	}

	// If no changes, just return
	if !hasChanges {
		// Copy computed fields from state
		plan.ID = state.ID
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Note: Password changes would typically require recreation or a separate API call
	// For now, we handle password changes by requiring replacement (RequiresReplace modifier)

	// Determine organization ID from state or default to client's org
	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	user, err := r.client.UpdateUser(ctx, orgID, plan.ID.ValueString(), userRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating user",
			"Could not update user, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate values
	plan.ID = types.StringValue(user.ID)
	plan.Username = types.StringValue(user.Username)
	plan.FirstName = types.StringValue(user.FirstName)
	plan.LastName = types.StringValue(user.LastName)
	plan.Email = types.StringValue(user.Email)
	plan.MfaVerificationExcluded = types.BoolValue(user.MfaVerificationExcluded)

	// Handle optional phone number
	if user.PhoneNumber != "" {
		plan.PhoneNumber = types.StringValue(user.PhoneNumber)
	} else {
		plan.PhoneNumber = types.StringNull()
	}

	// Note: Password is preserved from the plan since it's not returned by API

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserResourceModel

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

	// Delete the user
	err := r.client.DeleteUser(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user",
			"Could not delete user, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
