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
  description = "Secret group ID to list certificate pinsets from"
  type        = string
}

data "anypoint_secret_group_certificate_pinsets" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = var.secret_group_id
}

output "all_certificate_pinsets" {
  description = "All certificate pinsets in the secret group"
  value       = data.anypoint_secret_group_certificate_pinsets.all.certificate_pinsets
}

locals {
  my_pinset = one([
    for pin in data.anypoint_secret_group_certificate_pinsets.all.certificate_pinsets
    : pin if pin.name == "my-pinset"
  ])
}

output "my_pinset_id" {
  description = "ID of the certificate pinset named 'my-pinset'"
  value       = local.my_pinset != null ? local.my_pinset.id : null
}
