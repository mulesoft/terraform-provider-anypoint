---
page_title: "anypoint_environment Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  Manages an Anypoint Platform environment.
---

# anypoint_environment (Resource)

Manages an Anypoint Platform environment.

~> **Note:** Creating, updating, or deleting an environment requires an **on-behalf-of-user connected app** using `auth_type = "user"` (admin user credentials together with a connected app's `client_id`/`client_secret`). Use the `anypoint.admin` provider alias and set `provider = anypoint.admin` on this resource. A standard connected app (client credentials only) does not have sufficient privileges for these write operations.

-> **Reading vs. writing:** Only create/update/delete need user auth. *Reading* an environment — via the [`anypoint_environment` data source](../data-sources/anypoint_environment.md) or a `terraform plan` refresh — works with a standard connected app (client credentials, the default `auth_type = "connected_app"`) that has the appropriate environment read scopes.

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

resource "anypoint_environment" "example" {
  provider = anypoint.admin
  name            = "my-sandbox-env"
  type            = "sandbox"
  is_production   = false
  organization_id = "your-org-id"
}
```

## Schema

### Required

- `name` (String) The name of the environment.

### Optional

- `arc_namespace` (String) The ARC namespace for the environment.
- `client_id` (String) The client ID associated with the environment.
- `is_production` (Boolean) Whether this is a production environment. Defaults to `false`.
- `organization_id` (String) The organization ID where the environment will be created. If not provided, the organization ID will be inferred from the connected app credentials.
- `type` (String) The type of the environment (e.g., 'design', 'sandbox', 'production'). Defaults to `"sandbox"`.

### Read-Only

- `id` (String) The unique identifier for the environment.

## Import

An existing environment can be imported using its environment ID (UUID).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_environment.imported
  id       = "<environment_id>"
}

resource "anypoint_environment" "imported" {
  provider        = anypoint.admin
  organization_id = "<organization_id>"
  name            = "<environment_name>"
  type            = "sandbox"
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
terraform import anypoint_environment.imported <environment_id>
```
