---
page_title: "anypoint_transit_gateway Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Creates and manages a Transit Gateway in a CloudHub 2.0 private space.
---

# anypoint_transit_gateway (Resource)

Creates and manages a Transit Gateway in a CloudHub 2.0 private space.

## Example Usage

```terraform
resource "anypoint_transit_gateway" "example" {
  private_space_id       = var.private_space_id
  name                   = "my-transit-gateway"
  resource_share_id      = "arn:aws:ram:us-east-1:123456789012:resource-share/abc-123"
  resource_share_account = "123456789012"
  routes                 = ["10.0.0.0/16", "172.16.0.0/12"]
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space.
- `name` (String) The name of the Transit Gateway.
- `resource_share_id` (String) The resource share ID for the Transit Gateway.
- `resource_share_account` (String) The resource share account for the Transit Gateway.
- `routes` (List of String) List of route CIDR blocks for the Transit Gateway.

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) The unique identifier for the Transit Gateway.
- `spec` (Block) The specification of the Transit Gateway. See [below for nested schema](#nestedschema--spec).
- `status` (Block) The status of the Transit Gateway. See [below for nested schema](#nestedschema--status).

<a id="nestedschema--spec"></a>
### Nested Schema for `spec`

Read-Only:

- `resource_share` (Block) Resource share information. See [below for nested schema](#nestedschema--spec--resource_share).
- `region` (String) The region of the Transit Gateway.
- `space_name` (String) The space name.

<a id="nestedschema--spec--resource_share"></a>
### Nested Schema for `spec.resource_share`

Read-Only:

- `id` (String) Resource share ID.
- `account` (String) Resource share account.

<a id="nestedschema--status"></a>
### Nested Schema for `status`

Read-Only:

- `gateway` (String) Gateway status.
- `attachment` (String) Attachment status.
- `tgw_resource` (String) TGW resource link.
- `routes` (List of String) List of active routes.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_transit_gateway.example <private_space_id>:<transit_gateway_id>
```
