<!--
  INTERNAL NOTE (not rendered): the arm-standalone-manager-service base path used by
  the underlying client (GET /standalone/api/v1/.../gateways/{id}) was VERIFIED against
  production on 2026-07-21 (live 404 body "Gateway not found by id" for a valid-but-absent
  id; live 200 for a real registered gateway).
  See internal/client/apimanagement/selfmanagedgateway.go for the full evidence trail.
-->
---
page_title: "anypoint_self_managed_gateway Data Source - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Fetches a single self-managed (connected-mode) Flex gateway by its ID.
---

# anypoint_self_managed_gateway (Data Source)

Fetches a single **self-managed (connected-mode) Flex gateway** by its ID.

Unlike the plural [`anypoint_self_managed_gateways`](anypoint_self_managed_gateways.md) data
source — which lists every registered gateway and, by default, hides soft-deleted (status
`DELETED`) tombstones — this singular data source returns exactly the gateway you request by
id, **including one that has been soft-deleted** (deleting a self-managed gateway is a
soft-delete: the object lingers in the platform forever with status `DELETED`). It also
surfaces the reported runtime `versions`, which the plural list does not expose.

Use it to reference an existing gateway's live status, tags, replica health, or reported
runtime versions from elsewhere in your configuration. Referencing an id that does not exist
is a configuration error and fails the plan.

## Example Usage

```terraform
data "anypoint_self_managed_gateway" "one" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  id              = "8f2a1c34-5b6d-4e7f-90ab-cdef12345678"
}

output "gateway_status" {
  value = data.anypoint_self_managed_gateway.one.status
}

# Total running replicas across all connectivity buckets.
output "running_replicas" {
  value = sum([
    for r in data.anypoint_self_managed_gateway.one.replicas : r.count
  ])
}
```

## Schema

### Required

- `id` (String) The unique identifier of the self-managed gateway to fetch.
- `environment_id` (String) The environment ID the gateway is registered in.

### Optional

- `organization_id` (String) The organization ID. Defaults to the provider credentials organization.

### Read-Only

- `name` (String) The name of the gateway.
- `status` (String) The current status of the gateway (e.g. `CONNECTED`, `DISCONNECTED`, `DELETED`).
- `last_update` (String) Timestamp of the gateway's last status update (RFC 3339).
- `tags` (List of String) Tags associated with the gateway.
- `versions` (List of String) Runtime versions reported by the gateway's replicas. Empty until a replica reports a version. Not exposed by the plural data source.
- `replicas` (List of Object) Replica (runtime instance) status buckets reported by the gateway. The platform reports one entry per connectivity status with a running count. See [`replicas`](#nestedschema--replicas) below.

<a id="nestedschema--replicas"></a>
### Nested Schema for `replicas`

Read-Only:

- `status` (String) The connectivity status of this replica bucket (e.g. `CONNECTED`, `DISCONNECTED`).
- `count` (Number) The number of replicas currently in this status.
- `certificate_expiration_dates` (List of String) Certificate expiration timestamps reported by replicas in this bucket.
