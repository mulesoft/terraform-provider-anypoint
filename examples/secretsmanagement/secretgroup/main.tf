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
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Target environment ID (e.g. Sandbox or Production)"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
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
