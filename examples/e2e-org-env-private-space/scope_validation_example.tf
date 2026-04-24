# Example: Scope Validation in Connected App Scopes
#
# This file demonstrates the scope validation feature that prevents
# invalid scope names from being used in connected app configurations.

# ✅ VALID EXAMPLE - All scopes are valid
# resource "anypoint_connected_app_scopes" "valid_scopes" {
#   provider = anypoint.admin

#   connected_app_id = var.connected_app_client_id

#   scopes = [
#     # CloudHub 2.0 access - VALID
#     {
#       scope = "admin:cloudhub"
#       context_params = {
#         org = var.org_id
#       }
#     },
#     # Runtime Fabrics management - VALID
#     {
#       scope = "manage:runtime_fabrics"
#       context_params = {
#         org = var.org_id
#       }
#     },
#     # Environment creation - VALID
#     {
#       scope = "create:environment"
#       context_params = {
#         org = var.org_id
#         env = var.environment_id
#       }
#     },
#     # Private space management - VALID
#     {
#       scope = "manage:private_spaces"
#       context_params = {
#         org = var.org_id
#       }
#     },
#     # API Manager access - VALID
#     {
#       scope = "admin:api_manager"
#       context_params = {
#         org = var.org_id
#       }
#     },
#     # API Query operations - VALID
#     {
#       scope = "read:api_query"
#       context_params = {
#         org   = var.org_id
#         envId = var.environment_id
#       }
#     },
#     {
#       scope = "edit:api_query"
#       context_params = {
#         org   = var.org_id
#         envId = var.environment_id
#       }
#     },
#     {
#       scope = "manage:api_query"
#       context_params = {
#         org   = var.org_id
#         envId = var.environment_id
#       }
#     }
#   ]
# }

# ❌ INVALID EXAMPLE 1 - Typo in scope name
# This will be REJECTED with a validation error
/*
resource "anypoint_connected_app_scopes" "invalid_typo" {
  provider = anypoint.admin

  connected_app_id = var.connected_app_client_id

  scopes = [
    {
      scope = "admin:cloudhb"  # ❌ TYPO: should be "admin:cloudhub"
      context_params = {
        org = var.org_id
      }
    }
  ]
}

Error message you'll see:
╷
│ Error: Invalid Scope Name
│
│   with anypoint_connected_app_scopes.invalid_typo,
│   on scope_validation_example.tf line XX, in resource "anypoint_connected_app_scopes" "invalid_typo":
│   XX:   resource "anypoint_connected_app_scopes" "invalid_typo" {
│
│ The scope 'admin:cloudhb' at index 0 is not a valid Anypoint Platform scope.
│ Please check the scope name for typos. Valid scopes include: admin:cloudhub,
│ manage:runtime_fabrics, create:environment, manage:private_spaces,
│ admin:api_manager, read:api_query, edit:api_query, manage:api_query, etc.
│ For a complete list of valid scopes, see the provider documentation.
╵
*/

# ❌ INVALID EXAMPLE 2 - Wrong separator
# This will be REJECTED with a validation error
/*
resource "anypoint_connected_app_scopes" "invalid_separator" {
  provider = anypoint.admin

  connected_app_id = var.connected_app_client_id

  scopes = [
    {
      scope = "admin-cloudhub"  # ❌ WRONG: uses dash instead of colon
      context_params = {
        org = var.org_id
      }
    }
  ]
}
*/

# ❌ INVALID EXAMPLE 3 - Non-existent scope
# This will be REJECTED with a validation error
/*
resource "anypoint_connected_app_scopes" "invalid_nonexistent" {
  provider = anypoint.admin

  connected_app_id = var.connected_app_client_id

  scopes = [
    {
      scope = "admin:kubernetes"  # ❌ INVALID: this scope doesn't exist
      context_params = {
        org = var.org_id
      }
    }
  ]
}
*/

# COMPLETE LIST OF VALID SCOPES (119 total)
#
# Admin Scopes:
# - admin:access_controls
# - admin:ang_governance_profiles
# - admin:api_query
# - admin:cloudhub
# - admin:data_exporter_configurations
# - admin:orgclientproviderclients
# - admin:orgclientproviders
# - admin:orgclients
# - admin:partner_manager
#
# Administer Scopes:
# - administer:destinations
#
# AEH Admin:
# - aeh_admin
#
# Clear Scopes:
# - clear:destinations
#
# Create Scopes:
# - create:applications
# - create:client_applications
# - create:design_center
# - create:environment
# - create:exchange
# - create:exchange_genai
# - create:orgclients
# - create:suborgs
#
# Delete Scopes:
# - delete:applications
#
# Download Scopes:
# - download:applications
#
# Edit Scopes:
# - edit:api_catalog
# - edit:api_query
# - edit:design_center
# - edit:environment
# - edit:flow_designer
# - edit:identityproviders
# - edit:monitoring
# - edit:organization
# - edit:orginvites
# - edit:orgusers
# - edit:rpa
# - edit:visualizer
#
# Execute Scopes:
# - execute:document_actions
#
# Manage Scopes:
# - manage:activity
# - manage:api_alerts
# - manage:api_configuration
# - manage:api_contracts
# - manage:api_groups
# - manage:api_policies
# - manage:api_proxies
# - manage:api_query
# - manage:apis
# - manage:application_alerts
# - manage:application_data
# - manage:application_flows
# - manage:application_queues
# - manage:application_schedules
# - manage:application_settings
# - manage:application_tenants
# - manage:clients
# - manage:cloudhub_networking
# - manage:data_gateway
# - manage:envclientproviders
# - manage:exchange
# - manage:host
# - manage:identityproviders
# - manage:partners
# - manage:runtime_fabrics
# - manage:secret_groups
# - manage:secrets
# - manage:servers
# - manage:store
# - manage:store_clients
# - manage:store_data
#
# Promote Scopes:
# - promote:api_query
#
# Publish Scopes:
# - publish:destinations
#
# Read Scopes:
# - read:activity
# - read:api_configuration
# - read:api_contracts
# - read:api_policies
# - read:api_query
# - read:application_alerts
# - read:applications
# - read:audit_logs
# - read:client_applications
# - read:cloudhub_networking
# - read:data_gateway
# - read:exchange
# - read:host_partners
# - read:orgclientproviderclients
# - read:orgclientproviders
# - read:orgclients
# - read:orgconnapps
# - read:orgenvironments
# - read:orginvites
# - read:organization
# - read:orgusers
# - read:runtime_fabrics
# - read:secrets
# - read:secrets_metadata
# - read:servers
# - read:stats
# - read:store
# - read:store_clients
# - read:store_metrics
#
# Restart Scopes:
# - restart:applications
#
# Subscribe Scopes:
# - subscribe:destinations
#
# View Scopes:
# - view:access_controls
# - view:ang_governance_profiles
# - view:clients
# - view:design_center
# - view:destinations
# - view:envclientproviders
# - view:environment
# - view:identityproviders
# - view:metering
# - view:monitoring
#
# Write Scopes:
# - write:audit_log_settings
