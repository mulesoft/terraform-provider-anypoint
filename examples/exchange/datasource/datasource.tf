# List REST APIs in the org matching "tf-demo". Safe on a new account (empty list).
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

# Read one REST asset by GAV. Commented so plan succeeds before rest_api/ has
# been applied (otherwise Exchange 404s). Uncomment after:
#   cd ../rest_api && terraform apply
# The GAV below matches rest_api/ (tf-demo-rest-api / 1.0.0).
#
# data "anypoint_exchange_asset" "rest_api" {
#   group_id = var.org_id
#   asset_id = "tf-demo-rest-api"
#   version  = "1.0.0"
# }
#
# output "rest_api_status" {
#   description = "Lifecycle status read back from the data source"
#   value       = data.anypoint_exchange_asset.rest_api.status
# }
#
# output "rest_api_type" {
#   description = "Asset type read back from the data source"
#   value       = data.anypoint_exchange_asset.rest_api.type
# }
#
# output "rest_api_tags" {
#   description = "Tags (labels) on the asset"
#   value       = data.anypoint_exchange_asset.rest_api.tags
# }
#
# output "rest_api_page_names" {
#   description = "Documentation page names"
#   value       = [for p in data.anypoint_exchange_asset.rest_api.pages : p.page_name]
# }
#
# output "rest_api_instances" {
#   description = "External API instances, including their computed instance_id"
#   value       = data.anypoint_exchange_asset.rest_api.instances
# }
#
# output "rest_api_terms" {
#   description = "Terms & Conditions markdown (empty string if none)"
#   value       = data.anypoint_exchange_asset.rest_api.terms_and_conditions
# }
#
# output "rest_api_categories" {
#   description = "Category assignments (key + values); empty until an org taxonomy exists"
#   value       = data.anypoint_exchange_asset.rest_api.categories
# }
#
# output "rest_api_custom_fields" {
#   description = "Custom-field assignments (key + values); empty until org custom fields exist"
#   value       = data.anypoint_exchange_asset.rest_api.custom_fields
# }
