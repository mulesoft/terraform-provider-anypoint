---
page_title: "anypoint_connected_app_scopes Resource - terraform-provider-anypoint"
subcategory: "Access Management"
description: |-
  DEPRECATED. Manages scopes for an Anypoint Connected Application using user authentication.
---

# anypoint_connected_app_scopes (Resource)

!> **Deprecated:** Use the inline `scopes` attribute on the [`anypoint_connected_app`](anypoint_connected_app.md) resource instead, which manages the app and its scopes as a single authoritative resource. This standalone resource will be removed in a future release. **Migration:** move the `scopes = [...]` block into the corresponding `anypoint_connected_app` resource and remove this resource (a follow-up `terraform apply` reconciles the state — no scope changes occur on the platform since the inline attribute is authoritative over the same list).

Manages scopes for an Anypoint Connected Application using user authentication.

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

resource "anypoint_connected_app_scopes" "example" {
  provider = anypoint.admin
  connected_app_id = "my-connected-app-id"

  scopes = [
    {
      scope = "admin:cloudhub"
      context_params = {
        org = "your-org-id"
      }
    },
    {
      scope = "read:applications"
      context_params = {
        org = "your-org-id"
        envId = "your-env-id"
      }
    }
  ]
}
```

## Schema

### Required

- `connected_app_id` (String) The ID of the connected application to manage scopes for.
- `scopes` (Block Set) The set of scopes to assign to the connected application. See [below for nested schema](#nestedschema--scopes).

### Read-Only

- `id` (String) The unique identifier for the connected app scopes (same as connected_app_id).

<a id="nestedschema--scopes"></a>
### Nested Schema for `scopes`

Required:

- `scope` (String) The scope name (e.g., 'admin:cloudhub', 'read:applications').

Optional:

- `context_params` (Map of String) Context parameters for the scope (e.g., organization ID).

## Import

An existing Connected App's scopes can be imported using its connected app ID (UUID).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  provider = anypoint.admin
  to       = anypoint_connected_app_scopes.imported
  id       = "<connected_app_id>"
}

resource "anypoint_connected_app_scopes" "imported" {
  provider         = anypoint.admin
  connected_app_id = "<connected_app_id>"
  scopes = []
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
terraform import anypoint_connected_app_scopes.imported <connected_app_id>
```
