<!--
  INTERNAL NOTE (not rendered): the arm-standalone-manager-service base paths used by
  the underlying client (/standalone/api/v1/...) were VERIFIED against production on
  2026-07-21 via live READ-only method-discovery probes plus a UI network capture:
  mint = POST .../gatewaytokens (GET->405 Allow: POST); list = GET .../gateways (200);
  item GET/DELETE = .../gateways/{id} (Allow: GET,HEAD,DELETE,OPTIONS); no update
  (permissions report selfManagedGateways:{view,create,delete}). See
  internal/client/apimanagement/selfmanagedgateway.go for the full evidence trail.
-->
---
page_title: "anypoint_self_managed_gateway Resource - terraform-provider-anypoint"
subcategory: "API Management"
description: |-
  Manages a self-managed (connected-mode) Flex/Omni Gateway in Anypoint Platform. The platform mints a registration token; you run the Flex runtime on your own infrastructure and register it with that token.
---

# anypoint_self_managed_gateway (Resource)

Manages a **self-managed (connected-mode) Flex/Omni Gateway** in Anypoint Platform.

Unlike the [`anypoint_managed_omni_gateway`](anypoint_managed_omni_gateway.md) resource — where
the platform provisions and runs the gateway for you on CloudHub 2.0 — a self-managed gateway
runs on **your own infrastructure**. The platform never provisions a runtime, so there is no
"create gateway" API. Instead this resource models the genuine connected-mode primitives:

1. **Create** mints a short-lived `registration_token` scoped to the organization/environment.
2. You feed that token to your Flex runtime (for example
   `flexctl registration create <name> --connected=true --organization=<org> --token=<token>`).
   The runtime generates its own keypair, submits a CSR, and **self-registers** with the platform.
3. Once the runtime registers, the gateway object appears on the platform and this resource
   resolves its `gateway_id`, `status`, and `last_update` on the next read.
4. **Delete** removes the platform-side gateway object once it has registered.

-> **No private key in state:** The private key and CSR are generated on your runtime host by
`flexctl` and never leave it, so **no key material is ever written to Terraform state**. The
provider only mints the enrollment token and tracks/deletes the resulting gateway object.

-> **Connected App:** This resource requires a **standard connected app** (client credentials).
An admin connected app is not needed. The connected app must have the relevant API Manager /
gateway scopes.

~> **`registration_token` is a one-shot secret:** It is marked sensitive, is only returned when
the token is minted (during `apply`), and **cannot be recovered on import**. Capture it from the
apply output (or an output block) and hand it to your runtime promptly — it is short-lived.

## Example Usage

```terraform
resource "anypoint_self_managed_gateway" "example" {
  name           = "my-flex-gateway"
  environment_id = "env-id-here"
}

# The minted token is what you feed to flexctl on your runtime host.
output "flex_registration_token" {
  value     = anypoint_self_managed_gateway.example.registration_token
  sensitive = true
}
```

Register the runtime on your own host using the minted token. This mirrors the exact command the
Anypoint UI shows on the **Add Self-Managed Omni Gateway** screen (Docker/Podman), just with the
token sourced from the Terraform output:

```shell
docker run --entrypoint flexctl -u $UID \
  -v "$(pwd)":/registration mulesoft/flex-gateway \
  registration create \
  --organization=<your-org-id> \
  --token="$(terraform output -raw flex_registration_token)" \
  --output-directory=/registration \
  --connected=true \
  my-flex-gateway
```

If you run the `flexctl` binary directly (Linux install rather than a container), the equivalent is:

```shell
flexctl registration create my-flex-gateway \
  --connected=true \
  --organization=<your-org-id> \
  --token="$(terraform output -raw flex_registration_token)"
```

After the runtime registers, run `terraform refresh` (or the next `plan`) and the computed
`gateway_id`, `status`, and `last_update` attributes populate.

## Schema

### Required

- `name` (String) The name of the self-managed gateway. Must be unique within the organization/environment. Changing it forces a new registration.
- `environment_id` (String) The environment ID the gateway registers into. Changing it forces a new registration.

### Optional

- `organization_id` (String) The organization ID. If not specified, the organization ID is inferred from the connected app credentials.

### Read-Only

- `id` (String) Composite identifier: `<organization_id>/<environment_id>/<name>`. Stable across the gateway's lifecycle, even before the runtime registers.
- `registration_token` (String, Sensitive) The registration token minted for this gateway. Feed it to the Flex runtime so it can self-register. This is a short-lived, one-shot enrollment secret and cannot be recovered on import.
- `gateway_id` (String) The platform-assigned gateway ID, resolved once the runtime registers. Empty until the gateway appears on the platform.
- `status` (String) The current status of the registered gateway (e.g. `CONNECTED`, `DISCONNECTED`). Empty until it registers.
- `last_update` (String) Timestamp of the gateway's last status update, as reported by the platform (RFC 3339). Empty until the runtime registers.

## Import

An existing self-managed gateway can be imported using a composite ID. Use the 2-part form for
root-org gateways and the 3-part form when the gateway belongs to a Business Group (sub-org).

-> **Note:** `registration_token` is a one-shot enrollment secret and is **not recoverable on
import** — it will be null in state after an import. This does not affect a gateway that has
already registered.

### Using an import block (Terraform ≥ 1.5 — recommended)

**Root org (2-part ID):**

```terraform
import {
  to = anypoint_self_managed_gateway.imported
  id = "<environment_id>/<name>"
}

resource "anypoint_self_managed_gateway" "imported" {
  name           = "<gateway_name>"
  environment_id = "<environment_id>"
}
```

**Sub-org (3-part ID):**

```terraform
import {
  to = anypoint_self_managed_gateway.imported
  id = "<organization_id>/<environment_id>/<name>"
}

resource "anypoint_self_managed_gateway" "imported" {
  organization_id = "<organization_id>"
  name            = "<gateway_name>"
  environment_id  = "<environment_id>"
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
terraform import anypoint_self_managed_gateway.imported <environment_id>/<name>

# Sub-org:
terraform import anypoint_self_managed_gateway.imported <organization_id>/<environment_id>/<name>
```
