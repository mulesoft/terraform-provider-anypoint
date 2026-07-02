---
page_title: "anypoint_role_permission Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Assigns a permission (role) to an Anypoint role group.
---

# anypoint_role_permission (Resource)

Assigns a permission (role) to an Anypoint role group. Each instance represents a single role assignment. Changing any identifier attribute forces recreation (destroy + create). Requires Organization Administrator privileges.

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

# Assign an organization-level permission
resource "anypoint_role_permission" "example_org" {
  provider       = anypoint.admin
  role_group_id  = anypoint_role.example.id
  role_id        = data.anypoint_roles_available.all.roles[0].id
  context_params = {
    org = var.organization_id
  }
}

# Assign an environment-scoped permission
resource "anypoint_role_permission" "example_env" {
  provider       = anypoint.admin
  role_group_id  = anypoint_role.example.id
  role_id        = data.anypoint_roles_available.all.roles[1].id
  context_params = {
    org   = var.organization_id
    envId = var.environment_id
  }
}
```

## Schema

### Required

- `context_params` (Map of String) Context parameters for the assignment. Must include 'org' (organization ID). For environment-scoped permissions, also include 'envId'.
- `role_group_id` (String) The ID of the role group to assign the permission to.
- `role_id` (String) The ID of the role (permission) to assign. Use data.anypoint_roles_available to discover available role IDs.

### Optional

- `organization_id` (String) The organization ID. If not specified, uses the organization from provider credentials.

### Read-Only

- `created_at` (String) The timestamp when the assignment was created.
- `id` (String) The unique role_group_assignment_id for this assignment.
- `role_name` (String) The name of the assigned role (permission). Populated from the API on read.

## Import

An existing role permission assignment can be imported using a composite ID format: `{role_group_id}:{role_group_assignment_id}`.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_role_permission.imported
  id       = "<role_group_id>:<role_group_assignment_id>"
}

resource "anypoint_role_permission" "imported" {
  provider       = anypoint.admin
  role_group_id  = "<role_group_id>"
  role_id        = "<role_id>"
  context_params = {
    org = "<organization_id>"
  }
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
terraform import anypoint_role_permission.imported <role_group_id>:<role_group_assignment_id>
```
