---
page_title: "anypoint_exchange_assets Data Source - terraform-provider-anypoint"
subcategory: "Exchange"
description: |-
  Lists Exchange assets in an organization. Supports filtering by type and free-text search.
---

# anypoint_exchange_assets (Data Source)

Lists Exchange assets in an organization. Supports filtering by type and free-text search.

-> **Latest version per asset:** The listing returns one entry per asset (its latest version), not one entry per version. Use the singular [`anypoint_exchange_asset`](anypoint_exchange_asset.md) data source to read a specific version by its GAV coordinates.

## Example Usage

```terraform
# List Exchange assets in the org, filtered by type + free-text search.
data "anypoint_exchange_assets" "rest_apis" {
  organization_id = var.org_id
  type            = "rest-api"
  search          = "tf-demo"
  limit           = 50
}

output "rest_api_asset_ids" {
  description = "Asset IDs of the REST APIs matching the query"
  value       = [for a in data.anypoint_exchange_assets.rest_apis.assets : a.asset_id]
}
```

## Schema

### Required

- `organization_id` (String) The organization ID to list assets from.

### Optional

- `limit` (Number) Optional cap on the total number of assets to return. When **omitted**, ALL matching assets are returned (the data source paginates through every page automatically). When set to a positive value, at most that many assets are returned.
- `search` (String) Free-text search query to filter assets by name or description.
- `type` (String) Filter by asset type (`rest-api`, `http-api`, `evented-api`, `graphql-api`, `custom`, `connector`, `app`, `template`, `example`, `policy`, `agent`, `llm`, `mcp`).

### Read-Only

- `assets` (List of Object) The list of Exchange assets matching the query. See [`assets`](#nestedschema--assets) below.

<a id="nestedschema--assets"></a>
### Nested Schema for `assets`

Read-Only:

- `group_id` (String) The group ID of the asset.
- `asset_id` (String) The asset ID.
- `version` (String) The latest version of the asset.
- `name` (String) The display name of the asset.
- `description` (String) The asset description.
- `type` (String) The asset type.
- `status` (String) The lifecycle status.
- `is_public` (Boolean) Whether the asset is publicly visible.
- `created_date` (String) When the asset was created.
- `updated_date` (String) When the asset was last updated.
