package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &PrivateSpaceAdvancedConfigResource{}
	_ resource.ResourceWithConfigure   = &PrivateSpaceAdvancedConfigResource{}
	_ resource.ResourceWithImportState = &PrivateSpaceAdvancedConfigResource{}
)

// PrivateSpaceAdvancedConfigResource is the resource implementation.
type PrivateSpaceAdvancedConfigResource struct {
	client *cloudhub2.PrivateSpaceAdvancedConfigClient
}

// PrivateSpaceAdvancedConfigResourceModel describes the resource data model.
type PrivateSpaceAdvancedConfigResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	PrivateSpaceID       types.String `tfsdk:"private_space_id"`
	OrganizationID       types.String `tfsdk:"organization_id"`
	IngressConfiguration types.Object `tfsdk:"ingress_configuration"`
	EnableIAMRole        types.Bool   `tfsdk:"enable_iam_role"`
}

// IngressConfigurationModel represents the ingress configuration model
type IngressConfigurationModel struct {
	ReadResponseTimeout types.String `tfsdk:"read_response_timeout"`
	Logs                types.Object `tfsdk:"logs"`
	Protocol            types.String `tfsdk:"protocol"`
	Deployment          types.Object `tfsdk:"deployment"`
}

// IngressLogsModel represents the ingress logs model
type IngressLogsModel struct {
	Filters      types.List   `tfsdk:"filters"`
	PortLogLevel types.String `tfsdk:"port_log_level"`
}

// IngressLogFilterModel represents the ingress log filter model
type IngressLogFilterModel struct {
	IP    types.String `tfsdk:"ip"`
	Level types.String `tfsdk:"level"`
}

// IngressDeploymentModel represents the ingress deployment model
type IngressDeploymentModel struct {
	Status            types.String `tfsdk:"status"`
	LastSeenTimestamp types.Int64  `tfsdk:"last_seen_timestamp"`
}

// NewPrivateSpaceAdvancedConfigResource is a helper function to simplify the provider implementation.
func NewPrivateSpaceAdvancedConfigResource() resource.Resource {
	return &PrivateSpaceAdvancedConfigResource{}
}

// Metadata returns the resource type name.
func (r *PrivateSpaceAdvancedConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privatespace_advanced_config"
}

// Schema defines the schema for the resource.
func (r *PrivateSpaceAdvancedConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages advanced configuration for an Anypoint Private Space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the advanced configuration.",
			},
			"private_space_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the private space to configure.",
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"ingress_configuration": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Ingress configuration for the private space.",
				Attributes: map[string]schema.Attribute{
					"read_response_timeout": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("300"),
						Description: "Read response timeout in seconds.",
					},
					"logs": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Logs configuration for ingress.",
						Attributes: map[string]schema.Attribute{
							"filters": schema.ListNestedAttribute{
								Optional:    true,
								Computed:    true,
								Default:     listdefault.StaticValue(types.ListValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{"ip": types.StringType, "level": types.StringType}}, []attr.Value{})),
								Description: "List of log filters.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ip": schema.StringAttribute{
											Required:    true,
											Description: "IP address for the filter.",
										},
										"level": schema.StringAttribute{
											Required:    true,
											Description: "Log level for the filter.",
										},
									},
								},
							},
							"port_log_level": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("ERROR"),
								Description: "Port log level.",
							},
						},
					},
					"protocol": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("https-redirect"),
						Description: "Protocol for ingress configuration.",
					},
					"deployment": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Deployment configuration for ingress.",
						Attributes: map[string]schema.Attribute{
							"status": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("APPLIED"),
								Description: "Deployment status.",
							},
							"last_seen_timestamp": schema.Int64Attribute{
								Optional:    true,
								Computed:    true,
								Default:     int64default.StaticInt64(1753719215000),
								Description: "Last seen timestamp.",
							},
						},
					},
				},
			},
			"enable_iam_role": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to enable IAM role for the private space.",
			},
		},
	}
}

// Configure configures the resource.
func (r *PrivateSpaceAdvancedConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clientConfig, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	client, err := cloudhub2.NewPrivateSpaceAdvancedConfigClient(clientConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Client",
			fmt.Sprintf("Unable to create private space advanced config client: %s", err.Error()),
		)
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *PrivateSpaceAdvancedConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateSpaceAdvancedConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	request := r.buildRequest(ctx, &data)

	privateSpace, err := r.client.UpdatePrivateSpaceAdvancedConfig(ctx, orgID, data.PrivateSpaceID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create private space advanced config", err.Error())
		return
	}

	// Map response to state
	r.mapPrivateSpaceToState(ctx, privateSpace, &data)

	data.ID = data.PrivateSpaceID                  // Use private space ID as the resource ID
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *PrivateSpaceAdvancedConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateSpaceAdvancedConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Get the private space to read advanced config
	privateSpace, err := r.client.GetPrivateSpace(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read private space advanced config", err.Error())
		return
	}

	// Map response to state
	r.mapPrivateSpaceToState(ctx, privateSpace, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PrivateSpaceAdvancedConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PrivateSpaceAdvancedConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	request := r.buildRequest(ctx, &data)

	privateSpace, err := r.client.UpdatePrivateSpaceAdvancedConfig(ctx, orgID, data.PrivateSpaceID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update private space advanced config", err.Error())
		return
	}

	// Map response to state
	r.mapPrivateSpaceToState(ctx, privateSpace, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *PrivateSpaceAdvancedConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateSpaceAdvancedConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For advanced config, delete means resetting to defaults
	defaultRequest := &cloudhub2.UpdatePrivateSpaceAdvancedConfigRequest{
		IngressConfiguration: cloudhub2.IngressConfiguration{
			ReadResponseTimeout: "300",
			Logs: cloudhub2.IngressLogs{
				Filters:      []cloudhub2.IngressLogFilter{},
				PortLogLevel: "ERROR",
			},
			Protocol: "https-redirect",
			Deployment: cloudhub2.IngressDeployment{
				Status:            "APPLIED",
				LastSeenTimestamp: 1753719215000,
			},
		},
		EnableIAMRole: false,
	}

	// Determine organization ID from state or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	_, err := r.client.UpdatePrivateSpaceAdvancedConfig(ctx, orgID, data.PrivateSpaceID.ValueString(), defaultRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset private space advanced config to defaults", err.Error())
		return
	}
}

// ImportState imports the resource.
func (r *PrivateSpaceAdvancedConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the private space ID as both the ID and private_space_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), req.ID)...)
}

// Helper functions

func (r *PrivateSpaceAdvancedConfigResource) buildRequest(ctx context.Context, data *PrivateSpaceAdvancedConfigResourceModel) *cloudhub2.UpdatePrivateSpaceAdvancedConfigRequest {
	request := &cloudhub2.UpdatePrivateSpaceAdvancedConfigRequest{
		EnableIAMRole: data.EnableIAMRole.ValueBool(),
	}

	// Handle ingress configuration
	if !data.IngressConfiguration.IsNull() {
		var ingressConfig IngressConfigurationModel
		data.IngressConfiguration.As(ctx, &ingressConfig, basetypes.ObjectAsOptions{})

		request.IngressConfiguration = cloudhub2.IngressConfiguration{
			ReadResponseTimeout: ingressConfig.ReadResponseTimeout.ValueString(),
			Protocol:            ingressConfig.Protocol.ValueString(),
		}

		// Handle logs
		if !ingressConfig.Logs.IsNull() {
			var logs IngressLogsModel
			ingressConfig.Logs.As(ctx, &logs, basetypes.ObjectAsOptions{})

			request.IngressConfiguration.Logs = cloudhub2.IngressLogs{
				PortLogLevel: logs.PortLogLevel.ValueString(),
				Filters:      []cloudhub2.IngressLogFilter{},
			}

			// Handle filters
			if !logs.Filters.IsNull() {
				var filters []IngressLogFilterModel
				logs.Filters.ElementsAs(ctx, &filters, false)

				for _, filter := range filters {
					request.IngressConfiguration.Logs.Filters = append(
						request.IngressConfiguration.Logs.Filters,
						cloudhub2.IngressLogFilter{
							IP:    filter.IP.ValueString(),
							Level: filter.Level.ValueString(),
						},
					)
				}
			}
		}

		// Handle deployment
		if !ingressConfig.Deployment.IsNull() {
			var deployment IngressDeploymentModel
			ingressConfig.Deployment.As(ctx, &deployment, basetypes.ObjectAsOptions{})

			request.IngressConfiguration.Deployment = cloudhub2.IngressDeployment{
				Status:            deployment.Status.ValueString(),
				LastSeenTimestamp: deployment.LastSeenTimestamp.ValueInt64(),
			}
		}
	}

	return request
}

func (r *PrivateSpaceAdvancedConfigResource) mapPrivateSpaceToState(_ context.Context, privateSpace *cloudhub2.PrivateSpace, data *PrivateSpaceAdvancedConfigResourceModel) {
	// Set enableIAMRole
	data.EnableIAMRole = types.BoolValue(privateSpace.EnableIAMRole)

	// Map ingress configuration from the private space response
	// Note: We extract only the fields we manage in this resource
	ingressConfigAttrs := map[string]attr.Value{
		"read_response_timeout": types.StringValue(fmt.Sprintf("%d", privateSpace.IngressConfiguration.ReadResponseTimeout)),
		"protocol":              types.StringValue(privateSpace.IngressConfiguration.Protocol),
	}

	// Map logs
	logFilters := make([]attr.Value, 0, len(privateSpace.IngressConfiguration.Logs.Filters))
	for _, filter := range privateSpace.IngressConfiguration.Logs.Filters {
		filterAttrs := map[string]attr.Value{
			"ip":    types.StringValue(filter.IP),
			"level": types.StringValue(filter.Level),
		}
		filterObj, _ := types.ObjectValue(map[string]attr.Type{
			"ip":    types.StringType,
			"level": types.StringType,
		}, filterAttrs)
		logFilters = append(logFilters, filterObj)
	}

	filtersList, _ := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip":    types.StringType,
			"level": types.StringType,
		},
	}, logFilters)

	logsAttrs := map[string]attr.Value{
		"filters":        filtersList,
		"port_log_level": types.StringValue(privateSpace.IngressConfiguration.Logs.PortLogLevel),
	}

	logsObj, _ := types.ObjectValue(map[string]attr.Type{
		"filters":        types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"ip": types.StringType, "level": types.StringType}}},
		"port_log_level": types.StringType,
	}, logsAttrs)

	// Map deployment - using computed defaults since this may not be in response
	deploymentAttrs := map[string]attr.Value{
		"status":              types.StringValue("APPLIED"),
		"last_seen_timestamp": types.Int64Value(1753719215000),
	}

	deploymentObj, _ := types.ObjectValue(map[string]attr.Type{
		"status":              types.StringType,
		"last_seen_timestamp": types.Int64Type,
	}, deploymentAttrs)

	ingressConfigAttrs["logs"] = logsObj
	ingressConfigAttrs["deployment"] = deploymentObj

	ingressConfigObj, _ := types.ObjectValue(map[string]attr.Type{
		"read_response_timeout": types.StringType,
		"protocol":              types.StringType,
		"logs": types.ObjectType{AttrTypes: map[string]attr.Type{
			"filters":        types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"ip": types.StringType, "level": types.StringType}}},
			"port_log_level": types.StringType,
		}},
		"deployment": types.ObjectType{AttrTypes: map[string]attr.Type{
			"status":              types.StringType,
			"last_seen_timestamp": types.Int64Type,
		}},
	}, ingressConfigAttrs)

	data.IngressConfiguration = ingressConfigObj
}
