terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  alias         = "admin"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
  auth_type     = "user"
}

# Create a design environment
resource "anypoint_environment" "my_env" {
  provider = anypoint.admin
  organization_id = var.organization_id
  name            = var.environment_name
  type            = var.environment_type
  is_production   = var.is_production
}

# Create a design environment
resource "anypoint_environment" "my_env_1" {
  provider = anypoint.admin
  organization_id = var.organization_id
  name            = "${var.environment_name}-1"
  type            = "production"
  is_production   = true
}

# Output the environment details
output "environment_id" {
  description = "The ID of the created environment"
  value       = anypoint_environment.my_env.id
}

output "environment_name" {
  description = "The name of the created environment"
  value       = anypoint_environment.my_env.name
}

output "environment_type" {
  description = "The type of the created environment"
  value       = anypoint_environment.my_env.type
}

output "environment_is_production" {
  description = "Whether the environment is production"
  value       = anypoint_environment.my_env.is_production
}

output "environment_organization_id" {
  description = "The organization ID of the environment"
  value       = anypoint_environment.my_env.organization_id
}

output "environment_client_id" {
  description = "The client ID of the environment"
  value       = anypoint_environment.my_env.client_id
}

output "environment_arc_namespace" {
  description = "The ARC namespace of the environment"
  value       = anypoint_environment.my_env.arc_namespace
} 