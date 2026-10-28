# Self-Managed Gateways Data Source Example

This example demonstrates how to list **self-managed (connected-mode) Flex gateways** that have
registered in an environment, using the `anypoint_self_managed_gateways` data source.

## Overview

Only gateways whose runtime has completed registration appear in this list. A gateway for which a
registration token was minted (via [`anypoint_self_managed_gateway`](../self_managed_gateway)) but
never used to enroll a runtime is **not** listed — the platform object does not exist until the
runtime self-registers.

## What This Example Shows

- Listing all registered self-managed gateways in an environment
- Filtering to only those currently reporting `CONNECTED`
- Looking up a specific gateway by name with a `local`
- Auditing soft-deleted (`DELETED`) tombstones with `include_deleted = true`

## Usage

### Step 1: Set Required Variables

Create a `terraform.tfvars` file:

```hcl
anypoint_client_id     = "your-client-id"
anypoint_client_secret = "your-client-secret"
anypoint_base_url      = "https://anypoint.mulesoft.com"

organization_id = "your-org-id"
environment_id  = "your-env-id"
gateway_name    = "my-flex-gateway"  # optional
```

### Step 2: Apply and inspect outputs

```bash
terraform init
terraform apply

terraform output all_gateways
terraform output connected_gateways
terraform output matched_gateway_id
```

## Schema Reference

Each gateway in `gateways` exposes:

- `id` — the platform-assigned gateway UUID
- `name` — the gateway name
- `status` — current connectivity status (e.g. `CONNECTED`, `DISCONNECTED`, `DELETED`)
- `last_update` — RFC 3339 timestamp of the last status update
- `tags` — list of tags on the gateway
- `replicas` — one entry per connectivity-status bucket, each with `status`, `count`, and
  `certificate_expiration_dates`

The data source also accepts an optional top-level `include_deleted` (bool). Deleting a
self-managed gateway is an async **soft-delete**: the platform flips the object's status to
`DELETED` but keeps it in the list forever (a tombstone). By default these tombstones are
filtered out; set `include_deleted = true` to surface them for auditing.

## Related

- [`self_managed_gateway`](../self_managed_gateway) — register a self-managed gateway (mints the token)
- [`managed_omni_gateway_ds`](../managed_omni_gateway_ds) — list platform-managed gateways
