terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0"
    }
  }
}

# Reading an organization only needs a standard connected app that "acts on
# its own behalf" (client credentials). This is the default auth_type, so no
# username/password is required.
#
# Grant the connected app these scopes on the target organization:
#   - Profile            (profile)
#   - View Organization  (read:organization)
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
  auth_type     = "connected_app" # default; shown here for clarity
}

data "anypoint_organization" "this" {
  id = var.organization_id
}

output "organization_name" {
  value = data.anypoint_organization.this.name
}

output "organization_type" {
  value = data.anypoint_organization.this.org_type
}
