---
page_title: "anypoint_private_space_upgrade Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Schedules an upgrade for a CloudHub 2.0 private space. Scheduled upgrades can be cancelled by deleting this resource.
---

# anypoint_private_space_upgrade (Resource)

Schedules an upgrade for a CloudHub 2.0 private space. Scheduled upgrades can be cancelled by deleting this resource.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_private_space_upgrade" "example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id
  date             = "2025-09-12"
  opt_in           = true
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space to upgrade.
- `date` (String) The date when the upgrade should be scheduled (format: YYYY-MM-DD).
- `opt_in` (Boolean) Whether to opt in to the upgrade.

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) The unique identifier for the upgrade operation.
- `scheduled_update_time` (String) The scheduled update time returned by the API.
- `status` (String) The status of the upgrade operation.

## Import

An existing scheduled upgrade can be imported using a composite ID. Use the 3-part form for root-org private spaces and the 4-part form when the private space belongs to a Business Group (sub-org).

### Using an import block (Terraform ≥ 1.5 — recommended)

**Root org (3-part ID):**

```terraform
import {
  to = anypoint_private_space_upgrade.imported
  id = "<private_space_id>:<date>:<opt_in>"
}

resource "anypoint_private_space_upgrade" "imported" {
  private_space_id = "<private_space_id>"
  date             = "<date>"
  opt_in           = true
}
```

**Sub-org (4-part ID):**

```terraform
import {
  to = anypoint_private_space_upgrade.imported
  id = "<org_id>:<private_space_id>:<date>:<opt_in>"
}

resource "anypoint_private_space_upgrade" "imported" {
  private_space_id = "<private_space_id>"
  organization_id  = "<org_id>"
  date             = "<date>"
  opt_in           = true
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
# Root org:
terraform import anypoint_private_space_upgrade.imported <private_space_id>:<date>:<opt_in>

# Sub-org:
terraform import anypoint_private_space_upgrade.imported <org_id>:<private_space_id>:<date>:<opt_in>
```
