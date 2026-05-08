terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Create Private Space Associations
resource "anypoint_private_space_association" "example" {
  private_space_id = var.private_space_id
  associations = [
    # {
    #   organization_id = "<org_id>"
    #   environment     = "all"
    # },
    {
      organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"
      environment     = "all"
    }
  ]
}

# Outputs
output "private_space_association_id" {
  description = "The ID of the Private Space Association resource"
  value       = anypoint_private_space_association.example.id
}

output "created_associations" {
  description = "List of created associations with their IDs"
  value       = anypoint_private_space_association.example.created_associations
}

output "association_count" {
  description = "Number of associations created"
  value       = length(anypoint_private_space_association.example.created_associations)
}

output "association_ids" {
  description = "List of association IDs"
  value       = [for assoc in anypoint_private_space_association.example.created_associations : assoc.id]
}

output "environment_associations" {
  description = "Map of environment and organization to association IDs"
  value = {
    for assoc in anypoint_private_space_association.example.created_associations : 
    "${assoc.environment}-${assoc.organization_id}" => assoc.id
  }
}