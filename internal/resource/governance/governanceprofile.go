package governance

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/governance"
)

var (
	_ resource.Resource                = &GovernanceProfileResource{}
	_ resource.ResourceWithConfigure   = &GovernanceProfileResource{}
	_ resource.ResourceWithImportState = &GovernanceProfileResource{}
)

type GovernanceProfileResource struct {
	client *governance.GovernanceProfileClient
}

type GovernanceProfileResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Filter             types.String `tfsdk:"filter"`
	Rulesets           types.List   `tfsdk:"rulesets"`
	Allowing           types.List   `tfsdk:"allowing"`
	Denying            types.List   `tfsdk:"denying"`
	NotificationConfig types.Object `tfsdk:"notification_config"`
}

type RulesetModel struct {
	GroupID  types.String `tfsdk:"group_id"`
	AssetID  types.String `tfsdk:"asset_id"`
	Version  types.String `tfsdk:"version"`
}

type NotificationConfigModel struct {
	Enabled       types.Bool `tfsdk:"enabled"`
	Notifications types.List `tfsdk:"notifications"`
}

type NotificationModel struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	Condition  types.String `tfsdk:"condition"`
	Recipients types.List   `tfsdk:"recipients"`
}

type RecipientModel struct {
	ContactType      types.String `tfsdk:"contact_type"`
	NotificationType types.String `tfsdk:"notification_type"`
	Value            types.String `tfsdk:"value"`
	Label            types.String `tfsdk:"label"`
}

var rulesetObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"group_id": types.StringType,
		"asset_id": types.StringType,
		"version":  types.StringType,
	},
}

var recipientObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"contact_type":      types.StringType,
		"notification_type": types.StringType,
		"value":             types.StringType,
		"label":             types.StringType,
	},
}

var notificationObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"enabled":    types.BoolType,
		"condition":  types.StringType,
		"recipients": types.ListType{ElemType: recipientObjectType},
	},
}

var notificationConfigAttrTypes = map[string]attr.Type{
	"enabled":       types.BoolType,
	"notifications": types.ListType{ElemType: notificationObjectType},
}

func NewGovernanceProfileResource() resource.Resource {
	return &GovernanceProfileResource{}
}

func (r *GovernanceProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_governance_profile"
}

func (r *GovernanceProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API Governance profile in Anypoint Platform. " +
			"A governance profile applies Exchange rulesets to APIs matching a filter criteria.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the governance profile.",
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
			"name": schema.StringAttribute{
				Description: "Name of the governance profile.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the governance profile.",
				Optional:    true,
			},
			"filter": schema.StringAttribute{
				Description: "Filter expression to select APIs (e.g. 'scope:http-api').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("scope:http-api"),
			},
			"rulesets": schema.ListNestedAttribute{
				Description: "List of Exchange rulesets to apply. Each ruleset is identified by group_id, asset_id, and version.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.StringAttribute{
							Description: "Exchange group ID of the ruleset.",
							Required:    true,
						},
						"asset_id": schema.StringAttribute{
							Description: "Exchange asset ID of the ruleset (e.g. 'anypoint-best-practices').",
							Required:    true,
						},
						"version": schema.StringAttribute{
							Description: "Version of the ruleset (e.g. '1.0.0' or 'latest').",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("latest"),
						},
					},
				},
			},
			"allowing": schema.ListAttribute{
				Description: "List of API identifiers to explicitly allow (exceptions).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"denying": schema.ListAttribute{
				Description: "List of API identifiers to explicitly deny.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"notification_config": schema.SingleNestedAttribute{
				Description: "Notification configuration for governance violations.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Whether notifications are enabled.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"notifications": schema.ListNestedAttribute{
						Description: "List of notification rules.",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"enabled": schema.BoolAttribute{
									Description: "Whether this notification rule is enabled.",
									Optional:    true,
									Computed:    true,
									Default:     booldefault.StaticBool(true),
								},
								"condition": schema.StringAttribute{
									Description: "When to send notification (e.g. 'OnFailure').",
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("OnFailure"),
								},
								"recipients": schema.ListNestedAttribute{
									Description: "Notification recipients.",
									Optional:    true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"contact_type": schema.StringAttribute{
												Description: "Contact type (e.g. 'Publisher').",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString("Publisher"),
											},
											"notification_type": schema.StringAttribute{
												Description: "Notification channel type (e.g. 'Email').",
												Optional:    true,
												Computed:    true,
												Default:     stringdefault.StaticString("Email"),
											},
											"value": schema.StringAttribute{
												Description: "Recipient value (e.g. email address). Empty for publisher-type contacts.",
												Optional:    true,
												Computed:    true,
											},
											"label": schema.StringAttribute{
												Description: "Recipient label.",
												Optional:    true,
												Computed:    true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *GovernanceProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	govClient, err := governance.NewGovernanceProfileClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Governance Profile Client",
			"An unexpected error occurred when creating the Governance Profile client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	r.client = govClient
}

func (r *GovernanceProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GovernanceProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	createReq, diags := r.expandProfile(ctx, &data, orgID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, err := r.client.CreateProfile(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating governance profile", "Could not create governance profile: "+err.Error())
		return
	}

	flatDiags := r.flattenProfile(ctx, profile, &data, orgID)
	resp.Diagnostics.Append(flatDiags...)
	tflog.Trace(ctx, "created governance profile", map[string]interface{}{"id": profile.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GovernanceProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GovernanceProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	profile, err := r.client.GetProfile(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading governance profile", "Could not read governance profile: "+err.Error())
		return
	}

	flatDiags := r.flattenProfile(ctx, profile, &data, orgID)
	resp.Diagnostics.Append(flatDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GovernanceProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state GovernanceProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	updateReq, diags := r.expandProfile(ctx, &plan, orgID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, err := r.client.UpdateProfile(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating governance profile", "Could not update governance profile: "+err.Error())
		return
	}

	flatDiags := r.flattenProfile(ctx, profile, &plan, orgID)
	resp.Diagnostics.Append(flatDiags...)
	tflog.Trace(ctx, "updated governance profile", map[string]interface{}{"id": profile.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GovernanceProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GovernanceProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteProfile(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting governance profile", "Could not delete governance profile: "+err.Error())
		return
	}

	tflog.Trace(ctx, "deleted governance profile", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *GovernanceProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- Helpers ---

// rulesetToGAV converts structured ruleset fields to a GAV URI.
func rulesetToGAV(groupID, assetID, version string) string {
	return fmt.Sprintf("gav://%s/%s/%s", groupID, assetID, version)
}

// gavToRuleset parses a GAV URI into group_id, asset_id, version.
func gavToRuleset(gav string) (groupID, assetID, version string) {
	trimmed := strings.TrimPrefix(gav, "gav://")
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2]
	}
	return "", "", ""
}

func (r *GovernanceProfileResource) expandProfile(ctx context.Context, data *GovernanceProfileResourceModel, orgID string) (*governance.CreateGovernanceProfileRequest, diag.Diagnostics) {
	var allDiags diag.Diagnostics

	// Expand rulesets
	var rulesetModels []RulesetModel
	diags := data.Rulesets.ElementsAs(ctx, &rulesetModels, false)
	allDiags.Append(diags...)
	if allDiags.HasError() {
		return nil, allDiags
	}

	rulesetGAVs := make([]string, len(rulesetModels))
	for i, rs := range rulesetModels {
		rulesetGAVs[i] = rulesetToGAV(rs.GroupID.ValueString(), rs.AssetID.ValueString(), rs.Version.ValueString())
	}

	// Expand allowing/denying
	var allowing, denying []string
	if !data.Allowing.IsNull() && !data.Allowing.IsUnknown() {
		diags = data.Allowing.ElementsAs(ctx, &allowing, false)
		allDiags.Append(diags...)
	}
	if allowing == nil {
		allowing = []string{}
	}
	if !data.Denying.IsNull() && !data.Denying.IsUnknown() {
		diags = data.Denying.ElementsAs(ctx, &denying, false)
		allDiags.Append(diags...)
	}
	if denying == nil {
		denying = []string{}
	}

	// Expand notification config
	notifConfig := governance.NotificationConfig{Enabled: true}
	if !data.NotificationConfig.IsNull() && !data.NotificationConfig.IsUnknown() {
		var ncModel NotificationConfigModel
		diags = data.NotificationConfig.As(ctx, &ncModel, basetypes.ObjectAsOptions{})
		allDiags.Append(diags...)

		notifConfig.Enabled = ncModel.Enabled.ValueBool()

		if !ncModel.Notifications.IsNull() && !ncModel.Notifications.IsUnknown() {
			var notifModels []NotificationModel
			diags = ncModel.Notifications.ElementsAs(ctx, &notifModels, false)
			allDiags.Append(diags...)

			notifConfig.Notifications = make([]governance.Notification, len(notifModels))
			for i, n := range notifModels {
				notif := governance.Notification{
					Enabled:   n.Enabled.ValueBool(),
					Condition: n.Condition.ValueString(),
				}

				if !n.Recipients.IsNull() && !n.Recipients.IsUnknown() {
					var recipModels []RecipientModel
					diags = n.Recipients.ElementsAs(ctx, &recipModels, false)
					allDiags.Append(diags...)

					notif.Recipients = make([]governance.NotifyRecipient, len(recipModels))
					for j, rec := range recipModels {
						notif.Recipients[j] = governance.NotifyRecipient{
							ContactType:      rec.ContactType.ValueString(),
							NotificationType: rec.NotificationType.ValueString(),
							Value:            rec.Value.ValueString(),
							Label:            rec.Label.ValueString(),
						}
					}
				}
				notifConfig.Notifications[i] = notif
			}
		}
	}

	return &governance.CreateGovernanceProfileRequest{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		Org:                orgID,
		Rulesets:           rulesetGAVs,
		Filter:             data.Filter.ValueString(),
		Allowing:           allowing,
		Denying:            denying,
		NotificationConfig: notifConfig,
	}, allDiags
}

func (r *GovernanceProfileResource) flattenProfile(ctx context.Context, profile *governance.GovernanceProfile, data *GovernanceProfileResourceModel, orgID string) diag.Diagnostics {
	var allDiags diag.Diagnostics

	data.ID = types.StringValue(profile.ID)
	data.OrganizationID = types.StringValue(orgID)
	data.Name = types.StringValue(profile.Name)

	if profile.Description != "" {
		data.Description = types.StringValue(profile.Description)
	}
	if profile.Filter != "" {
		data.Filter = types.StringValue(profile.Filter)
	}

	// Flatten rulesets (GAV URIs -> structured objects)
	rulesetModels := make([]RulesetModel, len(profile.Rulesets))
	for i, gav := range profile.Rulesets {
		groupID, assetID, version := gavToRuleset(gav)
		rulesetModels[i] = RulesetModel{
			GroupID: types.StringValue(groupID),
			AssetID: types.StringValue(assetID),
			Version: types.StringValue(version),
		}
	}
	rulesetList, diags := types.ListValueFrom(ctx, rulesetObjectType, rulesetModels)
	allDiags.Append(diags...)
	data.Rulesets = rulesetList

	// Flatten allowing/denying
	if len(profile.Allowing) > 0 {
		allowList, diags := types.ListValueFrom(ctx, types.StringType, profile.Allowing)
		allDiags.Append(diags...)
		data.Allowing = allowList
	} else {
		data.Allowing = types.ListValueMust(types.StringType, []attr.Value{})
	}

	if len(profile.Denying) > 0 {
		denyList, diags := types.ListValueFrom(ctx, types.StringType, profile.Denying)
		allDiags.Append(diags...)
		data.Denying = denyList
	} else {
		data.Denying = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Flatten notification config
	notifModels := make([]NotificationModel, len(profile.NotificationConfig.Notifications))
	for i, n := range profile.NotificationConfig.Notifications {
		recipModels := make([]RecipientModel, len(n.Recipients))
		for j, rec := range n.Recipients {
			recipModels[j] = RecipientModel{
				ContactType:      types.StringValue(rec.ContactType),
				NotificationType: types.StringValue(rec.NotificationType),
				Value:            types.StringValue(rec.Value),
				Label:            types.StringValue(rec.Label),
			}
		}

		recipList, diags := types.ListValueFrom(ctx, recipientObjectType, recipModels)
		allDiags.Append(diags...)

		notifModels[i] = NotificationModel{
			Enabled:    types.BoolValue(n.Enabled),
			Condition:  types.StringValue(n.Condition),
			Recipients: recipList,
		}
	}

	notifList, diags := types.ListValueFrom(ctx, notificationObjectType, notifModels)
	allDiags.Append(diags...)

	ncObj, diags := types.ObjectValueFrom(ctx, notificationConfigAttrTypes, NotificationConfigModel{
		Enabled:       types.BoolValue(profile.NotificationConfig.Enabled),
		Notifications: notifList,
	})
	allDiags.Append(diags...)
	data.NotificationConfig = ncObj

	return allDiags
}
