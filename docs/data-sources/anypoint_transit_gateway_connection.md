---
page_title: "anypoint_transit_gateway_connection Data Source - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Fetches a single Transit Gateway connection (attachment) in a CloudHub 2.0 Private Space by its ID.
---

# anypoint_transit_gateway_connection (Data Source)

Fetches a single Transit Gateway connection (attachment) in a CloudHub 2.0 Private Space by its ID.

Unlike the plural [`anypoint_transit_gateway_connections`](anypoint_transit_gateway_connections.md) data source — which lists every connection but surfaces only `id`, `name`, `status`, and `routes` — this singular data source returns the **full attribute set** for one connection: the AWS Transit Gateway ID the platform discovered, the AWS RAM resource share (ID and owner account), the AWS region, and the attachment state. Use it to reference an existing connection's computed details from elsewhere in your configuration.

## Example Usage

```terraform
data "anypoint_transit_gateway_connection" "one" {
  organization_id  = var.organization_id
  private_space_id = var.private_space_id
  id               = "tgw-0af6a0a1b5ae060b1"
}

output "aws_transit_gateway_id" {
  value = data.anypoint_transit_gateway_connection.one.aws_transit_gateway_id
}

output "connection_region" {
  value = data.anypoint_transit_gateway_connection.one.region
}
```

## Schema

### Required

- `id` (String) The unique identifier of the transit gateway connection to fetch.
- `organization_id` (String) The organization ID.
- `private_space_id` (String) The ID of the Private Space the transit gateway connection belongs to.

### Read-Only

- `name` (String) The name of the transit gateway connection.
- `aws_transit_gateway_id` (String) The AWS Transit Gateway ID the platform discovered from the resource share.
- `resource_share_id` (String) The AWS RAM resource share ID (UUID format).
- `resource_share_account` (String) The AWS account ID that owns the Transit Gateway.
- `region` (String) The AWS region of the transit gateway connection.
- `status` (String) The current gateway status (e.g. `Pending`, `Available`).
- `attachment` (String) The current attachment status of the transit gateway connection.
- `routes` (List of String) The CIDR routes configured on this transit gateway connection.
