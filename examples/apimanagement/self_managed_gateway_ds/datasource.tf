terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# List all self-managed (connected-mode) Flex gateways that have registered
# in the environment. Only gateways whose runtime has completed registration
# appear here. Soft-deleted (DELETED) tombstones are filtered out by default.
data "anypoint_self_managed_gateways" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

# The same list including soft-deleted tombstones (status DELETED). Deleting a
# self-managed gateway is an async soft-delete: the object lingers in the list
# forever. Set include_deleted = true to audit those tombstones.
data "anypoint_self_managed_gateways" "with_deleted" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  include_deleted = true
}

# Output the full list
output "all_gateways" {
  description = "All self-managed gateways that have registered in the environment"
  value       = data.anypoint_self_managed_gateways.all.gateways
}

# Output just the connected ones
output "connected_gateways" {
  description = "Names of gateways currently reporting CONNECTED"
  value = [
    for gw in data.anypoint_self_managed_gateways.all.gateways :
    gw.name if gw.status == "CONNECTED"
  ]
}

# Look up a specific gateway by name using a local
locals {
  gateway = one([
    for gw in data.anypoint_self_managed_gateways.all.gateways :
    gw if gw.name == var.gateway_name
  ])
}

output "matched_gateway_id" {
  description = "ID of the gateway matching var.gateway_name (null if not found)"
  value       = local.gateway != null ? local.gateway.id : null
}

# Read ONE gateway by its ID using the singular data source. The singular data
# source additionally surfaces the reported runtime `versions` array, which the
# plural list above does NOT expose (both expose status, last_update, tags, and
# replicas). Guarded with count so the config stays valid when var.gateway_name
# matched nothing (local.gateway == null).
data "anypoint_self_managed_gateway" "one" {
  count = local.gateway != null ? 1 : 0

  id              = local.gateway.id
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "matched_gateway_detail" {
  description = "Full detail (status, last_update, tags, versions, replicas) of the matched gateway"
  value       = local.gateway != null ? data.anypoint_self_managed_gateway.one[0] : null
}
