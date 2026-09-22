# Self-Managed (Connected-Mode) Flex/Omni Gateway Example

This example demonstrates how to register a **self-managed** Flex/Omni Gateway in Anypoint
Platform using Terraform.

## Overview

Unlike a **managed** Omni Gateway (see the [`managed_omni_gateway`](../managed_omni_gateway)
example), where the platform provisions and runs the gateway for you on CloudHub 2.0, a
**self-managed** gateway runs on **your own infrastructure** (VM, container, Kubernetes, bare
metal). The platform never provisions a runtime — so there is no "create gateway" API.

Instead, the connected-mode flow is:

1. **Mint a registration token** — `anypoint_self_managed_gateway` issues a short-lived,
   org/env-scoped token during `terraform apply`.
2. **Register your runtime** — you feed that token to the Flex runtime on your host
   (`flexctl registration create <name> --mode=connected --token=<token>`). The runtime generates
   its own keypair, submits a CSR, and self-registers with the platform.
3. **Resolve** — once the runtime registers, the gateway object appears on the platform and the
   resource's computed `gateway_id`, `status`, and `last_update` populate on the next
   `terraform refresh` / `plan`.
4. **Delete** — `terraform destroy` removes the platform-side gateway object once it has
   registered. If the runtime never registered, there is nothing to delete (a minted token
   simply expires).

> **No key material in state.** The private key and CSR are generated on your runtime host by
> `flexctl` and never leave it. Terraform state only ever holds the minted enrollment token —
> never a private key.

## What This Example Creates

- **1 self-managed gateway registration** named `my-flex-gateway`, which mints a registration
  token you then use to enroll your runtime.

## Prerequisites

1. **Anypoint Platform Account** with API Manager / gateway permissions
2. **Standard Connected App Credentials** (Client ID and Secret) with the relevant scopes
3. **Environment ID** — the environment the gateway registers into
4. **A host to run the Flex runtime** with `flexctl` installed

## Usage

### Step 1: Set Required Variables

Create a `terraform.tfvars` file:

```hcl
anypoint_client_id     = "your-client-id"
anypoint_client_secret = "your-client-secret"
anypoint_base_url      = "https://anypoint.mulesoft.com"

environment_id = "your-env-id"
```

### Step 2: Apply

```bash
terraform init
terraform apply
```

### Step 3: Capture the token and register your runtime

The token is **sensitive** and **one-shot** — capture it promptly. This matches the command the
Anypoint UI shows on **Add Self-Managed Omni Gateway** (Docker/Podman), with the token sourced
from the Terraform output:

```bash
docker run --entrypoint flexctl -u $UID \
  -v "$(pwd)":/registration mulesoft/flex-gateway \
  registration create \
  --organization=<your-org-id> \
  --token="$(terraform output -raw flex_registration_token)" \
  --output-directory=/registration \
  --mode=connected \
  my-flex-gateway
```

> `--mode` accepts `connected` / `local` / `disconnected`. The older `--connected` /
> `--connected=true` flag is deprecated in current `flexctl`; use `--mode=connected`.

### Step 4: Refresh to resolve the registered gateway

After the runtime registers, re-read state so the computed fields populate:

```bash
terraform refresh
terraform output gateway_id
terraform output gateway_status
```

## Important Notes

- **`registration_token` is a one-shot secret.** It is only returned at mint time (during
  `apply`) and is **not recoverable on import**. Store it securely and use it promptly — it is
  short-lived.
- **Computed fields are empty until registration.** `gateway_id`, `status`, and `last_update` stay
  empty (not unknown) right after `apply`; they populate once your runtime self-registers and you
  refresh.
- **Changing `name` or `environment_id` forces a new registration** (and thus a new token).
- **Delete is a soft-delete.** `terraform destroy` issues an async delete (HTTP 202) that flips the
  platform object's status to `DELETED`; the object then lingers in the platform list forever as a
  tombstone (there is no hard-delete). Destroy is idempotent — a repeated destroy of an
  already-deleted gateway is treated as success — and name resolution skips `DELETED` tombstones so
  a re-registered gateway that reuses the name binds to the live object, not the dead one.

## Import

An existing self-managed gateway can be imported using a composite ID — see
[`import.tf`](./import.tf) in this directory. Note that `registration_token` is not recoverable on
import.

## Related

- [`self_managed_gateway_ds`](../self_managed_gateway_ds) — list registered self-managed gateways
- [`managed_omni_gateway`](../managed_omni_gateway) — platform-managed (CloudHub 2.0) gateways
- [Flex Gateway connected mode docs](https://docs.mulesoft.com/gateway/flex-conn-reg-run)
