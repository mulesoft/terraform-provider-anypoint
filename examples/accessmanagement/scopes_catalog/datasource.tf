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

# Fetch the full scopes catalog (excluding internal scopes)
data "anypoint_available_scopes" "public_scopes" {
  include_internal = false
}

# Fetch all scopes including internal ones
data "anypoint_available_scopes" "all_scopes" {
  include_internal = true
}

# Output all public scopes
output "public_scopes" {
  value = data.anypoint_available_scopes.public_scopes.scopes
}

output "public_scope_count" {
  value = length(data.anypoint_available_scopes.public_scopes.scopes)
}

# Filter for CloudHub-related scopes
output "cloudhub_scopes" {
  value = [
    for scope in data.anypoint_available_scopes.public_scopes.scopes :
    scope if strcontains(lower(scope.product_label), "cloudhub") || strcontains(lower(scope.display_name), "cloudhub")
  ]
}

# Filter for Runtime Manager scopes
output "runtime_manager_scopes" {
  value = [
    for scope in data.anypoint_available_scopes.public_scopes.scopes :
    scope if strcontains(lower(scope.product_label), "runtime manager")
  ]
}

# Filter for Exchange scopes
output "exchange_scopes" {
  value = [
    for scope in data.anypoint_available_scopes.public_scopes.scopes :
    scope if strcontains(lower(scope.product_label), "exchange")
  ]
}

# Filter for Design Center scopes
output "design_center_scopes" {
  value = [
    for scope in data.anypoint_available_scopes.public_scopes.scopes :
    scope if strcontains(lower(scope.product_label), "design center")
  ]
}

# Output display names (use these in anypoint_connected_app scopes blocks)
output "all_scope_display_names" {
  value = [for scope in data.anypoint_available_scopes.public_scopes.scopes : scope.display_name]
}

# Example: Find a scope by its display name (as shown in the Anypoint UI)
output "read_applications_scope" {
  value = [
    for scope in data.anypoint_available_scopes.public_scopes.scopes :
    scope if scope.display_name == "Read Applications"
  ]
}

# Example: Find a scope by its identifier (for advanced use)
output "create_generations_scope" {
  value = [
    for scope in data.anypoint_available_scopes.public_scopes.scopes :
    scope if scope.scope == "create:generations"
  ]
}
