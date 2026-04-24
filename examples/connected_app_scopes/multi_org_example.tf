# Advanced example: Managing scopes for multiple organizations
# This example shows how to use the connected app scopes resource
# to grant access to multiple organizations at once

# Define the organizations to grant access to
# locals {
#   target_organizations = {
#     "production" = {
#       org_id = "30aaff6f-c6d0-4555-b19e-ca6c72a6ef60"
#       scopes = ["admin:cloudhub", "read:applications", "write:applications"]
#     }
#     "staging" = {
#       org_id = "42aaff6f-c6d0-4555-b19e-ca6c72a6ef61"
#       scopes = ["admin:cloudhub", "read:applications"]
#     }
#     "development" = {
#       org_id = "52aaff6f-c6d0-4555-b19e-ca6c72a6ef62"
#       scopes = ["read:applications"]
#     }
#   }

#   # Flatten the scope definitions for all organizations
#   all_scopes = flatten([
#     for env_name, env_config in local.target_organizations : [
#       for scope in env_config.scopes : {
#         scope = scope
#         context_params = {
#           org = env_config.org_id
#         }
#       }
#     ]
#   ])
# }

# # Configure scopes for all target organizations
# resource "anypoint_connected_app_scopes" "multi_org" {
#   connected_app_id = var.connected_app_id
  
#   # Use the flattened scope list
#   scopes = local.all_scopes
# }

# # Output scope summary by organization
# output "scopes_by_organization" {
#   description = "Summary of scopes granted per organization"
#   value = {
#     for env_name, env_config in local.target_organizations : env_name => {
#       organization_id = env_config.org_id
#       scopes         = env_config.scopes
#     }
#   }
# }