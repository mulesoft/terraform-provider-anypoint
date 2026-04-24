package apimanagement

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

var (
	_ resource.Resource                = &APIGroupResource{}
	_ resource.ResourceWithConfigure   = &APIGroupResource{}
	_ resource.ResourceWithImportState = &APIGroupResource{}
)

// APIGroupResource manages an Anypoint API Group.
type APIGroupResource struct {
	client *apimanagement.APIGroupClient
}

// --- Terraform State Models ---

type APIGroupResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Name           types.String `tfsdk:"name"`
	Versions       types.List   `tfsdk:"versions"`
}

type APIGroupVersionModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Instances types.List   `tfsdk:"instances"`
}

type APIGroupInstanceModel struct {
	EnvironmentID      types.String `tfsdk:"environment_id"`
	GroupInstanceLabel types.String `tfsdk:"group_instance_label"`
	APIInstances       types.List   `tfsdk:"api_instances"`
}

// attr type maps used for List/Object conversions
var apiGroupInstanceAttrTypes = map[string]attr.Type{
	"environment_id":       types.StringType,
	"group_instance_label": types.StringType,
	"api_instances":        types.ListType{ElemType: types.Int64Type},
}

var apiGroupVersionAttrTypes = map[string]attr.Type{
	"id":        types.StringType,
	"name":      types.StringType,
	"instances": types.ListType{ElemType: types.ObjectType{AttrTypes: apiGroupInstanceAttrTypes}},
}

func NewAPIGroupResource() resource.Resource {
	return &APIGroupResource{}
}

func (r *APIGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_group"
}

func (r *APIGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API Group in Anypoint API Manager. " +
			"An API Group bundles multiple API instances across environments under a shared versioned contract.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the API Group (numeric, assigned by the platform).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "Organization ID. Defaults to the provider's org if omitted.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name of the API Group.",
				Required:    true,
			},
			"versions": schema.ListNestedAttribute{
				Description: "List of named versions defined in this API Group.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Version ID assigned by the platform (computed on create).",
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							Description: "Name of the version (e.g. 'v1', 'v2').",
							Required:    true,
						},
						"instances": schema.ListNestedAttribute{
							Description: "API instances associated with this version.",
							Required:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"environment_id": schema.StringAttribute{
										Description: "Environment ID that owns the API instances.",
										Required:    true,
									},
									"group_instance_label": schema.StringAttribute{
										Description: "Optional label for this instance group.",
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString(""),
									},
									"api_instances": schema.ListAttribute{
										Description: "Numeric IDs of the API instances to include.",
										Required:    true,
										ElementType: types.Int64Type,
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

func (r *APIGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cfg, ok := req.ProviderData.(*client.ClientConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientConfig, got: %T.", req.ProviderData),
		)
		return
	}

	c, err := apimanagement.NewAPIGroupClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create API Group Client",
			"An unexpected error occurred when creating the API Group client.\n\n"+
				"Anypoint Client Error: "+err.Error(),
		)
		return
	}
	r.client = c
}

// --- CRUD ---

func (r *APIGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	createReq, diags := r.expand(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.CreateAPIGroup(ctx, orgID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating API group", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, group, &data, orgID)...)
	tflog.Trace(ctx, "created API group", map[string]interface{}{"id": group.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	groupID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API group ID", "Could not parse group ID: "+err.Error())
		return
	}

	group, err := r.client.GetAPIGroup(ctx, orgID, groupID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API group", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, group, &data, orgID)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state APIGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	groupID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API group ID", "Could not parse group ID: "+err.Error())
		return
	}

	updateReq, diags := r.expand(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.UpdateAPIGroup(ctx, orgID, groupID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating API group", err.Error())
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, group, &plan, orgID)...)
	tflog.Trace(ctx, "updated API group", map[string]interface{}{"id": group.ID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := data.OrganizationID.ValueString()
	if orgID == "" {
		orgID = r.client.OrgID
	}

	groupID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid API group ID", "Could not parse group ID: "+err.Error())
		return
	}

	if err := r.client.DeleteAPIGroup(ctx, orgID, groupID); err != nil {
		resp.Diagnostics.AddError("Error deleting API group", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted API group", map[string]interface{}{"id": data.ID.ValueString()})
}

func (r *APIGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: organization_id/group_id  OR  group_id
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected format: group_id  or  organization_id/group_id")
	}
}

// --- Helpers ---

// expand converts the Terraform model into a CreateAPIGroupRequest.
func (r *APIGroupResource) expand(ctx context.Context, data *APIGroupResourceModel) (*apimanagement.CreateAPIGroupRequest, diag.Diagnostics) {
	var allDiags diag.Diagnostics

	var versionModels []APIGroupVersionModel
	allDiags.Append(data.Versions.ElementsAs(ctx, &versionModels, false)...)
	if allDiags.HasError() {
		return nil, allDiags
	}

	versions := make([]apimanagement.APIGroupVersion, len(versionModels))
	for i, v := range versionModels {
		var instanceModels []APIGroupInstanceModel
		allDiags.Append(v.Instances.ElementsAs(ctx, &instanceModels, false)...)
		if allDiags.HasError() {
			return nil, allDiags
		}

		instances := make([]apimanagement.APIGroupInstance, len(instanceModels))
		for j, inst := range instanceModels {
			var apiIDs []int64
			allDiags.Append(inst.APIInstances.ElementsAs(ctx, &apiIDs, false)...)

			intIDs := make([]int, len(apiIDs))
			for k, id := range apiIDs {
				intIDs[k] = int(id)
			}
			instances[j] = apimanagement.APIGroupInstance{
				EnvironmentID:      inst.EnvironmentID.ValueString(),
				GroupInstanceLabel: inst.GroupInstanceLabel.ValueString(),
				APIInstances:       intIDs,
			}
		}
		versions[i] = apimanagement.APIGroupVersion{
			Name:      v.Name.ValueString(),
			Instances: instances,
		}
	}

	return &apimanagement.CreateAPIGroupRequest{
		Name:     data.Name.ValueString(),
		Versions: versions,
	}, allDiags
}

// flatten maps the API response back into the Terraform state model.
// Versions (and instances within each version) are re-ordered to match the
// current state so that a read-with-no-changes never produces a diff.
func (r *APIGroupResource) flatten(ctx context.Context, group *apimanagement.APIGroup, data *APIGroupResourceModel, orgID string) diag.Diagnostics {
	var allDiags diag.Diagnostics

	data.ID = types.StringValue(strconv.Itoa(group.ID))
	data.OrganizationID = types.StringValue(orgID)
	data.Name = types.StringValue(group.Name)

	// Index API versions by their ID so we can look them up quickly.
	apiVersionByID := make(map[int]*apimanagement.APIGroupVersion, len(group.Versions))
	for i := range group.Versions {
		apiVersionByID[group.Versions[i].ID] = &group.Versions[i]
	}

	// Recover the prior state's version list for ordering.
	var stateVersions []APIGroupVersionModel
	if !data.Versions.IsNull() && !data.Versions.IsUnknown() {
		allDiags.Append(data.Versions.ElementsAs(ctx, &stateVersions, false)...)
	}

	// Build ordered slice: state order first, then any new versions.
	orderedVersions := make([]*apimanagement.APIGroupVersion, 0, len(group.Versions))
	seen := make(map[int]bool)
	for _, sv := range stateVersions {
		id, _ := strconv.Atoi(sv.ID.ValueString())
		if v, ok := apiVersionByID[id]; ok {
			orderedVersions = append(orderedVersions, v)
			seen[id] = true
		}
	}
	for i := range group.Versions {
		if !seen[group.Versions[i].ID] {
			orderedVersions = append(orderedVersions, &group.Versions[i])
		}
	}

	// Convert ordered versions to Terraform objects.
	versionObjs := make([]APIGroupVersionModel, len(orderedVersions))
	for i, v := range orderedVersions {
		// Find the corresponding state version's instance order.
		var stateInstances []APIGroupInstanceModel
		for _, sv := range stateVersions {
			svID, _ := strconv.Atoi(sv.ID.ValueString())
			if svID == v.ID && !sv.Instances.IsNull() && !sv.Instances.IsUnknown() {
				allDiags.Append(sv.Instances.ElementsAs(ctx, &stateInstances, false)...)
				break
			}
		}

		instanceObjs := flattenInstancesOrdered(ctx, v.Instances, stateInstances, &allDiags)
		instList, diags := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: apiGroupInstanceAttrTypes}, instanceObjs)
		allDiags.Append(diags...)

		versionObjs[i] = APIGroupVersionModel{
			ID:        types.StringValue(strconv.Itoa(v.ID)),
			Name:      types.StringValue(v.Name),
			Instances: instList,
		}
	}

	versionList, diags := types.ListValueFrom(ctx,
		types.ObjectType{AttrTypes: apiGroupVersionAttrTypes}, versionObjs)
	allDiags.Append(diags...)
	data.Versions = versionList

	return allDiags
}

// flattenInstancesOrdered converts API instances to Terraform models, re-ordering
// them to match the prior state order (matched by environment_id + label key).
func flattenInstancesOrdered(
	ctx context.Context,
	apiInstances []apimanagement.APIGroupInstance,
	stateInstances []APIGroupInstanceModel,
	allDiags *diag.Diagnostics,
) []APIGroupInstanceModel {
	// Index API instances by (environment_id, group_instance_label).
	type instKey struct{ env, label string }
	apiByKey := make(map[instKey]*apimanagement.APIGroupInstance, len(apiInstances))
	for i := range apiInstances {
		k := instKey{apiInstances[i].EnvironmentID, apiInstances[i].GroupInstanceLabel}
		apiByKey[k] = &apiInstances[i]
	}

	ordered := make([]*apimanagement.APIGroupInstance, 0, len(apiInstances))
	seen := make(map[instKey]bool)
	for _, si := range stateInstances {
		k := instKey{si.EnvironmentID.ValueString(), si.GroupInstanceLabel.ValueString()}
		if inst, ok := apiByKey[k]; ok {
			ordered = append(ordered, inst)
			seen[k] = true
		}
	}
	for i := range apiInstances {
		k := instKey{apiInstances[i].EnvironmentID, apiInstances[i].GroupInstanceLabel}
		if !seen[k] {
			ordered = append(ordered, &apiInstances[i])
		}
	}

	result := make([]APIGroupInstanceModel, len(ordered))
	for i, inst := range ordered {
		apiIDs := make([]int64, len(inst.APIInstances))
		for k, id := range inst.APIInstances {
			apiIDs[k] = int64(id)
		}

		// Preserve the plan/state order of api_instances to avoid spurious diffs.
		// If the set of IDs matches, keep the prior order; otherwise use API order.
		if i < len(stateInstances) {
			var priorIDs []int64
			stateInstances[i].APIInstances.ElementsAs(ctx, &priorIDs, false)
			if sameElements(priorIDs, apiIDs) {
				apiIDs = priorIDs
			}
		}

		apiIDList, diags := types.ListValueFrom(ctx, types.Int64Type, apiIDs)
		allDiags.Append(diags...)
		result[i] = APIGroupInstanceModel{
			EnvironmentID:      types.StringValue(inst.EnvironmentID),
			GroupInstanceLabel: types.StringValue(inst.GroupInstanceLabel),
			APIInstances:       apiIDList,
		}
	}
	return result
}

func sameElements(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[int64]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

// ensure basetypes is imported (used implicitly via ElementsAs options).
var _ = basetypes.ObjectAsOptions{}
