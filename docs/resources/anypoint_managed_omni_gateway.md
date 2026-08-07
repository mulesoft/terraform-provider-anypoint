---
page_title: "anypoint_managed_omni_gateway Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages a CloudHub 2.0 Managed Omni Gateway instance in Anypoint Platform.
---

# anypoint_managed_omni_gateway (Resource)

Manages a CloudHub 2.0 Managed Omni Gateway instance in Anypoint Platform.

-> **Gateway type:** This resource manages a **MuleSoft Managed Omni Gateway** (CloudHub 2.0), where the platform provisions and runs the gateway for you. For a **self-managed (connected-mode) Flex/Omni Gateway** that you run on your own infrastructure, use [`anypoint_self_managed_gateway`](anypoint_self_managed_gateway.md) instead.

-> **Authentication:** This resource calls **Gateway Manager / API Manager control-plane APIs** (the gateway lifecycle and/or the `gateway_id` pre-flight). A `client_credentials` Connected App works — grant it **Manage Servers**, **Read Servers**, and **View Organization** (plus API Manager scopes such as **Manage APIs Configuration**, **Manage Policies**, **Deploy API Proxies**, and **Exchange Viewer** for Omni Gateway operations). A Connected App missing these scopes is rejected with `HTTP 401`/`403` before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

-> **Read/list route (managed vs self-managed):** Managed Omni Gateways are read from the Gateway Manager route (`gatewaymanager/api/v1/.../gateways`), which lists *only* managed gateways. Self-managed gateways do **not** appear there — they are read from the standalone route (`standalone/api/v1/.../gateways`); see [`anypoint_self_managed_gateway`](anypoint_self_managed_gateway.md).

-> **Tracing note:** The Gateway Manager API does not echo back `configuration.tracing` in POST/PUT responses. The provider retains the plan-requested value in state after create/update so that `tracing.enabled = true` works correctly. On the next `terraform refresh` or `plan`, the provider reads the live value from the API for accurate drift detection.

## Example Usage

```terraform
resource "anypoint_managed_omni_gateway" "example" {
  name           = "my-omni-gateway"
  environment_id = "env-id-here"
  target_id      = "target-private-space-id"

  release_channel = "lts"
  size            = "small"

  ingress = {
    forward_ssl_session = true
    last_mile_security  = true
  }

  properties = {
    upstream_response_timeout = 15
    connection_idle_timeout   = 60
  }

  logging = {
    level        = "info"
    forward_logs = true
  }

  tracing = {
    enabled = false
  }
}
```

## Schema

### Required

- `name` (String) The name of the managed Omni Gateway.
- `environment_id` (String) The environment ID where the gateway will be deployed.
- `target_id` (String) The target (private space) ID for the gateway deployment.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `runtime_version` (String) The Omni Gateway runtime version (e.g., '1.9.9'). If omitted, the provider auto-selects the latest version for the chosen release_channel.
- `release_channel` (String) The release channel for the gateway. Valid values: `lts`, `edge`. Defaults to `lts`.
- `size` (String) The size of the gateway instance. Valid values: `small`, `large`. Defaults to `small`.
- `ingress` (Block) Ingress configuration for the gateway. See [below for nested schema](#nestedschema--ingress).
- `properties` (Block) Runtime properties for the gateway. See [below for nested schema](#nestedschema--properties).
- `logging` (Block) Logging configuration for the gateway. See [below for nested schema](#nestedschema--logging).
- `tracing` (Block) Distributed tracing configuration for the gateway. See [below for nested schema](#nestedschema--tracing).

### Read-Only

- `id` (String) The unique identifier of the managed Omni Gateway.
- `status` (String) The current status of the managed Omni Gateway.

<a id="nestedschema--ingress"></a>
### Nested Schema for `ingress`

Optional:

- `public_url` (String) The public URL for the gateway ingress. Auto-derived from the target domain when empty.
- `internal_url` (String) The internal URL for the gateway ingress. Auto-derived from the target domain when empty.
- `forward_ssl_session` (Boolean) Whether to forward SSL sessions to upstream services. Defaults to `true`.
- `last_mile_security` (Boolean) Whether to enable last-mile security (TLS between gateway and upstream). Defaults to `true`.

<a id="nestedschema--properties"></a>
### Nested Schema for `properties`

Optional:

- `upstream_response_timeout` (Number) Timeout in seconds for upstream service responses. Defaults to `15`.
- `connection_idle_timeout` (Number) Timeout in seconds for idle connections. Defaults to `60`.

<a id="nestedschema--logging"></a>
### Nested Schema for `logging`

Optional:

- `level` (String) The log level. Valid values: `debug`, `info`, `warn`, `error`. Defaults to `info`.
- `forward_logs` (Boolean) Whether to forward logs to Anypoint Monitoring. Defaults to `true`.

<a id="nestedschema--tracing"></a>
### Nested Schema for `tracing`

Optional:

- `enabled` (Boolean) Whether distributed tracing is enabled. Defaults to `false`.

## Import

An existing Managed Omni Gateway can be imported using a composite ID. Use the 2-part form for root-org gateways and the 3-part form when the gateway belongs to a Business Group (sub-org).

The `gateway_id` is the UUID of the gateway visible in Runtime Manager or the Gateway Manager API.

### Using an import block (Terraform ≥ 1.5 — recommended)

**Root org (2-part ID):**

```terraform
import {
  to = anypoint_managed_omni_gateway.imported
  id = "<environment_id>/<gateway_id>"
}

resource "anypoint_managed_omni_gateway" "imported" {
  name           = "<gateway_name>"
  environment_id = "<environment_id>"
  target_id      = "<target_id>"
}
```

**Sub-org (3-part ID):**

```terraform
import {
  to = anypoint_managed_omni_gateway.imported
  id = "<organization_id>/<environment_id>/<gateway_id>"
}

resource "anypoint_managed_omni_gateway" "imported" {
  organization_id = "<organization_id>"
  name            = "<gateway_name>"
  environment_id  = "<environment_id>"
  target_id       = "<target_id>"
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
terraform import anypoint_managed_omni_gateway.imported <environment_id>/<gateway_id>

# Sub-org:
terraform import anypoint_managed_omni_gateway.imported <organization_id>/<environment_id>/<gateway_id>
```
