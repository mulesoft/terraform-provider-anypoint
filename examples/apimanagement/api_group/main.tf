terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

###############################################################################
# Example 1 – Simple single-version API Group
###############################################################################
resource "anypoint_api_group" "payments" {
  organization_id = var.organization_id
  name            = "Payments API Group"

  versions = [
    {
      name = "v1"
      instances = [
        {
          environment_id       = var.environment_id
          group_instance_label = ""
          api_instances        = var.api_instance_ids
        }
      ]
    }
  ]
}

###############################################################################
# Example 2 – Multi-version API Group with instances in multiple environments
###############################################################################
resource "anypoint_api_group" "orders" {
  organization_id = var.organization_id
  name            = "Orders API Group"

  versions = [
    {
      name = "v1"
      instances = [
        {
          environment_id       = var.environment_id
          group_instance_label = "sandbox"
          api_instances        = var.api_instance_ids
        }
      ]
    },
    {
      name = "v2"
      instances = [
        {
          environment_id       = var.environment_id
          group_instance_label = "sandbox-v2"
          api_instances        = var.api_instance_ids_v2
        },
        {
          environment_id       = var.staging_environment_id
          group_instance_label = "staging-v2"
          api_instances        = var.staging_api_instance_ids
        }
      ]
    }
  ]
}

###############################################################################
# Outputs
###############################################################################
output "payments_group_id" {
  description = "ID of the Payments API Group."
  value       = anypoint_api_group.payments.id
}

output "orders_group_id" {
  description = "ID of the Orders API Group."
  value       = anypoint_api_group.orders.id
}
