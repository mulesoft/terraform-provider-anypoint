package apimanagement

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &AlertResource{}
	_ resource.ResourceWithConfigure   = &AlertResource{}
	_ resource.ResourceWithImportState = &AlertResource{}
)

type AlertResource struct {
	client *apimanagement.AlertClient
}

type AlertResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	APIInstanceID  types.String `tfsdk:"api_instance_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Severity       types.String `tfsdk:"severity"`
	ResourceType   types.String `tfsdk:"resource_type"`
	DeploymentType types.String `tfsdk:"deployment_type"`
	MetricType     types.String `tfsdk:"metric_type"`
	WildcardAlert  types.Bool   `tfsdk:"wildcard_alert"`
	Condition      types.Object `tfsdk:"condition"`
	Notifications  types.List   `tfsdk:"notifications"`
}

type AlertConditionModel struct {
	Operator  types.String `tfsdk:"operator"`
	Threshold types.Int64  `tfsdk:"threshold"`
	Interval  types.Int64  `tfsdk:"interval"`
}

type AlertNotificationModel struct {
	Type       types.String `tfsdk:"type"`
	Recipients types.List   `tfsdk:"recipients"`
	Subject    types.String `tfsdk:"subject"`
	Message    types.String `tfsdk:"message"`
}

var alertConditionAttrTypes = map[string]attr.Type{
	"operator":  types.StringType,
	"threshold": types.Int64Type,
	"interval":  types.Int64Type,
}

var alertNotificationAttrTypes = map[string]attr.Type{
	"type":       types.StringType,
	"recipients": types.ListType{ElemType: types.StringType},
	"subject":    types.StringType,
	"message":    types.StringType,
}

func NewAlertResource() resource.Resource {
	return &AlertResource{}
}

func (r *AlertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_instance_alert"
}

func (r *AlertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an alert for an API instance in Anypoint Monitoring.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the alert.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_id": schema.StringAttribute{
				Description: "Environment ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_instance_id": schema.StringAttribute{
				Description: "Numeric ID of the API instance this alert is for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the alert.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Alert type (e.g. 'basic').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("basic"),
			},
			"severity": schema.StringAttribute{
				Description: "Alert severity: info, warning, or critical.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("info", "warning", "critical"),
				},
			},
			"resource_type": schema.StringAttribute{
				Description: "Resource type (e.g. 'api').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("api"),
			},
			"deployment_type": schema.StringAttribute{
				Description: "Deployment type. Accepts shortcodes (CH, CH2, HY, RF, SM) " +
					"which are mapped to the API enum values (cloudHub, cloudHub2, hybrid, runtimeFabric, serviceMesh).",
				Required: true,
			},
			"metric_type": schema.StringAttribute{
				Description: "Metric to alert on (e.g. 'api_request_count', 'api_response_time').",
				Required:    true,
			},
			"wildcard_alert": schema.BoolAttribute{
				Description: "Whether this is a wildcard alert that applies to all APIs.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"condition": schema.SingleNestedAttribute{
				Description: "Alert trigger condition.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"operator": schema.StringAttribute{
						Description: "Comparison operator (e.g. 'above', 'below').",
						Required:    true,
					},
					"threshold": schema.Int64Attribute{
						Description: "Threshold value that triggers the alert.",
						Required:    true,
					},
					"interval": schema.Int64Attribute{
						Description: "Time interval in minutes over which the metric is evaluated.",
						Required:    true,
					},
				},
			},
			"notifications": schema.ListNestedAttribute{
				Description: "Notification channels for the alert.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Notification type (e.g. 'email').",
							Required:    true,
						},
						"recipients": schema.ListAttribute{
							Description: "List of recipient email addresses.",
							Required:    true,
							ElementType: types.StringType,
						},
						"subject": schema.StringAttribute{
							Description: "Email subject template. Supports variables like ${severity}, ${api}, ${condition}.",
							Optional:    true,
						},
						"message": schema.StringAttribute{
							Description: "Email body template. Supports variables like ${api}, ${condition}, ${value}, ${timestamp}.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *AlertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T.", req.ProviderData),
		)
		return
	}

	alertClient, err := apimanagement.NewAlertClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Alert Client",
			"An unexpected error occurred when creating the Alert client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = alertClient
}

func (r *AlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	createReq, diags := r.expandAlert(&data, orgID, envID, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert, err := r.client.CreateAlert(ctx, orgID, envID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating alert", "Could not create alert: "+err.Error())
		return
	}

	flatDiags := r.flattenAlert(ctx, alert, &data, orgID, envID)
	resp.Diagnostics.Append(flatDiags...)
	tflog.Trace(ctx, "created API alert", map[string]interface{}{"id": alert.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	alert, err := r.client.GetAlert(ctx, orgID, envID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading alert", "Could not read alert: "+err.Error())
		return
	}

	flatDiags := r.flattenAlert(ctx, alert, &data, orgID, envID)
	resp.Diagnostics.Append(flatDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AlertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := state.EnvironmentID.ValueString()

	updateReq, diags := r.expandAlert(&plan, orgID, envID, ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	alert, err := r.client.UpdateAlert(ctx, orgID, envID, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating alert", "Could not update alert: "+err.Error())
		return
	}

	flatDiags := r.flattenAlert(ctx, alert, &plan, orgID, envID)
	resp.Diagnostics.Append(flatDiags...)
	tflog.Trace(ctx, "updated API alert", map[string]interface{}{"id": alert.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AlertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}
	envID := data.EnvironmentID.ValueString()

	if err := r.client.DeleteAlert(ctx, orgID, envID, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting alert", "Could not delete alert: "+err.Error())
		return
	}

	tflog.Trace(ctx, "deleted API alert", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *AlertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: organization_id/environment_id/alert_id")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// --- Helpers ---

func (r *AlertResource) expandAlert(data *AlertResourceModel, orgID, envID string, ctx context.Context) (*apimanagement.CreateAlertRequest, diag.Diagnostics) {
	var allDiags diag.Diagnostics

	var condModel AlertConditionModel
	diags := data.Condition.As(ctx, &condModel, basetypes.ObjectAsOptions{})
	allDiags.Append(diags...)
	if allDiags.HasError() {
		return nil, allDiags
	}

	var notifModels []AlertNotificationModel
	diags = data.Notifications.ElementsAs(ctx, &notifModels, false)
	allDiags.Append(diags...)
	if allDiags.HasError() {
		return nil, allDiags
	}

	notifications := make([]apimanagement.AlertNotification, len(notifModels))
	for i, n := range notifModels {
		var recipients []string
		diags = n.Recipients.ElementsAs(ctx, &recipients, false)
		allDiags.Append(diags...)

		notifications[i] = apimanagement.AlertNotification{
			Type:       n.Type.ValueString(),
			Recipients: recipients,
			Subject:    n.Subject.ValueString(),
			Message:    n.Message.ValueString(),
		}
	}

	apiInstanceID := data.APIInstanceID.ValueString()
	deploymentType := apimanagement.NormalizeDeploymentType(data.DeploymentType.ValueString())

	return &apimanagement.CreateAlertRequest{
		Name:                 data.Name.ValueString(),
		Type:                 data.Type.ValueString(),
		Severity:             data.Severity.ValueString(),
		MasterOrganizationID: orgID,
		OrganizationID:       orgID,
		EnvironmentID:        envID,
		ResourceType:         data.ResourceType.ValueString(),
		DeploymentType:       deploymentType,
		MetricType:           data.MetricType.ValueString(),
		Resources: []apimanagement.AlertResource{
			{
				APIVersionID:   apiInstanceID,
				APIID:          apiInstanceID,
				Type:           data.ResourceType.ValueString(),
				DeploymentType: deploymentType,
			},
		},
		Condition: apimanagement.AlertCondition{
			Operator:  condModel.Operator.ValueString(),
			Threshold: int(condModel.Threshold.ValueInt64()),
			Interval:  int(condModel.Interval.ValueInt64()),
		},
		WildcardAlert: data.WildcardAlert.ValueBool(),
		Notifications: notifications,
	}, allDiags
}

func (r *AlertResource) flattenAlert(ctx context.Context, alert *apimanagement.Alert, data *AlertResourceModel, orgID, envID string) diag.Diagnostics {
	var allDiags diag.Diagnostics

	data.ID = types.StringValue(alert.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.EnvironmentID = types.StringValue(envID)
	data.Name = types.StringValue(alert.Name)
	data.Type = types.StringValue(alert.Type)
	data.Severity = types.StringValue(alert.Severity)
	data.ResourceType = types.StringValue(alert.ResourceType)
	// Preserve the user's original deployment_type value (e.g. "HY") rather than
	// overwriting with the API's canonical enum name (e.g. "Hybrid") to avoid
	// perpetual diffs between config shortcodes and API response values.
	if data.DeploymentType.IsNull() || data.DeploymentType.IsUnknown() {
		data.DeploymentType = types.StringValue(string(alert.DeploymentType))
	}
	data.MetricType = types.StringValue(alert.MetricType)
	data.WildcardAlert = types.BoolValue(alert.WildcardAlert)

	condObj, diags := types.ObjectValueFrom(ctx, alertConditionAttrTypes, AlertConditionModel{
		Operator:  types.StringValue(alert.Condition.Operator),
		Threshold: types.Int64Value(int64(alert.Condition.Threshold)),
		Interval:  types.Int64Value(int64(alert.Condition.Interval)),
	})
	allDiags.Append(diags...)
	data.Condition = condObj

	notifModels := make([]AlertNotificationModel, len(alert.Notifications))
	for i, n := range alert.Notifications {
		recipientList, diags := types.ListValueFrom(ctx, types.StringType, n.Recipients)
		allDiags.Append(diags...)

		notifModels[i] = AlertNotificationModel{
			Type:       types.StringValue(n.Type),
			Recipients: recipientList,
			Subject:    types.StringValue(n.Subject),
			Message:    types.StringValue(n.Message),
		}
	}

	notifList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: alertNotificationAttrTypes}, notifModels)
	allDiags.Append(diags...)
	data.Notifications = notifList

	return allDiags
}
