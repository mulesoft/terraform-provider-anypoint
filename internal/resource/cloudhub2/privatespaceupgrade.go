package cloudhub2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &PrivateSpaceUpgradeResource{}
	_ resource.ResourceWithConfigure   = &PrivateSpaceUpgradeResource{}
	_ resource.ResourceWithImportState = &PrivateSpaceUpgradeResource{}
)

// PrivateSpaceUpgradeResource is the resource implementation.
type PrivateSpaceUpgradeResource struct {
	client *cloudhub2.PrivateSpaceUpgradeClient
}

// PrivateSpaceUpgradeResourceModel describes the resource data model.
type PrivateSpaceUpgradeResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	PrivateSpaceID      types.String `tfsdk:"private_space_id"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	Date                types.String `tfsdk:"date"`
	OptIn               types.Bool   `tfsdk:"opt_in"`
	ScheduledUpdateTime types.String `tfsdk:"scheduled_update_time"`
	Status              types.String `tfsdk:"status"`
}

func NewPrivateSpaceUpgradeResource() resource.Resource {
	return &PrivateSpaceUpgradeResource{}
}

// Metadata returns the resource type name.
func (r *PrivateSpaceUpgradeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_upgrade"
}

// Schema defines the schema for the resource.
func (r *PrivateSpaceUpgradeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schedules an upgrade for a CloudHub 2.0 private space. Scheduled upgrades can be cancelled by deleting this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the upgrade operation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space to upgrade.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"date": schema.StringAttribute{
				Description: "The date when the upgrade should be scheduled (format: YYYY-MM-DD).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"opt_in": schema.BoolAttribute{
				Description: "Whether to opt in to the upgrade.",
				Required:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"scheduled_update_time": schema.StringAttribute{
				Description: "The scheduled update time returned by the API.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the upgrade operation.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *PrivateSpaceUpgradeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	upgradeClient, err := cloudhub2.NewPrivateSpaceUpgradeClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Upgrade Client",
			"An unexpected error occurred when creating the Private Space Upgrade client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = upgradeClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *PrivateSpaceUpgradeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateSpaceUpgradeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate date format
	date := data.Date.ValueString()
	if _, err := time.Parse("2006-01-02", date); err != nil {
		resp.Diagnostics.AddError(
			"Invalid date format",
			fmt.Sprintf("Date must be in YYYY-MM-DD format, got: %s", date),
		)
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Create the upgrade request
	upgradeRequest := &cloudhub2.UpgradePrivateSpaceRequest{
		Date:  date,
		OptIn: data.OptIn.ValueBool(),
	}

	// Schedule the upgrade
	upgradeResponse, err := r.client.UpgradePrivateSpace(ctx, orgID, data.PrivateSpaceID.ValueString(), upgradeRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error scheduling private space upgrade",
			"Could not schedule private space upgrade: "+err.Error(),
		)
		return
	}

	// Set the ID as a combination of private space ID and timestamp for uniqueness
	data.ID = types.StringValue(fmt.Sprintf("%s-%d", data.PrivateSpaceID.ValueString(), time.Now().Unix()))
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used
	data.ScheduledUpdateTime = types.StringValue(upgradeResponse.ScheduledUpdateTime)
	data.Status = types.StringValue(upgradeResponse.Status)

	tflog.Trace(ctx, "scheduled private space upgrade")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *PrivateSpaceUpgradeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateSpaceUpgradeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For upgrade operations, we don't typically need to refresh state
	// as it's a one-time operation. Keep the existing state.
	tflog.Trace(ctx, "read private space upgrade state")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PrivateSpaceUpgradeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Upgrade operations typically don't support updates
	// The schema is configured to force replacement for any changes
	resp.Diagnostics.AddError(
		"Update not supported",
		"Private space upgrade operations cannot be updated. Any changes will require replacement.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *PrivateSpaceUpgradeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateSpaceUpgradeResourceModel

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

	// Delete/cancel the scheduled upgrade
	err := r.client.DeletePrivateSpaceUpgrade(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting private space upgrade",
			"Could not delete/cancel private space upgrade: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted private space upgrade")
}

// ImportState imports the resource into Terraform state.
func (r *PrivateSpaceUpgradeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id:date:opt_in
	// Example: my-space-id:2025-08-12:true
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: private_space_id:date:opt_in",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("date"), parts[1])...)

	optIn := parts[2] == "true"
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("opt_in"), optIn)...)
}
