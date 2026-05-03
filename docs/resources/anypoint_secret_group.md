---
page_title: "anypoint_secret_group Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a secret group in Anypoint Secrets Manager.
---

# anypoint_secret_group (Resource)

Manages a secret group in Anypoint Secrets Manager.

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

Import is supported using the following syntax:

```shell
terraform import anypoint_secret_group.example organization_id/environment_id/secret_group_id
```
