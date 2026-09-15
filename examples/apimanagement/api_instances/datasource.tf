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

# List all API instances in the environment
data "anypoint_api_instances" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

# Output the full list
output "all_api_instances" {
  description = "All API instances registered in API Manager for the environment"
  value       = data.anypoint_api_instances.all.instances
}

# Output a concise summary (includes the gateway each instance is deployed to)
output "api_instance_summary" {
  description = "ID, asset, technology, status and gateway for each API instance"
  value = [
    for inst in data.anypoint_api_instances.all.instances : {
      id             = inst.id
      asset_id       = inst.asset_id
      asset_version  = inst.asset_version
      technology     = inst.technology
      instance_label = inst.instance_label
      status         = inst.status
      gateway_id     = inst.gateway_id
    }
  ]
}

# Filter by technology (e.g. only OmniGateway instances)
output "omni_gateway_instances" {
  description = "API instances deployed on OmniGateway"
  value = [
    for inst in data.anypoint_api_instances.all.instances :
    inst if inst.technology == "omniGateway"
  ]
}

# List only the API instances deployed to a specific gateway (server-side of
# the write path: an anypoint_api_instance references its gateway via
# gateway_id — this data source is the reverse lookup). Surfaces instances
# created outside Terraform too (e.g. via the Anypoint UI).
data "anypoint_api_instances" "on_gateway" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id
}

output "instances_on_gateway" {
  description = "IDs of the API instances attached to var.gateway_id"
  value       = [for inst in data.anypoint_api_instances.on_gateway.instances : inst.id]
}

# Look up a specific instance by label
locals {
  matched_instance = one([
    for inst in data.anypoint_api_instances.all.instances :
    inst if inst.instance_label == var.instance_label
  ])
}

output "matched_instance_id" {
  description = "ID of the API instance matching var.instance_label (null if not found)"
  value       = local.matched_instance != null ? local.matched_instance.id : null
}
