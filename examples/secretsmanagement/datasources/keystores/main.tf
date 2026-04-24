terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
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

variable "secret_group_id" {
  description = "Secret group ID to list keystores from"
  type        = string
}

data "anypoint_secret_group_keystores" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = var.secret_group_id
}

output "all_keystores" {
  description = "All keystores in the secret group"
  value       = data.anypoint_secret_group_keystores.all.keystores
}

locals {
  # Look up a keystore by name to reference its ID elsewhere
  my_keystore = one([
    for ks in data.anypoint_secret_group_keystores.all.keystores
    : ks if ks.name == "my-keystore"
  ])
}

output "my_keystore_id" {
  description = "ID of the keystore named 'my-keystore'"
  value       = local.my_keystore != null ? local.my_keystore.id : null
}
