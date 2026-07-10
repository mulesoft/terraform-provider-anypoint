---
page_title: "anypoint_available_scopes Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches the catalog of all available scopes for connected applications in the Anypoint Platform.
---

# anypoint_available_scopes (Data Source)

Fetches the complete catalog of scopes available for connected applications. This includes all scope names, display names, descriptions, and product labels.

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

# Retrieve the full catalog of available scopes
data "anypoint_available_scopes" "all" {
  provider = anypoint.admin
}

# Retrieve the catalog including internal scopes
data "anypoint_available_scopes" "with_internal" {
  provider         = anypoint.admin
  include_internal = true
}

# Output all available scopes (display_name is what you use in connected_app scopes)
output "available_scopes" {
  value = data.anypoint_available_scopes.all.scopes
}

# List all display names — use these directly in anypoint_connected_app scopes blocks
output "scope_display_names" {
  value = [for s in data.anypoint_available_scopes.all.scopes : s.display_name]
}

# Filter to CloudHub scopes
output "cloudhub_scopes" {
  value = [
    for scope in data.anypoint_available_scopes.all.scopes :
    scope if can(regex("(?i)cloudhub", scope.display_name))
  ]
}

# Group scopes by product label
locals {
  scopes_by_product = {
    for scope in data.anypoint_available_scopes.all.scopes :
    scope.product_label => scope...
  }
}

output "scopes_by_product" {
  value = local.scopes_by_product
}
```

## Schema

### Optional

- `include_internal` (Boolean) Whether to include internal scopes in the catalog. Defaults to `false`.

### Read-Only

- `id` (String) A generated identifier for this data source (static value: "catalog").
- `scopes` (List of Object) The list of available scopes. See [`scopes`](#nestedschema--scopes) below.

<a id="nestedschema--scopes"></a>
### Nested Schema for `scopes`

Read-Only:

- `display_name` (String) The scope name as shown in the Anypoint UI (e.g., `Read Applications`, `Cloudhub Organization Admin`). **Use this value in `anypoint_connected_app` scopes blocks.**
- `scope` (String) The scope identifier (e.g., `admin:cloudhub`, `read:applications`). Also accepted by connected apps but display names are preferred.
- `description` (String) Description of what the scope permits.
- `product_label` (String) The product or service this scope applies to.
- `internal` (Boolean) Whether this is an internal scope (only visible when `include_internal = true`).
