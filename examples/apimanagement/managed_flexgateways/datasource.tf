terraform {
  required_providers {
    anypoint = {
      source  = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# List all managed Flex Gateways in the environment
data "anypoint_managed_flexgateways" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

# Output the full list
output "all_gateways" {
  description = "All managed Flex Gateways in the environment"
  value       = data.anypoint_managed_flexgateways.all.gateways
}

# Output just names and IDs
output "gateway_summary" {
  description = "Names, IDs and status of all managed Flex Gateways"
  value = [
    for gw in data.anypoint_managed_flexgateways.all.gateways : {
      id           = gw.id
      name         = gw.name
      status       = gw.status
      target_id    = gw.target_id
      last_updated = gw.last_updated
    }
  ]
}

# Look up a specific gateway by name using a local
locals {
  gateway = one([
    for gw in data.anypoint_managed_flexgateways.all.gateways :
    gw if gw.name == var.gateway_name
  ])
}

output "matched_gateway_id" {
  description = "ID of the gateway matching var.gateway_name (null if not found)"
  value       = local.gateway != null ? local.gateway.id : null
}
