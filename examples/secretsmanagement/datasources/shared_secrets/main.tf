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

variable "secret_group_id" {
  description = "Secret group ID to list shared secrets from"
  type        = string
}

# Note: the API returns only non-sensitive fields (name, type, expiration_date, username,
# access_key_id). Sensitive values (passwords, keys, secret access keys) are never returned.
data "anypoint_secret_group_shared_secrets" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = var.secret_group_id
}

output "all_shared_secrets" {
  description = "All shared secrets in the secret group (non-sensitive fields only)"
  value       = data.anypoint_secret_group_shared_secrets.all.shared_secrets
}

locals {
  my_secret = one([
    for ss in data.anypoint_secret_group_shared_secrets.all.shared_secrets
    : ss if ss.name == "my-db-credentials"
  ])
}

output "my_shared_secret_id" {
  description = "ID of the shared secret named 'my-db-credentials'"
  value       = local.my_secret != null ? local.my_secret.id : null
}
