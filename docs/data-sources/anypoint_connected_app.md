---
page_title: "anypoint_connected_app Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Reads a single connected app by ID.
---

# anypoint_connected_app (Data Source)

Reads a single connected app by ID. Use this to look up details of an existing connected app.

~> **Note:** This is an Access Management data source and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this data source. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

-> **Connected App:** This data source requires an **admin connected app** configured with `auth_type = "user"` (user credentials + connected app client credentials). Use the `anypoint.admin` provider alias.

## Example Usage

```terraform
# Admin provider – authenticates on behalf of a user using connected app credentials
provider "anypoint" {
  alias         = "admin"
  auth_type     = "user"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
}

# Read a connected app by ID
data "anypoint_connected_app" "example" {
  provider = anypoint.admin
  id       = "abc123-def456-ghi789"
}

# Output the app name
output "app_name" {
  value = data.anypoint_connected_app.example.name
}

# Output the grant types
output "grant_types" {
  value = data.anypoint_connected_app.example.grant_types
}

# Output the scopes
output "scopes" {
  value = data.anypoint_connected_app.example.scopes
}
```

## Schema

### Required

- `id` (String) The client_id of the connected app.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider's org.

### Read-Only

- `name` (String) The name of the connected app.
- `grant_types` (List of String) The OAuth grant types.
- `redirect_uris` (List of String) OAuth redirect URIs (if configured).
- `audience` (String) Who can use this application ('internal' or 'everyone').
- `client_uri` (String) Website URL for the app (if configured).
- `enabled` (Boolean) Whether the app is enabled.
- `owner_user_id` (String) The user who owns this app.
- `scopes` (List of Object) The scopes assigned to this connected app. See [`scopes`](#nestedschema--scopes) below.
- `created_at` (String) When the app was created.
- `updated_at` (String) When the app was last updated.

<a id="nestedschema--scopes"></a>
### Nested Schema for `scopes`

Read-Only:

- `scope` (String) The scope display name or identifier.
- `context_params` (Map of String) Context parameters for the scope (e.g., `org`, `envId`).
