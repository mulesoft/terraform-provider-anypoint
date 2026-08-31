---
page_title: "anypoint_private_spaces Data Source - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Lists all Private Spaces in an organization. Use this to discover the private_space_id needed by other CloudHub 2.0 resources.
---

# anypoint_private_spaces (Data Source)

Lists all Private Spaces in an organization. Use this to discover the `id` (used as `private_space_id`) needed by resources such as [`anypoint_transit_gateway_connection`](../resources/anypoint_transit_gateway_connection.md) and [`anypoint_vpn_connection`](../resources/anypoint_vpn_connection.md).

## Example Usage

```terraform
data "anypoint_private_spaces" "all" {
  organization_id = var.organization_id
}

output "total_private_spaces" {
  value = length(data.anypoint_private_spaces.all.private_spaces)
}

# Map of Private Space name -> id, handy for looking up a private_space_id by name.
output "private_space_ids_by_name" {
  value = { for ps in data.anypoint_private_spaces.all.private_spaces : ps.name => ps.id }
}
```

## Schema

### Required

- `organization_id` (String) The organization ID to list Private Spaces for.

### Read-Only

- `private_spaces` (List of Object) The list of Private Spaces in the organization. See [`private_spaces`](#nestedschema--private_spaces) below.

<a id="nestedschema--private_spaces"></a>
### Nested Schema for `private_spaces`

Read-Only:

- `id` (String) The unique identifier of the Private Space (use this as `private_space_id`).
- `name` (String) The name of the Private Space.
- `status` (String) The current status of the Private Space.
- `status_message` (String) A human-readable message describing the current status.
- `region` (String) The region where the Private Space is provisioned.
- `organization_id` (String) The organization ID that owns the Private Space.
- `root_organization_id` (String) The root organization ID.
