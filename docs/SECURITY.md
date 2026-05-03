# Security Guide — Credentials & State Hygiene

This document explains how to store Anypoint credentials safely when using the
`sf.com/mulesoft/anypoint` Terraform provider, how credentials flow through the
provider, and how the provider handles sensitive values inside `terraform.tfstate`.
The guidance here is deliberately aligned with what the official AWS, GCP,
Azure, Kong, Kubernetes, and Snowflake Terraform providers recommend — it is
the industry-standard posture, not Anypoint-specific practice.

---

## 1. Credential storage on the customer side

The provider accepts four credentials for the "admin" connected app used by
Access Management resources (`anypoint_organization`, `anypoint_environment`,
`anypoint_team`, …):

| Attribute | Environment variable | Used for |
|---|---|---|
| `client_id` | `ANYPOINT_CLIENT_ID` | Connected-app client identifier |
| `client_secret` | `ANYPOINT_CLIENT_SECRET` | Connected-app client secret |
| `username` | `ANYPOINT_USERNAME` or `ANYPOINT_ADMIN_USERNAME` | Platform user for password-grant |
| `password` | `ANYPOINT_PASSWORD` or `ANYPOINT_ADMIN_PASSWORD` | Password for password-grant |

All four are marked `Sensitive: true` on the provider schema, so they are
**redacted from `terraform plan` / `terraform apply` output**, but that does
not protect them from the places the secret actually lives on disk.

### 1.1. Storage ranking (least → most mature)

From weakest to strongest posture. Most teams combine several of these.

1. **Hard-coded in `main.tf`.**
   **Do not do this.** Anything in a `.tf` file is typically committed to
   version control. This is the equivalent of an `aws_access_key_id` literal —
   every provider explicitly warns against it.

2. **Committed `terraform.tfvars`.**
   Also a no-go. `terraform.tfvars` / `*.auto.tfvars` are loaded automatically
   by Terraform; if they contain real credentials and land in Git, you have a
   secret leak. Always `.gitignore` them and commit only `.tfvars.example`.

3. **Local environment variables** set by the shell (e.g. `~/.zshrc.local` or
   a `direnv` `.envrc`), ideally sourced from a local password manager:

   ```bash
   # Sourced from 1Password CLI at shell startup; not committed anywhere.
   export ANYPOINT_CLIENT_ID=$(op read 'op://Anypoint/terraform-admin/client_id')
   export ANYPOINT_CLIENT_SECRET=$(op read 'op://Anypoint/terraform-admin/client_secret')
   export ANYPOINT_ADMIN_USERNAME=$(op read 'op://Anypoint/terraform-admin/username')
   export ANYPOINT_ADMIN_PASSWORD=$(op read 'op://Anypoint/terraform-admin/password')
   ```

   This is acceptable for solo development. Use `op read`, `vault kv get`,
   `aws secretsmanager get-secret-value`, `gcloud secrets versions access`,
   `bw get`, or whatever your team standardizes on. The common invariant is
   **the secret is never at rest in plaintext on disk**.

4. **Terraform data source backed by a secrets manager.**
   Works well when multiple people / pipelines share the config. See:

   - [`examples/security/aws-secrets-manager/`](../examples/security/aws-secrets-manager/)
   - [`examples/security/vault/`](../examples/security/vault/)

   Caveat: as soon as a secret flows through a Terraform data source, its
   value lands in state. Treat state as sensitive (Section 2).

5. **Terraform Cloud / Enterprise workspace variables** (marked **Sensitive**,
   category **Environment**). HashiCorp hosts state encrypted at rest, audits
   every read, and never re-displays sensitive variables in the UI.

6. **CI/CD secret store + ephemeral job env.** GitHub Actions Secrets, GitLab
   CI Masked Variables, Azure DevOps Variable Groups, CircleCI Contexts,
   Jenkins Credentials Plugin, etc. Secrets are injected as env vars only for
   the duration of the job; GitHub Actions auto-redacts them from logs.
   Example in [`examples/security/github-actions/`](../examples/security/github-actions/).

### 1.2. How major providers do this

| Provider | Preferred posture for humans | Preferred posture for CI |
|---|---|---|
| **AWS** (`hashicorp/aws`) | `aws sso login` → short-lived STS creds cached in `~/.aws/sso/cache`; or named profile in `~/.aws/credentials`. Static keys in `.tf` are discouraged in docs. | EC2/ECS/EKS role (IMDSv2 / task role / IRSA) or GitHub OIDC → `assume_role_with_web_identity`. **No long-lived keys at all.** |
| **GCP** (`hashicorp/google`) | `gcloud auth application-default login` (ADC) — federates through Google Workspace SSO. | Workload Identity Federation (GitHub/GitLab/AWS OIDC → GCP federated credential → impersonated service account). JSON keys on disk discouraged. |
| **Azure** (`hashicorp/azurerm`) | `az login` interactive browser auth. | Managed Identity, Workload Identity, OIDC federation. Client secrets are a "last resort" per docs. |
| **Kong Konnect** | PAT generated after SSO login to the UI. | **System Account token** (service principal that bypasses SSO). |
| **Snowflake** | `authenticator = "externalbrowser"` (SSO). | **Key-pair auth** with a dedicated non-SSO service user. |
| **Vault** | `vault login -method=oidc`. | AppRole / JWT / OIDC auth. |

The uniform pattern: **short-lived, federated creds for CI** and
**interactive SSO for developers**. Long-lived static credentials sit at the
bottom of every provider's recommendation list.

---

## 2. SSO and the admin connected app

### 2.1. Why password-grant breaks with SSO users

The "admin" connected app uses OAuth 2.0 **Resource Owner Password Credentials
grant** — the provider POSTs the admin user's username and password to
Anypoint's token endpoint, and Anypoint validates them.

When your organization enables SSO, **Anypoint delegates password validation
to your IdP** (Okta, Azure AD, Google Workspace, Ping, ForgeRock, …). The
human's password is owned by the IdP, not by Anypoint. That breaks password
grant in several ways:

- Anypoint does not have the password to validate against.
- Most enterprise IdPs enforce MFA; password grant has no way to satisfy a
  second factor.
- Conditional access (device posture, IP pinning, session limits) will reject
  non-interactive logins.

### 2.2. What every mature Terraform provider tells SSO-gated customers

> **Do not use SSO credentials for automation. Provision a dedicated, non-SSO
> service account with narrowly-scoped permissions, network restrictions, and
> audited usage. Prefer short-lived federated tokens for CI where the platform
> supports them.**

Concrete examples:

- **AWS IAM Identity Center (SSO)** has no password-grant equivalent. Humans
  use `aws sso login`; CI uses Workload Identity Federation. Long-lived
  service creds are actively discouraged.
- **Okta's own Terraform provider** requires an API token for a service user
  that is "**not subject to MFA or session-based SSO policies**" (quoted from
  the official docs).
- **Kong Konnect** issues **System Account tokens** (service principals
  detached from SSO) for CI.
- **Snowflake** uses **key-pair auth** for a dedicated non-SSO service user.
- **Azure** pushes customers to Managed Identity / OIDC federation so there
  are no passwords involved at all.

### 2.3. Concrete guidance for Anypoint with SSO

1. **Create a dedicated local (non-SSO) Anypoint user for Terraform.**

   - Access Management → Users → create a local user (not federated via your
     IdP).
   - Assign the narrowest admin role set needed, scoped only to the sub-orgs
     and environments Terraform must manage (never grant `Root Organization
     Administrator` unless truly required).
   - Generate a long random password (40+ chars) from a password manager.
   - If your IdP / security team supports it, add this user to a "service
     accounts" group that is **excluded** from:
     - SSO enforcement policy
     - MFA enforcement policy
     - Interactive-login restrictions

2. **Use `client_credentials` grant wherever possible.**
   Most CloudHub 2.0, API Manager, and DLB endpoints accept client_credentials
   and do not need user context. Reserve the admin username/password for
   Access Management resources (`anypoint_organization`, `anypoint_role`,
   `anypoint_team_roles`, …) that genuinely need user-context tokens.

3. **Keep one connected app per automation boundary.**
   One god-mode connected app used by three teams is a rotation hazard. Issue
   a separate connected app per pipeline, each with the minimum scopes it
   actually uses. Rotate the client secret on a 90-day cadence.

4. **If security refuses to allow a non-SSO local user:**

   - **Option A (preferred):** Run Terraform from a CI system your IdP
     trusts via OIDC (GitHub Actions, GitLab, Azure DevOps). The CI job
     presents an OIDC token; Anypoint still needs a service account, but its
     creds can live in an IdP-managed secret store and be short-lived.
   - **Option B:** A break-glass workflow through a PAM system (CyberArk,
     HashiCorp Boundary, Teleport) that issues a 1-hour credential per run.
     High operational cost; only worthwhile for highly regulated customers.
   - **Option C (long-term):** Escalate to Salesforce to expose a
     service-principal / JIT-token flow for the Access Management APIs.
     This is the direction AWS, GCP, and Azure took over 2019–2024 and is
     the right strategic endpoint.

### 2.4. Short policy statement you can paste into internal runbooks

> The Anypoint admin connected app requires OAuth 2.0 password grant against
> the Access Management APIs. **Password grant is incompatible with
> SSO-federated user accounts** because the password is owned by the IdP, not
> by Anypoint. Terraform automation MUST use a dedicated non-SSO local
> Anypoint user with narrowly-scoped admin roles, mirroring Okta's service-user
> recommendation, Snowflake's key-pair auth for automation, and Kong Konnect's
> System Account pattern.

---

## 3. Secrets in Terraform state (PEM passphrases, private keys, passwords)

### 3.1. Ground truth

**Terraform state contains every attribute value in plaintext, including
secrets.** `Sensitive: true` on a schema attribute only redacts it from
`terraform plan` and `terraform apply` output. It does **not** encrypt, hash,
or otherwise protect the value inside `terraform.tfstate`.

This is true across every Terraform provider. Direct paraphrases from the
official docs of comparable providers:

- **AWS provider**, `aws_db_instance.password`: "the password is stored in
  the state file in plaintext."
- **AWS provider**, `aws_secretsmanager_secret_version.secret_string`: "the
  secret value is stored in plain text in the state file."
- **GCP provider**, `google_secret_manager_secret_version.secret_data`:
  "**Warning:** this value will be stored in the raw state as plain text."
- **Kubernetes provider**, `kubernetes_secret.data`: "the data is stored in
  raw state as plaintext."
- **Kong provider**, `kong_certificate.key`: "private key in PEM format.
  Stored in state."
- **Vault provider**, `vault_generic_secret.data_json`: "all sensitive data
  will be persisted in the state file."

### 3.2. What the Anypoint provider does today

Every secret-valued attribute in the `anypoint_keystore`, `anypoint_truststore`,
`anypoint_certificate`, `anypoint_certificate_pinset`, `anypoint_shared_secret`,
`anypoint_tls_context`, and `anypoint_connected_app` schemas is marked
`Sensitive: true`. That guarantees:

- They are redacted from CLI output.
- They are redacted from CI logs that capture stdout/stderr.
- Terraform will not render them in JSON plan output unless you pass
  `-json`, in which case the `sensitive_values` map still flags them.

The values are nevertheless written into `terraform.tfstate` as plaintext
JSON, because the plugin-framework serializes every attribute to state.
This is the same behavior as AWS, GCP, Kubernetes, Kong, etc.

### 3.3. The four-tier mitigation strategy

**Tier 1 — Encrypted remote state (mandatory baseline).**

Never keep `terraform.tfstate` on developer laptops or in Git. Use one of:

- S3 backend with `encrypt = true` + `kms_key_id = <arn>` + a bucket policy
  denying non-role access + CloudTrail auditing on the bucket + DynamoDB
  state locking.
- GCS backend with CMEK + bucket-level IAM + uniform bucket-level access.
- Azure Storage backend with CMK via Key Vault + ADLS Gen2 +
  `require_infrastructure_encryption`.
- Terraform Cloud / Enterprise (state is encrypted at rest by HashiCorp,
  access-controlled per workspace, and audit-logged).

**Tier 2 — Terraform 1.7+ native state encryption (defense in depth).**

Layer a second envelope on top of the backend's at-rest encryption. Even if
the backend bucket is exfiltrated, the state is useless without KMS access.

```hcl
terraform {
  encryption {
    key_provider "aws_kms" "state_key" {
      kms_key_id = "alias/terraform-anypoint-state"
      region     = "us-east-1"
    }
    method "aes_gcm" "state_method" {
      keys = key_provider.aws_kms.state_key
    }
    state {
      method = method.aes_gcm.state_method
    }
    plan {
      method = method.aes_gcm.state_method
    }
  }
}
```

GCP KMS, passphrase-based, and PBKDF2 key providers are also supported.

**Tier 3 — Write-only attributes (Terraform ≥ 1.11, framework ≥ 1.11; shipped).**

This is the modern answer. A schema attribute marked `WriteOnly: true` is
accepted from config but **never persisted to state**. AWS migrated
`aws_db_instance.password` to a write-only form in 2025; the Kubernetes
provider did the same for `kubernetes_secret_v1.data`.

The Anypoint provider is **on the plugin-framework version that supports
write-only attributes** (`v1.19.0`). A migration plan for the secret-valued
attributes is tracked in [`docs/ROADMAP_SECRETS.md`](./ROADMAP_SECRETS.md).

**Tier 4 — Reference-by-ID (if the API supports it).**

The resource takes a pointer (ARN, URI, Vault path, Secret Manager ID) and
the runtime service resolves it. Terraform never sees the secret value.

- ECS task definition → `secrets = [{ name = "DB_PASS", valueFrom = aws_secretsmanager_secret.db.arn }]`.
- Kubernetes Pod → `envFrom.secretRef`.
- For Anypoint: the Secrets Manager API currently requires inline bytes, so
  this tier is not yet applicable to `anypoint_keystore` / `anypoint_certificate`
  / `anypoint_shared_secret`. If the API adds reference-by-URI in future, the
  provider will adopt Tier 4 and drop inline material from the schema.

### 3.4. Short policy statement for runbooks

> The Anypoint provider marks every PEM body, passphrase, private key, and
> password attribute as `Sensitive`, which redacts them from `terraform plan`
> and `terraform apply` output. **However, the values are still written into
> `terraform.tfstate` in plaintext — this is a Terraform-wide design
> constraint and identical to the behavior of AWS, GCP, Kong, and Kubernetes
> providers.** Customers must therefore (1) use an encrypted remote backend,
> (2) restrict state access to the apply role only, (3) enable Terraform 1.7+
> native state encryption for defense-in-depth, and (4) rotate secrets on a
> schedule assuming state may leak. The provider roadmap for migrating
> secret-valued attributes to `WriteOnly` is documented in
> `docs/ROADMAP_SECRETS.md`.

---

## 4. Quick-reference checklist

Run through this before shipping any Anypoint Terraform config to production.

- [ ] No credentials in any `.tf` file.
- [ ] No credentials in any committed `.tfvars`. Only `.tfvars.example` is
      tracked, with placeholders.
- [ ] `.gitignore` contains `terraform.tfvars`, `*.auto.tfvars`,
      `terraform.tfstate*`, `.terraform/`, `.terraform.lock.hcl`.
- [ ] A dedicated **non-SSO** Anypoint user is used for Terraform automation,
      with least-privilege roles scoped to the managed orgs/envs.
- [ ] Connected-app client secrets are rotated at least every 90 days.
- [ ] State is stored in an encrypted remote backend (S3+KMS, GCS+CMEK,
      TFC/TFE, or equivalent) — never in Git and never on developer laptops.
- [ ] State bucket / workspace access is restricted to the CI apply role and
      a small set of auditors; read access is logged.
- [ ] CI injects secrets through the CI provider's secret store (GitHub
      Actions Secrets, GitLab CI Masked Variables, etc.), never from repo
      files.
- [ ] Terraform 1.7+ native state encryption is enabled on top of the
      backend's encryption-at-rest (recommended for regulated workloads).

---

## See also

- [`examples/security/aws-secrets-manager/`](../examples/security/aws-secrets-manager/)
  — Terraform data source pattern with AWS Secrets Manager.
- [`examples/security/vault/`](../examples/security/vault/) — HashiCorp
  Vault KV v2 pattern.
- [`examples/security/terraform-cloud/`](../examples/security/terraform-cloud/)
  — Terraform Cloud / Enterprise workspace variable pattern.
- [`examples/security/github-actions/`](../examples/security/github-actions/)
  — CI pattern using GitHub Actions Secrets.
- [`docs/ROADMAP_SECRETS.md`](./ROADMAP_SECRETS.md) — provider roadmap for
  migrating secret-valued schema attributes to `WriteOnly`.
