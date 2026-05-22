terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# ---------------------------------------------------------------------------
# Import an existing API instance into Terraform state.
#
# Usage:
#   1. Set the variables below (or provide them via terraform.tfvars).
#   2. Run: terraform import anypoint_api_instance.imported <org_id>/<env_id>/<api_instance_id>
#      OR use the declarative import block below (Terraform >= 1.5).
#   3. Run: terraform plan  — review the diff and adjust the resource block.
#   4. Run: terraform apply — reconcile state with your config.
#
# The import ID must be: organization_id/environment_id/api_instance_id
# where api_instance_id is the numeric ID visible in Anypoint API Manager.
# ---------------------------------------------------------------------------

import {
  to = anypoint_api_instance.imported
  id = "${var.organization_id}/${var.environment_id}/4717715"
}

resource "anypoint_api_instance" "imported" {
  organization_id = var.organization_id
  environment_id  = var.environment_id

  # Technology defaults to "omniGateway"; override if your instance uses mule4.
  technology = "omniGateway"

  # Populate these with the values that match the existing API instance.
  instance_label  = var.instance_label
  approval_method = var.approval_method

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  # For Omni Gateway, use base_path. For Mule4, use uri instead.
  endpoint = {
    base_path = var.api_base_path
  }

  # Simple single-upstream routing. Replace with a full routing block if needed.
  upstream_uri = var.upstream_uri
}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App client ID"
  type        = string
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Anypoint Platform organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Environment ID where the API instance lives"
  type        = string
  default     = "<environment_id>"
}

variable "api_instance_id" {
  description = "Numeric ID of the existing API instance to import (e.g. 12345678)"
  type        = string
  default     = "<api_instance_id>"
}

variable "instance_label" {
  description = "Human-readable label of the existing API instance"
  type        = string
  default     = "<instance_label>"
}

variable "approval_method" {
  description = "Client approval method: 'manual' or 'automatic'"
  type        = string
  default     = null
}

variable "api_asset_id" {
  description = "Exchange asset ID for the API specification"
  type        = string
  default     = "<api_asset_id>"
}

variable "api_asset_version" {
  description = "Exchange asset version (e.g. 1.0.0)"
  type        = string
  default     = "1.0.0"
}

variable "api_base_path" {
  description = "Base path of the API (used for Omni Gateway endpoint)"
  type        = string
  default     = "<base_path>"
}

variable "upstream_uri" {
  description = "Backend upstream URI (e.g. http://backend.internal:8080)"
  type        = string
  default     = "<upstream_uri>"
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "imported_api_instance_id" {
  description = "Numeric ID of the imported API instance"
  value       = anypoint_api_instance.imported.id
}

output "imported_api_instance_status" {
  description = "Status of the imported API instance"
  value       = anypoint_api_instance.imported.status
}

output "imported_api_instance_asset_id" {
  description = "Exchange asset ID of the imported API instance"
  value       = anypoint_api_instance.imported.asset_id
}

output "imported_api_instance_asset_version" {
  description = "Exchange asset version of the imported API instance"
  value       = anypoint_api_instance.imported.asset_version
}
