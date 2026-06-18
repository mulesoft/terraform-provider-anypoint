package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &PrivateSpaceAdvancedConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpaceAdvancedConfigDataSource{}
)

// PrivateSpaceAdvancedConfigDataSource is the data source implementation.
type PrivateSpaceAdvancedConfigDataSource struct {
	client *cloudhub2.PrivateSpaceAdvancedConfigClient
}

// PrivateSpaceAdvancedConfigDataSourceModel describes the data source data model.
type PrivateSpaceAdvancedConfigDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	PrivateSpaceID       types.String `tfsdk:"private_space_id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	EnableIAMRole        types.Bool   `tfsdk:"enable_iam_role"`
	IngressConfiguration types.Object `tfsdk:"ingress_configuration"`
}

func NewPrivateSpaceAdvancedConfigDataSource() datasource.DataSource {
	return &PrivateSpaceAdvancedConfigDataSource{}
}

// Metadata returns the data source type name.
func (d *PrivateSpaceAdvancedConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatespace_advanced_config"
}

// Schema defines the schema for the data source.
func (d *PrivateSpaceAdvancedConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the advanced configuration for a CloudHub 2.0 private space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for the data source.",
				Computed:    true,
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space to fetch advanced configuration for.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not provided, the provider's default organization will be used.",
				Optional:    true,
				Computed:    true,
			},
			"enable_iam_role": schema.BoolAttribute{
				Description: "Whether IAM role is enabled for the private space.",
				Computed:    true,
			},
			"ingress_configuration": schema.SingleNestedAttribute{
				Description: "Ingress configuration for the private space.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"read_response_timeout": schema.StringAttribute{
						Description: "Read response timeout in milliseconds.",
						Computed:    true,
					},
					"protocol": schema.StringAttribute{
						Description: "Protocol used for ingress.",
						Computed:    true,
					},
					"logs": schema.SingleNestedAttribute{
						Description: "Logs configuration for ingress.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"filters": schema.ListNestedAttribute{
								Description: "List of log filters.",
								Computed:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip": schema.StringAttribute{
											Description: "IP address for the filter.",
											Computed:    true,
										},
										"level": schema.StringAttribute{
											Description: "Log level for the filter.",
											Computed:    true,
										},
									},
								},
							},
							"port_log_level": schema.StringAttribute{
								Description: "Port log level.",
								Computed:    true,
							},
						},
					},
					"deployment": schema.SingleNestedAttribute{
						Description: "Deployment status information.",
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"status": schema.StringAttribute{
								Description: "Deployment status.",
								Computed:    true,
							},
							"last_seen_timestamp": schema.Int64Attribute{
								Description: "Last seen timestamp for the deployment.",
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateSpaceAdvancedConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	privateSpaceAdvancedConfigClient, err := cloudhub2.NewPrivateSpaceAdvancedConfigClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Advanced Config Client",
			"An unexpected error occurred when creating the Private Space Advanced Config client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	d.client = privateSpaceAdvancedConfigClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateSpaceAdvancedConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateSpaceAdvancedConfigDataSourceModel

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

	// Get private space from API
	privateSpace, err := d.client.GetPrivateSpace(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Private Space Advanced Configuration",
			"Could not read Private Space Advanced Configuration for private space ID "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Set the data source ID
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString())
	data.OrganizationID = types.StringValue(orgID)
	data.EnableIAMRole = types.BoolValue(privateSpace.EnableIAMRole)

	// Map ingress configuration filters
	var filterElements []attr.Value
	for _, filter := range privateSpace.IngressConfiguration.Logs.Filters {
		filterObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"ip":    types.StringType,
				"level": types.StringType,
			},
			map[string]attr.Value{
				"ip":    types.StringValue(filter.IP),
				"level": types.StringValue(filter.Level),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		filterElements = append(filterElements, filterObj)
	}

	filtersList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"ip":    types.StringType,
				"level": types.StringType,
			},
		},
		filterElements,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map ingress logs
	logsObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"filters": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"ip":    types.StringType,
						"level": types.StringType,
					},
				},
			},
			"port_log_level": types.StringType,
		},
		map[string]attr.Value{
			"filters":        filtersList,
			"port_log_level": types.StringValue(privateSpace.IngressConfiguration.Logs.PortLogLevel),
		},
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map ingress deployment
	deploymentObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"status":              types.StringType,
			"last_seen_timestamp": types.Int64Type,
		},
		map[string]attr.Value{
			"status":              types.StringValue(privateSpace.IngressConfiguration.Deployment.Status),
			"last_seen_timestamp": types.Int64Value(privateSpace.IngressConfiguration.Deployment.LastSeenTimestamp),
		},
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert ReadResponseTimeout to string
	readResponseTimeoutStr := fmt.Sprintf("%d", privateSpace.IngressConfiguration.ReadResponseTimeout)

	// Map ingress configuration
	ingressConfigObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"read_response_timeout": types.StringType,
			"protocol":              types.StringType,
			"logs": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"filters": types.ListType{
						ElemType: types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"ip":    types.StringType,
								"level": types.StringType,
							},
						},
					},
					"port_log_level": types.StringType,
				},
			},
			"deployment": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"status":              types.StringType,
					"last_seen_timestamp": types.Int64Type,
				},
			},
		},
		map[string]attr.Value{
			"read_response_timeout": types.StringValue(readResponseTimeoutStr),
			"protocol":              types.StringValue(privateSpace.IngressConfiguration.Protocol),
			"logs":                  logsObj,
			"deployment":            deploymentObj,
		},
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.IngressConfiguration = ingressConfigObj

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
