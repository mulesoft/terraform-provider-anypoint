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

# Look up a specific API instance by ID
data "anypoint_api_instance" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  id              = var.api_instance_id
}

# Output key details about the API instance
output "api_instance_label" {
  description = "The human-readable label of the API instance"
  value       = data.anypoint_api_instance.example.instance_label
}

output "api_instance_status" {
  description = "Current status of the API instance"
  value       = data.anypoint_api_instance.example.status
}

output "api_consumer_endpoint" {
  description = "Consumer-facing endpoint URL"
  value       = data.anypoint_api_instance.example.consumer_endpoint
}

output "api_deployment_target" {
  description = "Gateway target where the API is deployed"
  value = {
    target_id   = data.anypoint_api_instance.example.deployment.target_id
    target_name = data.anypoint_api_instance.example.deployment.target_name
    version     = data.anypoint_api_instance.example.deployment.gateway_version
  }
}

output "api_routing_summary" {
  description = "Summary of routing configuration"
  value = [
    for route in data.anypoint_api_instance.example.routing : {
      label     = route.label
      upstreams = [for us in route.upstreams : us.uri]
    }
  ]
}
