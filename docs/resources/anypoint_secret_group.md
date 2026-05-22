---
page_title: "anypoint_secret_group Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a secret group in Anypoint Secrets Manager.
---

# anypoint_secret_group (Resource)

Manages a secret group in Anypoint Secrets Manager.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

-> **Lifecycle note:** Deleting this resource also cascade-deletes all sub-resources on the Platform (keystores, truststores, certificates, shared secrets, TLS contexts, certificate pinsets). Sub-resource Terraform resources (`anypoint_secret_group_keystore`, etc.) must be declared as dependents — destroy them first in your config or Terraform will remove them from state automatically when the secret group is destroyed.

## Example Usage

```terraform
resource "anypoint_secret_group" "example" {
  environment_id = var.environment_id
  name           = "terraform-secrets"
  downloadable   = false
}
```

## Schema

### Required

- `environment_id` (String) Environment ID where the secret group is created.
- `name` (String) Name of the secret group.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `downloadable` (Boolean) Whether the secrets in this group can be downloaded. Defaults to `false`.

### Read-Only

- `id` (String) Unique identifier of the secret group.
- `current_state` (String) Current state of the secret group.

## Import

An existing `anypoint_secret_group` can be imported using its composite ID: `organization_id/environment_id/secret_group_id`.

- `organization_id` — UUID of the organization
- `environment_id` — UUID of the environment
- `secret_group_id` — UUID of the secret group

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_secret_group.imported
  id = "<organization_id>/<environment_id>/<secret_group_id>"
}

resource "anypoint_secret_group" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  name            = "<secret_group_name>"
  downloadable    = false
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
terraform import anypoint_secret_group.imported <organization_id>/<environment_id>/<secret_group_id>
```
