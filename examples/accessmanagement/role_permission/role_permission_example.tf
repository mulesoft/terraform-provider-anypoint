terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  auth_type     = "user"
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = var.anypoint_base_url
}

# Assign CloudHub Admin permission to a role group (organization-scoped)
resource "anypoint_role_permission" "cloudhub_admin" {
  role_group_id = var.role_group_id
  role_id       = "<cloudhub_admin_role_id>" # Use data.anypoint_available_roles to discover role IDs

  context_params = {
    org = var.org_id
  }
}

# Assign CloudHub Developer permission to a role group for a specific environment
resource "anypoint_role_permission" "cloudhub_developer_sandbox" {
  role_group_id = var.role_group_id
  role_id       = "<cloudhub_developer_role_id>"

  context_params = {
    org   = var.org_id
    envId = var.environment_id
  }
}

# Assign Exchange Contributor permission (organization-scoped)
resource "anypoint_role_permission" "exchange_contributor" {
  role_group_id = var.role_group_id
  role_id       = "<exchange_contributor_role_id>"

  context_params = {
    org = var.org_id
  }
}

output "cloudhub_admin_assignment_id" {
  description = "Assignment ID for CloudHub Admin permission"
  value       = anypoint_role_permission.cloudhub_admin.id
}

output "cloudhub_developer_assignment_id" {
  description = "Assignment ID for CloudHub Developer permission (environment-scoped)"
  value       = anypoint_role_permission.cloudhub_developer_sandbox.id
}

output "exchange_contributor_role_name" {
  description = "Name of the Exchange Contributor role"
  value       = anypoint_role_permission.exchange_contributor.role_name
}
