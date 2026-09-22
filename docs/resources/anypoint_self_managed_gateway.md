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

-> **Authentication:** This resource calls the **standalone (self-managed) gateway control-plane
API** — token mint, gateway list/get, and delete all ride
`standalone/api/v1/organizations/{org}/environments/{env}/gateways`. A `client_credentials`
Connected App works — grant it **Manage Servers**, **Read Servers**, and **View Organization**
(the Runtime Manager server scopes). A Connected App missing these scopes is rejected with
`HTTP 401`/`403`; the fix is to add the scopes (or use `auth_type = "user"` with a user that has
the equivalent permissions). See
[Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

-> **Read/list route (managed vs self-managed):** Self-managed gateways are read from the
**standalone** service above. The **managed** Gateway Manager route
(`gatewaymanager/api/v1/.../gateways`) lists only *managed* Omni Gateways and returns an empty
`content` array for self-managed ones, so it cannot be used to verify a self-managed gateway.
(The unified Anypoint UI list is a facade served by
`gatewaymanager/xapi/v1/.../gateways?kind=selfManaged`.)

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
`gateway_id` and `last_update` attributes populate. At this point `status` is `DISCONNECTED`:
the gateway object exists, but nothing is running yet.

### Registering is not the same as running

`flexctl registration create` only **enrols** the gateway. It contacts the platform, generates
the keypair, and writes `registration.yaml` to the output directory — it does not start a
gateway. Until you start a runtime the gateway shows `DISCONNECTED` with **0 replicas**, which
is expected rather than a fault.

Start the runtime by mounting the registration into the Flex Gateway container:

```shell
docker run --rm \
  -v "$(pwd)":/usr/local/share/mulesoft/flex-gateway/conf.d \
  -p 8080:8080 \
  mulesoft/flex-gateway
```

~> **Run this from the directory that holds `registration.yaml`.** The command mounts
`$(pwd)`, so running it from anywhere else silently mounts a directory with no
registration in it. The failure is loud but misleading: the agent logs a few hundred
lines of `Extension default/<name>: the registration configuration is missing` and
never prints `Gateway: ... Mode=connected`. The quickest way to tell the two apart is
to look for this line near the top of the log, which only appears when the mount is
correct:

```
[flex-gateway-agent][info] Reading config file '/usr/local/share/mulesoft/flex-gateway/conf.d/registration.yaml'
```

On Apple Silicon add `--platform linux/amd64`; the published image is `linux/amd64` and Docker
otherwise warns about the architecture mismatch. Once the agent logs `Anypoint websocket:
connected`, another `terraform refresh` reports `status = "CONNECTED"`.

~> Re-running `registration create` in a directory that already has a `registration.yaml` fails
with `file already exists`. To re-register, delete that file **and** mint a fresh token — the
previous one is spent.

### Installation method does not change the Terraform configuration

The Anypoint UI offers several ways to install a self-managed gateway (Docker, a Linux package,
Kubernetes/Helm, and so on). They all consume **the same registration token**, and this resource
mints it the same way for all of them: the token is scoped only to the organization and
environment. So there is nothing per-method to configure here — pick whichever installation the
UI offers and feed it the `registration_token` output.

-> This resource models **connected mode** only. A gateway running in *local* mode never
registers with the control plane, so no gateway object exists for Terraform to manage.

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
- `status` (String) The current status of the registered gateway. Moves through three states: empty immediately after `apply` (the token is minted but nothing has registered), `DISCONNECTED` once a runtime has registered but is not running, and `CONNECTED` once a runtime is up.
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
