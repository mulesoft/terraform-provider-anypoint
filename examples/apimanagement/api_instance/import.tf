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
# Import an existing API instance into Terraform state.
#
# Steps:
#   1. Uncomment the relevant block below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_api_instance.imported \
#          <env_id>/<api_instance_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   Root org:  anypoint_api_instance -> <env_id>/<api_instance_id>
#   Sub-org:   anypoint_api_instance -> <org_id>/<env_id>/<api_instance_id>
#
# Use the sub-org format when the API instance belongs to a Business Group.
# ---------------------------------------------------------------------------

# --- Root org (2-part ID) ---
# locals {
#   env_id          = "<env_id>"
#   api_instance_id = "<api_instance_id>"  # numeric ID from Anypoint API Manager
# }
#
# import {
#   to = anypoint_api_instance.imported
#   id = "${local.env_id}/${local.api_instance_id}"
# }

# --- Sub-org (3-part ID) ---
# locals {
#   org_id          = "<org_id>"
#   env_id          = "<env_id>"
#   api_instance_id = "<api_instance_id>"  # numeric ID from Anypoint API Manager
# }
#
# import {
#   to = anypoint_api_instance.imported
#   id = "${local.org_id}/${local.env_id}/${local.api_instance_id}"
# }

# resource "anypoint_api_instance" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#
#   # Technology: "omniGateway" (Flex Gateway) or "mule4".
#   technology = "omniGateway"
#
#   instance_label  = "<instance_label>"
#   approval_method = null   # "manual" | "automatic" | null
#
#   spec = {
#     asset_id = "<api_asset_id>"
#     group_id = local.org_id
#     version  = "1.0.0"
#   }
#
#   # For Omni Gateway use base_path; for Mule 4 use uri instead.
#   endpoint = {
#     base_path = "/<base_path>"
#   }
#
#   upstream_uri = "http://<backend-host>:8080"
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

# output "imported_api_instance_id" {
#   description = "Numeric ID of the imported API instance"
#   value       = anypoint_api_instance.imported.id
# }
#
# output "imported_api_instance_status" {
#   description = "Status of the imported API instance"
#   value       = anypoint_api_instance.imported.status
# }
#
# output "imported_api_instance_asset_id" {
#   description = "Exchange asset ID of the imported API instance"
#   value       = anypoint_api_instance.imported.asset_id
# }
