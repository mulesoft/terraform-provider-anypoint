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

# Output a concise summary
output "api_instance_summary" {
  description = "ID, asset, technology and status for each API instance"
  value = [
    for inst in data.anypoint_api_instances.all.instances : {
      id             = inst.id
      asset_id       = inst.asset_id
      asset_version  = inst.asset_version
      technology     = inst.technology
      instance_label = inst.instance_label
      status         = inst.status
    }
  ]
}

# Filter by technology (e.g. only FlexGateway instances)
output "flex_gateway_instances" {
  description = "API instances deployed on FlexGateway"
  value = [
    for inst in data.anypoint_api_instances.all.instances :
    inst if inst.technology == "flexGateway"
  ]
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
