package exchange

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	_ resource.ResourceWithModifyPlan  = &AssetResource{}
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

	// AdditionalFiles holds EXTRA files for multi-file asset types published in the
	// same request as the primary file_path. The canonical case is type="policy",
	// which needs (schema.json + metadata.yaml) or (mule-policy.jar +
	// policy-definition.yaml). Local-only + upload-only, exactly like file_path:
	// preserved from prior state on Read, never reconciled from the API. Null/empty
	// for every single-file and fileless type.
	AdditionalFiles types.List `tfsdk:"additional_file"`

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

// AdditionalFileModel describes one extra file attached to a multi-file publish
// (alongside the primary file_path). Both fields mirror the primary file_path /
// classifier semantics: local-only, used only at creation/replacement time.
type AdditionalFileModel struct {
	Path       types.String `tfsdk:"path"`
	Classifier types.String `tfsdk:"classifier"`
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
				Description: "The Exchange asset type. This is a free-form value forwarded as-is to Exchange — it is NOT restricted to a fixed enum, so any type Exchange accepts works. Common values: custom, rest-api, http-api, evented-api (AsyncAPI), graphql-api, grpc-api, soap-api, connector, app, template, example, policy, ruleset, agent, llm, mcp, extension. API-spec fragments (RAML/OAS fragments) are not a distinct type: publish them as an API-spec asset and select the fragment via `classifier` (e.g. raml-fragment, oas-fragment/oas-components). The mule-plugin family (a JAR with classifier `mule-plugin`, e.g. a `policy` or `connector`) is stored by Exchange under the single super-type `extension`; you may declare the semantic type (`policy`/`connector`) and the provider preserves it (no drift), or declare `extension` directly — both apply cleanly. A bare `terraform import` of such an asset surfaces the stored `extension` (the semantic sub-type cannot be recovered from the API); you can then set `type = \"policy\"` (or `connector`) in config and the provider treats it as the SAME asset — the change is reconciled in place, NOT as a destroy+recreate, because both normalize to `extension`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// Reuse the prior state when type is omitted from config, exactly like
					// the immutable siblings classifier/api_version/main_file. Without this,
					// an Optional+Computed attribute with config null plans as UNKNOWN, so the
					// RequiresReplace modifier below sees "rest-api" -> (unknown) and forces a
					// destroy+recreate on the FIRST plan after import (import seeds type via
					// Read, but config typically omits it). That is catastrophic: importing an
					// existing asset must never recreate it. With UseStateForUnknown the plan
					// keeps the seeded value, so RequiresReplace no-ops on the omit path.
					// Safe against "inconsistent result after apply": type is immutable, so on
					// an in-place update Read maps the API type back to the same state value
					// (normalizeType), and on a genuine type change the plan value is KNOWN
					// (not unknown) so UseStateForUnknown no-ops and the replace modifier still
					// forces replacement. On replacement, create re-plans with null prior
					// state so this no-ops and the value is recomputed fresh.
					//
					// RequiresReplaceOnTypeChange (not the built-in RequiresReplace) compares
					// the API-NORMALIZED type, so alias members of the same super-type are NOT
					// treated as a change: declaring `type = "policy"` against an imported state
					// of `extension` (the only thing a bare import can surface for the
					// mule-plugin family) reconciles in place instead of destroying+recreating
					// the asset. A real cross-type change (rest-api -> soap-api) still replaces.
					stringplanmodifier.UseStateForUnknown(),
					RequiresReplaceOnTypeChange(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The lifecycle status of this asset version. One of `development`, " +
					"`published` (default), or `deprecated`. The value is case-sensitive. " +
					"Note the API asymmetry (validated at plan time and at apply time): " +
					"`development` is only accepted when first publishing a version — it CANNOT be " +
					"set on an existing version (the platform rejects it with HTTP 400), so moving " +
					"an already-published version back to `development` requires republishing " +
					"(a new version). `deprecated` can only be set on an existing version, not at " +
					"initial publish. `published` is valid in both cases.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					// Full valid set = union of the create-path (development, published) and the
					// update-path (published, deprecated), LIVE-VERIFIED against the platform's own
					// 400 bodies (2026-07-22). Case-sensitive on purpose: the API does NOT normalize
					// case ("Published" -> HTTP 400), so catching a typo here at plan time is exactly
					// what prevents a bad status from reaching a `version`-RequiresReplace apply.
					stringvalidator.OneOf("development", "published", "deprecated"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"file_path": schema.StringAttribute{
				Description: "Path to the file to upload (JAR, ZIP, RAML, OAS, etc.). Used only at creation time. " +
					"After import, one apply settles this field (non-destructive). Changing to a different value triggers replacement.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					RequiresReplaceExceptOnImport(),
				},
			},
			"classifier": schema.StringAttribute{
				Description: "The file classifier — it is the FILE kind, not the asset type, and for the spec types it does NOT equal the type. Common values: custom, oas (rest-api), raml (rest-api), wsdl (soap-api), graphql (graphql-api), proto (grpc-api), evented-api (evented-api / AsyncAPI — the classifier equals the type here; `asyncapi` is NOT accepted and yields 400 COULD_NOT_DETERMINE_ASSET_TYPE), raml-fragment, oas-fragment/oas-components. Required when file_path is set. Fragment assets are published by setting this to a fragment classifier (e.g. raml-fragment). Exchange bundles the spec and stores it as `fat-<classifier>`; the provider normalizes it back on read.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// Reuse the prior state on unrelated in-place updates instead of rendering
					// "(known after apply)". Safe because on an in-place update the API returns
					// the same immutable classifier (normalizeClassifier preserves the state
					// value; Read seeds it identically), so the frozen plan equals what Update
					// re-derives. On a genuine change the value is KNOWN in the plan, so this
					// modifier no-ops and RequiresReplaceExceptOnImport still forces replacement
					// (the two are mutually exclusive on plan-known-ness). On replacement, core
					// re-plans the create with null prior state, so UseStateForUnknown no-ops and
					// the value is recomputed fresh.
					stringplanmodifier.UseStateForUnknown(),
					RequiresReplaceExceptOnImport(),
				},
			},
			"additional_file": schema.ListNestedAttribute{
				Description: "Additional files to upload alongside file_path in the SAME publish request, " +
					"for multi-file asset types. The canonical case is type=\"policy\", which requires two " +
					"files: (schema.json + metadata.yaml) or (mule-policy.jar + policy-definition.yaml). Each " +
					"entry is written as its own files.{classifier}.{ext} part. Like file_path, this is a " +
					"local-only, create-time field: it is preserved from state on read (never reconciled from " +
					"the API), and changing it triggers replacement (except the null→value settle after import).",
				Optional: true,
				PlanModifiers: []planmodifier.List{
					// Mirror file_path/classifier: the uploaded file set is immutable, so a
					// change forces replacement — EXCEPT the post-import null→value settle,
					// which is non-destructive (ImportState does not seed additional_file, so
					// the first apply after import goes null→value and must not recreate).
					RequiresReplaceListExceptOnImport(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Local path to the additional file to upload (e.g. specs/metadata.yaml).",
							Required:    true,
						},
						"classifier": schema.StringAttribute{
							Description: "The classifier for this file (e.g. metadata, policy-definition, schema). " +
								"Combined with the file extension to form the files.{classifier}.{ext} field name.",
							Required: true,
						},
					},
				},
			},
			"keywords": schema.StringAttribute{
				Description: "Comma-separated keywords for search discovery.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					RequiresReplaceExceptOnImport(),
				},
			},
			"api_version": schema.StringAttribute{
				Description: "The API version (properties.apiVersion), e.g. \"v1\". REQUIRED at create for the API-spec types rest-api, evented-api, and grpc-api — publishing one of these without api_version fails with `400 MISSING_REQUIRED_PROPERTIES: apiVersion`. This is the human-facing API contract version, distinct from the immutable GAV version.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See classifier: reuse prior state on unrelated in-place updates. api_version
					// is create-only metadata read back via extractAttributeValue; on an in-place
					// update it is unchanged (any change forces replacement), and Read seeds it
					// identically, so the frozen plan equals the Update re-derivation. No-ops on a
					// known plan value (so RequiresReplaceExceptOnImport still governs real changes)
					// and on replacement (create re-plans with null state).
					stringplanmodifier.UseStateForUnknown(),
					RequiresReplaceExceptOnImport(),
				},
			},
			"main_file": schema.StringAttribute{
				Description: "The main file within the uploaded archive (properties.mainFile). Used for multi-file specs.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See classifier: reuse prior state on unrelated in-place updates. main_file is
					// create-only metadata read back via extractFileMetadata from the immutable
					// uploaded file; on an in-place update it is unchanged (any change forces
					// replacement), and Read seeds it identically, so the frozen plan equals the
					// Update re-derivation. No-ops on a known plan value and on replacement.
					stringplanmodifier.UseStateForUnknown(),
					RequiresReplaceExceptOnImport(),
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
				// READ-ONLY. The Exchange metadata PATCH endpoint does not accept an
				// arbitrary manager value: LIVE-VERIFIED (2026-07-22) that PATCH
				// {"manager":"<username>"} returns HTTP 403 Forbidden and
				// {"manager":"<uuid>"} returns HTTP 400 ("must be equal to one of the
				// allowed values: ,  ... must match exactly one schema in oneOf"). There
				// is no supported way to SET the manager through this API from Terraform,
				// so exposing it as writable only produced apply-time 403/400 failures
				// that killed the entire apply. It is therefore Computed-only: the value
				// is surfaced from Exchange (e.g. when set via the UI) but cannot be
				// managed here. Setting it in configuration is a plan-time error.
				Description: "The manager of this asset, as reported by Exchange. Read-only: " +
					"the Exchange API does not permit setting the manager via automation " +
					"(attempting to do so returns HTTP 403/400), so this attribute cannot be " +
					"configured — it only reflects a value set elsewhere (e.g. the Exchange UI).",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				Description: "Search tags for the asset version. Each element is a tag value string.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					// Reuse the prior tags list on unrelated in-place updates instead of
					// rendering "(known after apply)". The Update path gates the tags sync on
					// !plan.Tags.Equal(state.Tags) && !IsNull && !IsUnknown, so an explicit []
					// (a KNOWN value) still clears tags; only the false churn is suppressed.
					// Crash-safe because mapAssetToState reorders the API labels to the prior
					// tag order (labels are order-unstable from the API), so the frozen plan
					// order equals the applied order — no "inconsistent result after apply".
					listplanmodifier.UseStateForUnknown(),
				},
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
				// NO UseStateForUnknown here — deliberately. updated_date is a SERVER-BUMPED
				// timestamp that changes on every write. UseStateForUnknown would freeze the
				// plan to the prior timestamp, but Update re-reads the NEW timestamp from the
				// API (mapAssetToState) with no preservation, so apply would fail with
				// "Provider produced inconsistent result after apply" (planned old timestamp
				// != applied new timestamp). Showing "(known after apply)" on an in-place
				// update is CORRECT here: the value genuinely changes. (Contrast created_date,
				// which is immutable, so UseStateForUnknown is safe there.)
			},
			"pages": schema.ListNestedAttribute{
				Description: "Documentation pages for the asset portal. Each page has a name and markdown content.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					// Keep the whole prior list (including the computed page_path on each
					// element) in the plan when pages are not being edited. Without this,
					// any unrelated in-place update (e.g. a tags-only change) renders pages
					// as "(known after apply)". The Update path gates the pages sync on
					// !plan.Pages.Equal(state.Pages) && !IsNull && !IsUnknown. The .Equal()
					// term is load-bearing: reusing prior state (which makes plan==state)
					// means the gate does NOT fire on the omit path, so pages are not
					// re-synced. A real edit or an explicit [] is a KNOWN value that differs
					// from state, so it still clears/updates pages. Do NOT reduce this to
					// !IsUnknown() alone — that would re-sync on every unrelated update.
					listplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					// Keep the prior T&C value in the plan on unrelated in-place updates
					// instead of "(known after apply)". The Update path gates the T&C sync
					// on !plan.TermsAndConditions.Equal(state...) && !IsNull && !IsUnknown.
					// The .Equal() term means reusing prior state (plan==state) does NOT
					// re-sync on the omit path; an explicit "" (a KNOWN value that differs)
					// still clears. Do NOT reduce to !IsUnknown() alone.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instances": schema.ListNestedAttribute{
				Description: "Non-managed (external) API instances for this asset version.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					// Keep the whole prior list (including the computed instance_id on each
					// element) in the plan when instances are not being edited. The Update
					// path gates the instance sync on !plan.Instances.Equal(state...) &&
					// !IsNull && !IsUnknown. The .Equal() term means reusing prior state
					// (plan==state) does NOT re-sync on the omit path; an explicit [] (a
					// KNOWN value that differs) still clears. Do NOT reduce to !IsUnknown().
					listplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.List{
					// Keep the prior categories list in the plan when categories are not
					// being edited, instead of "(known after apply)" on unrelated updates.
					// The Update path gates the category sync on !plan.Categories.Equal(
					// state...) && !IsNull && !IsUnknown. The .Equal() term means reusing
					// prior state (plan==state) does NOT re-sync on the omit path; an
					// explicit [] (a KNOWN value that differs) still clears. Do NOT reduce
					// to !IsUnknown() alone.
					listplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.List{
					// Keep the prior custom_fields list in the plan when they are not being
					// edited, instead of "(known after apply)" on unrelated updates. The
					// Update path gates the sync on !plan.CustomFields.Equal(state...) &&
					// !IsNull && !IsUnknown. The .Equal() term means reusing prior state
					// (plan==state) does NOT re-sync on the omit path; an explicit [] (a
					// KNOWN value that differs) still clears. Do NOT reduce to !IsUnknown().
					listplanmodifier.UseStateForUnknown(),
				},
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

// assetTypesRequiringFile is the allowlist of Exchange asset types whose
// creation REQUIRES an uploaded spec file. Exchange infers/validates the asset
// type from the uploaded file, so publishing one of these WITHOUT a file fails
// mid-apply with COULD_NOT_DETERMINE_ASSET_TYPE.
//
// This is an ALLOWLIST on purpose, and intentionally conservative:
//   - A false positive (blocking a type that does not actually need a file) would
//     break a legitimate fileless workflow — exactly what we must avoid.
//   - A false negative (not blocking a type that does need one) merely preserves
//     today's behavior (the apply still fails with the same server error), so it
//     is the safe direction to err in.
//
// Therefore only types with a MANDATORY formal spec are listed:
//   - rest-api    (RAML/OAS)  — proven in the field to reject a fileless publish
//   - soap-api    (WSDL)      — WSDL is mandatory for SOAP
//   - graphql-api (SDL)       — a GraphQL schema is mandatory
//   - evented-api (AsyncAPI)  — live-verified 2026-07-16: fileless publish 400s
//     COULD_NOT_DETERMINE_ASSET_TYPE. The correct file classifier is `evented-api`
//     (same string as the type; Exchange stores it as fat-evented-api) — NOT
//     `asyncapi`, which also 400s COULD_NOT_DETERMINE_ASSET_TYPE. Confirmed
//     2026-08-06 from a real prod evented-api asset (files[].classifier=evented-api)
//     and a devx publish with classifier=evented-api returning 201.
//   - grpc-api    (protobuf)  — live-verified 2026-07-16: fileless publish 400s
//     MISSING_FILES_ERROR (protobuf.proto|protobuf.zip)
//   - ruleset     (profile)   — live-verified 2026-07-16: fileless publish 400s
//     COULD_NOT_DETERMINE_ASSET_TYPE
//
// Deliberately EXCLUDED (may legitimately be published without a file, so blocking
// them would be a false positive): custom, app, template, example, connector,
// policy, agent, llm (live-verified fileless 2026-07-16), mcp, http-api, and any
// future/unknown type. Extend this map only when a type is confirmed to require a file.
//
// NOTE: rest-api, evented-api and grpc-api ALSO require properties.apiVersion (a
// versionless publish 400s MISSING_REQUIRED_PROPERTIES: apiVersion) — a distinct
// constraint from this file-only allowlist. It is modeled separately by
// assetTypesRequiringAPIVersion / the api_version guard in ModifyPlan (below).
// See the E2E report Finding B.
var assetTypesRequiringFile = map[string]bool{
	"rest-api":    true,
	"soap-api":    true,
	"graphql-api": true,
	"evented-api": true,
	"grpc-api":    true,
	"ruleset":     true,
}

// assetTypeRequiresFile reports whether the given asset type must be published
// with an uploaded spec file. Unknown/unlisted types return false (never blocked).
func assetTypeRequiresFile(t string) bool {
	return assetTypesRequiringFile[strings.ToLower(strings.TrimSpace(t))]
}

// assetTypesRequiringAPIVersion lists asset types whose multipart CREATE publish is
// rejected when properties.apiVersion is omitted, with
// `400 MISSING_REQUIRED_PROPERTIES: apiVersion`. This is a SEPARATE constraint from
// assetTypesRequiringFile (a type can need a file, an apiVersion, or both), so it is
// modeled as its own allowlist rather than reusing that map.
//
// As with the file allowlist, membership is conservative — ONLY types whose apiVersion
// requirement was verified against the platform's own 400 body are listed, so the guard
// can never be a false positive for a type that legitimately publishes without one:
//   - rest-api    — live-verified 2026-07-27: create without properties.apiVersion 400s
//     MISSING_REQUIRED_PROPERTIES: apiVersion (both mainFile and apiVersion when both omitted).
//   - evented-api — live-verified 2026-07-16 (E2E report Finding B).
//   - grpc-api    — live-verified 2026-07-16 (E2E report Finding B).
//
// Deliberately EXCLUDED (no confirmed apiVersion requirement — blocking them would be a
// false positive): soap-api and graphql-api (file-backed but apiVersion not confirmed
// mandatory), and every fileless type (custom/app/template/policy/llm/mcp/http-api/…).
// Extend this map only when a type is confirmed to reject a versionless publish.
var assetTypesRequiringAPIVersion = map[string]bool{
	"rest-api":    true,
	"evented-api": true,
	"grpc-api":    true,
}

// assetTypeRequiresAPIVersion reports whether the given asset type's create publish
// requires properties.apiVersion. Unknown/unlisted types return false (never blocked).
func assetTypeRequiresAPIVersion(t string) bool {
	return assetTypesRequiringAPIVersion[strings.ToLower(strings.TrimSpace(t))]
}

// stringChanged reports whether a KNOWN planned value differs from the prior state
// value. Unknown or null plan values (and null/unknown prior state) are treated as
// "no change" so an unresolved computed value is never mistaken for a real edit.
func stringChanged(state, plan types.String) bool {
	if plan.IsUnknown() || plan.IsNull() {
		return false
	}
	if state.IsNull() || state.IsUnknown() {
		return false
	}
	return !plan.Equal(state)
}

// assetReplaceTriggered reports whether the planned change replaces the asset
// version (destroy + recreate) rather than updating it in place. Terraform recreates
// the asset when any replace-forcing attribute changes, and the recreate re-runs the
// file-dependent create path — so a replace with a missing file is just as dangerous
// as a fresh create. Only the replace-forcing identity/immutable attributes are
// compared; changing any of them forces replacement per the schema plan modifiers.
func assetReplaceTriggered(state, plan *AssetResourceModel) bool {
	return stringChanged(state.OrganizationID, plan.OrganizationID) ||
		stringChanged(state.GroupID, plan.GroupID) ||
		stringChanged(state.AssetID, plan.AssetID) ||
		stringChanged(state.Version, plan.Version) ||
		stringChanged(state.Type, plan.Type)
}

// ModifyPlan converts a specific silent-data-loss footgun into a loud plan-time
// error: publishing (create) or replacing a file-backed asset type with no
// file_path set. Without this guard, such a plan applies, Exchange rejects the
// upload with COULD_NOT_DETERMINE_ASSET_TYPE, and — on a version bump, which is a
// RequiresReplace — the OLD version has already been destroyed by the time the
// create fails. Failing at plan time (before any destroy) is strictly safer.
//
// Carefully scoped to avoid false positives:
//   - Fileless types (custom/app/template/…) and any unknown type are never blocked
//     (see assetTypesRequiringFile).
//   - Only CREATE and REPLACE are guarded. An in-place update — including the
//     zero-drift import→apply workflow, where the asset already exists on the server
//     with its file and file_path is a local-only field that legitimately stays
//     null — is never blocked.
func (r *AssetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy plan: nothing to validate.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan AssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Classify the operation ONCE, up front, because both guards below depend on
	// it and status validation is type-independent (so it must run before the
	// file-type early-returns):
	//   - creating  : no prior state.
	//   - replacing : a RequiresReplace attribute (e.g. version) changed, so the
	//                  version is destroyed and recreated via the multipart publish.
	//   - otherwise : an in-place update (PUT /status, PUT /tags, PATCH metadata).
	// Both create and replace publish through the multipart endpoint; only an
	// in-place update ever calls PUT /status.
	creating := req.State.Raw.IsNull()
	replacing := false
	var state AssetResourceModel
	if !creating {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		replacing = assetReplaceTriggered(&state, &plan)
	}

	// --- Guard #0: version-collision pre-check (Q-PROV-1 Option B). ---
	// Catch, at PLAN time, a create (or version bump) that targets a GAV already
	// published in Exchange — before any destroy runs — instead of letting apply hit
	// the raw 409 ASSET_PRE_CONDITIONS_FAILED. Adds a diagnostic only; falls through so
	// the file/api_version/status guards can also report on the same plan.
	r.checkVersionCollisionAtPlan(ctx, creating, &state, &plan, resp)

	// --- #67: keyed UseStateForUnknown for computed children of CONFIGURED lists. ---
	// pages (page_path) and instances (instance_id, is_public) are Optional+Computed
	// nested lists. The list-level UseStateForUnknown only fires when the WHOLE list is
	// unknown; when the user CONFIGURES the list, the list is known but each element's
	// computed child is unknown. That renders "page_path = (known after apply)" churn on
	// every unrelated in-place update AND trips the Update-path !plan.X.Equal(state.X)
	// sync gate (an unknown child never equals a concrete one), forcing a needless
	// re-sync. Fill each configured element's UNKNOWN computed children from the prior
	// state element MATCHED BY KEY (page_name / instance name) — the keyed generalization
	// of UseStateForUnknown. Done ONLY for an in-place update: on create/replace a new
	// version's children are genuinely unknown, and a positional copy of the old version's
	// values would be wrong (and crash with "inconsistent result after apply"). Keyed (not
	// positional) so reorder/insert/delete is handled correctly; only UNKNOWN plan children
	// are filled, so a user-set is_public is preserved; a new element with no keyed match
	// keeps its unknown child (correctly rendered as "known after apply").
	if !creating && !replacing {
		r.fillComputedChildrenFromState(ctx, &state, &plan, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// --- Guard #1: `development` cannot be set on an EXISTING version. ---
	// The multipart CREATE endpoint accepts {development, published}; the PUT
	// /status UPDATE endpoint accepts only {published, deprecated}. Both facts are
	// LIVE-VERIFIED against the platform's own 400 bodies (2026-07-22):
	//   PUT /status {"status":"development"} -> HTTP 400
	//     "must be equal to one of the allowed values: published, deprecated"
	// So moving an already-published version's status TO `development` in place is
	// impossible; the only way to get a `development` version is create/replace
	// (both go through multipart). Flag it at plan time — before apply issues the
	// PUT /status and fails with a raw 400 — with an actionable message. The plan
	// falls through so the file_path guard can also report if applicable.
	if !creating && !replacing &&
		!plan.Status.IsNull() && !plan.Status.IsUnknown() &&
		plan.Status.ValueString() == "development" &&
		!plan.Status.Equal(state.Status) {
		resp.Diagnostics.AddAttributeError(
			path.Root("status"),
			"Cannot set status to \"development\" on an existing asset version",
			"The Exchange status endpoint only accepts \"published\" or \"deprecated\" for an "+
				"existing version; \"development\" is valid only when first publishing a version. "+
				"The platform rejects an in-place change to \"development\" with HTTP 400.\n\n"+
				"To have a version in the development state, publish a NEW version with "+
				"status = \"development\" (change the version attribute, which forces replacement) "+
				"rather than editing the status of an already-published version in place.",
		)
	}

	// --- Guard #1.5: API-spec types need api_version on create/replace. ---
	// rest-api / evented-api / grpc-api reject a versionless multipart publish with
	// `400 MISSING_REQUIRED_PROPERTIES: apiVersion`. Like the file_path guard this only
	// bites when Exchange (re)publishes the version — CREATE or REPLACE — and on a
	// REPLACE (version bump) the old version is destroyed BEFORE the failing publish, so
	// raising it at plan time (before any destroy) is strictly safer than the raw
	// apply-time 400. Skipped when the type is unknown (can't classify) and for any type
	// not on the confirmed assetTypesRequiringAPIVersion allowlist. Falls through (no
	// early return) so the file_path guard below can also report on the same plan.
	//
	// The provided-check reads CONFIG, not plan — this is the key difference from the
	// file_path guard. api_version is Optional+Computed, so an UNCONFIGURED value plans
	// as Unknown; but so does a value bound to another resource's not-yet-known output.
	// Those two are indistinguishable in the PLAN, so a plan-based check would false-
	// positive on a legitimate reference and block a valid apply. In the CONFIG they ARE
	// distinct: omitted => null, unresolved reference => unknown, literal => known. Fire
	// only when the user genuinely supplied nothing (config null or empty ""); when it is
	// an unresolved reference (config unknown) defer to apply (the server 400 is the
	// backstop) rather than risk a false positive. (file_path is Optional-only, not
	// Computed, so an omitted file_path is already null in the plan — hence that guard can
	// safely read the plan directly.)
	if (creating || replacing) && !plan.Type.IsUnknown() &&
		assetTypeRequiresAPIVersion(plan.Type.ValueString()) {
		var cfgAPIVersion types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("api_version"), &cfgAPIVersion)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !cfgAPIVersion.IsUnknown() && (cfgAPIVersion.IsNull() || cfgAPIVersion.ValueString() == "") {
			apiVersionAction := "created"
			if replacing {
				apiVersionAction = "replaced (destroyed and recreated)"
			}
			resp.Diagnostics.AddAttributeError(
				path.Root("api_version"),
				"Missing api_version for an API-spec asset type",
				fmt.Sprintf(
					"Asset type %q requires api_version (properties.apiVersion) when it is published, "+
						"but api_version is not set. When the asset version is %s, Exchange rejects the "+
						"publish with \"400 MISSING_REQUIRED_PROPERTIES: apiVersion\".\n\n"+
						"Set api_version to the API contract version (for example \"v1\"). On a version "+
						"change this is especially important: the version attribute forces replacement, so "+
						"the failing publish can occur after the previous version has already been destroyed.",
					plan.Type.ValueString(), apiVersionAction,
				),
			)
		}
	}

	// --- Guard #2: file-backed types need a file on create/replace. ---
	// If the type is not yet known (e.g. it references another resource that has
	// not been created), we cannot classify it — skip rather than risk a false
	// positive. The apply-time server error remains as the backstop in that case.
	if plan.Type.IsUnknown() {
		return
	}
	if !assetTypeRequiresFile(plan.Type.ValueString()) {
		return // fileless type — never blocked
	}

	// The type needs a file. If the config provides one, nothing to flag.
	if !plan.FilePath.IsNull() && !plan.FilePath.IsUnknown() && plan.FilePath.ValueString() != "" {
		return
	}

	// file_path is absent for a file-backed type. This is only a problem when
	// Exchange will (re)upload the spec — i.e. on CREATE or on a REPLACE. On an
	// in-place update the asset already exists with its file, so file_path being
	// null is expected (this is the post-import steady state) and must not error.
	if !creating && !replacing {
		return // in-place update — file_path legitimately absent
	}

	action := "created"
	if replacing {
		action = "replaced (destroyed and recreated)"
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("file_path"),
		"Missing file_path for a file-backed asset type",
		fmt.Sprintf(
			"Asset type %q must be published with a spec file, but file_path is not set. "+
				"When the asset version is %s, Exchange uploads the file and infers/validates "+
				"the type from it; without a file the apply fails with "+
				"COULD_NOT_DETERMINE_ASSET_TYPE.\n\n"+
				"Set file_path to the spec file for this version. On a version change this is "+
				"especially important: the version attribute forces replacement, so a failed "+
				"upload can occur after the previous version has already been destroyed.\n\n"+
				"(file_path is expected to be null only when you import an existing asset and "+
				"leave it unchanged in place.)",
			plan.Type.ValueString(), action,
		),
	)
}

// fillComputedChildrenFromState is the keyed generalization of UseStateForUnknown for
// the computed CHILDREN of a configured nested list (#67). The list-level
// UseStateForUnknown plan modifiers on pages/instances only fire when the ENTIRE list is
// unknown; once the user configures the list, the list is a known value but each
// element's computed child (page_path; instance_id / is_public) is unknown, which both
// renders "(known after apply)" churn on unrelated in-place updates and trips the
// Update-path !plan.X.Equal(state.X) sync gate into a pointless re-sync.
//
// For each configured plan element whose computed child is UNKNOWN, copy the prior-state
// value of that child from the state element with the SAME KEY (page_name for pages, name
// for instances). Matching is by key — never by position — so reordering, inserting, or
// deleting elements is handled correctly (a positional copy would assign the wrong
// page_path/instance_id and crash the apply with "inconsistent result after apply"). Only
// UNKNOWN plan children are touched, so an explicitly configured is_public is preserved;
// a new element (no keyed match in state) keeps its unknown child, correctly shown as
// "known after apply". The caller invokes this ONLY for an in-place update (never on
// create/replace, where a new version's children are genuinely unknown).
func (r *AssetResource) fillComputedChildrenFromState(ctx context.Context, state, plan *AssetResourceModel, resp *resource.ModifyPlanResponse) {
	// pages: fill each unknown page_path from the prior state page with the same page_name.
	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() &&
		!state.Pages.IsNull() && !state.Pages.IsUnknown() {
		var planPages, statePages []PageModel
		resp.Diagnostics.Append(plan.Pages.ElementsAs(ctx, &planPages, false)...)
		resp.Diagnostics.Append(state.Pages.ElementsAs(ctx, &statePages, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		statePathByName := make(map[string]types.String, len(statePages))
		for _, sp := range statePages {
			if !sp.PageName.IsNull() && !sp.PageName.IsUnknown() {
				statePathByName[sp.PageName.ValueString()] = sp.PagePath
			}
		}
		pagesChanged := false
		for i := range planPages {
			if !planPages[i].PagePath.IsUnknown() ||
				planPages[i].PageName.IsNull() || planPages[i].PageName.IsUnknown() {
				continue
			}
			if prior, ok := statePathByName[planPages[i].PageName.ValueString()]; ok {
				planPages[i].PagePath = prior
				pagesChanged = true
			}
		}
		if pagesChanged {
			newList, diags := types.ListValueFrom(ctx, pageObjectType(), planPages)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("pages"), newList)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// instances: fill each unknown instance_id / is_public from the prior state instance
	// with the same name.
	if !plan.Instances.IsNull() && !plan.Instances.IsUnknown() &&
		!state.Instances.IsNull() && !state.Instances.IsUnknown() {
		var planInstances, stateInstances []InstanceModel
		resp.Diagnostics.Append(plan.Instances.ElementsAs(ctx, &planInstances, false)...)
		resp.Diagnostics.Append(state.Instances.ElementsAs(ctx, &stateInstances, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateByName := make(map[string]InstanceModel, len(stateInstances))
		for _, si := range stateInstances {
			if !si.Name.IsNull() && !si.Name.IsUnknown() {
				stateByName[si.Name.ValueString()] = si
			}
		}
		instancesChanged := false
		for i := range planInstances {
			if planInstances[i].Name.IsNull() || planInstances[i].Name.IsUnknown() {
				continue
			}
			prior, ok := stateByName[planInstances[i].Name.ValueString()]
			if !ok {
				continue
			}
			if planInstances[i].InstanceID.IsUnknown() {
				planInstances[i].InstanceID = prior.InstanceID
				instancesChanged = true
			}
			if planInstances[i].IsPublic.IsUnknown() {
				planInstances[i].IsPublic = prior.IsPublic
				instancesChanged = true
			}
		}
		if instancesChanged {
			newList, diags := types.ListValueFrom(ctx, instanceObjectType(), planInstances)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("instances"), newList)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}
}

// isAssetVersionConflict reports whether a CreateAsset error is the Exchange
// "this version already exists" precondition failure (HTTP 409
// ASSET_PRE_CONDITIONS_FAILED). The client returns a formatted error carrying
// the status code and response body, so we match on the stable error code plus
// the 409 status as a fallback.
func isAssetVersionConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "ASSET_PRE_CONDITIONS_FAILED") {
		return true
	}
	return strings.Contains(msg, "status 409") && strings.Contains(msg, "already exists")
}

// nextPatchVersionHint suggests the next patch version for the "bump the version"
// guidance (e.g. "1.0.0" -> "1.0.1"). It is best-effort: when the current value
// is not a dotted numeric version it falls back to appending a suffix.
func nextPatchVersionHint(version string) string {
	parts := strings.Split(version, ".")
	last := parts[len(parts)-1]
	// Strip any pre-release/build suffix (e.g. "1-SNAPSHOT") before incrementing.
	num := last
	if i := strings.IndexAny(last, "-+"); i >= 0 {
		num = last[:i]
	}
	n := 0
	for _, c := range num {
		if c < '0' || c > '9' {
			return version + "-2"
		}
		n = n*10 + int(c-'0')
	}
	if num == "" {
		return version + "-2"
	}
	parts[len(parts)-1] = fmt.Sprintf("%d", n+1)
	return strings.Join(parts, ".")
}

// checkVersionCollisionAtPlan raises a PLAN-TIME error when a create — or a version
// bump — targets a group/asset/version that ALREADY exists in Exchange. Exchange
// versions are immutable and permanently reserved, so publishing onto an existing GAV
// fails at apply with 409 ASSET_PRE_CONDITIONS_FAILED. On a version bump the OLD version
// is destroyed BEFORE that failing publish (version is RequiresReplace), so surfacing the
// collision at plan time — before any destroy — is strictly safer than the apply-time
// 409 (this is "Option B" for Q-PROV-1; the friendly apply-time message in Create stays
// as the backstop).
//
// Scope — mirrors the file_path / api_version guards, with a same-version carve-out:
//   - Fires ONLY when the plan will PUBLISH onto a not-yet-owned version: a create (null
//     prior state) or a genuine version change. A SAME-version replace (e.g. a
//     classifier or file_path edit that forces replacement without changing the version
//     number) is deliberately NOT flagged — that path destroys the current version
//     first (hard-delete frees the GAV) and republishes at the same number, which the
//     platform allows, so blocking it would be a false positive.
//   - Requires group_id / asset_id / version all KNOWN. An unresolved reference (unknown
//     value) defers to apply, exactly like the api_version guard, to avoid false
//     positives on a legitimate cross-resource reference.
//   - Requires a configured client (nil in pure-schema unit tests → skip).
//   - A NotFound (404) means the version is free → proceed silently. Any OTHER GET error
//     (network, auth, 5xx) is NON-fatal: we never block a plan on a transient probe
//     failure; the apply-time 409 remains the backstop.
func (r *AssetResource) checkVersionCollisionAtPlan(ctx context.Context, creating bool, state, plan *AssetResourceModel, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	// Only a create or a real version change publishes onto a new, un-owned version.
	if !creating && !stringChanged(state.Version, plan.Version) {
		return
	}
	g, a, v := plan.GroupID, plan.AssetID, plan.Version
	if g.IsNull() || g.IsUnknown() || a.IsNull() || a.IsUnknown() || v.IsNull() || v.IsUnknown() {
		return
	}

	_, err := r.client.GetAsset(ctx, g.ValueString(), a.ValueString(), v.ValueString())
	if err != nil {
		if !client.IsNotFound(err) {
			// Transient / non-404: never block the plan on it — apply's 409 is the backstop.
			tflog.Warn(ctx, "exchange asset plan-time version-collision probe failed; deferring to apply", map[string]interface{}{
				"group_id": g.ValueString(),
				"asset_id": a.ValueString(),
				"version":  v.ValueString(),
				"error":    err.Error(),
			})
		}
		return
	}

	// GET 200 → the version already exists in Exchange and we do not own it at this
	// version (create, or a bump onto a taken number). The apply would 409.
	action := "created"
	if !creating {
		action = "published at this new version"
	}
	resp.Diagnostics.AddAttributeError(
		path.Root("version"),
		"Exchange asset version already exists",
		fmt.Sprintf(
			"Version %q of asset %q/%q already exists in Exchange, so this plan would fail at "+
				"apply time when the version is %s: Exchange versions are immutable and permanently "+
				"reserved, and the publish is rejected with 409 ASSET_PRE_CONDITIONS_FAILED.\n\n"+
				"This is caught at plan time (before any destroy) so a version bump cannot delete the "+
				"old version and then fail to publish the new one. Resolve it by either:\n"+
				"  - bump `version` to a new value (e.g. %q) to publish fresh content, or\n"+
				"  - import the existing version instead of creating it:\n"+
				"      terraform import <address> %s/%s/%s",
			v.ValueString(), g.ValueString(), a.ValueString(), action,
			nextPatchVersionHint(v.ValueString()),
			g.ValueString(), a.ValueString(), v.ValueString(),
		),
	)
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

	// Extra files for multi-file types (e.g. policy). Each is uploaded as its own
	// files.{classifier}.{ext} part in the same publish request as file_path.
	extraFiles, extraDiags := additionalFilesToUploads(ctx, plan.AdditionalFiles)
	resp.Diagnostics.Append(extraDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

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
		ExtraFiles:     extraFiles,
		Keywords:       plan.Keywords.ValueString(),
		APIVersion:     plan.APIVersion.ValueString(),
		MainFile:       plan.MainFile.ValueString(),
	}

	asset, err := r.client.CreateAsset(ctx, createReq)
	if err != nil {
		// Exchange versions are immutable AND non-reusable: once a version is
		// published it is reserved forever, and a soft-delete does NOT free it up.
		// Re-publishing the same group/asset/version (e.g. after changing the
		// file, classifier, or tags, which forces a destroy+recreate) fails with
		// 409 ASSET_PRE_CONDITIONS_FAILED. Surface actionable "bump the version"
		// guidance instead of the raw platform error.
		if isAssetVersionConflict(err) {
			resp.Diagnostics.AddError(
				"Exchange asset version already exists",
				fmt.Sprintf(
					"Version %q of asset %q already exists in Exchange and cannot be reused. "+
						"Exchange versions are immutable and are reserved permanently — even after the "+
						"asset version is deleted, the same version number cannot be re-published "+
						"(the platform returns 409 ASSET_PRE_CONDITIONS_FAILED).\n\n"+
						"To publish new content (a different file, classifier, or tags), bump the "+
						"`version` to a new value (e.g. %q). Changing `version`, `classifier`, or "+
						"`file_path` already forces Terraform to replace this resource; give it a "+
						"fresh version number so the replacement can publish.\n\n"+
						"Original error: %s",
					plan.Version.ValueString(),
					plan.AssetID.ValueString(),
					nextPatchVersionHint(plan.Version.ValueString()),
					err.Error(),
				),
			)
			return
		}
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
	// NOTE: `manager` is intentionally NOT written here. The Exchange metadata PATCH
	// endpoint rejects any manager value (403 for a username, 400 for a UUID —
	// LIVE-VERIFIED 2026-07-22), so it is a read-only/Computed attribute. See its
	// schema definition. The value is populated from the API in Read.

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
			// Reconcile against instances that ALREADY exist on this versionGroup.
			// External instances are not cascade-deleted with the asset version, so a
			// recreate at the same versionGroup can inherit orphans from a prior delete.
			// Passing them as "current" makes syncInstances adopt/update (and prune)
			// them instead of blindly re-POSTing and hitting 409 EXTERNAL_API_CONFLICT.
			_, existingInstances, listErr := r.fetchExternalInstances(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), plan.Version.ValueString())
			if listErr != nil {
				resp.Diagnostics.AddError(
					"Error creating external instances",
					"Asset was created but reading existing instances failed: "+listErr.Error(),
				)
				return
			}
			err := r.syncInstances(ctx, plan.GroupID.ValueString(), plan.AssetID.ValueString(), asset.VersionGroup, existingInstances, planInstances)
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

	// Ensure Computed fields are concrete in state (not Unknown).
	// For types without file upload, classifier/api_version/main_file stay null.
	if plan.Classifier.IsUnknown() {
		plan.Classifier = types.StringNull()
	}
	if plan.APIVersion.IsUnknown() {
		plan.APIVersion = types.StringNull()
	}
	if plan.MainFile.IsUnknown() {
		plan.MainFile = types.StringNull()
	}

	// Compute file hash for drift detection
	if !plan.FilePath.IsNull() && plan.FilePath.ValueString() != "" {
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

	// Preserve file_path and file_sha256 from state (truly local-only, not in API).
	// additional_file is the same: a local, upload-only field the API never echoes
	// back as such, so it is preserved verbatim rather than reconciled (no drift).
	filePath := state.FilePath
	fileSHA256 := state.FileSHA256
	additionalFiles := state.AdditionalFiles

	r.mapAssetToState(&state, asset)

	// Restore truly local-only fields
	state.FilePath = filePath
	state.FileSHA256 = fileSHA256
	state.AdditionalFiles = additionalFiles

	// Extract api_version from attributes (available in API response for some types).
	// Some asset types (e.g. GraphQL) don't expose api-version in attributes,
	// so we preserve from prior state when not found (it's create-only / RequiresReplace).
	if apiVer := extractAttributeValue(asset.Attributes, "api-version"); apiVer != "" {
		state.APIVersion = types.StringValue(apiVer)
	}
	// else: preserve whatever was already in state (could be user-set or null)

	// Extract classifier and mainFile from the files array (user-uploaded file).
	// In MULTI-FILE mode (additional_file set), skip this reconciliation entirely and
	// preserve the config classifier/main_file verbatim: extractFileMetadata returns
	// only the FIRST matching file, which is ambiguous when several user files are
	// uploaded (e.g. policy's schema + metadata). The uploaded file set is immutable
	// and local-only, so freezing these from config avoids any drift the first-file
	// heuristic could introduce if the API's file ordering ever changes.
	if state.AdditionalFiles.IsNull() || state.AdditionalFiles.IsUnknown() {
		classifier, mainFile := extractFileMetadata(asset.Files)
		if classifier != "" {
			state.Classifier = types.StringValue(normalizeClassifier(classifier, state.Classifier))
		}
		// else: preserve from state (some asset types may not have user-uploaded files)
		if mainFile != "" {
			state.MainFile = types.StringValue(mainFile)
		}
		// else: preserve from state
	}

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

	// Only these fields are writable via the metadata PATCH: name, description,
	// contactName, contactEmail. `manager` is NOT writable — the endpoint rejects any
	// value (403/400, LIVE-VERIFIED 2026-07-22), so it is read-only/Computed and is
	// never sent here. We only send fields that actually changed AND have a meaningful
	// value in the plan.
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

	// Preserve file_path from plan (config value). After import, state may have null
	// but plan has the config value. Using plan ensures one apply settles it permanently.
	filePath := plan.FilePath
	// additional_file follows file_path exactly: a local, upload-only field. Preserve
	// the plan (config) value so the post-import null→value settle lands from config
	// and no drift is shown thereafter.
	additionalFiles := plan.AdditionalFiles
	// Preserve file_sha256 from state — never recompute during Update because the plan
	// uses UseStateForUnknown and expects the value to stay unchanged. For imported
	// resources where file_sha256 starts as null, it stays null (file_sha256 is only
	// computed fresh during Create).
	fileSHA256 := state.FileSHA256

	r.mapAssetToState(&plan, asset)

	plan.FilePath = filePath
	plan.AdditionalFiles = additionalFiles
	plan.FileSHA256 = fileSHA256

	// Extract api_version from attributes (some types like GraphQL don't expose it)
	if apiVer := extractAttributeValue(asset.Attributes, "api-version"); apiVer != "" {
		plan.APIVersion = types.StringValue(apiVer)
	}
	// else: preserve plan value (create-only field, already set from plan)

	// Extract classifier and mainFile from files array. In MULTI-FILE mode
	// (additional_file set) skip the first-file reconciliation and keep the plan
	// values (frozen from state via UseStateForUnknown) — see the matching guard in
	// Read for why the first-file heuristic is ambiguous with several user files.
	if plan.AdditionalFiles.IsNull() || plan.AdditionalFiles.IsUnknown() {
		classifierVal, mainFileVal := extractFileMetadata(asset.Files)
		if classifierVal != "" {
			plan.Classifier = types.StringValue(normalizeClassifier(classifierVal, state.Classifier))
		} else if plan.Classifier.IsUnknown() {
			plan.Classifier = types.StringNull()
		}
		if mainFileVal != "" {
			plan.MainFile = types.StringValue(mainFileVal)
		} else if plan.MainFile.IsUnknown() {
			// API doesn't have mainFile for this type — set to null so state is concrete
			plan.MainFile = types.StringNull()
		}
	} else {
		// Multi-file: ensure classifier/main_file are concrete even if config omitted
		// them (Optional+Computed → could be unknown at plan when never set).
		if plan.Classifier.IsUnknown() {
			plan.Classifier = types.StringNull()
		}
		if plan.MainFile.IsUnknown() {
			plan.MainFile = types.StringNull()
		}
	}

	// Ensure api_version is concrete (not Unknown) after Update
	if plan.APIVersion.IsUnknown() {
		plan.APIVersion = types.StringNull()
	}

	// Keywords: preserve from plan (user's config value)
	if plan.Keywords.IsNull() || plan.Keywords.IsUnknown() || plan.Keywords.ValueString() == "" {
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

	// Delete external instances first. They live in api-metadata-service keyed by
	// versionGroup and are NOT cascade-deleted with the asset version, so leaving them
	// orphans the instances — a later recreate at the same versionGroup then fails with
	// 409 EXTERNAL_API_CONFLICT ("instance already exists"). We read the live set
	// (best-effort) so we also clean up instances that never made it into state.
	//
	// LIMITATION: the set to delete comes from a VERSION-scoped GetAsset(...version).Instances,
	// but the DELETE endpoint is versionGroup-scoped. If two managed asset VERSIONS share a
	// versionGroup and the API reports the same external instance under both, deleting one
	// version would remove an instance the sibling version still references (the sibling then
	// shows drift on its next plan). Observed behaviour is version-scoped instances (one asset
	// version == one instance set), so this is currently latent; revisit if a group-wide LIST
	// endpoint is added. We do NOT skip the cleanup on this hypothesis — skipping reintroduces
	// the verified 409 orphan bug, which is strictly worse than the unverified sibling case.
	if versionGroup, existing, listErr := r.fetchExternalInstances(ctx,
		state.GroupID.ValueString(), state.AssetID.ValueString(), state.Version.ValueString()); listErr == nil {
		for _, inst := range existing {
			instanceID := inst.InstanceID.ValueString()
			if instanceID == "" {
				continue
			}
			if delErr := r.client.DeleteExternalInstance(ctx,
				state.GroupID.ValueString(), state.AssetID.ValueString(), versionGroup, instanceID); delErr != nil {
				resp.Diagnostics.AddError(
					"Error deleting exchange asset",
					fmt.Sprintf("Could not delete external instance %q before removing the asset: %s", inst.Name.ValueString(), delErr.Error()),
				)
				return
			}
		}
	}

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

	groupID := parts[0]
	assetID := parts[1]
	version := parts[2]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("asset_id"), assetID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), version)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), groupID)...)

	// Fetch the asset from the API to seed immutable fields so that the first plan
	// after import shows zero drift (avoids RequiresReplace on file_path, classifier, etc.)
	asset, err := r.client.GetAsset(ctx, groupID, assetID, version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Exchange Asset During Import",
			"Could not read asset to seed state: "+err.Error(),
		)
		return
	}

	// Seed classifier and main_file from the files array
	classifier, mainFile := extractFileMetadata(asset.Files)
	if classifier != "" {
		// Store the user-facing classifier (e.g. "oas" not "fat-oas",
		// "raml-fragment" not "fat-raml-fragment").
		userClassifier := apiClassifierToUserClassifier(classifier)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("classifier"), userClassifier)...)
	}
	if mainFile != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("main_file"), mainFile)...)
	}

	// Seed api_version from the attributes array. For GraphQL and some types
	// api-version is not present there (the API stores it as properties.apiVersion);
	// in that case there is nothing to seed here — Read reconciles the value from
	// config during plan via the custom plan modifier.
	if apiVer := extractAttributeValue(asset.Attributes, "api-version"); apiVer != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("api_version"), apiVer)...)
	}

	// Seed file_path as a sentinel value so RequiresReplace doesn't trigger.
	// The actual local path comes from config; we just need state to be non-null
	// so the plan modifier sees "old value exists" and compares properly.
	// We use a special marker that the plan modifier recognizes.
	// Actually, we handle this via useConfigValueAfterImport plan modifier instead.
}

// --- Helpers ---

// apiTypeToUserType maps API response type names back to the user-facing type names
// used in the create request. The Exchange API normalizes some types differently in responses.
var apiTypeToUserType = map[string]string{
	"graphql": "graphql-api",
}

// userTypeToAPIType maps user-facing type names to what the API stores/returns,
// so normalizeType can PRESERVE the user's declared type when Exchange normalizes
// it to a different stored value on publish.
//
// The "extension" super-type: Exchange stores the whole mule-plugin family under
// the single stored type "extension". A policy or a connector is a JAR published
// with the mule-plugin bundle classifier; users declare the semantically
// meaningful "policy"/"connector", but the API stores AND returns "extension".
// Without a mapping, Create's readback rewrites state to "extension" and Terraform
// fails the post-apply check: `inconsistent result after apply: .type: was
// "policy", but now "extension"` (STGX EXC-01i, 2026-08-13). Mapping these to
// "extension" makes normalizeType return the user's value instead.
//
// This is SAFE even if a given member is NOT actually normalized on some tenant
// (i.e. stored as-is): normalizeType's identity branch (userType == apiType) still
// returns the user's value, so the mapping can only ADD tolerance, never break the
// as-is case. NOTE the inverse is intentionally ambiguous — "extension" maps back
// to several user types — so it is NOT added to apiTypeToUserType; a bare import of
// a mule-plugin asset therefore surfaces the stored "extension" (see normalizeType).
var userTypeToAPIType = map[string]string{
	"graphql-api": "graphql",
	"policy":      "extension",
	"connector":   "extension",
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

// apiClassifierToUserClassifier converts an API-stored classifier back to the
// user-facing value. When the Exchange API ingests an API-spec file it bundles
// the spec (inlining dependencies) and stores the result with a "fat-" prefix:
// user "raml" -> stored "fat-raml", "oas" -> "fat-oas", "raml-fragment" ->
// "fat-raml-fragment", "oas-components" -> "fat-oas-components", etc. Stripping
// the "fat-" prefix is the general inverse and covers every current and future
// spec family, not just the two we happened to hardcode originally. Classifiers
// that are not bundled (wsdl, graphql, json-schema, proto, custom, ...) have no
// "fat-" prefix and pass through unchanged.
func apiClassifierToUserClassifier(apiClassifier string) string {
	return strings.TrimPrefix(apiClassifier, "fat-")
}

// normalizeClassifier resolves the API response classifier to the user-facing value.
// If the state already has a classifier set (from user config), preserve it to avoid
// perpetual drift / forced replacement when the API bundles the spec into "fat-*".
func normalizeClassifier(apiClassifier string, currentStateClassifier types.String) string {
	if !currentStateClassifier.IsNull() && !currentStateClassifier.IsUnknown() {
		userClassifier := currentStateClassifier.ValueString()
		// API returned exactly what the user set.
		if userClassifier == apiClassifier {
			return userClassifier
		}
		// API bundled the user's classifier with the "fat-" prefix — preserve theirs.
		if apiClassifier == "fat-"+userClassifier {
			return userClassifier
		}
	}
	// Otherwise (e.g. import): strip the "fat-" bundling prefix if present.
	return apiClassifierToUserClassifier(apiClassifier)
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

// additionalFilesToUploads converts the additional_file plan list into the client's
// ExtraFiles slice. Returns nil (no extra files) when the list is null/unknown/empty —
// the normal case for every single-file and fileless type. Each entry maps path+classifier
// to an exchange.AssetFileUpload, which buildAssetMultipart writes as its own
// files.{classifier}.{ext} part in the same publish request as the primary file.
func additionalFilesToUploads(ctx context.Context, list types.List) ([]exchange.AssetFileUpload, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var models []AdditionalFileModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(models) == 0 {
		return nil, diags
	}
	uploads := make([]exchange.AssetFileUpload, 0, len(models))
	for _, m := range models {
		uploads = append(uploads, exchange.AssetFileUpload{
			FilePath:   m.Path.ValueString(),
			Classifier: m.Classifier.ValueString(),
		})
	}
	return uploads, diags
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

// reorderByKey reorders apiItems so their order matches desiredKeys (matched items
// first, in desiredKeys order), then appends any apiItems whose key is not present in
// desiredKeys (preserving their original API order).
//
// WHY THIS EXISTS: several computed collections (pages, categories, custom_fields,
// tags) carry the UseStateForUnknown plan modifier so unrelated in-place updates don't
// render them as "(known after apply)". UseStateForUnknown is a plan-time PROMISE that
// the applied value will equal the frozen plan value. The readback helpers re-derive
// these collections from the API on every Update, and the API does not guarantee
// element ORDER. If the readback emitted a different order than the frozen plan,
// Terraform would fail with "Provider produced inconsistent result after apply"
// (planned order != applied order) — even though the SET of elements is identical.
// Reordering the readback to the plan/state model order makes applied == frozen plan.
// (This mirrors the long-standing reorder in readInstancesIntoState.)
//
// desiredKeys comes from the model the readback writes into (plan on Update, state on
// Read): on a real edit it is the config order; on the omit path it is the
// UseStateForUnknown-frozen prior-state order. Either way, matching it is correct.
// The returned slice is always a PERMUTATION of apiItems — it never adds, drops, or
// de-duplicates elements (doing so would itself produce an inconsistent-result error
// against the frozen plan). Multiplicity and per-key API order are preserved: if the
// API returns a key twice, both copies survive, matched copies going into the desired
// section and any extras appended afterward.
func reorderByKey[T any](apiItems []T, desiredKeys []string, keyOf func(T) string) []T {
	if len(apiItems) == 0 || len(desiredKeys) == 0 {
		return apiItems
	}
	// Bucket API items by key, preserving multiplicity and original order within a key.
	buckets := make(map[string][]T, len(apiItems))
	apiKeyOrder := make([]string, 0, len(apiItems)) // distinct keys in first-seen API order
	for _, it := range apiItems {
		k := keyOf(it)
		if _, ok := buckets[k]; !ok {
			apiKeyOrder = append(apiKeyOrder, k)
		}
		buckets[k] = append(buckets[k], it)
	}
	ordered := make([]T, 0, len(apiItems))
	// First: emit one item per desiredKeys entry, draining that key's bucket (so a key
	// repeated in desiredKeys consumes multiple API copies, in API order).
	for _, k := range desiredKeys {
		if b := buckets[k]; len(b) > 0 {
			ordered = append(ordered, b[0])
			buckets[k] = b[1:]
		}
	}
	// Then: append every remaining (unconsumed) API item, in original API key order.
	for _, k := range apiKeyOrder {
		ordered = append(ordered, buckets[k]...)
	}
	return ordered
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

	// contact_name / contact_email / manager carry UseStateForUnknown, so on an
	// omit-path update the plan is frozen to the prior value and the Update readback
	// MUST reproduce that exact value. When the API returns no value we therefore
	// preserve the prior (user-set) value verbatim — re-materialised via StringValue so
	// the readback is idempotent: f(known x) == x for both "Alice" and "". The earlier
	// code forced "" here, which is NOT a fixed point (f("Alice")="" but f("")=null) and
	// would fail apply with "Provider produced inconsistent result after apply"; it also
	// caused perpetual churn when config set a value the API did not echo back. IsUnknown
	// is excluded so an omitted value at Create still settles to a concrete null.
	if asset.ContactName != nil && *asset.ContactName != "" {
		state.ContactName = types.StringValue(*asset.ContactName)
	} else if !state.ContactName.IsNull() && !state.ContactName.IsUnknown() {
		state.ContactName = types.StringValue(state.ContactName.ValueString())
	} else {
		state.ContactName = types.StringNull()
	}
	if asset.ContactEmail != nil && *asset.ContactEmail != "" {
		state.ContactEmail = types.StringValue(*asset.ContactEmail)
	} else if !state.ContactEmail.IsNull() && !state.ContactEmail.IsUnknown() {
		state.ContactEmail = types.StringValue(state.ContactEmail.ValueString())
	} else {
		state.ContactEmail = types.StringNull()
	}
	if asset.Manager != nil && *asset.Manager != "" {
		state.Manager = types.StringValue(*asset.Manager)
	} else if !state.Manager.IsNull() && !state.Manager.IsUnknown() {
		state.Manager = types.StringValue(state.Manager.ValueString())
	} else {
		state.Manager = types.StringNull()
	}

	// Map tags (labels) from API response — labels are simple strings.
	// Reorder to match the model's prior tag order (config order on Create/edit, or the
	// UseStateForUnknown-frozen prior order on an omit-path Update). The labels API does
	// not guarantee order; tags carries UseStateForUnknown, so a differently-ordered
	// readback would fail with "Provider produced inconsistent result after apply". A
	// label IS its own key, so keyOf is identity. Elements() is used (not ElementsAs) to
	// keep this helper ctx-free; it returns an empty slice when Tags is null/unknown.
	if len(asset.Labels) > 0 {
		var priorTagOrder []string
		for _, e := range state.Tags.Elements() {
			if s, ok := e.(types.String); ok {
				priorTagOrder = append(priorTagOrder, s.ValueString())
			}
		}
		orderedLabels := reorderByKey(asset.Labels, priorTagOrder, func(s string) string { return s })

		tagValues := make([]attr.Value, len(orderedLabels))
		for i, label := range orderedLabels {
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

	// Reorder to match the model's page order (config order on Update, or the
	// UseStateForUnknown-frozen prior order otherwise). The portal-pages API does not
	// guarantee element order; without this, the pages attribute carries
	// UseStateForUnknown, so a differently-ordered readback would fail with "Provider
	// produced inconsistent result after apply". Keyed by page_name.
	var desiredPageNames []string
	if !state.Pages.IsNull() && !state.Pages.IsUnknown() {
		var currentPages []PageModel
		if diags := state.Pages.ElementsAs(ctx, &currentPages, false); !diags.HasError() {
			for _, cp := range currentPages {
				desiredPageNames = append(desiredPageNames, cp.PageName.ValueString())
			}
		}
	}
	userPages = reorderByKey(userPages, desiredPageNames, func(p exchange.PortalPage) string { return p.Name })

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

	// Create or update pages. Iterate the ORDERED desiredPages slice (config order),
	// NOT the desiredByName map — Exchange fixes portal display order to page CREATION
	// order (there is no reorder endpoint; verified live), so creating pages in config
	// order is the only way to control what a customer sees in the Exchange portal.
	// desiredByName is retained only for O(1) existence lookups above/below.
	for _, desired := range desiredPages {
		if desired.PageName.IsNull() || desired.PageName.IsUnknown() {
			continue
		}
		name := desired.PageName.ValueString()
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
			// New page — create then set content.
			page, err := r.client.CreateDraftPage(ctx, groupID, assetID, version, name)
			if err != nil {
				if client.IsConflict(err) {
					// The page already exists in the draft even though it is not in our
					// "current" set. Exchange auto-provisions a "home" page on every asset
					// version, so a fresh Create (which passes no current pages) hits this
					// 409 on the very first release. Rather than fail the whole apply,
					// adopt the existing page: resolve its real draft path and upsert the
					// desired content. Mirrors the external-instances idempotent-create
					// reconcile (see syncInstances / client.IsConflict).
					existingPath, lookupErr := r.lookupDraftPagePath(ctx, groupID, assetID, version, name)
					if lookupErr != nil {
						return fmt.Errorf("failed to adopt existing page '%s': %w", name, lookupErr)
					}
					if existingPath == "" {
						return fmt.Errorf("page '%s' already exists but was not found in the draft page listing", name)
					}
					if content != "" {
						if err := r.client.UpdateDraftPageContent(ctx, groupID, assetID, version, existingPath, content); err != nil {
							return fmt.Errorf("failed to set content for existing page '%s': %w", name, err)
						}
					}
					needsPublish = true
					continue
				}
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

// lookupDraftPagePath returns the concrete draft page path for a page identified by
// its name (e.g. "home"), matching either the page's name or the trailing segment of
// its path. CreateDraftPage POSTs into the DRAFT namespace, and Exchange assigns the
// real path (which may carry a prefix), so we resolve it from the draft listing rather
// than the published listing (an auto-provisioned page may not be published yet).
// Returns "" (no error) when the page is not present in the draft.
func (r *AssetResource) lookupDraftPagePath(ctx context.Context, groupID, assetID, version, pageName string) (string, error) {
	pages, err := r.client.ListDraftPages(ctx, groupID, assetID, version)
	if err != nil {
		return "", err
	}
	for _, p := range pages {
		if p.Name == pageName || p.Path == pageName {
			return p.Path, nil
		}
	}
	// Fall back to matching the trailing path segment (paths may be prefixed).
	for _, p := range pages {
		if idx := strings.LastIndex(p.Path, "/"); idx >= 0 && p.Path[idx+1:] == pageName {
			return p.Path, nil
		}
	}
	return "", nil
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

// fetchExternalInstances returns the external (non-managed) instances currently
// attached to the asset's version group, as InstanceModel values with InstanceID
// populated, plus the asset's versionGroup.
//
// External instances live in api-metadata-service keyed by versionGroup and are NOT
// cascade-deleted when an asset version is deleted. A recreate at the same versionGroup
// can therefore encounter pre-existing ("orphaned") instances left behind by a prior
// delete. Reading the live set lets Create and Delete reconcile against reality instead
// of assuming a clean slate — without it, Create blindly re-POSTs and the platform
// rejects it with 409 API_METADATA_EXTERNAL_API_CONFLICT ("... already exists").
func (r *AssetResource) fetchExternalInstances(ctx context.Context, groupID, assetID, version string) (string, []InstanceModel, error) {
	asset, err := r.client.GetAsset(ctx, groupID, assetID, version)
	if err != nil {
		return "", nil, err
	}
	var out []InstanceModel
	for _, inst := range asset.Instances {
		instMap, ok := inst.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := instMap["type"].(string); t != "external" {
			continue
		}
		name, _ := instMap["name"].(string)
		endpoint, _ := instMap["endpointUri"].(string)
		isPublic, _ := instMap["isPublic"].(bool)
		id, _ := instMap["id"].(string)
		out = append(out, InstanceModel{
			Name:        types.StringValue(name),
			EndpointURI: types.StringValue(endpoint),
			IsPublic:    types.BoolValue(isPublic),
			InstanceID:  types.StringValue(id),
		})
	}
	return asset.VersionGroup, out, nil
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
				// Idempotent create: an instance with this name already exists on the
				// version group — typically orphaned by a prior asset delete, since
				// external instances are NOT cascade-deleted with the asset version.
				// Rather than fail the whole apply with 409 EXTERNAL_API_CONFLICT, treat
				// it as already-present and continue. The caller's readInstancesIntoState
				// (which reads with the concrete version) then reflects the actual
				// instance into state; if its endpoint/visibility differs from config,
				// the next plan shows drift and Update reconciles it via PATCH.
				if client.IsConflict(err) {
					continue
				}
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

	// Reorder to match the model's category order (config order on Update, or the
	// UseStateForUnknown-frozen prior order otherwise). The asset API does not guarantee
	// category order; without this, the categories attribute carries UseStateForUnknown,
	// so a differently-ordered readback would fail with "Provider produced inconsistent
	// result after apply". Keyed by category key.
	var desiredCatKeys []string
	// priorCatValues maps a category key to its prior inner-values order. The parent
	// UseStateForUnknown freezes the WHOLE nested value (including each category's inner
	// `values` list), so the inner list order must ALSO match on apply or Terraform fails
	// with "inconsistent result after apply". The asset API does not guarantee inner value
	// order either, so we reorder each category's values to its prior order below.
	priorCatValues := make(map[string][]string)
	if !state.Categories.IsNull() && !state.Categories.IsUnknown() {
		var currentCats []CategoryModel
		if diags := state.Categories.ElementsAs(ctx, &currentCats, false); !diags.HasError() {
			for _, cc := range currentCats {
				k := cc.Key.ValueString()
				desiredCatKeys = append(desiredCatKeys, k)
				var vals []string
				if d := cc.Values.ElementsAs(ctx, &vals, false); !d.HasError() {
					priorCatValues[k] = vals
				}
			}
		}
	}
	orderedCats := reorderByKey(asset.Categories, desiredCatKeys, func(c exchange.Category) string { return c.Key })

	catValues := make([]attr.Value, 0, len(orderedCats))
	for _, cat := range orderedCats {
		// Build the values list, reordered to the prior inner order (see priorCatValues).
		orderedVals := reorderByKey(cat.Value, priorCatValues[cat.Key], func(s string) string { return s })
		valueAttrs := make([]attr.Value, len(orderedVals))
		for i, v := range orderedVals {
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

	// Reorder to match the model's custom-field order (config order on Update, or the
	// UseStateForUnknown-frozen prior order otherwise). The asset API does not guarantee
	// custom-field order; without this, the custom_fields attribute carries
	// UseStateForUnknown, so a differently-ordered readback would fail with "Provider
	// produced inconsistent result after apply". Keyed by field key.
	var desiredFieldKeys []string
	// priorFieldValues maps a field key to its prior inner-values order — same rationale as
	// categories: the parent UseStateForUnknown freezes each field's inner `values` list, so
	// the inner order must also match on apply to avoid "inconsistent result after apply".
	priorFieldValues := make(map[string][]string)
	if !state.CustomFields.IsNull() && !state.CustomFields.IsUnknown() {
		var currentFields []CustomFieldModel
		if diags := state.CustomFields.ElementsAs(ctx, &currentFields, false); !diags.HasError() {
			for _, cf := range currentFields {
				k := cf.Key.ValueString()
				desiredFieldKeys = append(desiredFieldKeys, k)
				var vals []string
				if d := cf.Values.ElementsAs(ctx, &vals, false); !d.HasError() {
					priorFieldValues[k] = vals
				}
			}
		}
	}
	orderedFields := reorderByKey(asset.CustomFields, desiredFieldKeys, func(f exchange.CustomField) string { return f.Key })

	fieldValues := make([]attr.Value, 0, len(orderedFields))
	for _, field := range orderedFields {
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

		// Reorder to the prior inner order (see priorFieldValues).
		values = reorderByKey(values, priorFieldValues[field.Key], func(s string) string { return s })
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

// --- Custom Plan Modifiers ---

// requiresReplaceExceptOnImport is a plan modifier that triggers RequiresReplace ONLY when
// the prior state value is non-null AND differs from the plan value. After import,
// file_path/api_version may be null in state (the API doesn't store local paths or
// certain metadata for all types). Going from null → value should NOT force recreation
// because the asset already exists with the correct content.
//
// After import, `terraform plan` may show an in-place update for file_path (null → "path/...").
// This is a NON-DESTRUCTIVE change — one `terraform apply` settles it permanently.
// For classifier/api_version/main_file, ImportState seeds from API so no diff is shown.
type requiresReplaceExceptOnImport struct{}

func (m requiresReplaceExceptOnImport) Description(_ context.Context) string {
	return "Requires replacement only when changing from one non-null value to another (not after import)."
}

func (m requiresReplaceExceptOnImport) MarkdownDescription(_ context.Context) string {
	return "Requires replacement only when changing from one non-null value to another (not after import)."
}

func (m requiresReplaceExceptOnImport) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the resource is being created (no prior state), no action needed
	if req.State.Raw.IsNull() {
		return
	}

	// If the plan is being destroyed, no action needed
	if req.Plan.Raw.IsNull() {
		return
	}

	// If the plan value is unknown, we can't compare yet
	if req.PlanValue.IsUnknown() {
		return
	}

	// If old value is null/unknown (post-import state), do NOT trigger replacement.
	// The asset already exists — file_path is a local-only field used at creation.
	// Plan will show an in-place update (null→value) which is non-destructive.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// Both old and new are set. If they differ, require replacement.
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// RequiresReplaceExceptOnImport returns a plan modifier that requires replacement
// only when changing from one non-null value to another. Null → value transitions
// (which occur after import) do NOT trigger replacement.
func RequiresReplaceExceptOnImport() planmodifier.String {
	return requiresReplaceExceptOnImport{}
}

// apiTypeFor returns the API-STORED type for a user-facing type name, so that alias
// members of the same Exchange super-type compare equal. The mule-plugin family
// (policy, connector) is all stored as "extension" (see userTypeToAPIType), so
// apiTypeFor("policy") == apiTypeFor("connector") == apiTypeFor("extension") ==
// "extension". Any type without a mapping passes through unchanged (identity), so a
// plain type like "rest-api" is only ever equal to itself.
func apiTypeFor(userType string) string {
	if api, ok := userTypeToAPIType[userType]; ok {
		return api
	}
	return userType
}

// requiresReplaceOnTypeChange forces replacement when the asset `type` changes to a
// GENUINELY different Exchange type — comparing the API-normalized form (apiTypeFor)
// rather than the raw string. Exchange versions are immutable, so a real cross-type
// change (rest-api -> soap-api) must destroy+recreate. But members of the same
// super-type are NOT a real change: Exchange stores the whole mule-plugin family
// (policy, connector) as "extension", and a bare `terraform import` can only ever
// surface the stored "extension" (the semantic sub-type is unrecoverable from the
// API). Declaring `type = "policy"` against an imported state of "extension" therefore
// used to force a destroy+recreate of a live asset — catastrophic and pointless, since
// the platform object is identical. Comparing the normalized form makes that edit an
// in-place reconcile (Update path), which mapAssetToState+normalizeType then settle
// back to the user's declared "policy" with no drift.
type requiresReplaceOnTypeChange struct{}

func (m requiresReplaceOnTypeChange) Description(_ context.Context) string {
	return "Requires replacement only when the API-normalized asset type actually changes; mule-plugin aliases (policy/connector/extension) are equivalent and reconcile in place."
}

func (m requiresReplaceOnTypeChange) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m requiresReplaceOnTypeChange) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Create (no prior state): nothing to compare; value is computed fresh.
	if req.State.Raw.IsNull() {
		return
	}
	// Destroy: no plan value to compare.
	if req.Plan.Raw.IsNull() {
		return
	}
	// Plan still unknown (e.g. config omitted type before UseStateForUnknown ran):
	// can't compare yet, and the omit path must never force replacement.
	if req.PlanValue.IsUnknown() {
		return
	}
	// Post-import state can be null/unknown for some fields; be defensive.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	// Replace ONLY when the normalized (API-stored) types differ. This is what lets
	// policy/connector/extension coexist without a spurious replace after import.
	if apiTypeFor(req.PlanValue.ValueString()) != apiTypeFor(req.StateValue.ValueString()) {
		resp.RequiresReplace = true
	}
}

// RequiresReplaceOnTypeChange returns a plan modifier that requires replacement only
// when the API-normalized asset type changes. See requiresReplaceOnTypeChange.
func RequiresReplaceOnTypeChange() planmodifier.String {
	return requiresReplaceOnTypeChange{}
}

// requiresReplaceListExceptOnImport is the List-typed sibling of
// requiresReplaceExceptOnImport, used for the additional_file block. Same rule:
// the uploaded file set is immutable (a change destroys+recreates the version),
// but a null→value transition after import is a non-destructive settle and must
// NOT force replacement. ImportState does not seed additional_file, so the first
// apply after importing a multi-file asset goes null→value here.
type requiresReplaceListExceptOnImport struct{}

func (m requiresReplaceListExceptOnImport) Description(_ context.Context) string {
	return "Requires replacement only when changing from one non-null value to another (not after import)."
}

func (m requiresReplaceListExceptOnImport) MarkdownDescription(_ context.Context) string {
	return "Requires replacement only when changing from one non-null value to another (not after import)."
}

func (m requiresReplaceListExceptOnImport) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Being created (no prior state): nothing to compare.
	if req.State.Raw.IsNull() {
		return
	}
	// Being destroyed: nothing to compare.
	if req.Plan.Raw.IsNull() {
		return
	}
	// Plan value not yet resolved: can't compare.
	if req.PlanValue.IsUnknown() {
		return
	}
	// Prior state null/unknown (post-import): null→value is a non-destructive settle.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	// Both set and different: the uploaded file set changed → replace.
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

// RequiresReplaceListExceptOnImport returns the List-typed plan modifier that
// requires replacement only when changing from one non-null value to another.
func RequiresReplaceListExceptOnImport() planmodifier.List {
	return requiresReplaceListExceptOnImport{}
}
