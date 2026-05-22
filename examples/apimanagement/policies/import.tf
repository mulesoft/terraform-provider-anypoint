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
# Import an existing API policy into Terraform state.
#
# Two resource types can be used to manage API policies:
#
#   anypoint_api_policy          — generic resource, suitable for any policy
#                                  type including custom/Exchange policies.
#   anypoint_api_policy_<type>   — type-specific resource with a structured
#                                  `configuration` block (e.g. rate_limiting,
#                                  jwt_validation, cors, …).  The import ID
#                                  format is identical for both.
#
# Import ID format (both resource types):
#   organization_id/environment_id/api_instance_id/policy_id
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Choose Option A (generic) or Option B (type-specific) and uncomment
#      the corresponding import + resource blocks.
#   4. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_api_policy.imported \
#          <org_id>/<env_id>/<api_instance_id>/<policy_id>
#   5. Run: terraform plan — review the diff and adjust the resource block.
# ---------------------------------------------------------------------------

# locals {
#   org_id          = "<org_id>"
#   env_id          = "<env_id>"
#   api_instance_id = "<api_instance_id>"
#   policy_id       = "<policy_id>"
# }

# ─────────────────────────────────────────────────────────────────────────────
# Option A — Generic anypoint_api_policy
# Use this when the policy type is not yet covered by a type-specific resource,
# or when you prefer to manage configuration_data as raw JSON.
# ─────────────────────────────────────────────────────────────────────────────

# import {
#   to = anypoint_api_policy.imported
#   id = "${local.org_id}/${local.env_id}/${local.api_instance_id}/${local.policy_id}"
# }
#
# resource "anypoint_api_policy" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_instance_id
#
#   # policy_type is a convenience alias; the provider resolves group_id, asset_id,
#   # and asset_version from it.  Set to the asset ID of the policy, e.g.:
#   #   "rate-limiting", "jwt-validation", "cors", "client-id-enforcement", …
#   policy_type = "<policy_type>"
#
#   # Alternatively, supply Exchange coordinates directly and omit policy_type:
#   # group_id      = "68ef9520-24e9-4cf2-b2f5-620025690913"
#   # asset_id      = "<asset_id>"
#   # asset_version = "1.4.1"
#
#   # Raw JSON string of the policy configuration.  After import, run
#   # `terraform show` to discover the exact structure and fill this in.
#   configuration_data = jsonencode({})
#
#   order    = 1
#   disabled = false
#   label    = "<policy_label>"
# }

# ─────────────────────────────────────────────────────────────────────────────
# Option B — Type-specific anypoint_api_policy_rate_limiting (example)
# All anypoint_api_policy_<type> resources share the same import ID format.
# Replace "rate_limiting" with the desired policy type suffix and update the
# configuration block accordingly (see policies/main.tf for all type examples).
# ─────────────────────────────────────────────────────────────────────────────

# import {
#   to = anypoint_api_policy_rate_limiting.imported
#   id = "${local.org_id}/${local.env_id}/${local.api_instance_id}/${local.policy_id}"
# }
#
# resource "anypoint_api_policy_rate_limiting" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_instance_id
#
#   order    = 1
#   disabled = false
#   label    = "<policy_label>"
#
#   configuration = {
#     key_selector = "#[attributes.queryParams['identifier']]"
#     rate_limits = [
#       {
#         maximum_requests            = 100
#         time_period_in_milliseconds = 60000
#       }
#     ]
#     expose_headers = true
#     clusterizable  = true
#   }
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

# output "imported_policy_id" {
#   description = "ID of the imported API policy"
#   value       = anypoint_api_policy.imported.id
# }
#
# output "imported_policy_asset_id" {
#   description = "Exchange asset ID of the imported policy"
#   value       = anypoint_api_policy.imported.asset_id
# }
#
# output "imported_policy_order" {
#   description = "Execution order of the imported policy"
#   value       = anypoint_api_policy.imported.order
# }
