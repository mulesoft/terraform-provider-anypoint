---
page_title: "anypoint_exchange_asset Data Source - terraform-provider-anypoint"
subcategory: "Exchange"
description: |-
  Reads a specific Exchange asset version by its GAV coordinates (groupId, assetId, version).
---

# anypoint_exchange_asset (Data Source)

Reads a specific Exchange asset version by its GAV coordinates (groupId, assetId, version).

-> **Full asset, not just metadata:** Alongside scalar fields like `name`, `type`, and `status`, this data source surfaces the asset's `tags`, documentation `pages` (with their markdown `content`), external API `instances`, `categories`, `custom_fields`, and the `terms_and_conditions` page — mirroring what the [`anypoint_exchange_asset`](../resources/anypoint_exchange_asset.md) resource manages. Nested collections always materialize as (possibly empty) lists, so you can index them without a null check.

## Example Usage

```terraform
# Read a single Exchange asset version by its GAV coordinates.
data "anypoint_exchange_asset" "rest_api" {
  group_id = var.org_id
  asset_id = "tf-demo-rest-api"
  version  = "1.0.0"
}

# Scalar metadata.
output "rest_api_status" {
  value = data.anypoint_exchange_asset.rest_api.status
}

# Nested collections — the data source surfaces the whole asset.
output "rest_api_tags" {
  value = data.anypoint_exchange_asset.rest_api.tags
}

output "rest_api_page_names" {
  value = [for p in data.anypoint_exchange_asset.rest_api.pages : p.page_name]
}

output "rest_api_instances" {
  value = data.anypoint_exchange_asset.rest_api.instances
}
```

## Schema

### Required

- `asset_id` (String) The asset ID.
- `group_id` (String) The group ID of the asset.
- `version` (String) The semantic version of the asset.

### Read-Only

- `id` (String) The composite identifier (`groupId/assetId/version`).
- `name` (String) The display name of the asset.
- `description` (String) The asset description.
- `type` (String) The asset type (`custom`, `rest-api`, `graphql-api`, etc.).
- `status` (String) The lifecycle status (`published`, `deprecated`, `development`).
- `contact_name` (String) Contact person name.
- `contact_email` (String) Contact email.
- `manager` (String) Asset manager.
- `is_public` (Boolean) Whether the asset is publicly visible.
- `is_snapshot` (Boolean) Whether this is a snapshot version.
- `minor_version` (String) The minor version (e.g. `1.0`).
- `version_group` (String) The version group.
- `created_date` (String) When the asset was created.
- `updated_date` (String) When the asset was last updated.
- `tags` (List of String) The asset tags (labels).
- `terms_and_conditions` (String) The markdown content of the asset's Terms & Conditions portal page (empty if none).
- `pages` (List of Object) The asset's documentation portal pages (excludes synthetic and Terms & Conditions pages). See [`pages`](#nestedschema--pages) below.
- `instances` (List of Object) The asset's external (non-managed) API instances. See [`instances`](#nestedschema--instances) below.
- `categories` (List of Object) The category assignments on the asset. See [`categories`](#nestedschema--categories) below.
- `custom_fields` (List of Object) The custom-field assignments on the asset. See [`custom_fields`](#nestedschema--custom_fields) below.

<a id="nestedschema--pages"></a>
### Nested Schema for `pages`

Read-Only:

- `page_name` (String) The page name.
- `content` (String) The markdown content of the page.
- `page_path` (String) The API-assigned path of the page.

<a id="nestedschema--instances"></a>
### Nested Schema for `instances`

Read-Only:

- `name` (String) The instance name.
- `endpoint_uri` (String) The instance endpoint URI.
- `is_public` (Boolean) Whether the instance is publicly visible.
- `instance_id` (String) The API-assigned instance identifier.

<a id="nestedschema--categories"></a>
### Nested Schema for `categories`

Read-Only:

- `key` (String) The category key (display name).
- `values` (List of String) The assigned category values.

<a id="nestedschema--custom_fields"></a>
### Nested Schema for `custom_fields`

Read-Only:

- `key` (String) The custom-field key.
- `values` (List of String) The assigned custom-field values.
