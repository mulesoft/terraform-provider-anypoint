# List all connected apps in the organization
data "anypoint_connected_apps" "all_apps" {
  organization_id = var.org_id
}

# Output all connected apps
output "all_connected_apps" {
  value = data.anypoint_connected_apps.all_apps.apps
}

output "total_apps" {
  value = length(data.anypoint_connected_apps.all_apps.apps)
}

# Filter for enabled apps only
output "enabled_apps" {
  value = [
    for app in data.anypoint_connected_apps.all_apps.apps :
    app if app.enabled
  ]
}

# Filter for client_credentials apps (service apps)
output "service_apps" {
  value = [
    for app in data.anypoint_connected_apps.all_apps.apps :
    app if contains(app.grant_types, "client_credentials")
  ]
}

# Filter for user-facing apps (authorization_code grant)
output "user_facing_apps" {
  value = [
    for app in data.anypoint_connected_apps.all_apps.apps :
    app if contains(app.grant_types, "authorization_code")
  ]
}

# Filter for internal-only apps
output "internal_apps" {
  value = [
    for app in data.anypoint_connected_apps.all_apps.apps :
    app if app.audience == "internal"
  ]
}
