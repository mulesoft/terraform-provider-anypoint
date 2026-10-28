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

# ---------------------------------------------------------------------------
# List all managed Omni Gateways in the environment
# ---------------------------------------------------------------------------

data "anypoint_managed_omni_gateways" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "all_gateways" {
  description = "All managed Omni Gateways in the environment"
  value       = data.anypoint_managed_omni_gateways.all.gateways
}

output "gateway_summary" {
  description = "Names, IDs, and status of all managed Omni Gateways"
  value = [
    for gw in data.anypoint_managed_omni_gateways.all.gateways : {
      id           = gw.id
      name         = gw.name
      status       = gw.status
      target_id    = gw.target_id
      last_updated = gw.last_updated
    }
  ]
}

# Look up one gateway by name, then read its full details

locals {
  gateway = one([
    for gw in data.anypoint_managed_omni_gateways.all.gateways :
    gw if gw.name == var.gateway_name
  ])
}

output "matched_gateway_id" {
  description = "ID of the gateway matching var.gateway_name (null if not found)"
  value       = local.gateway != null ? local.gateway.id : null
}

data "anypoint_managed_omni_gateway" "matched" {
  count = local.gateway != null ? 1 : 0

  organization_id = var.organization_id
  environment_id  = var.environment_id
  id              = local.gateway.id
}

output "matched_gateway_details" {
  description = "Full details for the matched gateway"
  value = length(data.anypoint_managed_omni_gateway.matched) > 0 ? {
    id             = data.anypoint_managed_omni_gateway.matched[0].id
    name           = data.anypoint_managed_omni_gateway.matched[0].name
    status         = data.anypoint_managed_omni_gateway.matched[0].status
    desired_status = data.anypoint_managed_omni_gateway.matched[0].desired_status
    api_limit      = data.anypoint_managed_omni_gateway.matched[0].api_limit
    target_id      = data.anypoint_managed_omni_gateway.matched[0].target_id
  } : null
}
