---
page_title: "anypoint_available_scopes Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches the catalog of all available scopes for connected applications in the Anypoint Platform.
---

# anypoint_available_scopes (Data Source)

Fetches the complete catalog of scopes available for connected applications. This includes all scope names, display names, descriptions, and product labels.

~> **Authentication:** This Access Management data source works with a standard **client_credentials** connected app configured in the provider block — a separate admin/user provider is **not** required. What matters is that the connected app holds the right **scopes / organization permissions** to read this data; a missing scope surfaces as `HTTP 401`/`403` (a missing-scope error, not a wrong auth mode). See [Authentication](../index.md).

## Example Usage

```terraform
# A standard client_credentials connected app is enough for this read.
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Retrieve the full catalog of available scopes
data "anypoint_available_scopes" "all" {}

# Retrieve the catalog including internal scopes
data "anypoint_available_scopes" "with_internal" {
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

- `scopes` (List of Object) The list of available scopes. See [`scopes`](#nestedschema--scopes) below.

<a id="nestedschema--scopes"></a>
### Nested Schema for `scopes`

Read-Only:

- `display_name` (String) The scope name as shown in the Anypoint UI (e.g., `Read Applications`, `Cloudhub Organization Admin`). **Use this value in `anypoint_connected_app` scopes blocks.**
- `scope` (String) The scope identifier (e.g., `admin:cloudhub`, `read:applications`). Also accepted by connected apps but display names are preferred.
- `description` (String) Description of what the scope permits.
- `product_label` (String) The product or service this scope applies to.
- `internal` (Boolean) Whether this is an internal scope (only visible when `include_internal = true`).
