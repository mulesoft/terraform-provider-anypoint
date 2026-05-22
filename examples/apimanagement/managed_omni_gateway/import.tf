# terraform {
#   required_providers {
#     anypoint = {
#       source = "mulesoft/anypoint"
#     }
#   }
# }
#
# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# ---------------------------------------------------------------------------
# Import an existing Managed Omni Gateway into Terraform state.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values with real IDs.
#      IDs can be found in Anypoint Runtime Manager or via the Anypoint API.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_managed_omni_gateway.imported \
#          <organization_id>/<environment_id>/<gateway_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   anypoint_managed_omni_gateway -> <organization_id>/<environment_id>/<gateway_id>
# ---------------------------------------------------------------------------

# locals {
#   org_id     = "<organization_id>"
#   env_id     = "<environment_id>"
#   gateway_id = "<gateway_id>"
# }

# import {
#   to = anypoint_managed_omni_gateway.imported
#   id = "${local.org_id}/${local.env_id}/${local.gateway_id}"
# }

# resource "anypoint_managed_omni_gateway" "imported" {
#   name           = "<gateway_name>"
#   environment_id = "<environment_id>"
#   target_id      = "<target_id>"       # private space / target UUID
#
#   # Optional — omit to keep the existing values.
#   # size            = "small"
#   # release_channel = "lts"
#
#   # logging = {
#   #   level        = "info"
#   #   forward_logs = true
#   # }
#
#   # tracing = {
#   #   enabled  = false
#   #   sampling = 1
#   #   labels   = []
#   # }
# }

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

# variable "anypoint_client_id" {
#   description = "Anypoint Platform Connected App client ID"
#   type        = string
# }
#
# variable "anypoint_client_secret" {
#   description = "Anypoint Platform Connected App client secret"
#   type        = string
#   sensitive   = true
# }
#
# variable "anypoint_base_url" {
#   description = "Anypoint Platform base URL"
#   type        = string
#   default     = "https://anypoint.mulesoft.com"
# }

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

# output "imported_gateway_id" {
#   description = "ID of the imported Managed Omni Gateway"
#   value       = anypoint_managed_omni_gateway.imported.id
# }
#
# output "imported_gateway_status" {
#   description = "Current status of the imported gateway"
#   value       = anypoint_managed_omni_gateway.imported.status
# }
#
# output "imported_gateway_runtime_version" {
#   description = "Runtime version running on the imported gateway"
#   value       = anypoint_managed_omni_gateway.imported.runtime_version
# }
