package cloudhub2

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &PrivateSpaceAssociationResource{}
	_ resource.ResourceWithConfigure   = &PrivateSpaceAssociationResource{}
	_ resource.ResourceWithImportState = &PrivateSpaceAssociationResource{}
)

// PrivateSpaceAssociationResource is the resource implementation.
type PrivateSpaceAssociationResource struct {
	client *cloudhub2.PrivateSpaceAssociationClient
}

// PrivateSpaceAssociationResourceModel describes the resource data model.
type PrivateSpaceAssociationResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	PrivateSpaceID      types.String `tfsdk:"private_space_id"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	Associations        types.List   `tfsdk:"associations"`
	CreatedAssociations types.List   `tfsdk:"created_associations"`
}

func NewPrivateSpaceAssociationResource() resource.Resource {
	return &PrivateSpaceAssociationResource{}
}

// Metadata returns the resource type name.
func (r *PrivateSpaceAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_space_association"
}

// Schema defines the schema for the resource.
func (r *PrivateSpaceAssociationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages associations between a CloudHub 2.0 private space and environments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the Private Space Association resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_space_id": schema.StringAttribute{
				Description: "The ID of the private space.",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID. If not provided, the provider's default organization will be used.",
				Optional:    true,
				Computed:    true,
			},
			"associations": schema.ListNestedAttribute{
				Description: "List of associations to create between the private space and environments.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"organization_id": schema.StringAttribute{
							Description: "The organization ID for the association.",
							Required:    true,
						},
						"environment": schema.StringAttribute{
							Description: "The environment for the association. Can be an environment UUID, 'all', 'production', or 'sandbox'.",
							Required:    true,
						},
					},
				},
			},
			"created_associations": schema.ListNestedAttribute{
				Description: "List of created associations with their IDs.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the created association.",
							Computed:    true,
						},
						"organization_id": schema.StringAttribute{
							Description: "The organization ID of the association.",
							Computed:    true,
						},
						"environment": schema.StringAttribute{
							Description: "The environment of the association.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *PrivateSpaceAssociationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	privateSpaceAssociationClient, err := cloudhub2.NewPrivateSpaceAssociationClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Private Space Association Client",
			"An unexpected error occurred when creating the Private Space Association client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Client Error: "+err.Error(),
		)
		return
	}

	r.client = privateSpaceAssociationClient
}

// Create creates the resource and sets the initial Terraform state.
func (r *PrivateSpaceAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PrivateSpaceAssociationResourceModel

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

	// Convert associations from Terraform list to request format
	var associations []cloudhub2.AssociationRequest
	for _, elem := range data.Associations.Elements() {
		assocObj := elem.(types.Object)
		assocAttrs := assocObj.Attributes()

		associations = append(associations, cloudhub2.AssociationRequest{
			OrganizationID: assocAttrs["organization_id"].(types.String).ValueString(),
			Environment:    assocAttrs["environment"].(types.String).ValueString(),
		})
	}

	// Build the request
	createRequest := &cloudhub2.CreatePrivateSpaceAssociationRequest{
		Associations: associations,
	}

	// Create the Private Space Associations
	createdAssociations, err := r.client.CreatePrivateSpaceAssociations(ctx, orgID, data.PrivateSpaceID.ValueString(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Private Space Associations",
			"Could not create Private Space Associations: "+err.Error(),
		)
		return
	}

	// Generate a resource ID
	resourceID := fmt.Sprintf("%s-associations", data.PrivateSpaceID.ValueString())
	data.ID = types.StringValue(resourceID)

	// Map created associations to computed attribute
	var createdAssocElements []attr.Value
	for _, assoc := range createdAssociations {
		createdAssocObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"id":              types.StringType,
				"organization_id": types.StringType,
				"environment":     types.StringType,
			},
			map[string]attr.Value{
				"id":              types.StringValue(assoc.ID),
				"organization_id": types.StringValue(assoc.OrganizationID),
				"environment":     types.StringValue(assoc.EnvironmentID),
			},
		)
		createdAssocElements = append(createdAssocElements, createdAssocObj)
	}

	createdAssociationsList, _ := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":              types.StringType,
				"organization_id": types.StringType,
				"environment":     types.StringType,
			},
		},
		createdAssocElements,
	)

	data.CreatedAssociations = createdAssociationsList
	data.OrganizationID = types.StringValue(orgID)

	tflog.Trace(ctx, "created Private Space Associations")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *PrivateSpaceAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PrivateSpaceAssociationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Note: No read API available for private space associations
	// Maintaining existing state as-is

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PrivateSpaceAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PrivateSpaceAssociationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := plan.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Convert associations from Terraform list to request format
	var associations []cloudhub2.AssociationRequest
	for _, elem := range plan.Associations.Elements() {
		assocObj := elem.(types.Object)
		assocAttrs := assocObj.Attributes()

		associations = append(associations, cloudhub2.AssociationRequest{
			OrganizationID: assocAttrs["organization_id"].(types.String).ValueString(),
			Environment:    assocAttrs["environment"].(types.String).ValueString(),
		})
	}

	// Build the request
	createRequest := &cloudhub2.CreatePrivateSpaceAssociationRequest{
		Associations: associations,
	}

	// Update the Private Space Associations by POSTing the full list
	createdAssociations, err := r.client.CreatePrivateSpaceAssociations(ctx, orgID, plan.PrivateSpaceID.ValueString(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Private Space Associations",
			"Could not update Private Space Associations: "+err.Error(),
		)
		return
	}

	// Map created associations to computed attribute
	var createdAssocElements []attr.Value
	for _, assoc := range createdAssociations {
		createdAssocObj, diags := types.ObjectValue(
			getPSAssociationAttrTypes(),
			map[string]attr.Value{
				"id":              types.StringValue(assoc.ID),
				"organization_id": types.StringValue(assoc.OrganizationID),
				"environment":     types.StringValue(assoc.EnvironmentID),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createdAssocElements = append(createdAssocElements, createdAssocObj)
	}

	createdAssociationsList, diags := types.ListValue(types.ObjectType{AttrTypes: getPSAssociationAttrTypes()}, createdAssocElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.CreatedAssociations = createdAssociationsList
	plan.OrganizationID = types.StringValue(orgID)

	tflog.Trace(ctx, "updated Private Space Associations")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *PrivateSpaceAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PrivateSpaceAssociationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine organization ID - use provided value or default to client's org
	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	// Delete each association individually
	var deleteErrors []string
	for _, elem := range data.CreatedAssociations.Elements() {
		assocObj := elem.(types.Object)
		assocAttrs := assocObj.Attributes()

		associationID := assocAttrs["id"].(types.String).ValueString()

		// Delete the association
		err := r.client.DeletePrivateSpaceAssociation(ctx, orgID, data.PrivateSpaceID.ValueString(), associationID)
		if err != nil {
			deleteErrors = append(deleteErrors, fmt.Sprintf("Failed to delete association %s: %s", associationID, err.Error()))
		}
	}

	// If there were any delete errors, report them
	if len(deleteErrors) > 0 {
		resp.Diagnostics.AddError(
			"Error deleting Private Space Associations",
			"Could not delete some Private Space Associations:\n"+strings.Join(deleteErrors, "\n"),
		)
		return
	}

	tflog.Trace(ctx, "deleted Private Space Associations")
}

// ImportState imports the resource into Terraform state.
func (r *PrivateSpaceAssociationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: private_space_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("private_space_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID+"-associations")...)
}

func getPSAssociationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":              types.StringType,
		"organization_id": types.StringType,
		"environment":     types.StringType,
	}
}
