###############################################################################
# Data Source: anypoint_agent_instance
#
# Fetch details of a single agent instance by its numeric ID.
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

data "anypoint_agent_instance" "example" {
  id              = var.agent_instance_id
  environment_id  = var.environment_id
  organization_id = var.organization_id
}

output "agent_instance_status" {
  description = "The current status of the agent instance"
  value       = data.anypoint_agent_instance.example.status
}

output "agent_instance_label" {
  description = "The label of the agent instance"
  value       = data.anypoint_agent_instance.example.instance_label
}

output "agent_instance_consumer_endpoint" {
  description = "The consumer-facing endpoint for the agent"
  value       = data.anypoint_agent_instance.example.consumer_endpoint
}

output "agent_instance_asset" {
  description = "The Exchange asset backing this agent"
  value = {
    asset_id = data.anypoint_agent_instance.example.asset_id
    version  = data.anypoint_agent_instance.example.asset_version
  }
}
