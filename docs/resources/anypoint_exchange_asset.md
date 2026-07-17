---
page_title: "anypoint_exchange_asset Resource - terraform-provider-anypoint"
subcategory: "Exchange"
description: |-
  Manages an Exchange asset (publish, update metadata, delete). Asset versions are immutable — changing the version, type, or file triggers a replacement.
---

# anypoint_exchange_asset (Resource)

Manages an Exchange asset (publish, update metadata, delete). Asset versions are immutable — changing the version, type, or file triggers a replacement.

-> **Immutable versions:** `version`, `type`, and `file_path` are replacement-forcing. Changing any of them destroys the existing asset version and creates a new one. To publish an additional version side-by-side (like the Exchange UI "Add version" button), add a separate resource block rather than editing `version` in place.

-> **Recreate safety (`create_before_destroy`):** A replacement destroys the current version with a **hard delete** and *then* publishes the replacement — so a failed publish can leave you with neither (no rollback). When the replacement is triggered by a **`version` bump**, the new version is a distinct GAV that can coexist with the old one, so adding `lifecycle { create_before_destroy = true }` flips the order (publish the new version first, delete the old one only after it succeeds) and closes that window. Do **not** rely on this for a *same-version* replacement — changing `file_path`, `type`, `classifier`, `api_version`, or `main_file` while `version` stays the same re-publishes to the **same version URL**, so create-before-destroy would collide with (or delete) the version it just wrote. For an in-place spec/file change, bump `version` as well so the backstop applies.

-> **Classifier normalization:** `classifier` is the user-facing value (e.g. `oas`). Exchange stores some classifiers bundled with a `fat-` prefix (e.g. `fat-oas`); the provider normalizes it back on read, so there is no perpetual diff.

-> **Authoritative instances:** `instances` are managed authoritatively — an instance removed from the list is deleted from the api-metadata-service on the next apply.

## Example Usage

### Metadata-only custom asset

```terraform
resource "anypoint_exchange_asset" "custom" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-custom"
  version         = "1.0.0"
  name            = "TF Demo Custom Asset"
  type            = "custom"
  description     = "A custom Exchange asset published by Terraform."
  keywords        = "terraform,demo,custom"
}
```

### REST API with a spec file, tags, docs, and external instances

```terraform
resource "anypoint_exchange_asset" "rest_api" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-rest-api"
  version         = "1.0.0"
  name            = "TF Demo REST API"
  type            = "rest-api"

  classifier = "oas"
  file_path  = "${path.module}/test-assets/petstore.json"
  main_file  = "petstore.json"

  tags          = ["terraform", "demo"]
  contact_name  = "Platform Team"
  contact_email = "platform@example.com"

  pages = [
    {
      page_name = "Overview"
      content   = "# Overview\n\nTerraform-managed documentation page."
    }
  ]

  instances = [
    {
      name         = "Production"
      endpoint_uri = "https://api.example.com/petstore"
      is_public    = true
    },
    {
      name         = "Sandbox"
      endpoint_uri = "https://sandbox.example.com/petstore"
    },
  ]

  terms_and_conditions = "# Terms and Conditions\n\nProvided as-is."
}
```

## Schema

### Required

- `organization_id` (String) The organization ID to publish the asset to.
- `group_id` (String) The group ID of the asset (usually the same as `organization_id`).
- `asset_id` (String) The asset ID (slug/identifier). Must be unique within the group.
- `version` (String) The semantic version of the asset (e.g. `1.0.0`). Asset versions are immutable.
- `name` (String) The display name of the asset.

### Optional

- `type` (String) The asset type: `custom`, `rest-api`, `http-api`, `evented-api` (AsyncAPI), `graphql-api`, `connector`, `app`, `template`, `example`, `policy`, `agent`, `llm`, `mcp`.
- `status` (String) The lifecycle status: `published` (default) or `development`.
- `description` (String) A description of the asset.
- `keywords` (String) Comma-separated keywords for search discovery.
- `contact_name` (String) Contact person name for this asset.
- `contact_email` (String) Contact email for this asset.
- `manager` (String) Manager for this asset.
- `api_version` (String) The API version (`properties.apiVersion`). Used for API spec asset types.
- `classifier` (String) The file classifier: `custom`, `raml`, `oas`, `wsdl`, `graphql`, etc. Required when `file_path` is set.
- `file_path` (String) Path to the file to upload (JAR, ZIP, RAML, OAS, etc.). Used only at creation time. After import, one apply settles this field (non-destructive). Changing to a different value triggers replacement.
- `main_file` (String) The main file within the uploaded archive (`properties.mainFile`). Used for multi-file specs.
- `tags` (List of String) Search tags for the asset version. Each element is a tag value string.
- `terms_and_conditions` (String) Terms and conditions content (markdown). Displayed as the T&C page in the asset portal.
- `pages` (Block List) Documentation pages for the asset portal. Each page has a name and markdown content. See [`pages`](#nestedschema--pages) below.
- `instances` (Block List) Non-managed (external) API instances for this asset version. See [`instances`](#nestedschema--instances) below.
- `categories` (Block List) Category assignments on this asset version. Categories are org-level taxonomy (the category key must already exist in the org via the Exchange UI or API). Each entry assigns one or more values to a category key. See [`categories`](#nestedschema--categories) below.
- `custom_fields` (Block List) Custom field assignments on this asset version. Custom fields are org-level metadata (the field key must already exist in the org via the Exchange UI or API). Each entry assigns one or more values to a custom field key. See [`custom_fields`](#nestedschema--custom_fields) below.

### Read-Only

- `id` (String) The composite identifier (`groupId/assetId/version`).
- `minor_version` (String) The minor version (e.g. `1.0`).
- `version_group` (String) The version group.
- `is_public` (Boolean) Whether the asset is publicly visible.
- `is_snapshot` (Boolean) Whether this is a snapshot version.
- `file_sha256` (String) SHA256 hash of the uploaded file (for drift detection).
- `created_date` (String) When the asset was created.
- `updated_date` (String) When the asset was last updated.

<a id="nestedschema--pages"></a>
### Nested Schema for `pages`

Required:

- `page_name` (String) The page name (used as URL slug). Cannot contain: `% @ * + / _ \`
- `content` (String) The markdown content of the page.

Read-Only:

- `page_path` (String) The full page path assigned by the API (includes random prefix). Computed after creation.

<a id="nestedschema--instances"></a>
### Nested Schema for `instances`

Required:

- `name` (String) The display name of the instance (e.g. `Production`, `Sandbox`).
- `endpoint_uri` (String) The endpoint URL of the external instance.

Optional:

- `is_public` (Boolean) Whether this instance is publicly visible. Defaults to `false`.

Read-Only:

- `instance_id` (String) The unique ID assigned by the API. Computed after creation.

<a id="nestedschema--categories"></a>
### Nested Schema for `categories`

Required:

- `key` (String) The category key (must match an existing org-level category definition).
- `values` (List of String) The category values to assign (e.g. `["Finance", "HR"]`).

<a id="nestedschema--custom_fields"></a>
### Nested Schema for `custom_fields`

Required:

- `key` (String) The custom field key (must match an existing org-level field definition).
- `values` (List of String) The field values to assign (e.g. `["v1.2", "stable"]`).

## Import

An existing Exchange asset version can be imported using its composite ID: `group_id/asset_id/version` (`group_id` is usually the organization ID).

On import the provider reads the live asset and seeds the immutable fields (`classifier`, `main_file`, `api_version`) and external instances into state, so the first plan after import shows zero drift. The local `file_path` settles on the next apply without recreating the asset.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_exchange_asset.imported
  id = "<group_id>/<asset_id>/<version>" # e.g. "6c3c4eb3-.../tf-demo-rest-api/1.0.0"
}

resource "anypoint_exchange_asset" "imported" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "<asset_id>"
  version         = "<version>"
  name            = "<name>"
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_exchange_asset.imported <group_id>/<asset_id>/<version>
```
