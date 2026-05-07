terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
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
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Target environment ID (e.g. Sandbox or Production)"
  type        = string
  default     = "<private_space_id>"
}


# Basic secret group (non-downloadable)
resource "anypoint_secret_group" "basic" {
  environment_id = var.environment_id
  name           = "terraform-secrets"
  downloadable   = false
}

# Downloadable secret group
resource "anypoint_secret_group" "downloadable" {
  environment_id = var.environment_id
  name           = "terraform-downloadable-secrets"
  downloadable   = true
}

output "basic_secret_group_id" {
  value = anypoint_secret_group.basic.id
}

output "downloadable_secret_group_id" {
  value = anypoint_secret_group.downloadable.id
}
