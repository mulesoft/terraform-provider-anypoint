---
page_title: "anypoint_role Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform role group (custom or default).
---

# anypoint_role (Resource)

Manages an Anypoint Platform role group (custom or default). Requires Organization Administrator privileges.

~> **Note:** This is an Access Management resource and requires the **admin provider** (`anypoint.admin`), which uses admin user credentials along with the `client_id` and `client_secret` of a connected app to authenticate on behalf of the user (`auth_type = "user"`). You must set `provider = anypoint.admin` on this resource. The default provider (connected app credentials only) does not have sufficient privileges for Access Management operations.

-> **Connected App:** This resource requires an **admin connected app** configured with `auth_type = "user"` (user credentials + connected app client credentials). Use the `anypoint.admin` provider alias. A standard connected app (client credentials only) does not have sufficient privileges for Access Management operations.

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

resource "anypoint_role" "example" {
  provider    = anypoint.admin
  name        = "API Developers"
  description = "Role group for API development team"
}
```

## Schema

### Required

- `name` (String) The name of the role group.

### Optional

- `description` (String) A description of the role group.
- `organization_id` (String) The organization ID where the role group will be created. If not specified, uses the organization from provider credentials.

### Read-Only

- `created_at` (String) The timestamp when the role group was created.
- `editable` (Boolean) Whether the role group can be edited. Default (system) role groups are not editable.
- `external_names` (List of String) External group names mapped to this role group (for SSO/SAML integration). Read-only.
- `id` (String) The unique identifier for the role group.
- `updated_at` (String) The timestamp when the role group was last updated.

## Import

An existing role group can be imported using its role group ID (UUID).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_role.imported
  id       = "<role_group_id>"
}

resource "anypoint_role" "imported" {
  provider        = anypoint.admin
  organization_id = "<organization_id>"
  name            = "<role_name>"
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_role.imported <role_group_id>
```
