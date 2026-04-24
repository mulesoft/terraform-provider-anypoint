package cloudhub2

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

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &FirewallRulesResource{}
	_ resource.ResourceWithConfigure   = &FirewallRulesResource{}
	_ resource.ResourceWithImportState = &FirewallRulesResource{}
)

// FirewallRulesResource is the resource implementation.
type FirewallRulesResource struct {
	client *cloudhub2.FirewallRulesClient
}

// FirewallRulesResourceModel describes the resource data model.
type FirewallRulesResourceModel struct {
	ID             types.String        `tfsdk:"id"`
	PrivateSpaceID types.String        `tfsdk:"private_space_id"`
	OrganizationID types.String        `tfsdk:"organization_id"`
	Rules          []FirewallRuleModel `tfsdk:"rules"`
}

// FirewallRuleModel describes individual firewall rule data model.
type FirewallRuleModel struct {
	CidrBlock types.String `tfsdk:"cidr_block"`
	Protocol  types.String `tfsdk:"protocol"`
	FromPort  types.Int64  `tfsdk:"from_port"`
	ToPort    types.Int64  `tfsdk:"to_port"`
	Type      types.String `tfsdk:"type"`
}

func NewFirewallRulesResource() resource.Resource {
	return &FirewallRulesResource{}
}

// Metadata returns the resource type name.
func (r *FirewallRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_rules"
}

// Schema defines the schema for the resource.
func (r *FirewallRulesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages firewall rules for an Anypoint Private Space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the firewall rules (same as private_space_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space for which to manage firewall rules.",
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
			"rules": schema.ListNestedAttribute{
				Description: "List of firewall rules.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr_block": schema.StringAttribute{
							Description: "The CIDR block for the firewall rule.",
							Required:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "The protocol for the firewall rule (tcp, udp, icmp).",
							Required:    true,
						},
						"from_port": schema.Int64Attribute{
							Description: "The starting port for the firewall rule.",
							Required:    true,
						},
						"to_port": schema.Int64Attribute{
							Description: "The ending port for the firewall rule.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the firewall rule (inbound, outbound).",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *FirewallRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	firewallRulesClient, err := cloudhub2.NewFirewallRulesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Firewall Rules Client",
			"An unexpected error occurred when creating the Firewall Rules client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = firewallRulesClient
}

// mapFirewallRulesToAPI converts Terraform firewall rule models to API format
func mapFirewallRulesToAPI(rules []FirewallRuleModel) []cloudhub2.FirewallRule {
	apiRules := make([]cloudhub2.FirewallRule, len(rules))
	for i, rule := range rules {
		apiRules[i] = cloudhub2.FirewallRule{
			CidrBlock: rule.CidrBlock.ValueString(),
			Protocol:  rule.Protocol.ValueString(),
			FromPort:  int(rule.FromPort.ValueInt64()),
			ToPort:    int(rule.ToPort.ValueInt64()),
			Type:      rule.Type.ValueString(),
		}
	}
	return apiRules
}

// mapFirewallRulesFromAPI converts API firewall rules to Terraform models
func mapFirewallRulesFromAPI(apiRules []cloudhub2.FirewallRule) []FirewallRuleModel {
	rules := make([]FirewallRuleModel, len(apiRules))
	for i, apiRule := range apiRules {
		rules[i] = FirewallRuleModel{
			CidrBlock: types.StringValue(apiRule.CidrBlock),
			Protocol:  types.StringValue(apiRule.Protocol),
			FromPort:  types.Int64Value(int64(apiRule.FromPort)),
			ToPort:    types.Int64Value(int64(apiRule.ToPort)),
			Type:      types.StringValue(apiRule.Type),
		}
	}
	return rules
}

// Create creates the resource and sets the initial Terraform state.
func (r *FirewallRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FirewallRulesResourceModel

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

	// Convert Terraform rules to API format
	apiRules := mapFirewallRulesToAPI(data.Rules)

	// Create the update request
	updateRequest := &cloudhub2.UpdateFirewallRulesRequest{
		ManagedFirewallRules: apiRules,
	}

	// Update the firewall rules
	privateSpace, err := r.client.UpdateFirewallRules(ctx, orgID, data.PrivateSpaceID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating firewall rules",
			"Could not create firewall rules: "+err.Error(),
		)
		return
	}

	// Map response back to Terraform model
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString())
	data.OrganizationID = types.StringValue(orgID) // Set the actual org ID used
	data.Rules = mapFirewallRulesFromAPI(privateSpace.ManagedFirewallRules)

	tflog.Trace(ctx, "created firewall rules")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *FirewallRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FirewallRulesResourceModel

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

	// Get the firewall rules from the API
	privateSpace, err := r.client.GetFirewallRules(ctx, orgID, data.PrivateSpaceID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading firewall rules",
			"Could not read firewall rules for private space "+data.PrivateSpaceID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response back to Terraform model
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString())
	data.Rules = mapFirewallRulesFromAPI(privateSpace.ManagedFirewallRules)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *FirewallRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FirewallRulesResourceModel

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

	// Convert Terraform rules to API format
	apiRules := mapFirewallRulesToAPI(data.Rules)

	// Create the update request
	updateRequest := &cloudhub2.UpdateFirewallRulesRequest{
		ManagedFirewallRules: apiRules,
	}

	// Update the firewall rules
	privateSpace, err := r.client.UpdateFirewallRules(ctx, orgID, data.PrivateSpaceID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating firewall rules",
			"Could not update firewall rules: "+err.Error(),
		)
		return
	}

	// Map response back to Terraform model
	data.ID = types.StringValue(data.PrivateSpaceID.ValueString())
	data.Rules = mapFirewallRulesFromAPI(privateSpace.ManagedFirewallRules)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *FirewallRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FirewallRulesResourceModel

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

	// Clear all firewall rules by setting empty list
	updateRequest := &cloudhub2.UpdateFirewallRulesRequest{
		ManagedFirewallRules: []cloudhub2.FirewallRule{},
	}

	// Update the firewall rules with empty list
	_, err := r.client.UpdateFirewallRules(ctx, orgID, data.PrivateSpaceID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting firewall rules",
			"Could not delete firewall rules: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *FirewallRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
