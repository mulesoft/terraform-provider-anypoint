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
	_ datasource.DataSource              = &EnvironmentDataSource{}
	_ datasource.DataSourceWithConfigure = &EnvironmentDataSource{}
)

// EnvironmentDataSource is the data source implementation.
type EnvironmentDataSource struct {
	client *accessmanagement.EnvironmentClient
}

// EnvironmentDataSourceModel describes the data source data model.
type EnvironmentDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	IsProduction   types.Bool   `tfsdk:"is_production"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ClientID       types.String `tfsdk:"client_id"`
	ArcNamespace   types.String `tfsdk:"arc_namespace"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

// Metadata returns the data source type name.
func (d *EnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

// Schema defines the schema for the data source.
func (d *EnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an Anypoint Platform environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the environment.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the environment (e.g., 'design', 'sandbox', 'production').",
				Computed:    true,
			},
			"is_production": schema.BoolAttribute{
				Description: "Whether this is a production environment.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the environment is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID associated with the environment.",
				Computed:    true,
			},
			"arc_namespace": schema.StringAttribute{
				Description: "The ARC namespace for the environment.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the environment was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the environment was last updated.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *EnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = environmentClient
}

// Read refreshes the Terraform state with the latest data.
func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

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

	// Get the environment from the API
	environment, err := d.client.GetEnvironment(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading environment",
			"Could not read environment ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(environment.ID)
	data.Name = types.StringValue(environment.Name)
	data.Type = types.StringValue(environment.Type)
	data.IsProduction = types.BoolValue(environment.IsProduction)
	data.OrganizationID = types.StringValue(environment.OrganizationID)
	data.ClientID = types.StringValue(environment.ClientID)

	// Handle nullable ArcNamespace field
	if environment.ArcNamespace != nil {
		data.ArcNamespace = types.StringValue(*environment.ArcNamespace)
	} else {
		data.ArcNamespace = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
