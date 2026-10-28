<!--
  INTERNAL NOTE (not rendered): the arm-standalone-manager-service base path used by
  the underlying client (GET /standalone/api/v1/.../gateways) was VERIFIED against
  production on 2026-07-21 (live 200 with {content,pageNumber,pageSize,totalElements}).
  See internal/client/apimanagement/selfmanagedgateway.go for the full evidence trail.
-->
---
page_title: "anypoint_self_managed_gateways Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Lists all self-managed (connected-mode) Flex gateways that have registered in the given environment.
---

# anypoint_self_managed_gateways (Data Source)

Lists all **self-managed (connected-mode) Flex gateways** that have registered in the given
environment. Only gateways whose runtime has completed registration appear here; a gateway for
which a [`anypoint_self_managed_gateway`](../resources/anypoint_self_managed_gateway.md) token was
minted but never used to register a runtime is not listed.

Deleting a self-managed gateway is a **soft-delete**: the object lingers in the platform list
forever with status `DELETED`. These tombstones are filtered out by default; set
`include_deleted = true` to surface them (e.g. for auditing). To fetch one gateway by its id —
including a tombstone, and with the reported runtime `versions` — use the singular
[`anypoint_self_managed_gateway`](anypoint_self_managed_gateway.md) data source instead.

## Example Usage

```terraform
data "anypoint_self_managed_gateways" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "connected_gateways" {
  value = [
    for gw in data.anypoint_self_managed_gateways.all.gateways :
    gw.name if gw.status == "CONNECTED"
  ]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID to list gateways from.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.
- `include_deleted` (Boolean) Whether to include soft-deleted gateways. Deleting a self-managed gateway is a soft-delete: the object lingers in the platform list forever with status `DELETED`. By default these tombstones are filtered out; set this to `true` to include them (e.g. for auditing).

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>`.
- `gateways` (List of Object) List of self-managed gateways that have registered. Soft-deleted (status `DELETED`) gateways are excluded unless `include_deleted` is `true`. See [`gateways`](#nestedschema--gateways) below.

<a id="nestedschema--gateways"></a>
### Nested Schema for `gateways`

Read-Only:

- `id` (String) The unique identifier of the gateway.
- `name` (String) The name of the gateway.
- `status` (String) The current status of the gateway (e.g. `CONNECTED`, `DISCONNECTED`, `DELETED`).
- `last_update` (String) Timestamp of the gateway's last status update (RFC 3339).
- `tags` (List of String) Tags associated with the gateway.
- `replicas` (List of Object) Replica (runtime instance) status buckets reported by the gateway. The platform reports one entry per connectivity status with a running count. See [`replicas`](#nestedschema--replicas) below.

<a id="nestedschema--replicas"></a>
### Nested Schema for `gateways.replicas`

Read-Only:

- `status` (String) The connectivity status of this replica bucket (e.g. `CONNECTED`, `DISCONNECTED`).
- `count` (Number) The number of replicas currently in this status.
- `certificate_expiration_dates` (List of String) Certificate expiration timestamps reported by replicas in this bucket.
