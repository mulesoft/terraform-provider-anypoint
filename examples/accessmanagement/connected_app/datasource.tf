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


# ---------------------------------------------------------------------------
# Example: fetch a single connected app by its client ID.
# ---------------------------------------------------------------------------
data "anypoint_connected_app" "existing" {
  id              = "<client_id>" # Replace with a valid connected app client ID
  organization_id = var.org_id
}

output "existing_connected_app_name" {
  value = data.anypoint_connected_app.existing.name
}

output "existing_connected_app_grant_types" {
  value = data.anypoint_connected_app.existing.grant_types
}

# Scopes granted to the app. This paginates internally — a bare API call returns
# only the first page, which is a common source of "missing scope" confusion.
output "existing_connected_app_scopes" {
  value = data.anypoint_connected_app.existing.scopes
}
