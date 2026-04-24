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
	_ datasource.DataSource              = &UserDataSource{}
	_ datasource.DataSourceWithConfigure = &UserDataSource{}
)

// UserDataSource is the data source implementation.
type UserDataSource struct {
	client *accessmanagement.UserClient
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Username                types.String `tfsdk:"username"`
	FirstName               types.String `tfsdk:"first_name"`
	LastName                types.String `tfsdk:"last_name"`
	Email                   types.String `tfsdk:"email"`
	PhoneNumber             types.String `tfsdk:"phone_number"`
	IsActive                types.Bool   `tfsdk:"is_active"`
	OrganizationID          types.String `tfsdk:"organization_id"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
	IsFederated             types.Bool   `tfsdk:"is_federated"`
	MfaVerificationExcluded types.Bool   `tfsdk:"mfa_verification_excluded"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// Metadata returns the data source type name.
func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source.
func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an Anypoint Platform user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the user.",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username of the user.",
				Computed:    true,
			},
			"first_name": schema.StringAttribute{
				Description: "The first name of the user.",
				Computed:    true,
			},
			"last_name": schema.StringAttribute{
				Description: "The last name of the user.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email address of the user.",
				Computed:    true,
			},
			"phone_number": schema.StringAttribute{
				Description: "The phone number of the user.",
				Computed:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the user is active.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the user is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the user was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the user was last updated.",
				Computed:    true,
			},
			"is_federated": schema.BoolAttribute{
				Description: "Whether the user is federated.",
				Computed:    true,
			},
			"mfa_verification_excluded": schema.BoolAttribute{
				Description: "Whether the user is excluded from MFA verification.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	// Extract the client configuration
	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
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

	// Create the user client
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

	d.client = userClient
}

// Read refreshes the Terraform state with the latest data.
func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = d.client.OrgID
	}

	// Get the user from the API
	user, err := d.client.GetUser(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading user",
			"Could not read user ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(user.ID)
	data.Username = types.StringValue(user.Username)
	data.FirstName = types.StringValue(user.FirstName)
	data.LastName = types.StringValue(user.LastName)
	data.Email = types.StringValue(user.Email)
	data.PhoneNumber = types.StringValue(user.PhoneNumber)
	data.IsActive = types.BoolValue(user.Enabled)
	data.OrganizationID = types.StringValue(user.Organization.ID)
	data.CreatedAt = types.StringValue(user.CreatedAt)
	data.UpdatedAt = types.StringValue(user.UpdatedAt)
	data.IsFederated = types.BoolValue(user.Organization.IsFederated)
	data.MfaVerificationExcluded = types.BoolValue(user.MfaVerificationExcluded)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
