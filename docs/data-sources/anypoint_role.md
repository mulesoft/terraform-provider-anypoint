---
page_title: "anypoint_role Data Source - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Fetches information about an Anypoint Platform role group.
---

# anypoint_role (Data Source)

Fetches information about a specific Anypoint Platform role group by ID.

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

# Retrieve information about a specific role group
data "anypoint_role" "example" {
  provider = anypoint.admin
  id       = "12345678-1234-1234-1234-123456789abc"
}

# Output role group details
output "role_name" {
  value = data.anypoint_role.example.name
}

output "role_description" {
  value = data.anypoint_role.example.description
}

output "is_editable" {
  value = data.anypoint_role.example.editable
}
```

## Schema

### Required

- `id` (String) The unique identifier for the role group.

### Optional

- `organization_id` (String) The organization ID where the role group is located. If not specified, uses the organization from provider credentials.

### Read-Only

- `name` (String) The name of the role group.
- `description` (String) A description of the role group.
- `editable` (Boolean) Whether the role group can be edited.
- `external_names` (List of String) External group names mapped to this role group.
- `created_at` (String) The timestamp when the role group was created.
- `updated_at` (String) The timestamp when the role group was last updated.
