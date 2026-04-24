package cloudhub2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &PrivateSpaceResource{}
	_ resource.ResourceWithConfigure   = &PrivateSpaceResource{}
	_ resource.ResourceWithImportState = &PrivateSpaceResource{}
)

// PrivateSpaceResource is the resource implementation.
type PrivateSpaceResource struct {
	client *cloudhub2.PrivateSpacesClient
}

// PrivateSpaceResourceModel describes the resource data model.
type PrivateSpaceResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Region                  types.String `tfsdk:"region"`
	EnableIAMRole           types.Bool   `tfsdk:"enable_iam_role"`
	EnableEgress            types.Bool   `tfsdk:"enable_egress"`
	Status                  types.String `tfsdk:"status"`
	OrganizationID          types.String `tfsdk:"organization_id"`
	RootOrganizationID      types.String `tfsdk:"root_organization_id"`
	MuleAppDeploymentCount  types.Int64  `tfsdk:"mule_app_deployment_count"`
	DaysLeftForRelaxedQuota types.Int64  `tfsdk:"days_left_for_relaxed_quota"`
	VPCMigrationInProgress  types.Bool   `tfsdk:"vpc_migration_in_progress"`
	ManagedFirewallRules    types.List   `tfsdk:"managed_firewall_rules"`
	FirewallRules           types.List   `tfsdk:"firewall_rules"`
	GlobalSpaceStatus       types.Map    `tfsdk:"global_space_status"`
}

func NewPrivateSpaceResource() resource.Resource {
	return &PrivateSpaceResource{}
}

// Metadata returns the resource type name.
func (r *PrivateSpaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space"
}

// Schema defines the schema for the resource.
func (r *PrivateSpaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Anypoint Private Space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private space.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the private space.",
				Required:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region where the private space is located.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enable_iam_role": schema.BoolAttribute{
				Description: "Whether to enable IAM role for the private space.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_egress": schema.BoolAttribute{
				Description: "Whether to enable egress for the private space.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		"status": schema.StringAttribute{
			Description: "The status of the private space.",
			Computed:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space will be created. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"root_organization_id": schema.StringAttribute{
				Description: "The root organization ID of the private space.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		"mule_app_deployment_count": schema.Int64Attribute{
			Description: "The number of mule apps deployed in the private space.",
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"days_left_for_relaxed_quota": schema.Int64Attribute{
			Description: "The number of days left for relaxed quota.",
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"vpc_migration_in_progress": schema.BoolAttribute{
			Description: "Whether the VPC migration is in progress.",
			Computed:    true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
			"managed_firewall_rules": schema.ListAttribute{
				Description: "The managed firewall rules for the private space.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"firewall_rules": schema.ListAttribute{
				Description: "The firewall rules for the private space.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"global_space_status": schema.MapAttribute{
				Description: "The global space status for the private space.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *PrivateSpaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	privateSpaceClient, err := cloudhub2.NewPrivateSpacesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Client",
			"An unexpected error occurred when creating the Private Space client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = privateSpaceClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *PrivateSpaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateSpaceResourceModel

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

	// Create the private space
	enableIAMRole := data.EnableIAMRole.ValueBool()
	enableEgress := data.EnableEgress.ValueBool()

	createRequest := &cloudhub2.CreatePrivateSpaceRequest{
		Name:          data.Name.ValueString(),
		Region:        data.Region.ValueString(),
		EnableIAMRole: &enableIAMRole,
		EnableEgress:  &enableEgress,
	}

	privateSpace, err := r.client.CreatePrivateSpace(ctx, orgID, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating private space",
			"Could not create private space: "+err.Error(),
		)
		return
	}

	// Preserve plan values for fields that may not be immediately
	// reflected in the API response after creation.
	plannedEnableEgress := data.EnableEgress
	plannedEnableIAMRole := data.EnableIAMRole

	data.ID = types.StringValue(privateSpace.ID)
	data.Name = types.StringValue(privateSpace.Name)
	data.Status = types.StringValue(privateSpace.Status)
	data.EnableIAMRole = plannedEnableIAMRole
	data.EnableEgress = plannedEnableEgress
	data.OrganizationID = types.StringValue(orgID)
	data.RootOrganizationID = types.StringValue(privateSpace.RootOrganizationID)
	data.MuleAppDeploymentCount = types.Int64Value(int64(privateSpace.MuleAppDeploymentCount))
	data.DaysLeftForRelaxedQuota = types.Int64Value(int64(privateSpace.DaysLeftForRelaxedQuota))
	data.VPCMigrationInProgress = types.BoolValue(privateSpace.VPCMigrationInProgress)

	data.ManagedFirewallRules = types.ListNull(types.StringType)
	data.FirewallRules = types.ListNull(types.StringType)
	data.GlobalSpaceStatus = types.MapNull(types.StringType)

	tflog.Trace(ctx, "created private space")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *PrivateSpaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateSpaceResourceModel

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

	// Get the private space from the API
	privateSpace, err := r.client.GetPrivateSpace(ctx, orgID, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading private space",
			"Could not read private space ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Preserve state values for enable_egress and enable_iam_role.
	// The API may not reflect these accurately (e.g., returns false when
	// the setting is still being provisioned), which causes Terraform to
	// detect false→true drift and attempt a 405-rejected update every cycle.
	stateEnableEgress := data.EnableEgress
	stateEnableIAMRole := data.EnableIAMRole

	// Map response body to schema
	data.ID = types.StringValue(privateSpace.ID)
	data.Name = types.StringValue(privateSpace.Name)
	data.Status = types.StringValue(privateSpace.Status)
	data.EnableIAMRole = stateEnableIAMRole
	data.EnableEgress = stateEnableEgress
	data.OrganizationID = types.StringValue(privateSpace.OrganizationID)
	data.RootOrganizationID = types.StringValue(privateSpace.RootOrganizationID)
	data.MuleAppDeploymentCount = types.Int64Value(int64(privateSpace.MuleAppDeploymentCount))
	data.DaysLeftForRelaxedQuota = types.Int64Value(int64(privateSpace.DaysLeftForRelaxedQuota))
	data.VPCMigrationInProgress = types.BoolValue(privateSpace.VPCMigrationInProgress)

	// Set complex fields to null for now (they can be properly implemented later if needed)
	data.ManagedFirewallRules = types.ListNull(types.StringType)
	data.FirewallRules = types.ListNull(types.StringType)
	data.GlobalSpaceStatus = types.MapNull(types.StringType)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PrivateSpaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PrivateSpaceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateRequest := &cloudhub2.UpdatePrivateSpaceRequest{}
	hasChanges := false

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		updateRequest.Name = &name
		hasChanges = true
	}

	if !plan.EnableIAMRole.Equal(state.EnableIAMRole) {
		enableIAMRole := plan.EnableIAMRole.ValueBool()
		updateRequest.EnableIAMRole = &enableIAMRole
		hasChanges = true
	}

	if !plan.EnableEgress.Equal(state.EnableEgress) {
		enableEgress := plan.EnableEgress.ValueBool()
		updateRequest.EnableEgress = &enableEgress
		hasChanges = true
	}

	if hasChanges {
		// Determine organization ID from state or default to client's org
		orgID := state.OrganizationID.ValueString()
		if orgID == "" {
			orgID = r.client.OrgID
		}

		privateSpace, err := r.client.UpdatePrivateSpace(ctx, orgID, plan.ID.ValueString(), updateRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating private space",
				"Could not update private space: "+err.Error(),
			)
			return
		}

		// Map response body to schema
		plan.ID = types.StringValue(privateSpace.ID)
		plan.Name = types.StringValue(privateSpace.Name)
		plan.Status = types.StringValue(privateSpace.Status)
		plan.EnableIAMRole = types.BoolValue(privateSpace.EnableIAMRole)
		plan.EnableEgress = types.BoolValue(privateSpace.EnableEgress)
		// Set the actual organization ID used, not what the API returned
		plan.OrganizationID = types.StringValue(orgID)
		plan.RootOrganizationID = types.StringValue(privateSpace.RootOrganizationID)
		plan.MuleAppDeploymentCount = types.Int64Value(int64(privateSpace.MuleAppDeploymentCount))
		plan.DaysLeftForRelaxedQuota = types.Int64Value(int64(privateSpace.DaysLeftForRelaxedQuota))
		plan.VPCMigrationInProgress = types.BoolValue(privateSpace.VPCMigrationInProgress)

		// Set complex fields to null for now (they can be properly implemented later if needed)
		plan.ManagedFirewallRules = types.ListNull(types.StringType)
		plan.FirewallRules = types.ListNull(types.StringType)
		plan.GlobalSpaceStatus = types.MapNull(types.StringType)
	} else {
		// If no changes, ensure all computed fields are populated from state
		plan.OrganizationID = state.OrganizationID
		plan.RootOrganizationID = state.RootOrganizationID
		plan.MuleAppDeploymentCount = state.MuleAppDeploymentCount
		plan.DaysLeftForRelaxedQuota = state.DaysLeftForRelaxedQuota
		plan.VPCMigrationInProgress = state.VPCMigrationInProgress
		plan.ManagedFirewallRules = state.ManagedFirewallRules
		plan.FirewallRules = state.FirewallRules
		plan.GlobalSpaceStatus = state.GlobalSpaceStatus
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *PrivateSpaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateSpaceResourceModel

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

	// Delete the private space
	err := r.client.DeletePrivateSpace(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting private space",
			"Could not delete private space: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *PrivateSpaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
