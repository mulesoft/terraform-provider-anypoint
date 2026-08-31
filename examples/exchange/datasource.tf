# Example: read a single Exchange asset version by its GAV coordinates.
#
# The singular data source surfaces the WHOLE asset — not just scalar metadata,
# but every nested collection the resource manages: tags, documentation pages
# (with markdown content), external instances, categories, custom_fields, and
# the terms_and_conditions page. Nested collections always come back as
# (possibly empty) lists, so they can be indexed without a null check.
data "anypoint_exchange_asset" "rest_api" {
  group_id = var.org_id
  asset_id = anypoint_exchange_asset.rest_api.asset_id
  version  = anypoint_exchange_asset.rest_api.version
}

# --- Scalar metadata ---------------------------------------------------------
output "rest_api_status" {
  description = "Lifecycle status read back from the data source"
  value       = data.anypoint_exchange_asset.rest_api.status
}

output "rest_api_type" {
  description = "Asset type read back from the data source"
  value       = data.anypoint_exchange_asset.rest_api.type
}

# --- Nested collections (the data source shows the full asset) ---------------
output "rest_api_tags" {
  description = "Tags (labels) on the asset"
  value       = data.anypoint_exchange_asset.rest_api.tags
}

output "rest_api_page_names" {
  description = "Documentation page names (content + page_path also available per page)"
  value       = [for p in data.anypoint_exchange_asset.rest_api.pages : p.page_name]
}

output "rest_api_instances" {
  description = "External API instances, including their computed instance_id"
  value       = data.anypoint_exchange_asset.rest_api.instances
}

output "rest_api_terms" {
  description = "Terms & Conditions markdown (empty string if none)"
  value       = data.anypoint_exchange_asset.rest_api.terms_and_conditions
}

output "rest_api_categories" {
  description = "Category assignments (key + values); empty until an org taxonomy exists"
  value       = data.anypoint_exchange_asset.rest_api.categories
}

output "rest_api_custom_fields" {
  description = "Custom-field assignments (key + values); empty until org custom fields exist"
  value       = data.anypoint_exchange_asset.rest_api.custom_fields
}

# Example: list Exchange assets in the org, filtered by type + free-text search.
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
