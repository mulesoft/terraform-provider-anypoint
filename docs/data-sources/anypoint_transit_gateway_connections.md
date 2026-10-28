---
page_title: "anypoint_transit_gateway_connections Data Source - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Lists all transit gateway connections (attachments) in a CloudHub 2.0 Private Space, including their routes.
---

# anypoint_transit_gateway_connections (Data Source)

Lists all transit gateway connections (attachments) in a CloudHub 2.0 Private Space, including their routes.

## Example Usage

```terraform
data "anypoint_transit_gateway_connections" "all" {
  organization_id  = var.organization_id
  private_space_id = var.private_space_id
}

output "total_transit_gateway_connections" {
  value = length(data.anypoint_transit_gateway_connections.all.transit_gateway_connections)
}

output "transit_gateway_connection_names" {
  value = [for tgw in data.anypoint_transit_gateway_connections.all.transit_gateway_connections : tgw.name]
}
```

## Schema

### Required

- `organization_id` (String) The organization ID.
- `private_space_id` (String) The ID of the Private Space to list transit gateway connections for.

### Read-Only

- `transit_gateway_connections` (List of Object) The list of transit gateway connections (attachments). See [`transit_gateway_connections`](#nestedschema--transit_gateway_connections) below.

<a id="nestedschema--transit_gateway_connections"></a>
### Nested Schema for `transit_gateway_connections`

Read-Only:

- `id` (String) The unique identifier of the transit gateway connection.
- `name` (String) The name of the transit gateway connection (attachment).
- `status` (String) The current status (e.g. `Pending`, `Available`).
- `routes` (List of String) The CIDR routes configured on this transit gateway connection. An empty
  routes array is returned as an empty list, never null. This is the same shape as the
  `routes` attribute on the [`anypoint_transit_gateway_connection`](anypoint_transit_gateway_connection)
  data source and the `anypoint_transit_gateway_connection` resource, so a value read here can be
  passed directly to the resource without unwrapping:

```terraform
resource "anypoint_transit_gateway_connection" "copy" {
  # ...
  routes = data.anypoint_transit_gateway_connections.all.transit_gateway_connections[0].routes
}
```
