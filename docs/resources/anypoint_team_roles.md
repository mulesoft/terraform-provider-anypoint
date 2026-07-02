---
page_title: "anypoint_team_roles Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Assigns a role (permission) to a team.
---

# anypoint_team_roles (Resource)

Assigns a role (permission) to a team. This grants all team members the specified permission scoped by the given context parameters (org, environment).

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

# Assign an organization-level permission to a team
resource "anypoint_team_roles" "example_org" {
  provider = anypoint.admin
  team_id  = anypoint_team.example.id
  role_id  = data.anypoint_available_roles.all.roles[0].id
  context_params = {
    org = var.organization_id
  }
}

# Assign an environment-scoped permission to a team
resource "anypoint_team_roles" "example_env" {
  provider = anypoint.admin
  team_id  = anypoint_team.example.id
  role_id  = data.anypoint_available_roles.all.roles[1].id
  context_params = {
    org   = var.organization_id
    envId = var.environment_id
  }
}
```

## Schema

### Required

- `context_params` (Map of String) Context parameters that scope the permission. Common keys: 'org' (organization ID), 'envId' (environment ID). Use the anypoint_available_roles data source to determine required context params for each role.
- `role_id` (String) The ID of the role (permission) to assign. Use the anypoint_available_roles data source to find valid role IDs.
- `team_id` (String) The ID of the team to assign the role to.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider's org.

### Read-Only

- `description` (String) The description of the assigned role (computed after creation).
- `id` (String) The role_group_assignment_id returned by the API.
- `name` (String) The name of the assigned role (computed after creation).

## Import

An existing team role assignment can be imported using a composite ID format: `{team_id}:{assignment_id}`.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_team_roles.imported
  id       = "<team_id>:<assignment_id>"
}

resource "anypoint_team_roles" "imported" {
  provider = anypoint.admin
  team_id  = "<team_id>"
  role_id  = "<role_id>"
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
terraform import anypoint_team_roles.imported <team_id>:<assignment_id>
```
