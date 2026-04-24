package cloudhub2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource              = &PrivateSpaceDataSource{}
	_ datasource.DataSourceWithConfigure = &PrivateSpaceDataSource{}
)

// PrivateSpaceDataSource is the data source implementation.
type PrivateSpaceDataSource struct {
	client *cloudhub2.PrivateSpacesClient
}

// PrivateSpaceDataSourceModel describes the data source data model.
type PrivateSpaceDataSourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Version                 types.String `tfsdk:"version"`
	Status                  types.String `tfsdk:"status"`
	StatusMessage           types.String `tfsdk:"status_message"`
	Region                  types.String `tfsdk:"region"`
	OrganizationID          types.String `tfsdk:"organization_id"`
	RootOrganizationID      types.String `tfsdk:"root_organization_id"`
	EnableIAMRole           types.Bool   `tfsdk:"enable_iam_role"`
	EnableEgress            types.Bool   `tfsdk:"enable_egress"`
	EnableNetworkIsolation  types.Bool   `tfsdk:"enable_network_isolation"`
	MuleAppDeploymentCount  types.Int64  `tfsdk:"mule_app_deployment_count"`
	DaysLeftForRelaxedQuota types.Int64  `tfsdk:"days_left_for_relaxed_quota"`
	VPCMigrationInProgress  types.Bool   `tfsdk:"vpc_migration_in_progress"`
	// Complex nested objects (keeping them as types.String for JSON)
	Provisioning         types.String `tfsdk:"provisioning"`
	ManagedFirewallRules types.String `tfsdk:"managed_firewall_rules"`
	Environments         types.String `tfsdk:"environments"`
	Network              types.String `tfsdk:"network"`
	FirewallRules        types.String `tfsdk:"firewall_rules"`
	LogForwarding        types.String `tfsdk:"log_forwarding"`
	IngressConfiguration types.String `tfsdk:"ingress_configuration"`
	GlobalSpaceStatus    types.String `tfsdk:"global_space_status"`
}

func NewPrivateSpaceDataSource() datasource.DataSource {
	return &PrivateSpaceDataSource{}
}

// Metadata returns the data source type name.
func (d *PrivateSpaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space"
}

// Schema defines the schema for the data source.
func (d *PrivateSpaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches information about a CloudHub 2.0 private space.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the private space.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the private space.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "The version of the private space.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the private space.",
				Computed:    true,
			},
			"status_message": schema.StringAttribute{
				Description: "The status message of the private space.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region of the private space.",
				Computed:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID where the private space is located. If not specified, uses the organization from provider credentials.",
				Optional:    true,
				Computed:    true,
			},
			"root_organization_id": schema.StringAttribute{
				Description: "The root organization ID of the private space.",
				Computed:    true,
			},
			"enable_iam_role": schema.BoolAttribute{
				Description: "Whether IAM role is enabled for the private space.",
				Computed:    true,
			},
			"enable_egress": schema.BoolAttribute{
				Description: "Whether egress is enabled for the private space.",
				Computed:    true,
			},
			"enable_network_isolation": schema.BoolAttribute{
				Description: "Whether network isolation is enabled for the private space.",
				Computed:    true,
			},
			"mule_app_deployment_count": schema.Int64Attribute{
				Description: "The number of Mule app deployments in the private space.",
				Computed:    true,
			},
			"days_left_for_relaxed_quota": schema.Int64Attribute{
				Description: "The number of days left for relaxed quota.",
				Computed:    true,
			},
			"vpc_migration_in_progress": schema.BoolAttribute{
				Description: "Whether VPC migration is in progress.",
				Computed:    true,
			},
			"provisioning": schema.StringAttribute{
				Description: "The provisioning status and message (JSON).",
				Computed:    true,
			},
			"managed_firewall_rules": schema.StringAttribute{
				Description: "The managed firewall rules (JSON).",
				Computed:    true,
			},
			"environments": schema.StringAttribute{
				Description: "The environments configuration (JSON).",
				Computed:    true,
			},
			"network": schema.StringAttribute{
				Description: "The network configuration (JSON).",
				Computed:    true,
			},
			"firewall_rules": schema.StringAttribute{
				Description: "The firewall rules (JSON).",
				Computed:    true,
			},
			"log_forwarding": schema.StringAttribute{
				Description: "The log forwarding configuration (JSON).",
				Computed:    true,
			},
			"ingress_configuration": schema.StringAttribute{
				Description: "The ingress configuration (JSON).",
				Computed:    true,
			},
			"global_space_status": schema.StringAttribute{
				Description: "The global space status (JSON).",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *PrivateSpaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	// Create the private space client
	privateSpaceClient, err := cloudhub2.NewPrivateSpacesClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create CloudHub 2.0 Private Space API Client",
			"An unexpected error occurred when creating the CloudHub 2.0 Private Space API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"CloudHub 2.0 Client Error: "+err.Error(),
		)
		return
	}

	d.client = privateSpaceClient
}

// Read refreshes the Terraform state with the latest data.
func (d *PrivateSpaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateSpaceDataSourceModel

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

	// Get the private space from the API
	privateSpace, err := d.client.GetPrivateSpace(ctx, orgID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private space",
			"Could not read private space ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate all attribute values
	data.ID = types.StringValue(privateSpace.ID)
	data.Name = types.StringValue(privateSpace.Name)
	data.Version = types.StringValue(privateSpace.Version)
	data.Status = types.StringValue(privateSpace.Status)
	data.StatusMessage = types.StringValue(privateSpace.StatusMessage)
	data.Region = types.StringValue(privateSpace.Region)
	data.OrganizationID = types.StringValue(privateSpace.OrganizationID)
	data.RootOrganizationID = types.StringValue(privateSpace.RootOrganizationID)
	data.EnableIAMRole = types.BoolValue(privateSpace.EnableIAMRole)
	data.EnableEgress = types.BoolValue(privateSpace.EnableEgress)
	data.EnableNetworkIsolation = types.BoolValue(privateSpace.EnableNetworkIsolation)
	data.MuleAppDeploymentCount = types.Int64Value(int64(privateSpace.MuleAppDeploymentCount))
	data.DaysLeftForRelaxedQuota = types.Int64Value(int64(privateSpace.DaysLeftForRelaxedQuota))
	data.VPCMigrationInProgress = types.BoolValue(privateSpace.VPCMigrationInProgress)

	// Convert complex objects to JSON strings
	if provisioningJSON, err := json.Marshal(privateSpace.Provisioning); err == nil {
		data.Provisioning = types.StringValue(string(provisioningJSON))
	} else {
		data.Provisioning = types.StringValue("")
	}

	if managedFirewallRulesJSON, err := json.Marshal(privateSpace.ManagedFirewallRules); err == nil {
		data.ManagedFirewallRules = types.StringValue(string(managedFirewallRulesJSON))
	} else {
		data.ManagedFirewallRules = types.StringValue("")
	}

	if environmentsJSON, err := json.Marshal(privateSpace.Environments); err == nil {
		data.Environments = types.StringValue(string(environmentsJSON))
	} else {
		data.Environments = types.StringValue("")
	}

	if networkJSON, err := json.Marshal(privateSpace.Network); err == nil {
		data.Network = types.StringValue(string(networkJSON))
	} else {
		data.Network = types.StringValue("")
	}

	if firewallRulesJSON, err := json.Marshal(privateSpace.FirewallRules); err == nil {
		data.FirewallRules = types.StringValue(string(firewallRulesJSON))
	} else {
		data.FirewallRules = types.StringValue("")
	}

	if logForwardingJSON, err := json.Marshal(privateSpace.LogForwarding); err == nil {
		data.LogForwarding = types.StringValue(string(logForwardingJSON))
	} else {
		data.LogForwarding = types.StringValue("")
	}

	if ingressConfigurationJSON, err := json.Marshal(privateSpace.IngressConfiguration); err == nil {
		data.IngressConfiguration = types.StringValue(string(ingressConfigurationJSON))
	} else {
		data.IngressConfiguration = types.StringValue("")
	}

	if globalSpaceStatusJSON, err := json.Marshal(privateSpace.GlobalSpaceStatus); err == nil {
		data.GlobalSpaceStatus = types.StringValue(string(globalSpaceStatusJSON))
	} else {
		data.GlobalSpaceStatus = types.StringValue("")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
