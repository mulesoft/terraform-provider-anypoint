---
page_title: "anypoint_connected_apps Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Lists all connected apps in the organization.
---

# anypoint_connected_apps (Data Source)

Lists all connected apps in the organization.

~> **Authentication:** This Access Management data source works with a standard **client_credentials** connected app configured in the provider block — a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** to read this data; a missing scope surfaces as `HTTP 401`/`403` (a missing-scope error, not a wrong auth mode). See [Authentication](../index.md).

## Example Usage

```terraform
# A standard client_credentials connected app is enough for this read.
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# List all connected apps in the organization
data "anypoint_connected_apps" "all" {}

# Output all connected apps
output "all_apps" {
  value = data.anypoint_connected_apps.all.apps
}

# Output app count
output "app_count" {
  value = length(data.anypoint_connected_apps.all.apps)
}

# Find a specific app by name
output "my_app" {
  value = [for a in data.anypoint_connected_apps.all.apps : a if a.name == "My Application"][0]
}

# Output enabled apps only
output "enabled_apps" {
  value = [for a in data.anypoint_connected_apps.all.apps : a if a.enabled]
}
```

## Schema

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider's org.

### Read-Only

- `apps` (List of Object) List of connected apps. See [`apps`](#nestedschema--apps) below.

<a id="nestedschema--apps"></a>
### Nested Schema for `apps`

Read-Only:

- `client_id` (String) The unique client ID of the app.
- `name` (String) The name of the app.
- `grant_types` (List of String) The OAuth grant types.
- `audience` (String) Who can use this application.
- `client_uri` (String) Website URL for the app.
- `enabled` (Boolean) Whether the app is enabled.
- `owner_user_id` (String) The user who owns this app.
- `created_at` (String) When the app was created.
- `updated_at` (String) When the app was last updated.
