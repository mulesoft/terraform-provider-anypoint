---
page_title: "anypoint_private_space Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages an Anypoint Private Space.
---

# anypoint_private_space (Resource)

Manages an Anypoint Private Space.

## Example Usage

```terraform
# Private space using default organization (from provider credentials)
resource "anypoint_private_space" "example" {
  name   = "my-private-space"
  region = "us-east-1"
}

# Private space in a specific organization
resource "anypoint_private_space" "custom_org" {
  name            = "my-private-space-custom-org"
  region          = "us-east-1"
  organization_id = "your-organization-id"
}
```

## Schema

### Required

- `name` (String) The name of the private space.
- `region` (String) The region where the private space is located.

### Optional

- `enable_iam_role` (Boolean) Whether to enable IAM role for the private space. Defaults to `false`.
- `enable_egress` (Boolean) Whether to enable egress for the private space. Defaults to `false`.
- `organization_id` (String) The organization ID where the private space will be created. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) The unique identifier for the private space.
- `status` (String) The status of the private space.
- `root_organization_id` (String) The root organization ID of the private space.
- `mule_app_deployment_count` (Number) The number of mule apps deployed in the private space.
- `days_left_for_relaxed_quota` (Number) The number of days left for relaxed quota.
- `vpc_migration_in_progress` (Boolean) Whether the VPC migration is in progress.
- `managed_firewall_rules` (List of String) The managed firewall rules for the private space.
- `firewall_rules` (List of String) The firewall rules for the private space.
- `global_space_status` (Map) The global space status for the private space.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_private_space.example <private_space_id>
```
