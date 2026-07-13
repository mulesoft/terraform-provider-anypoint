package exchange

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
)

var (
	_ resource.Resource                = &AssetResource{}
	_ resource.ResourceWithConfigure   = &AssetResource{}
	_ resource.ResourceWithImportState = &AssetResource{}
)

// AssetResource implements the anypoint_exchange_asset resource.
type AssetResource struct {
	client *exchange.AssetClient
}

// AssetResourceModel describes the resource data model.
type AssetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	GroupID        types.String `tfsdk:"group_id"`
	AssetID        types.String `tfsdk:"asset_id"`
	Version        types.String `tfsdk:"version"`

	// Mutable metadata
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	ContactName  types.String `tfsdk:"contact_name"`
	ContactEmail types.String `tfsdk:"contact_email"`
	Manager      types.String `tfsdk:"manager"`
	Tags         types.List   `tfsdk:"tags"`

	// Documentation pages
	Pages types.List `tfsdk:"pages"`

	// Terms and Conditions
	TermsAndConditions types.String `tfsdk:"terms_and_conditions"`

	// Non-managed instances
	Instances types.List `tfsdk:"instances"`

	// Categories (org-level taxonomy assigned to this asset)
	Categories types.List `tfsdk:"categories"`

	// Custom fields (org-level metadata assigned to this asset)
	CustomFields types.List `tfsdk:"custom_fields"`

	// Immutable after create
	Type       types.String `tfsdk:"type"`
	Status     types.String `tfsdk:"status"`
	FilePath   types.String `tfsdk:"file_path"`
	Classifier types.String `tfsdk:"classifier"`
	Keywords   types.String `tfsdk:"keywords"`

	// Properties for API spec types
	APIVersion types.String `tfsdk:"api_version"`
	MainFile   types.String `tfsdk:"main_file"`

	// Computed fields
	FileSHA256   types.String `tfsdk:"file_sha256"`
	IsPublic     types.Bool   `tfsdk:"is_public"`
	IsSnapshot   types.Bool   `tfsdk:"is_snapshot"`
	MinorVersion types.String `tfsdk:"minor_version"`
	VersionGroup types.String `tfsdk:"version_group"`
	CreatedDate  types.String `tfsdk:"created_date"`
	UpdatedDate  types.String `tfsdk:"updated_date"`
}

// PageModel describes a documentation page within the asset.
type PageModel struct {
	PageName types.String `tfsdk:"page_name"`
	Content  types.String `tfsdk:"content"`
	PagePath types.String `tfsdk:"page_path"`
}

// InstanceModel describes a non-managed (external) API instance.
type InstanceModel struct {
	Name        types.String `tfsdk:"name"`
	EndpointURI types.String `tfsdk:"endpoint_uri"`
	IsPublic    types.Bool   `tfsdk:"is_public"`
	InstanceID  types.String `tfsdk:"instance_id"`
}

// CategoryModel describes a category assignment on an asset.
// Categories are org-level taxonomy — the key must already exist in the org.
type CategoryModel struct {
	Key    types.String `tfsdk:"key"`
	Values types.List   `tfsdk:"values"` // List of string values
}

// CustomFieldModel describes a custom field assignment on an asset.
// Custom fields are org-level metadata — the key must already exist in the org.
type CustomFieldModel struct {
	Key    types.String `tfsdk:"key"`
	Values types.List   `tfsdk:"values"` // List of string values
}

func NewAssetResource() resource.Resource {
	return &AssetResource{}
}

func (r *AssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exchange_asset"
}

func (r *AssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Exchange asset (publish, update metadata, delete). " +
			"Asset versions are immutable — changing the version, type, or file triggers a replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The composite identifier (groupId/assetId/version).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID to publish the asset to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Description: "The group ID of the asset (usually the same as organization_id).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"asset_id": schema.StringAttribute{
				Description: "The asset ID (slug/identifier). Must be unique within the group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The semantic version of the asset (e.g. '1.0.0'). Asset versions are immutable.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The display name of the asset.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the asset.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The asset type: custom, rest-api, http-api, evented-api (AsyncAPI), graphql-api, connector, app, template, example, policy, agent, llm, mcp.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The lifecycle status: published (default) or development.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"file_path": schema.StringAttribute{
				Description: "Path to the file to upload (JAR, ZIP, RAML, OAS, etc.). Changes trigger replacement.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"classifier": schema.StringAttribute{
				Description: "The file classifier: custom, raml, oas, wsdl, graphql, etc. Required when file_path is set.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keywords": schema.StringAttribute{
				Description: "Comma-separated keywords for search discovery.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_version": schema.StringAttribute{
				Description: "The API version (properties.apiVersion). Used for API spec asset types.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"main_file": schema.StringAttribute{
				Description: "The main file within the uploaded archive (properties.mainFile). Used for multi-file specs.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"contact_name": schema.StringAttribute{
				Description: "Contact person name for this asset.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contact_email": schema.StringAttribute{
				Description: "Contact email for this asset.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"manager": schema.StringAttribute{
				Description: "Manager for this asset.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				Description: "Search tags for the asset version. Each element is a tag value string.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			// Computed fields
			"file_sha256": schema.StringAttribute{
				Description: "SHA256 hash of the uploaded file (for drift detection).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_public": schema.BoolAttribute{
				Description: "Whether the asset is publicly visible.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_snapshot": schema.BoolAttribute{
				Description: "Whether this is a snapshot version.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"minor_version": schema.StringAttribute{
				Description: "The minor version (e.g. '1.0').",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_group": schema.StringAttribute{
				Description: "The version group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_date": schema.StringAttribute{
				Description: "When the asset was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_date": schema.StringAttribute{
				Description: "When the asset was last updated.",
				Computed:    true,
			},
			"pages": schema.ListNestedAttribute{
				Description: "Documentation pages for the asset portal. Each page has a name and markdown content.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"page_name": schema.StringAttribute{
							Description: "The page name (used as URL slug). Cannot contain: % @ * + / _ \\",
							Required:    true,
						},
						"content": schema.StringAttribute{
							Description: "The markdown content of the page.",
							Required:    true,
						},
						"page_path": schema.StringAttribute{
							Description: "The full page path assigned by the API (includes random prefix). Computed after creation.",
							Computed:    true,
						},
					},
				},
			},
			"terms_and_conditions": schema.StringAttribute{
				Description: "Terms and conditions content (markdown). Displayed as the T&C page in the asset portal.",
				Optional:    true,
				Computed:    true,
			},
			"instances": schema.ListNestedAttribute{
				Description: "Non-managed (external) API instances for this asset version.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The display name of the instance (e.g. 'Production', 'Sandbox').",
							Required:    true,
						},
						"endpoint_uri": schema.StringAttribute{
							Description: "The endpoint URL of the external instance.",
							Required:    true,
						},
						"is_public": schema.BoolAttribute{
							Description: "Whether this instance is publicly visible. Defaults to false.",
							Optional:    true,
							Computed:    true,
						},
						"instance_id": schema.StringAttribute{
							Description: "The unique ID assigned by the API. Computed after creation.",
							Computed:    true,
						},
					},
				},
			},
			"categories": schema.ListNestedAttribute{
				Description: "Category assignments on this asset version. Categories are org-level taxonomy " +
					"(the category key must already exist in the org via the Exchange UI or API). " +
					"Each entry assigns one or more values to a category key.",
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The category key (must match an existing org-level category definition).",
							Required:    true,
						},
						"values": schema.ListAttribute{
							Description: "The category values to assign (e.g. [\"Finance\", \"HR\"]).",
							Required:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"custom_fields": schema.ListNestedAttribute{
				Description: "Custom field assignments on this asset version. Custom fields are org-level metadata " +
					"(the field key must already exist in the org via the Exchange UI or API). " +
					"Each entry assigns one or more values to a custom field key.",
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The custom field key (must match an existing org-level field definition).",
							Required:    true,
						},
						"values": schema.ListAttribute{
							Description: "The field values to assign (e.g. [\"v1.2\", \"stable\"]).",
							Required:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *AssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*client.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Config, got: %T.", req.ProviderData),
		)
		return
	}

	// Use client_credentials (works with any connected app that has Exchange scope).
	// Falls back to password grant only if username/password are explicitly provided
	// and client_credentials fails.
	assetClient, err := exchange.NewAssetClient(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Exchange Asset Client",
			"Client Error: "+err.Error(),
		)
		return
	}

	r.client = assetClient
}

func (r *AssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating exchange asset", map[string]interface{}{
		"asset_id": plan.AssetID.ValueString(),
		"version":  plan.Version.ValueString(),
	})

	createReq := &exchange.CreateAssetRequest{
		OrganizationID: plan.OrganizationID.ValueString(),
		GroupID:        plan.GroupID.ValueString(),
		AssetID:        plan.AssetID.ValueString(),
		Version:        plan.Version.ValueString(),
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		Type:           plan.Type.ValueString(),
		Status:         plan.Status.ValueString(),
		FilePath:       plan.FilePath.ValueString(),
		Classifier:     plan.Classifier.ValueString(),
		Keywords:       plan.Keywords.ValueString(),
		APIVersion:     plan.APIVersion.ValueString(),
		MainFile:       plan.MainFile.ValueString(),
	}

	asset, err := r.client.CreateAsset(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating exchange asset",
			"Could not publish asset: "+err.Error(),
		)
		return
	}

	// After create, PATCH mutable metadata fields to ensure they match the plan.
	// The multipart create may not update asset-level metadata (name, description)
	// when publishing a new version of an existing asset. Contact fields are PATCH-only.
	needsPatch := false
	updateReq := &exchange.UpdateAssetRequest{}

	// Always ensure name matches plan (asset-level name may not be set by multipart for new versions)
	if plan.Name.ValueString() != asset.Name {
		name := plan.Name.ValueString()
		updateReq.Name = &name
		needsPatch = true
	}
	// Always ensure description matches plan
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() != asset.Description {
		desc := plan.Description.ValueString()
		updateReq.Description = &desc
		needsPatch = true
	}
	if !plan.ContactName.IsNull() && !plan.ContactName.IsUnknown() && plan.ContactName.ValueString() != "" {
		cn := plan.ContactName.ValueString()
		updateReq.ContactName = &cn
		needsPatch = true
	}
	if !plan.ContactEmail.IsNull() && !plan.ContactEmail.IsUnknown() && plan.ContactEmail.ValueString() != "" {
		ce := plan.ContactEmail.ValueString()
		updateReq.ContactEmail = &ce
		needsPatch = true
	}
	if !plan.Manager.IsNull() && !plan.Manager.IsUnknown() && plan.Manager.ValueString() != "" {
		mgr := plan.Manager.ValueString()
		updateReq.Manager = &mgr
		needsPatch = true
	}

	if needsPatch {
		patchErr := r.client.UpdateAsset(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), updateReq)
		if patchErr != nil {
			resp.Diagnostics.AddError(
				"Error setting asset metadata after create",
				"Asset was created but metadata update failed: "+patchErr.Error(),
			)
			return
		}
		// Re-read after patch to get updated values
		asset, err = r.client.GetAsset(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading asset after metadata update",
				"Could not read asset: "+err.Error(),
			)
			return
		}
	}

	// Set tags if specified in the plan
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var planTags []string
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &planTags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(planTags) > 0 {
			tagRequests := make([]exchange.TagRequest, len(planTags))
			for i, tag := range planTags {
				tagRequests[i] = exchange.TagRequest{Value: tag}
			}
			tagErr := r.client.UpdateTags(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), tagRequests)
			if tagErr != nil {
				resp.Diagnostics.AddError(
					"Error setting asset tags",
					"Asset was created but tag update failed: "+tagErr.Error(),
				)
				return
			}
			// Re-read to get tags in response
			asset, err = r.client.GetAsset(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error reading asset after tag update",
					"Could not read asset: "+err.Error(),
				)
				return
			}
		}
	}

	// Create documentation pages if specified in the plan
	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() {
		var planPages []PageModel
		resp.Diagnostics.Append(plan.Pages.ElementsAs(ctx, &planPages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(planPages) > 0 {
			err := r.syncPages(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), nil, planPages)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating documentation pages",
					"Asset was created but page creation failed: "+err.Error(),
				)
				return
			}
		}
	}

	// Set Terms & Conditions if specified
	if !plan.TermsAndConditions.IsNull() && !plan.TermsAndConditions.IsUnknown() && plan.TermsAndConditions.ValueString() != "" {
		err := r.syncTermsAndConditions(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), "", plan.TermsAndConditions.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error setting terms and conditions",
				"Asset was created but T&C update failed: "+err.Error(),
			)
			return
		}
	}

	// Create non-managed instances if specified
	if !plan.Instances.IsNull() && !plan.Instances.IsUnknown() {
		var planInstances []InstanceModel
		resp.Diagnostics.Append(plan.Instances.ElementsAs(ctx, &planInstances, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(planInstances) > 0 {
			err := r.syncInstances(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), asset.VersionGroup, nil, planInstances)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating external instances",
					"Asset was created but instance creation failed: "+err.Error(),
				)
				return
			}
		}
	}

	// Set categories if specified in the plan
	if !plan.Categories.IsNull() && !plan.Categories.IsUnknown() {
		var planCategories []CategoryModel
		resp.Diagnostics.Append(plan.Categories.ElementsAs(ctx, &planCategories, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(planCategories) > 0 {
			err := r.syncCategories(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), nil, planCategories)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error setting categories",
					"Asset was created but category assignment failed: "+err.Error(),
				)
				return
			}
		}
	}

	// Set custom fields if specified in the plan
	if !plan.CustomFields.IsNull() && !plan.CustomFields.IsUnknown() {
		var planFields []CustomFieldModel
		resp.Diagnostics.Append(plan.CustomFields.ElementsAs(ctx, &planFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(planFields) > 0 {
			err := r.syncCustomFields(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), nil, planFields)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error setting custom fields",
					"Asset was created but custom field assignment failed: "+err.Error(),
				)
				return
			}
		}
	}

	// Compute file hash for drift detection
	if plan.FilePath.ValueString() != "" {
		hash, hashErr := computeFileHash(plan.FilePath.ValueString())
		if hashErr == nil {
			plan.FileSHA256 = types.StringValue(hash)
		}
	} else {
		plan.FileSHA256 = types.StringValue("")
	}

	// Map response to state
	r.mapAssetToState(&plan, asset)

	// Read pages and T&C from published portal and set in state
	r.readPagesIntoState(ctx, &plan)
	r.readTermsIntoState(ctx, &plan)
	r.readInstancesIntoState(ctx, &plan)
	r.readCategoriesIntoState(ctx, &plan)
	r.readCustomFieldsIntoState(ctx, &plan)

	tflog.Trace(ctx, "created exchange asset")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AssetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	asset, err := r.client.GetAsset(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
	)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Exchange asset not found, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading exchange asset",
			"Could not read asset: "+err.Error(),
		)
		return
	}

	// Preserve file_path and file_sha256 from state (truly local-only, not in API)
	filePath := state.FilePath
	fileSHA256 := state.FileSHA256

	r.mapAssetToState(&state, asset)

	// Restore truly local-only fields
	state.FilePath = filePath
	state.FileSHA256 = fileSHA256

	// Extract api_version from attributes (available in API response for some types).
	// Some asset types (e.g. GraphQL) don't expose api-version in attributes,
	// so we preserve from prior state when not found (it's create-only / RequiresReplace).
	if apiVer := extractAttributeValue(asset.Attributes, "api-version"); apiVer != "" {
		state.APIVersion = types.StringValue(apiVer)
	}
	// else: preserve whatever was already in state (could be user-set or null)

	// Extract classifier and mainFile from the files array (user-uploaded file)
	classifier, mainFile := extractFileMetadata(asset.Files)
	if classifier != "" {
		state.Classifier = types.StringValue(normalizeClassifier(classifier, state.Classifier))
	}
	// else: preserve from state (some asset types may not have user-uploaded files)
	if mainFile != "" {
		state.MainFile = types.StringValue(mainFile)
	}
	// else: preserve from state

	// Keywords: stored as null-key attributes but mixed with system-generated values.
	// We cannot reliably distinguish user keywords from system ones on import,
	// so we preserve from prior state if available.
	if state.Keywords.IsNull() || state.Keywords.ValueString() == "" {
		state.Keywords = types.StringNull()
	}

	// Read pages, T&C, instances, categories, custom fields from published portal
	r.readPagesIntoState(ctx, &state)
	r.readTermsIntoState(ctx, &state)
	r.readInstancesIntoState(ctx, &state)
	r.readCategoriesIntoState(ctx, &state)
	r.readCustomFieldsIntoState(ctx, &state)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only mutable fields can be updated: name, description, contactName, contactEmail, manager
	// We only send fields that actually changed AND have a meaningful value in the plan.
	updateReq := &exchange.UpdateAssetRequest{}
	hasChanges := false

	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		updateReq.Name = &name
		hasChanges = true
	}
	if !plan.Description.Equal(state.Description) && !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		updateReq.Description = &desc
		hasChanges = true
	}
	if !plan.ContactName.Equal(state.ContactName) && !plan.ContactName.IsNull() && !plan.ContactName.IsUnknown() {
		cn := plan.ContactName.ValueString()
		updateReq.ContactName = &cn
		hasChanges = true
	}
	if !plan.ContactEmail.Equal(state.ContactEmail) && !plan.ContactEmail.IsNull() && !plan.ContactEmail.IsUnknown() {
		ce := plan.ContactEmail.ValueString()
		updateReq.ContactEmail = &ce
		hasChanges = true
	}
	if !plan.Manager.Equal(state.Manager) && !plan.Manager.IsNull() && !plan.Manager.IsUnknown() {
		mgr := plan.Manager.ValueString()
		if mgr != "" { // API rejects empty string for manager
			updateReq.Manager = &mgr
			hasChanges = true
		}
	}

	if hasChanges {
		err := r.client.UpdateAsset(ctx,
			plan.GroupID.ValueString(),
			plan.AssetID.ValueString(),
			updateReq,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating exchange asset",
				"Could not update asset metadata: "+err.Error(),
			)
			return
		}
	}

	// Handle status transitions (development → published → deprecated)
	// Status changes require a separate PUT endpoint
	if !plan.Status.Equal(state.Status) && !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		newStatus := plan.Status.ValueString()
		err := r.client.UpdateStatus(ctx,
			plan.GroupID.ValueString(),
			plan.AssetID.ValueString(),
			plan.Version.ValueString(),
			newStatus,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating exchange asset status",
				fmt.Sprintf("Could not change status to '%s': %s", newStatus, err.Error()),
			)
			return
		}
	}

	// Handle tags changes
	if !plan.Tags.Equal(state.Tags) && !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var planTags []string
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &planTags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		tagRequests := make([]exchange.TagRequest, len(planTags))
		for i, tag := range planTags {
			tagRequests[i] = exchange.TagRequest{Value: tag}
		}

		err := r.client.UpdateTags(ctx,
			plan.GroupID.ValueString(),
			plan.AssetID.ValueString(),
			plan.Version.ValueString(),
			tagRequests,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating exchange asset tags",
				"Could not update tags: "+err.Error(),
			)
			return
		}
	}

	// Handle documentation pages changes
	if !plan.Pages.Equal(state.Pages) && !plan.Pages.IsNull() && !plan.Pages.IsUnknown() {
		var planPages []PageModel
		resp.Diagnostics.Append(plan.Pages.ElementsAs(ctx, &planPages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var statePages []PageModel
		if !state.Pages.IsNull() && !state.Pages.IsUnknown() {
			resp.Diagnostics.Append(state.Pages.ElementsAs(ctx, &statePages, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		err := r.syncPages(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), statePages, planPages)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating documentation pages",
				"Could not sync pages: "+err.Error(),
			)
			return
		}
	}

	// Handle Terms & Conditions changes
	if !plan.TermsAndConditions.Equal(state.TermsAndConditions) && !plan.TermsAndConditions.IsNull() && !plan.TermsAndConditions.IsUnknown() {
		currentTC := ""
		if !state.TermsAndConditions.IsNull() && !state.TermsAndConditions.IsUnknown() {
			currentTC = state.TermsAndConditions.ValueString()
		}
		err := r.syncTermsAndConditions(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), currentTC, plan.TermsAndConditions.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating terms and conditions",
				"Could not sync T&C: "+err.Error(),
			)
			return
		}
	}

	// Handle instances changes
	if !plan.Instances.Equal(state.Instances) && !plan.Instances.IsNull() && !plan.Instances.IsUnknown() {
		var planInstances []InstanceModel
		resp.Diagnostics.Append(plan.Instances.ElementsAs(ctx, &planInstances, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var stateInstances []InstanceModel
		if !state.Instances.IsNull() && !state.Instances.IsUnknown() {
			resp.Diagnostics.Append(state.Instances.ElementsAs(ctx, &stateInstances, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		// Get versionGroup from API
		currentAsset, getErr := r.client.GetAsset(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString())
		if getErr != nil {
			resp.Diagnostics.AddError(
				"Error reading asset for instance sync",
				"Could not read asset: "+getErr.Error(),
			)
			return
		}

		err := r.syncInstances(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), currentAsset.VersionGroup, stateInstances, planInstances)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating external instances",
				"Could not sync instances: "+err.Error(),
			)
			return
		}
	}

	// Handle categories changes
	if !plan.Categories.Equal(state.Categories) && !plan.Categories.IsNull() && !plan.Categories.IsUnknown() {
		var planCategories []CategoryModel
		resp.Diagnostics.Append(plan.Categories.ElementsAs(ctx, &planCategories, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var stateCategories []CategoryModel
		if !state.Categories.IsNull() && !state.Categories.IsUnknown() {
			resp.Diagnostics.Append(state.Categories.ElementsAs(ctx, &stateCategories, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		err := r.syncCategories(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), stateCategories, planCategories)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating categories",
				"Could not sync categories: "+err.Error(),
			)
			return
		}
	}

	// Handle custom fields changes
	if !plan.CustomFields.Equal(state.CustomFields) && !plan.CustomFields.IsNull() && !plan.CustomFields.IsUnknown() {
		var planFields []CustomFieldModel
		resp.Diagnostics.Append(plan.CustomFields.ElementsAs(ctx, &planFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var stateFields []CustomFieldModel
		if !state.CustomFields.IsNull() && !state.CustomFields.IsUnknown() {
			resp.Diagnostics.Append(state.CustomFields.ElementsAs(ctx, &stateFields, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		err := r.syncCustomFields(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString(), stateFields, planFields)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating custom fields",
				"Could not sync custom fields: "+err.Error(),
			)
			return
		}
	}

	// Read back the updated asset
	asset, err := r.client.GetAsset(ctx,
		plan.GroupID.ValueString(),
		plan.AssetID.ValueString(),
		plan.Version.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading exchange asset after update",
			"Could not read asset: "+err.Error(),
		)
		return
	}

	// Preserve truly local-only fields
	filePath := state.FilePath
	fileSHA256 := state.FileSHA256

	r.mapAssetToState(&plan, asset)

	plan.FilePath = filePath
	plan.FileSHA256 = fileSHA256

	// Extract api_version from attributes (some types like GraphQL don't expose it)
	if apiVer := extractAttributeValue(asset.Attributes, "api-version"); apiVer != "" {
		plan.APIVersion = types.StringValue(apiVer)
	}
	// else: preserve plan value (create-only field, already set from plan)

	// Extract classifier and mainFile from files array
	classifierVal, mainFileVal := extractFileMetadata(asset.Files)
	if classifierVal != "" {
		plan.Classifier = types.StringValue(normalizeClassifier(classifierVal, state.Classifier))
	}
	// else: preserve plan value
	if mainFileVal != "" {
		plan.MainFile = types.StringValue(mainFileVal)
	}
	// else: preserve plan value

	// Keywords: preserve from plan (user's config value)
	if plan.Keywords.IsNull() || plan.Keywords.ValueString() == "" {
		plan.Keywords = types.StringNull()
	}

	// Read pages, T&C, instances, categories, custom fields
	r.readPagesIntoState(ctx, &plan)
	r.readTermsIntoState(ctx, &plan)
	r.readInstancesIntoState(ctx, &plan)
	r.readCategoriesIntoState(ctx, &plan)
	r.readCustomFieldsIntoState(ctx, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AssetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AssetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting exchange asset", map[string]interface{}{
		"asset_id": state.AssetID.ValueString(),
		"version":  state.Version.ValueString(),
	})

	// Delete the specific version (not all versions of the asset)
	err := r.client.DeleteAssetVersion(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
		true, // hard delete
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting exchange asset",
			"Could not delete asset: "+err.Error(),
		)
		return
	}
}

// ImportState supports importing an existing exchange asset.
// Import ID format: "groupId/assetId/version"
func (r *AssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import ID format: groupId/assetId/version (e.g. org-id/my-api/1.0.0)",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("asset_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	// organization_id must be set by user after import (or we can try to infer from group_id)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
}

// --- Helpers ---

// apiTypeToUserType maps API response type names back to the user-facing type names
// used in the create request. The Exchange API normalizes some types differently in responses.
var apiTypeToUserType = map[string]string{
	"graphql": "graphql-api",
}

// userTypeToAPIType maps user-facing type names to what the API stores/returns.
// Used during import to recognize API response types.
var userTypeToAPIType = map[string]string{
	"graphql-api": "graphql",
}

// normalizeType resolves the API response type to the canonical user-facing type.
// If the state already has a type set (from user config), preserve it to avoid
// "inconsistent result after apply" errors when the API normalizes type names.
func normalizeType(apiType string, currentStateType types.String) string {
	// If the user already set a type and it maps to this API type, preserve theirs
	if !currentStateType.IsNull() && !currentStateType.IsUnknown() {
		userType := currentStateType.ValueString()
		if mapped, ok := userTypeToAPIType[userType]; ok && mapped == apiType {
			return userType
		}
		// If the user type exactly matches the API response, use it directly
		if userType == apiType {
			return userType
		}
	}
	// Otherwise (e.g. import), translate API type back to user-facing name if possible
	if userType, ok := apiTypeToUserType[apiType]; ok {
		return userType
	}
	return apiType
}

// apiClassifierToUserClassifier maps API-returned classifiers back to user-facing names.
// The Exchange API transforms uploaded classifiers during processing (e.g., bundles RAML/OAS).
var apiClassifierToUserClassifier = map[string]string{
	"fat-raml": "raml",
	"fat-oas":  "oas",
}

// userClassifierToAPIClassifier maps user-facing classifier names to what the API stores.
var userClassifierToAPIClassifier = map[string]string{
	"raml": "fat-raml",
	"oas":  "fat-oas",
}

// normalizeClassifier resolves the API response classifier to the user-facing value.
// If the state already has a classifier set (from user config), preserve it to avoid
// drift when the API transforms classifiers during processing.
func normalizeClassifier(apiClassifier string, currentStateClassifier types.String) string {
	// If the user already set a classifier and it maps to this API classifier, preserve theirs
	if !currentStateClassifier.IsNull() && !currentStateClassifier.IsUnknown() {
		userClassifier := currentStateClassifier.ValueString()
		if mapped, ok := userClassifierToAPIClassifier[userClassifier]; ok && mapped == apiClassifier {
			return userClassifier
		}
		// If exact match, use directly
		if userClassifier == apiClassifier {
			return userClassifier
		}
	}
	// Otherwise (e.g. import), translate API classifier back to user-facing name if possible
	if userClassifier, ok := apiClassifierToUserClassifier[apiClassifier]; ok {
		return userClassifier
	}
	return apiClassifier
}

// extractAttributeValue extracts a value from the asset's attributes array by key.
// The attributes array contains objects like {"key": "api-version", "value": "v1"}.
func extractAttributeValue(attributes []interface{}, key string) string {
	for _, attr := range attributes {
		if attrMap, ok := attr.(map[string]interface{}); ok {
			if attrKey, _ := attrMap["key"].(string); attrKey == key {
				if val, ok := attrMap["value"].(string); ok {
					return val
				}
			}
		}
	}
	return ""
}

// extractFileMetadata extracts the classifier and mainFile from the asset's files array.
// It finds the user-uploaded file (non-generated, with a non-null classifier) and returns
// its classifier and mainFile values.
func extractFileMetadata(files []exchange.AssetFile) (classifier string, mainFile string) {
	for _, f := range files {
		// Skip auto-generated files (like pom, fat-oas, etc.)
		if f.IsGenerated {
			continue
		}
		// Skip the pom file (always auto-created, classifier is empty)
		if f.Packaging == "pom" && f.Classifier == "" {
			continue
		}
		// This is the user-uploaded file
		if f.Classifier != "" {
			classifier = f.Classifier
			mainFile = f.MainFile
			return
		}
	}
	return "", ""
}

// mapAssetToState maps an API Asset response to the Terraform state model.
func (r *AssetResource) mapAssetToState(state *AssetResourceModel, asset *exchange.Asset) {
	state.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", asset.GroupID, asset.AssetID, asset.Version))
	state.GroupID = types.StringValue(asset.GroupID)
	state.AssetID = types.StringValue(asset.AssetID)
	state.Version = types.StringValue(asset.Version)
	state.Name = types.StringValue(asset.Name)
	state.Description = types.StringValue(asset.Description)
	state.Type = types.StringValue(normalizeType(asset.Type, state.Type))
	state.Status = types.StringValue(asset.Status)
	state.IsPublic = types.BoolValue(asset.IsPublic)
	state.IsSnapshot = types.BoolValue(asset.IsSnapshot)
	state.MinorVersion = types.StringValue(asset.MinorVersion)
	state.VersionGroup = types.StringValue(asset.VersionGroup)
	state.CreatedDate = types.StringValue(asset.CreatedDate)
	state.UpdatedDate = types.StringValue(asset.UpdatedDate)

	if asset.ContactName != nil && *asset.ContactName != "" {
		state.ContactName = types.StringValue(*asset.ContactName)
	} else if !state.ContactName.IsNull() && state.ContactName.ValueString() != "" {
		// Preserve user-set value that may not be reflected in API
		state.ContactName = types.StringValue("")
	} else {
		state.ContactName = types.StringNull()
	}
	if asset.ContactEmail != nil && *asset.ContactEmail != "" {
		state.ContactEmail = types.StringValue(*asset.ContactEmail)
	} else if !state.ContactEmail.IsNull() && state.ContactEmail.ValueString() != "" {
		state.ContactEmail = types.StringValue("")
	} else {
		state.ContactEmail = types.StringNull()
	}
	if asset.Manager != nil && *asset.Manager != "" {
		state.Manager = types.StringValue(*asset.Manager)
	} else if !state.Manager.IsNull() && state.Manager.ValueString() != "" {
		state.Manager = types.StringValue("")
	} else {
		state.Manager = types.StringNull()
	}

	// Map tags (labels) from API response — labels are simple strings
	if len(asset.Labels) > 0 {
		tagValues := make([]attr.Value, len(asset.Labels))
		for i, label := range asset.Labels {
			tagValues[i] = types.StringValue(label)
		}
		state.Tags, _ = types.ListValue(types.StringType, tagValues)
	} else {
		state.Tags = types.ListValueMust(types.StringType, []attr.Value{})
	}
}

// pageObjectType returns the object type for a page nested attribute.
func pageObjectType() basetypes.ObjectTypable {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"page_name": types.StringType,
			"content":   types.StringType,
			"page_path": types.StringType,
		},
	}
}

// readPagesIntoState reads published portal pages and sets them in the state model.
// Only reads pages that are NOT synthetic (system-generated like "home").
func (r *AssetResource) readPagesIntoState(ctx context.Context, state *AssetResourceModel) {
	pages, err := r.client.ListPortalPages(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
	)
	if err != nil {
		// Non-fatal — just leave pages empty if portal isn't available
		tflog.Warn(ctx, "Could not read portal pages", map[string]interface{}{"error": err.Error()})
		state.Pages = types.ListValueMust(pageObjectType(), []attr.Value{})
		return
	}

	// Filter out synthetic pages (like "home") and special pages (like ".terms")
	var userPages []exchange.PortalPage
	for _, p := range pages {
		if !p.Synthetic && p.Name != ".terms" && !strings.HasSuffix(p.Path, "/.terms") {
			userPages = append(userPages, p)
		}
	}

	if len(userPages) == 0 {
		state.Pages = types.ListValueMust(pageObjectType(), []attr.Value{})
		return
	}

	pageValues := make([]attr.Value, 0, len(userPages))
	for _, p := range userPages {
		// Read page content
		content, contentErr := r.client.GetPortalPageContent(ctx,
			state.GroupID.ValueString(),
			state.AssetID.ValueString(),
			state.Version.ValueString(),
			p.Path,
		)
		if contentErr != nil {
			tflog.Warn(ctx, "Could not read page content", map[string]interface{}{
				"page": p.Name, "error": contentErr.Error(),
			})
			content = ""
		}

		pageObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"page_name": types.StringType,
				"content":   types.StringType,
				"page_path": types.StringType,
			},
			map[string]attr.Value{
				"page_name": types.StringValue(p.Name),
				"content":   types.StringValue(content),
				"page_path": types.StringValue(p.Path),
			},
		)
		pageValues = append(pageValues, pageObj)
	}

	state.Pages, _ = types.ListValue(pageObjectType(), pageValues)
}

// syncPages synchronizes documentation pages between the desired (plan) state and the current state.
// It creates new pages, updates modified pages, and deletes removed pages.
// All changes go through the draft → publish workflow.
func (r *AssetResource) syncPages(ctx context.Context, groupID, assetID, version string, currentPages, desiredPages []PageModel) error {
	// Build lookup map of current pages by page_name
	currentByName := make(map[string]PageModel)
	for _, p := range currentPages {
		if !p.PageName.IsNull() && !p.PageName.IsUnknown() {
			currentByName[p.PageName.ValueString()] = p
		}
	}

	// Build lookup map of desired pages by page_name
	desiredByName := make(map[string]PageModel)
	for _, p := range desiredPages {
		if !p.PageName.IsNull() && !p.PageName.IsUnknown() {
			desiredByName[p.PageName.ValueString()] = p
		}
	}

	needsPublish := false

	// Delete pages that are in current but not in desired
	for name, current := range currentByName {
		if _, exists := desiredByName[name]; !exists {
			// Delete this page from draft
			pagePath := current.PagePath.ValueString()
			if pagePath != "" {
				if err := r.client.DeleteDraftPage(ctx, groupID, assetID, version, pagePath); err != nil {
					return fmt.Errorf("failed to delete page '%s': %w", name, err)
				}
				needsPublish = true
			}
		}
	}

	// Create or update pages
	for name, desired := range desiredByName {
		content := desired.Content.ValueString()

		if current, exists := currentByName[name]; exists {
			// Page exists — check if content changed
			if current.Content.ValueString() != content {
				// Update content via PUT to draft
				pagePath := current.PagePath.ValueString()
				if pagePath == "" {
					return fmt.Errorf("page '%s' exists in state but has no page_path", name)
				}
				if err := r.client.UpdateDraftPageContent(ctx, groupID, assetID, version, pagePath, content); err != nil {
					return fmt.Errorf("failed to update page '%s': %w", name, err)
				}
				needsPublish = true
			}
		} else {
			// New page — create then set content
			page, err := r.client.CreateDraftPage(ctx, groupID, assetID, version, name)
			if err != nil {
				return fmt.Errorf("failed to create page '%s': %w", name, err)
			}

			// Set the content
			if content != "" {
				if err := r.client.UpdateDraftPageContent(ctx, groupID, assetID, version, page.Path, content); err != nil {
					return fmt.Errorf("failed to set content for page '%s': %w", name, err)
				}
			}
			needsPublish = true
		}
	}

	// Publish the draft to make changes visible
	if needsPublish {
		if err := r.client.PublishDraft(ctx, groupID, assetID, version); err != nil {
			return fmt.Errorf("failed to publish portal draft: %w", err)
		}
	}

	return nil
}

// --- Terms & Conditions Helpers ---

// syncTermsAndConditions creates, updates, or removes the T&C page via the draft→publish workflow.
func (r *AssetResource) syncTermsAndConditions(ctx context.Context, groupID, assetID, version, currentContent, desiredContent string) error {
	if desiredContent == "" && currentContent != "" {
		// Delete T&C
		if err := r.client.DeleteDraftPage(ctx, groupID, assetID, version, ".terms"); err != nil {
			return fmt.Errorf("failed to delete T&C page: %w", err)
		}
		return r.client.PublishDraft(ctx, groupID, assetID, version)
	}

	if desiredContent != "" && desiredContent != currentContent {
		// Create or update T&C — PUT is an upsert
		if err := r.client.UpdateDraftPageContent(ctx, groupID, assetID, version, ".terms", desiredContent); err != nil {
			return fmt.Errorf("failed to set T&C content: %w", err)
		}
		return r.client.PublishDraft(ctx, groupID, assetID, version)
	}

	return nil
}

// readTermsIntoState reads the published T&C page content into state.
func (r *AssetResource) readTermsIntoState(ctx context.Context, state *AssetResourceModel) {
	content, err := r.client.GetPortalPageContent(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
		".terms",
	)
	if err != nil {
		// No T&C page — set empty
		state.TermsAndConditions = types.StringValue("")
		return
	}

	state.TermsAndConditions = types.StringValue(content)
}

// --- Non-Managed Instance Helpers ---

// instanceObjectType returns the object type for an instance nested attribute.
func instanceObjectType() basetypes.ObjectTypable {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":         types.StringType,
			"endpoint_uri": types.StringType,
			"is_public":    types.BoolType,
			"instance_id":  types.StringType,
		},
	}
}

// syncInstances synchronizes external instances between desired and current state.
// Since there's no LIST endpoint, we use state to track existing instances.
func (r *AssetResource) syncInstances(ctx context.Context, groupID, assetID, versionGroup string, currentInstances, desiredInstances []InstanceModel) error {
	// Build lookup of current instances by name (since we track instance_id in state)
	currentByName := make(map[string]InstanceModel)
	for _, inst := range currentInstances {
		if !inst.Name.IsNull() && !inst.Name.IsUnknown() {
			currentByName[inst.Name.ValueString()] = inst
		}
	}

	// Build lookup of desired instances by name
	desiredByName := make(map[string]InstanceModel)
	for _, inst := range desiredInstances {
		if !inst.Name.IsNull() && !inst.Name.IsUnknown() {
			desiredByName[inst.Name.ValueString()] = inst
		}
	}

	// Delete instances not in desired
	for name, current := range currentByName {
		if _, exists := desiredByName[name]; !exists {
			instanceID := current.InstanceID.ValueString()
			if instanceID != "" {
				if err := r.client.DeleteExternalInstance(ctx, groupID, assetID, versionGroup, instanceID); err != nil {
					return fmt.Errorf("failed to delete instance '%s': %w", name, err)
				}
			}
		}
	}

	// Create or update instances
	for name, desired := range desiredByName {
		if current, exists := currentByName[name]; exists {
			// Instance exists — check if it needs updating
			instanceID := current.InstanceID.ValueString()
			needsUpdate := false

			if desired.EndpointURI.ValueString() != current.EndpointURI.ValueString() {
				needsUpdate = true
			}
			if !desired.IsPublic.IsNull() && !desired.IsPublic.IsUnknown() && desired.IsPublic.ValueBool() != current.IsPublic.ValueBool() {
				needsUpdate = true
			}

			if needsUpdate && instanceID != "" {
				isPublic := desired.IsPublic.ValueBool()
				updateReq := &exchange.UpdateExternalInstanceRequest{
					Name:        desired.Name.ValueString(),
					EndpointURI: desired.EndpointURI.ValueString(),
					IsPublic:    &isPublic,
				}
				if err := r.client.UpdateExternalInstance(ctx, groupID, assetID, versionGroup, instanceID, updateReq); err != nil {
					return fmt.Errorf("failed to update instance '%s': %w", name, err)
				}
			}
		} else {
			// New instance — create
			isPublic := false
			if !desired.IsPublic.IsNull() && !desired.IsPublic.IsUnknown() {
				isPublic = desired.IsPublic.ValueBool()
			}
			createReq := &exchange.CreateExternalInstanceRequest{
				Name:        name,
				EndpointURI: desired.EndpointURI.ValueString(),
				IsPublic:    isPublic,
			}
			if _, err := r.client.CreateExternalInstance(ctx, groupID, assetID, versionGroup, createReq); err != nil {
				return fmt.Errorf("failed to create instance '%s': %w", name, err)
			}
		}
	}

	return nil
}

// readInstancesIntoState reads non-managed instances from the asset's instances list.
// Since there's no dedicated LIST endpoint for external instances, we read from the
// main asset GET response which includes instances.
// Instances are reordered to match the plan/state ordering (by name) so Terraform
// doesn't report spurious drift due to API returning instances in arbitrary order.
func (r *AssetResource) readInstancesIntoState(ctx context.Context, state *AssetResourceModel) {
	asset, err := r.client.GetAsset(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
	)
	if err != nil {
		state.Instances = types.ListValueMust(instanceObjectType(), []attr.Value{})
		return
	}

	// Parse instances from the asset response — filter for "external" type
	var externalInstances []map[string]interface{}
	for _, inst := range asset.Instances {
		if instMap, ok := inst.(map[string]interface{}); ok {
			if instType, _ := instMap["type"].(string); instType == "external" {
				externalInstances = append(externalInstances, instMap)
			}
		}
	}

	if len(externalInstances) == 0 {
		state.Instances = types.ListValueMust(instanceObjectType(), []attr.Value{})
		return
	}

	// Build a name→instance lookup so we can reorder to match the plan/state ordering
	instanceByName := make(map[string]map[string]interface{}, len(externalInstances))
	for _, inst := range externalInstances {
		name, _ := inst["name"].(string)
		instanceByName[name] = inst
	}

	// Determine desired order: use current state.Instances (which reflects config order)
	var desiredOrder []string
	if !state.Instances.IsNull() && !state.Instances.IsUnknown() {
		var currentInstances []InstanceModel
		if diags := state.Instances.ElementsAs(ctx, &currentInstances, false); !diags.HasError() {
			for _, ci := range currentInstances {
				desiredOrder = append(desiredOrder, ci.Name.ValueString())
			}
		}
	}

	// Build ordered list: first match by plan order, then append any new instances
	seen := make(map[string]bool, len(externalInstances))
	var ordered []map[string]interface{}
	for _, name := range desiredOrder {
		if inst, ok := instanceByName[name]; ok {
			ordered = append(ordered, inst)
			seen[name] = true
		}
	}
	// Append any instances from the API not in the current plan (shouldn't happen normally)
	for _, inst := range externalInstances {
		name, _ := inst["name"].(string)
		if !seen[name] {
			ordered = append(ordered, inst)
		}
	}

	instanceValues := make([]attr.Value, 0, len(ordered))
	for _, inst := range ordered {
		name, _ := inst["name"].(string)
		endpointURI, _ := inst["endpointUri"].(string)
		isPublic, _ := inst["isPublic"].(bool)
		id, _ := inst["id"].(string)

		instObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"name":         types.StringType,
				"endpoint_uri": types.StringType,
				"is_public":    types.BoolType,
				"instance_id":  types.StringType,
			},
			map[string]attr.Value{
				"name":         types.StringValue(name),
				"endpoint_uri": types.StringValue(endpointURI),
				"is_public":    types.BoolValue(isPublic),
				"instance_id":  types.StringValue(id),
			},
		)
		instanceValues = append(instanceValues, instObj)
	}

	state.Instances, _ = types.ListValue(instanceObjectType(), instanceValues)
}

// --- Category & Custom Field Helpers ---

// categoryObjectType returns the object type for a category nested attribute.
func categoryObjectType() basetypes.ObjectTypable {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"key":    types.StringType,
			"values": types.ListType{ElemType: types.StringType},
		},
	}
}

// customFieldObjectType returns the object type for a custom field nested attribute.
func customFieldObjectType() basetypes.ObjectTypable {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"key":    types.StringType,
			"values": types.ListType{ElemType: types.StringType},
		},
	}
}

// syncCategories synchronizes category assignments between desired and current state.
// It sets new/updated categories and removes categories that are no longer desired.
func (r *AssetResource) syncCategories(ctx context.Context, groupID, assetID, version string, currentCategories, desiredCategories []CategoryModel) error {
	// Build lookup of current categories by key
	currentByKey := make(map[string]CategoryModel)
	for _, cat := range currentCategories {
		if !cat.Key.IsNull() && !cat.Key.IsUnknown() {
			currentByKey[cat.Key.ValueString()] = cat
		}
	}

	// Build lookup of desired categories by key
	desiredByKey := make(map[string]CategoryModel)
	for _, cat := range desiredCategories {
		if !cat.Key.IsNull() && !cat.Key.IsUnknown() {
			desiredByKey[cat.Key.ValueString()] = cat
		}
	}

	// Delete categories not in desired
	for key := range currentByKey {
		if _, exists := desiredByKey[key]; !exists {
			if err := r.client.DeleteCategory(ctx, groupID, assetID, version, key); err != nil {
				return fmt.Errorf("failed to delete category '%s': %w", key, err)
			}
		}
	}

	// Set/update desired categories
	for key, desired := range desiredByKey {
		var values []string
		diags := desired.Values.ElementsAs(ctx, &values, false)
		if diags.HasError() {
			return fmt.Errorf("failed to extract category values for '%s'", key)
		}

		if err := r.client.SetCategory(ctx, groupID, assetID, version, key, values); err != nil {
			return fmt.Errorf("failed to set category '%s': %w", key, err)
		}
	}

	return nil
}

// syncCustomFields synchronizes custom field assignments between desired and current state.
func (r *AssetResource) syncCustomFields(ctx context.Context, groupID, assetID, version string, currentFields, desiredFields []CustomFieldModel) error {
	// Build lookup of current fields by key
	currentByKey := make(map[string]CustomFieldModel)
	for _, f := range currentFields {
		if !f.Key.IsNull() && !f.Key.IsUnknown() {
			currentByKey[f.Key.ValueString()] = f
		}
	}

	// Build lookup of desired fields by key
	desiredByKey := make(map[string]CustomFieldModel)
	for _, f := range desiredFields {
		if !f.Key.IsNull() && !f.Key.IsUnknown() {
			desiredByKey[f.Key.ValueString()] = f
		}
	}

	// Delete fields not in desired
	for key := range currentByKey {
		if _, exists := desiredByKey[key]; !exists {
			if err := r.client.DeleteCustomField(ctx, groupID, assetID, version, key); err != nil {
				return fmt.Errorf("failed to delete custom field '%s': %w", key, err)
			}
		}
	}

	// Set/update desired fields
	for key, desired := range desiredByKey {
		var values []string
		diags := desired.Values.ElementsAs(ctx, &values, false)
		if diags.HasError() {
			return fmt.Errorf("failed to extract custom field values for '%s'", key)
		}

		if err := r.client.SetCustomField(ctx, groupID, assetID, version, key, values); err != nil {
			return fmt.Errorf("failed to set custom field '%s': %w", key, err)
		}
	}

	return nil
}

// readCategoriesIntoState reads category assignments from the asset's API response into state.
func (r *AssetResource) readCategoriesIntoState(ctx context.Context, state *AssetResourceModel) {
	asset, err := r.client.GetAsset(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
	)
	if err != nil {
		state.Categories = types.ListValueMust(categoryObjectType(), []attr.Value{})
		return
	}

	if len(asset.Categories) == 0 {
		state.Categories = types.ListValueMust(categoryObjectType(), []attr.Value{})
		return
	}

	catValues := make([]attr.Value, 0, len(asset.Categories))
	for _, cat := range asset.Categories {
		// Build the values list
		valueAttrs := make([]attr.Value, len(cat.Value))
		for i, v := range cat.Value {
			valueAttrs[i] = types.StringValue(v)
		}
		valuesList, _ := types.ListValue(types.StringType, valueAttrs)

		catObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"key":    types.StringType,
				"values": types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"key":    types.StringValue(cat.Key),
				"values": valuesList,
			},
		)
		catValues = append(catValues, catObj)
	}

	state.Categories, _ = types.ListValue(categoryObjectType(), catValues)
}

// readCustomFieldsIntoState reads custom field assignments from the asset's API response into state.
func (r *AssetResource) readCustomFieldsIntoState(ctx context.Context, state *AssetResourceModel) {
	asset, err := r.client.GetAsset(ctx,
		state.GroupID.ValueString(),
		state.AssetID.ValueString(),
		state.Version.ValueString(),
	)
	if err != nil {
		state.CustomFields = types.ListValueMust(customFieldObjectType(), []attr.Value{})
		return
	}

	if len(asset.CustomFields) == 0 {
		state.CustomFields = types.ListValueMust(customFieldObjectType(), []attr.Value{})
		return
	}

	fieldValues := make([]attr.Value, 0, len(asset.CustomFields))
	for _, field := range asset.CustomFields {
		// The API returns value as interface{} — could be string, []string, or []interface{}
		var values []string
		switch v := field.Value.(type) {
		case string:
			values = []string{v}
		case []string:
			values = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					values = append(values, s)
				}
			}
		}

		valueAttrs := make([]attr.Value, len(values))
		for i, val := range values {
			valueAttrs[i] = types.StringValue(val)
		}
		valuesList, _ := types.ListValue(types.StringType, valueAttrs)

		fieldObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"key":    types.StringType,
				"values": types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"key":    types.StringValue(field.Key),
				"values": valuesList,
			},
		)
		fieldValues = append(fieldValues, fieldObj)
	}

	state.CustomFields, _ = types.ListValue(customFieldObjectType(), fieldValues)
}

// computeFileHash computes the SHA256 hash of a file for drift detection.
func computeFileHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}
