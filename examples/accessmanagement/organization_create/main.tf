terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0"
    }
  }
}

# Creating or updating an organization (business group) requires an
# on-behalf-of-user connected app: auth_type = "user" with a connected app's
# client_id/client_secret PLUS the credentials of a user who is an Org
# Administrator of the parent organization. A standard client-credentials
# connected app cannot set entitlements and will fail with HTTP 403.
provider "anypoint" {
  alias         = "admin"
  auth_type     = "user"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
}

resource "anypoint_organization" "example" {
  provider = anypoint.admin

  name                   = var.organization_name
  parent_organization_id = var.parent_organization_id
  owner_id               = var.owner_user_id

  entitlements = {
    create_environments = true
    create_sub_orgs     = true
    global_deployment   = false
  }
}

output "organization_id" {
  value = anypoint_organization.example.id
}
