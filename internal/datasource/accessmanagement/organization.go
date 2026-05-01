package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &OrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationDataSource{}
)

// OrganizationDataSource is the data source implementation.
type OrganizationDataSource struct {
	client *accessmanagement.OrganizationClient
}

// OrganizationDataSourceModel describes the data source data model.
type OrganizationDataSourceModel struct {
	ID                              types.String `tfsdk:"id"`
	Name                            types.String `tfsdk:"name"`
	CreatedAt                       types.String `tfsdk:"created_at"`
	UpdatedAt                       types.String `tfsdk:"updated_at"`
	OwnerID                         types.String `tfsdk:"owner_id"`
	ClientID                        types.String `tfsdk:"client_id"`
	IDProviderID                    types.String `tfsdk:"idprovider_id"`
	IsFederated                     types.Bool   `tfsdk:"is_federated"`
	ParentOrganizationIDs           types.List   `tfsdk:"parent_organization_ids"`
	SubOrganizationIDs              types.List   `tfsdk:"sub_organization_ids"`
	TenantOrganizationIDs           types.List   `tfsdk:"tenant_organization_ids"`
	MfaRequired                     types.String `tfsdk:"mfa_required"`
	IsAutomaticAdminPromotionExempt types.Bool   `tfsdk:"is_automatic_admin_promotion_exempt"`
	OrgType                         types.String `tfsdk:"org_type"`
	GdotID                          types.String `tfsdk:"gdot_id"`
	DeletedAt                       types.String `tfsdk:"deleted_at"`
	Domain                          types.String `tfsdk:"domain"`
	IsRoot                          types.Bool   `tfsdk:"is_root"`
	IsMaster                        types.Bool   `tfsdk:"is_master"`
	Entitlements                    types.String `tfsdk:"entitlements"`
	Subscription                    types.Object `tfsdk:"subscription"`
	Environments                    types.List   `tfsdk:"environments"`
	Owner                           types.Object `tfsdk:"owner"`
	SessionTimeout                  types.Int64  `tfsdk:"session_timeout"`
}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

// Metadata returns the data source type name.
func (d *OrganizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the data source.
func (d *OrganizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about an Anypoint Platform organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the organization.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the organization.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The last update timestamp of the organization.",
				Computed:    true,
			},
			"owner_id": schema.StringAttribute{
				Description: "The owner ID of the organization.",
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID associated with the organization.",
				Computed:    true,
			},
			"idprovider_id": schema.StringAttribute{
				Description: "The identity provider ID.",
				Computed:    true,
			},
			"is_federated": schema.BoolAttribute{
				Description: "Whether the organization is federated.",
				Computed:    true,
			},
			"parent_organization_ids": schema.ListAttribute{
				Description: "List of parent organization IDs.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"sub_organization_ids": schema.ListAttribute{
				Description: "List of sub-organization IDs.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"tenant_organization_ids": schema.ListAttribute{
				Description: "List of tenant organization IDs.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"mfa_required": schema.StringAttribute{
				Description: "Whether MFA is required for the organization.",
				Computed:    true,
			},
			"is_automatic_admin_promotion_exempt": schema.BoolAttribute{
				Description: "Whether the organization is exempt from automatic admin promotion.",
				Computed:    true,
			},
			"org_type": schema.StringAttribute{
				Description: "The type of the organization.",
				Computed:    true,
			},
			"gdot_id": schema.StringAttribute{
				Description: "The GDOT ID of the organization.",
				Computed:    true,
			},
			"deleted_at": schema.StringAttribute{
				Description: "The deletion timestamp of the organization.",
				Computed:    true,
			},
			"domain": schema.StringAttribute{
				Description: "The domain of the organization.",
				Computed:    true,
			},
			"is_root": schema.BoolAttribute{
				Description: "Whether this is a root organization.",
				Computed:    true,
			},
			"is_master": schema.BoolAttribute{
				Description: "Whether this is a master organization.",
				Computed:    true,
			},
			"session_timeout": schema.Int64Attribute{
				Description: "The session timeout for the organization.",
				Computed:    true,
			},
			"subscription": schema.SingleNestedAttribute{
				Description: "The subscription details for the organization.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"category": schema.StringAttribute{
						Description: "The subscription category.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "The subscription type.",
						Computed:    true,
					},
					"expiration": schema.StringAttribute{
						Description: "The subscription expiration date.",
						Computed:    true,
					},
					"justification": schema.StringAttribute{
						Description: "The subscription justification.",
						Computed:    true,
						Optional:    true,
					},
				},
			},
			"owner": schema.SingleNestedAttribute{
				Description: "The owner of the organization.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The owner's ID.",
						Computed:    true,
					},
					"first_name": schema.StringAttribute{
						Description: "The owner's first name.",
						Computed:    true,
					},
					"last_name": schema.StringAttribute{
						Description: "The owner's last name.",
						Computed:    true,
					},
					"email": schema.StringAttribute{
						Description: "The owner's email.",
						Computed:    true,
					},
					"username": schema.StringAttribute{
						Description: "The owner's username.",
						Computed:    true,
					},
					"enabled": schema.BoolAttribute{
						Description: "Whether the owner's account is enabled.",
						Computed:    true,
					},
					"created_at": schema.StringAttribute{
						Description: "The creation timestamp of the owner's account.",
						Computed:    true,
					},
					"updated_at": schema.StringAttribute{
						Description: "The last update timestamp of the owner's account.",
						Computed:    true,
					},
					"organization_id": schema.StringAttribute{
						Description: "The organization ID of the owner.",
						Computed:    true,
					},
					"phone_number": schema.StringAttribute{
						Description: "The owner's phone number.",
						Computed:    true,
					},
					"idprovider_id": schema.StringAttribute{
						Description: "The identity provider ID of the owner.",
						Computed:    true,
					},
					"deleted": schema.BoolAttribute{
						Description: "Whether the owner's account is deleted.",
						Computed:    true,
					},
					"last_login": schema.StringAttribute{
						Description: "The last login timestamp of the owner.",
						Computed:    true,
					},
					"mfa_verification_excluded": schema.BoolAttribute{
						Description: "Whether MFA verification is excluded for the owner.",
						Computed:    true,
					},
					"mfa_verifiers_configured": schema.StringAttribute{
						Description: "The MFA verifiers configured for the owner.",
						Computed:    true,
					},
					"email_verified_at": schema.StringAttribute{
						Description: "The email verification timestamp of the owner.",
						Computed:    true,
						Optional:    true,
					},
					"gdou_id": schema.StringAttribute{
						Description: "The GDOU ID of the owner.",
						Computed:    true,
					},
					"previous_last_login": schema.StringAttribute{
						Description: "The previous last login timestamp of the owner.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "The type of the owner.",
						Computed:    true,
					},
				},
			},
			"environments": schema.ListNestedAttribute{
				Description: "The environments within the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The environment ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The environment name.",
							Computed:    true,
						},
						"organization_id": schema.StringAttribute{
							Description: "The organization ID.",
							Computed:    true,
						},
						"is_production": schema.BoolAttribute{
							Description: "Whether the environment is a production environment.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The environment type.",
							Computed:    true,
						},
						"client_id": schema.StringAttribute{
							Description: "The environment client ID.",
							Computed:    true,
						},
						"arc_namespace": schema.StringAttribute{
							Description: "The ARC namespace of the environment.",
							Computed:    true,
							Optional:    true,
						},
					},
				},
			},
			"entitlements": schema.StringAttribute{
				Description: "The entitlements for the organization as a JSON string. Use jsondecode() to access individual fields.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *OrganizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	// Create user client config for organization operations (requires user authentication)
	userConfig := &client.UserClientConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		BaseURL:      config.BaseURL,
		Timeout:      config.Timeout,
		// Username and password will be filled by UserAnypointClient from env vars
		Username: "",
		Password: "",
	}

	// Create the organization client
	organizationClient, err := accessmanagement.NewOrganizationClient(userConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anypoint Organization API Client",
			"An unexpected error occurred when creating the Anypoint Organization API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}

	d.client = organizationClient
}

// Read refreshes the Terraform state with the latest data.
func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the organization from the API
	organization, err := d.client.GetOrganization(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading organization",
			"Could not read organization ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(organization.ID)
	data.Name = types.StringValue(organization.Name)
	data.CreatedAt = types.StringValue(organization.CreatedAt)
	data.UpdatedAt = types.StringValue(organization.UpdatedAt)
	data.OwnerID = types.StringValue(organization.OwnerID)
	data.ClientID = types.StringValue(organization.ClientID)
	data.IDProviderID = types.StringValue(organization.IDProviderID)
	data.IsFederated = types.BoolValue(organization.IsFederated)
	data.IsAutomaticAdminPromotionExempt = types.BoolValue(organization.IsAutomaticAdminPromotionExempt)
	data.OrgType = types.StringValue(organization.OrgType)
	data.IsRoot = types.BoolValue(organization.IsRoot)
	data.IsMaster = types.BoolValue(organization.IsMaster)

	// Handle nullable fields
	if organization.MfaRequired != "" {
		data.MfaRequired = types.StringValue(organization.MfaRequired)
	} else {
		data.MfaRequired = types.StringNull()
	}

	if organization.GdotID != nil {
		data.GdotID = types.StringValue(*organization.GdotID)
	} else {
		data.GdotID = types.StringNull()
	}

	if organization.DeletedAt != nil {
		data.DeletedAt = types.StringValue(*organization.DeletedAt)
	} else {
		data.DeletedAt = types.StringNull()
	}

	if organization.Domain != nil {
		data.Domain = types.StringValue(*organization.Domain)
	} else {
		data.Domain = types.StringNull()
	}

	// Convert string slices to Terraform lists
	parentOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.ParentOrganizationIDs)
	resp.Diagnostics.Append(diags...)
	data.ParentOrganizationIDs = parentOrgIDs

	subOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.SubOrganizationIDs)
	resp.Diagnostics.Append(diags...)
	data.SubOrganizationIDs = subOrgIDs

	tenantOrgIDs, diags := types.ListValueFrom(ctx, types.StringType, organization.TenantOrganizationIDs)
	resp.Diagnostics.Append(diags...)
	data.TenantOrganizationIDs = tenantOrgIDs

	data.SessionTimeout = types.Int64Value(int64(organization.SessionTimeout))

	// Map environments
	if organization.Environments != nil {
		environments := make([]attr.Value, len(organization.Environments))
		for i, env := range organization.Environments {
			envObj, diags := types.ObjectValueFrom(ctx, getEnvironmentsAttributeTypes(), env)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			environments[i] = envObj
		}
		envs, diags := types.ListValue(types.ObjectType{AttrTypes: getEnvironmentsAttributeTypes()}, environments)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Environments = envs
	}

	// Map owner
	if !reflect.ValueOf(organization.Owner).IsZero() {
		owner, diags := types.ObjectValueFrom(ctx, getOwnerAttributeTypes(), organization.Owner)
		resp.Diagnostics.Append(diags...)
		data.Owner = owner
	}

	// Map subscription
	if !reflect.ValueOf(organization.Subscription).IsZero() {
		subscription, diags := types.ObjectValueFrom(ctx, getSubscriptionAttributeTypes(), organization.Subscription)
		resp.Diagnostics.Append(diags...)
		data.Subscription = subscription
	}

	// Map entitlements as JSON string
	entJSON, err := json.Marshal(organization.Entitlements)
	if err != nil {
		resp.Diagnostics.AddError("Error encoding entitlements", fmt.Sprintf("Failed to marshal entitlements to JSON: %s", err))
		return
	}
	data.Entitlements = types.StringValue(string(entJSON))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func getSubscriptionAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"category":      types.StringType,
		"type":          types.StringType,
		"expiration":    types.StringType,
		"justification": types.StringType,
	}
}

func getOwnerAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                        types.StringType,
		"first_name":                types.StringType,
		"last_name":                 types.StringType,
		"email":                     types.StringType,
		"username":                  types.StringType,
		"enabled":                   types.BoolType,
		"created_at":                types.StringType,
		"updated_at":                types.StringType,
		"organization_id":           types.StringType,
		"phone_number":              types.StringType,
		"idprovider_id":             types.StringType,
		"deleted":                   types.BoolType,
		"last_login":                types.StringType,
		"mfa_verification_excluded": types.BoolType,
		"mfa_verifiers_configured":  types.StringType,
		"email_verified_at":         types.StringType,
		"gdou_id":                   types.StringType,
		"previous_last_login":       types.StringType,
		"type":                      types.StringType,
	}
}

func getEnvironmentsAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":              types.StringType,
		"name":            types.StringType,
		"organization_id": types.StringType,
		"is_production":   types.BoolType,
		"type":            types.StringType,
		"client_id":       types.StringType,
		"arc_namespace":   types.StringType,
	}
}
