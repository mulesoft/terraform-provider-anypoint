---
page_title: "anypoint_connected_app_scopes Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches scopes information for an Anypoint Connected Application.
---

# anypoint_connected_app_scopes (Data Source)

Fetches the set of scopes assigned to an Anypoint Connected Application.

~> **Authentication:** This Access Management data source works with a standard **client_credentials** connected app configured in the provider block — a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** to read this data; a missing scope surfaces as `HTTP 401`/`403` (a missing-scope error, not a wrong auth mode). See [Authentication](../index.md).

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

# Retrieve the current scopes for a connected application
data "anypoint_connected_app_scopes" "example" {
  provider         = anypoint.admin
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
```

## Schema

### Required

- `connected_app_id` (String) The ID of the connected application to read scopes for.

### Read-Only

- `id` (String) The unique identifier for the connected app scopes (same as `connected_app_id`).
- `scopes` (Set of Object) The set of scopes assigned to the connected application. See [`scopes`](#nestedschema--scopes) below.

<a id="nestedschema--scopes"></a>
### Nested Schema for `scopes`

Read-Only:

- `scope` (String) The scope name (e.g., `admin:cloudhub`, `read:applications`).
- `context_params` (Map of String) Context parameters for the scope (e.g., organization ID).
