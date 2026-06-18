# Retrieve the current scopes for a connected application
data "anypoint_connected_app_scopes" "example" {
  connected_app_id = var.connected_app_id
}

# Output all scopes
output "app_scopes" {
  value = data.anypoint_connected_app_scopes.example.scopes
}

# Output scope count
output "scope_count" {
  value = length(data.anypoint_connected_app_scopes.example.scopes)
}
