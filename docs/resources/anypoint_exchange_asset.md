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

### Managing multiple versions with `for_each`

There is no separate "asset version" resource — one `anypoint_exchange_asset` block already **is** one GAV (group / asset / version). To manage several versions of the same asset the way the Exchange UI's "Add version" button does, drive **one** resource block with `for_each` over a map of versions:

- **Add a map key** → publishes a **new** version (additive, like "Add version").
- **Remove a map key** → hard-deletes **that version only** (the others are untouched).
- **Edit a key's `version` string in place** → forces **replacement** of that entry (the destructive path — see the caveats).

Use a stable, human-meaningful map key (e.g. `v1`, `v2`) rather than the version number, so bumping `version` does not change the key (which would destroy and recreate under a new key instead of tracking the same instance).

```terraform
locals {
  # GROUP-scoped fields — declared ONCE and reused by every entry so they are
  # guaranteed identical across versions (see caveat 1 below).
  petstore_group = {
    name          = "TF Demo Petstore API (multi-version)"
    description   = "Petstore API published by Terraform, managed across versions with for_each."
    contact_name  = "Platform Team"
    contact_email = "platform@example.com"
  }

  # VERSION-scoped fields — may differ freely between versions.
  petstore_versions = {
    v1 = {
      version   = "1.0.0"
      file_path = "test-assets/petstore.json"
      status    = "published"
      tags      = ["terraform", "petstore", "v1", "stable"]
    }
    v2 = {
      version   = "2.0.0"
      file_path = "test-assets/petstore-v2.json" # a genuinely different spec
      status    = "published"
      tags      = ["terraform", "petstore", "v2", "adds-vaccinations"]
    }
  }
}

resource "anypoint_exchange_asset" "petstore" {
  for_each = local.petstore_versions

  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-petstore-multiversion"
  type            = "rest-api"
  classifier      = "oas"

  # VERSION-scoped — independent per version.
  version   = each.value.version
  file_path = "${path.module}/${each.value.file_path}"
  main_file = basename(each.value.file_path)
  status    = each.value.status
  tags      = each.value.tags

  # GROUP-scoped — identical across every entry (caveat 1).
  name          = local.petstore_group.name
  description   = local.petstore_group.description
  contact_name  = local.petstore_group.contact_name
  contact_email = local.petstore_group.contact_email

  # Recreate safety for the destructive path (caveat 3).
  lifecycle {
    create_before_destroy = true
  }
}
```

**Caveats** (all three were verified against the live platform):

1. **Group-scoped fields must be identical across every entry.** `name`, `description`, `contact_name`, `contact_email`, and `manager` are stored **once per asset**, not per version. If entries disagree, the platform silently keeps the existing group value and drops the others. Factor them into `locals` (as above) so they can never drift apart.
2. **External `instances` are shared within a major version.** Instances live at the version-group (major) level, so all versions that share a major (e.g. `1.0.0` and `1.1.0`) share one instance set. Define instances on a single entry per major (or keep them identical within a major) to avoid a tug-of-war between entries.
3. **`version` is replacement-forcing, so editing a version string in place is destructive.** Prefer adding/removing map keys (purely additive / version-scoped delete). When you must bump a version, `create_before_destroy = true` publishes the new GAV before hard-deleting the old one (safe because a version bump yields a distinct, coexisting GAV), and the `status` `OneOf` validator catches typos like `"Published"` at **plan** time — before any destroy runs.

## Schema

### Required

- `organization_id` (String) The organization ID to publish the asset to.
- `group_id` (String) The group ID of the asset (usually the same as `organization_id`).
- `asset_id` (String) The asset ID (slug/identifier). Must be unique within the group.
- `version` (String) The semantic version of the asset (e.g. `1.0.0`). Asset versions are immutable.
- `name` (String) The display name of the asset.

### Optional

- `type` (String) The asset type: `custom`, `rest-api`, `http-api`, `evented-api` (AsyncAPI), `graphql-api`, `connector`, `app`, `template`, `example`, `policy`, `agent`, `llm`, `mcp`.
- `status` (String) The lifecycle status of this asset **version**. One of `development`, `published` (default), or `deprecated`. **Case-sensitive** (`Published` is rejected). The Exchange API is asymmetric — validated both at plan time (via a `OneOf` validator) and by a plan-time guard: `development` is accepted **only when first publishing a version** and *cannot* be set on an existing version (the platform rejects an in-place change to `development` with HTTP 400); `deprecated` can only be set on an existing version, not at initial publish; `published` is valid in both cases. To move a published version back to `development`, publish a new version (bump `version`) rather than editing status in place.
- `description` (String) A description of the asset. **Group-scoped** — shared across all versions of the asset (see the multi-version note below).
- `keywords` (String) Comma-separated keywords for search discovery.
- `contact_name` (String) Contact person name for this asset. **Group-scoped** — shared across all versions.
- `contact_email` (String) Contact email for this asset. **Group-scoped** — shared across all versions.
- `api_version` (String) The API version (`properties.apiVersion`), e.g. `v1`. REQUIRED at create for the API-spec types `rest-api`, `evented-api`, and `grpc-api` — publishing one of these without api_version fails with `400 MISSING_REQUIRED_PROPERTIES: apiVersion`. This is the human-facing API contract version, distinct from the immutable GAV `version`.
- `classifier` (String) The file classifier: `custom`, `raml`, `oas`, `wsdl`, `graphql`, etc. Required when `file_path` is set.
- `file_path` (String) Path to the file to upload (JAR, ZIP, RAML, OAS, etc.). Used only at creation time. After import, one apply settles this field (non-destructive). Changing to a different value triggers replacement.
- `main_file` (String) The main file within the uploaded archive (`properties.mainFile`). Used for multi-file specs.
- `additional_file` (Block List) Extra files uploaded **alongside** `file_path` in the *same* publish request, for multi-file asset types. The canonical case is `type = "policy"`, which requires two files — e.g. `(mule-policy.jar + policy-definition.yaml)` or `(schema.json + metadata.yaml)`. Like `file_path`, this is a create-time, upload-only field (preserved from state on read, never reconciled from the API) and is **replacement-forcing** — except the non-destructive null→value settle on the first apply after import. See [`additional_file`](#nestedschema--additional_file) below.
- `tags` (List of String) Search tags for the asset version. Each element is a tag value string.
- `terms_and_conditions` (String) Terms and conditions content (markdown). Displayed as the T&C page in the asset portal.
- `pages` (Block List) Documentation pages for the asset portal. Each page has a name and markdown content. See [`pages`](#nestedschema--pages) below.
- `instances` (Block List) Non-managed (external) API instances for this asset version. See [`instances`](#nestedschema--instances) below.
- `categories` (Block List) Category assignments on this asset version. Categories are org-level taxonomy (the category key must already exist in the org via the Exchange UI or API). Each entry assigns one or more values to a category key. See [`categories`](#nestedschema--categories) below.
- `custom_fields` (Block List) Custom field assignments on this asset version. Custom fields are org-level metadata (the field key must already exist in the org via the Exchange UI or API). Each entry assigns one or more values to a custom field key. See [`custom_fields`](#nestedschema--custom_fields) below.

### Read-Only

- `id` (String) The composite identifier (`groupId/assetId/version`).
- `manager` (String) The manager of this asset, as reported by Exchange. **Read-only:** the Exchange API does not permit setting the manager via automation — attempting to do so returns HTTP 403 (username) or HTTP 400 (uuid) — so this attribute cannot be configured; it only reflects a value set elsewhere (e.g. the Exchange UI).
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

~> **Portal pages are flat and ordered by creation (platform limitations).** The Exchange
portal-pages API does **not** support page hierarchy/nesting (a `/` in `page_name` is rejected,
and there is no parent field), and it does **not** expose a reorder operation — pages are
displayed in the order they are first created. This provider therefore creates pages in the
order you list them in the `pages` block, which fixes their portal display order. To change the
order of existing pages you must recreate them (reorder the list *and* accept that a page whose
position changes is deleted and re-created). There is intentionally no `parent`/`order` attribute
because the platform has no API to honor one.

-> **Landing page (`home`).** Exchange auto-provisions a `home` landing page on every asset
version. It is hidden from state until you manage it. To set the portal landing content, add a
page named `home`: `pages = [{ page_name = "home", content = "# Welcome\n..." }, ...]`. The
provider adopts the auto-provisioned page (no create conflict) and publishes your content.

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

<a id="nestedschema--additional_file"></a>
### Nested Schema for `additional_file`

Required:

- `path` (String) Local path to the additional file to upload (e.g. `specs/metadata.yaml`).
- `classifier` (String) The classifier for this file (e.g. `metadata`, `policy-definition`, `schema`). Combined with the file extension to form the `files.{classifier}.{ext}` part name in the publish request.

#### Example: publishing a policy (two files)

```terraform
resource "anypoint_exchange_asset" "policy" {
  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-policy"
  version         = "1.0.0"
  name            = "TF Demo Policy"
  type            = "policy"

  # First file.
  file_path  = "${path.module}/test-assets/mule-policy.jar"
  classifier = "mule-policy"

  # Second file uploaded in the SAME publish request.
  additional_file = [
    {
      path       = "${path.module}/test-assets/policy-definition.yaml"
      classifier = "policy-definition"
    }
  ]
}
```

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
