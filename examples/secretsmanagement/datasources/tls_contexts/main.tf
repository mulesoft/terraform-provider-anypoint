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
  description = "Secret group ID to list TLS contexts from"
  type        = string
}

data "anypoint_secret_group_tls_contexts" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = var.secret_group_id
}

output "all_tls_contexts" {
  description = "All TLS contexts in the secret group"
  value       = data.anypoint_secret_group_tls_contexts.all.tls_contexts
}

locals {
  # Look up a TLS context by name — useful when referencing an existing TLS context
  # in a Flex Gateway deployment or API instance without managing it in Terraform.
  flex_tls_context = one([
    for tls in data.anypoint_secret_group_tls_contexts.all.tls_contexts
    : tls if tls.name == "flex-gateway-tls"
  ])
}

output "flex_tls_context_id" {
  description = "ID of the TLS context named 'flex-gateway-tls'"
  value       = local.flex_tls_context != null ? local.flex_tls_context.id : null
}
