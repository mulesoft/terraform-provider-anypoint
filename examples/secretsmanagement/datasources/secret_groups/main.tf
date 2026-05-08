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

variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
}

variable "environment_id" {
  description = "Target environment ID"
  type        = string
}

data "anypoint_secret_groups" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "all_secret_groups" {
  description = "All secret groups in the environment"
  value       = data.anypoint_secret_groups.all.secret_groups
}

locals {
  # Find a specific secret group by name
  my_group = one([
    for sg in data.anypoint_secret_groups.all.secret_groups
    : sg if sg.name == "my-secret-group"
  ])
}

output "my_group_id" {
  description = "ID of the secret group named 'my-secret-group'"
  value       = local.my_group != null ? local.my_group.id : null
}
