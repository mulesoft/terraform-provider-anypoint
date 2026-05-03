# Roadmap — Write-Only Migration for Secret-Valued Attributes

This document tracks the provider's plan to migrate secret-valued resource
attributes (PEM passphrases, private keys, keystore/truststore passphrases,
shared-secret passwords, …) from regular `Sensitive: true` attributes to
**write-only attributes** (`WriteOnly: true` on the plugin-framework schema).

The motivation is the long-standing Terraform-wide behavior that
`Sensitive: true` redacts values from plan/apply output but **does not protect
them in `terraform.tfstate`**. See [`docs/SECURITY.md`](./SECURITY.md) §3 for
the full explanation and comparison to the AWS / GCP / Kong / Kubernetes
providers.

## Framework readiness

| Requirement | Status |
|---|---|
| Terraform core ≥ 1.11 (write-only attributes GA) | Customers must install 1.11+ to benefit. We will keep 1.6+ supported by emitting a graceful validation error on older cores. |
| `terraform-plugin-framework` ≥ 1.11 | ✅ On `v1.19.0` today — no dep bump required. |
| Acceptance-test harness covers write-only paths | ⚠️ Needs new test helpers — see Phase 0. |

Write-only attributes were GA'd with Terraform core 1.11 (2025). The
plugin-framework has supported `WriteOnly: true` on string / number / bool
schema attributes since `v1.11.0`, so this provider has been framework-ready
since before v0.1.0 shipped.

## Attribute inventory

### Tier A — migrate to `WriteOnly` (true secrets, never re-derivable from server)

| Resource | Attribute | Today | Target |
|---|---|---|---|
| `anypoint_keystore` | `passphrase` | `Sensitive: true` | `WriteOnly: true` + `passphrase_wo_version` sidecar |
| `anypoint_keystore` | `key_base64` (PEM private key) | `Sensitive: true` | `WriteOnly: true` + `key_wo_version` sidecar |
| `anypoint_keystore` | `keystore_file_base64` (JKS/PKCS12) | `Sensitive: true` | `WriteOnly: true` + `keystore_file_wo_version` sidecar |
| `anypoint_truststore` | `passphrase` | `Sensitive: true` | `WriteOnly: true` + `passphrase_wo_version` sidecar |
| `anypoint_shared_secret` | `password` | `Sensitive: true` | `WriteOnly: true` + `password_wo_version` sidecar |
| `anypoint_shared_secret` | `secret_access_key` | `Sensitive: true` | `WriteOnly: true` + `secret_access_key_wo_version` sidecar |
| `anypoint_shared_secret` | `key` (symmetric) | `Sensitive: true` | `WriteOnly: true` + `key_wo_version` sidecar |
| `anypoint_shared_secret` | `content` (Blob) | `Sensitive: true` | `WriteOnly: true` + `content_wo_version` sidecar |
| `anypoint_tls_context` | `key` (PEM private key) | `Sensitive: true` | `WriteOnly: true` + `key_wo_version` sidecar |
| `anypoint_tls_context` | `store_passphrase` | `Sensitive: true` | `WriteOnly: true` + `store_passphrase_wo_version` sidecar |
| `anypoint_tls_context` | `key_passphrase` | `Sensitive: true` | `WriteOnly: true` + `key_passphrase_wo_version` sidecar |
| `anypoint_tls_context` | `keystore_base64` (JKS) | `Sensitive: true` | `WriteOnly: true` + `keystore_base64_wo_version` sidecar |

### Tier B — keep `Sensitive: true`, do **not** migrate

These attributes are marked `Sensitive` for UI redaction but are not
cryptographically secret — they're public certificate material, safe to live
in state. Migrating them to write-only would complicate drift detection with
no security upside.

| Resource | Attribute | Rationale |
|---|---|---|
| `anypoint_keystore` | `certificate_base64` | Public certificate (the cert half, not the key). |
| `anypoint_keystore` | `ca_path_base64` | CA chain, public. |
| `anypoint_truststore` | `content_base64` (or equivalent) | CA trust material, public. |
| `anypoint_certificate` | `certificate_base64` | Public certificate. |
| `anypoint_certificate_pinset` | `certificate_base64` | Public pin material. |
| `anypoint_tls_context` | `certificate` | Public certificate. |

### Tier C — special cases

| Resource | Attribute | Today | Notes |
|---|---|---|---|
| `anypoint_connected_app` | `client_secret` | `Sensitive: true` (returned by server on create) | **Cannot** migrate to pure write-only because the server only reveals the secret value on the create response. If we make it write-only we lose the ability to surface it back to the user via an output. Options: (1) leave as-is with a documented state-hygiene requirement, (2) add an explicit `client_secret_wo` write-only variant for imports, keeping the computed `client_secret` for create, (3) adopt the **ephemeral resource** shape (Terraform 1.10+) where the secret is handed to the user in an ephemeral output and never stored. Leaning toward (3) long-term; (1) for v0.2.0. |
| Provider config `password` | `Sensitive: true` | Provider-config attributes are never persisted in state (Terraform core never writes provider config to state). WriteOnly on provider config is therefore **unnecessary** — it's already effectively write-only. |

## Drift detection for write-only attributes

Once an attribute is `WriteOnly: true` the framework drops its value from
state entirely. That means Terraform has nothing to compare against on the
next plan, so there is **no automatic drift detection** — if an operator
rotates the passphrase out-of-band, Terraform will not notice.

The standard pattern, used by AWS `aws_db_instance.password` and Kubernetes
`kubernetes_secret_v1.data_wo`, is a **version sidecar**:

```hcl
resource "anypoint_keystore" "example" {
  organization_id = var.org_id
  environment_id  = var.env_id
  secret_group_id = anypoint_secret_group.example.id
  type            = "PEM"

  certificate_base64 = base64encode(file("cert.pem"))

  # Write-only: accepted from config, never stored in state.
  key_base64 = base64encode(file("key.pem"))
  passphrase = var.keystore_passphrase

  # Integer the operator bumps whenever the secret is rotated. Terraform
  # diffs the version in state vs. the version in config, not the secret
  # value itself. When they differ, Terraform calls the update path and the
  # new write-only value is sent to the API.
  key_wo_version        = 2
  passphrase_wo_version = 2
}
```

Semantics:

- Bump `*_wo_version` → Terraform plans an in-place update → provider reads
  the fresh `WriteOnly` value from config and PUTs it to the API. The new
  version is stored in state.
- Don't bump it → Terraform treats the write-only attribute as unchanged,
  even if the user pasted a new secret into config. (This is by design;
  without it, the attribute would always diff because state is empty.)
- Out-of-band rotation on the Anypoint UI → **Terraform will not detect
  drift** on a pure `WriteOnly` attribute. This is the same trade-off the
  AWS provider accepts for `aws_db_instance.password`. If stricter drift
  detection is required, add a hashed sidecar (`passphrase_sha256`) that
  the provider computes server-side and compares on every Read.

## Phased rollout

### Phase 0 — Test harness (pre-v0.2.0)

- Add `resource.TestCheckFunc` helpers for asserting that a write-only
  attribute is `"null"` in the JSON state representation after apply.
- Add a regression test per Tier-A attribute that verifies:
  1. The value is accepted from config.
  2. The value is NOT present in `terraform.tfstate`.
  3. Bumping `<attr>_wo_version` triggers an Update call to the API with
     the fresh value.

### Phase 1 — v0.2.0 (Tier-A migration, mechanical)

Migrate every Tier-A attribute behind a **new** schema attribute name with
the `_wo` suffix, keeping the legacy `Sensitive: true` attribute in place
for one release as a deprecation window:

```go
// v0.2.0 schema — both attributes present; exactly one must be set.
"passphrase": schema.StringAttribute{
    Description:        "Deprecated — use `passphrase_wo`. Will be removed in v0.3.0.",
    Optional:           true,
    Sensitive:          true,
    DeprecationMessage: "Use `passphrase_wo` + `passphrase_wo_version` to keep the passphrase out of state.",
},
"passphrase_wo": schema.StringAttribute{
    Description: "Passphrase for the keystore. Write-only: accepted from config, never stored in state.",
    Optional:    true,
    Sensitive:   true,
    WriteOnly:   true,
},
"passphrase_wo_version": schema.Int64Attribute{
    Description: "Integer version bumped whenever `passphrase_wo` changes. Required when `passphrase_wo` is set.",
    Optional:    true,
},
```

Validation: exactly one of `{passphrase, passphrase_wo}` is set; if
`passphrase_wo` is set, `passphrase_wo_version` is required.

Release notes: document the migration path. Existing customers keep working
unchanged; new customers are pushed to the `_wo` variant.

### Phase 2 — v0.3.0 (remove legacy attributes)

- Delete the non-`_wo` Tier-A attributes.
- Upgrade resource state schema version (the framework handles this
  via `StateUpgraders`) to migrate existing state automatically — the
  plaintext secret in v1 state is dropped; the `_wo_version` field defaults
  to `1`.
- Update every example in `examples/` to use the `_wo` form.
- Update `docs/resources/*.md` to document the new shape.

### Phase 3 — v0.4.0+ (Tier-C, connected_app)

Migrate `anypoint_connected_app.client_secret` to the **ephemeral resource**
shape (Terraform 1.10+). The generated client secret is surfaced to the
Terraform run as an ephemeral output and optionally written to a
downstream secrets-manager resource (AWS Secrets Manager, Vault KV) — never
to `terraform.tfstate`. AWS `aws_secretsmanager_random_password` uses this
pattern.

## Risks and open questions

1. **API does not accept rotation via PUT without destroying the resource.**
   If true for any Tier-A attribute, the `_wo_version` bump would force a
   replacement, which is poor UX. Must be verified per-attribute against
   the Anypoint API before Phase 1 lands.
2. **Customer scripts that read the plaintext value back from state.** Any
   external automation that does `terraform output -raw passphrase` or parses
   `terraform.tfstate` directly will break. Call this out prominently in the
   v0.2.0 release notes and CHANGELOG.
3. **Provider acceptance tests hitting a real Anypoint control plane** need
   the `_wo` code paths wired up before any production customer is migrated.
   Do not ship Phase 1 to any customer without green CI against a real org.

## References

- Terraform core 1.11 release notes — write-only attributes GA.
- `hashicorp/terraform-plugin-framework` changelog — `WriteOnly` attribute
  support.
- AWS provider: `aws_db_instance.password_wo` / `password_wo_version`
  (migration landed in 2025).
- Kubernetes provider: `kubernetes_secret_v1.data_wo`.
- HashiCorp blog: "Protect sensitive input variables with write-only
  arguments" (2025).
- This repo's [`docs/SECURITY.md`](./SECURITY.md) §3.3 for the four-tier
  mitigation strategy.
