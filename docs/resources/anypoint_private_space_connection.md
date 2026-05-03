---
page_title: "anypoint_private_space_connection Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages an Anypoint Private Space Connection.
---

# anypoint_private_space_connection (Resource)

Manages an Anypoint Private Space Connection.

## Example Usage

```terraform
resource "anypoint_private_space_connection" "example" {
  private_space_id = anypoint_private_space.example.id
  name             = "my-connection"
  type             = "vpn"
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space this connection belongs to.
- `name` (String) The name of the private space connection.
- `type` (String) The type of the private space connection.

### Read-Only

- `id` (String) The unique identifier for the private space connection.
- `status` (String) The status of the private space connection.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_private_space_connection.example <private_space_id>:<connection_id>
```
