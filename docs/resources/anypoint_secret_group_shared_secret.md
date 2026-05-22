---
page_title: "anypoint_secret_group_shared_secret Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a shared secret within a secret group in Anypoint Secrets Manager. Supports four types: UsernamePassword, S3Credential, SymmetricKey, and Blob.
---

# anypoint_secret_group_shared_secret (Resource)

Manages a shared secret within a secret group in Anypoint Secrets Manager. Supports four types: UsernamePassword, S3Credential, SymmetricKey, and Blob. Provide the type-specific fields based on the chosen type.

~> **Delete behaviour:** The Anypoint Secrets Manager API does not expose individual DELETE endpoints for sub-resources. `terraform destroy` removes this resource from Terraform state only — the shared secret is deleted on the Platform when the parent `anypoint_secret_group` is destroyed.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

### UsernamePassword

```terraform
resource "anypoint_secret_group_shared_secret" "db_creds" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "db-credentials"
  type            = "UsernamePassword"

  username = "admin"
  password = var.db_password
}
```

### S3Credential

```terraform
resource "anypoint_secret_group_shared_secret" "s3" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "s3-backup-creds"
  type            = "S3Credential"

  access_key_id     = var.aws_access_key
  secret_access_key = var.aws_secret_key
  expiration_date   = "2026-12-31"
}
```

### SymmetricKey

```terraform
resource "anypoint_secret_group_shared_secret" "symmetric" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "encryption-key"
  type            = "SymmetricKey"

  key = base64encode("my-256-bit-secret-key-value-here")
}
```

### Blob

```terraform
resource "anypoint_secret_group_shared_secret" "blob" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "api-token"
  type            = "Blob"

  content = var.api_token
}
```

## Schema

### Required

- `environment_id` (String) Environment ID.
- `secret_group_id` (String) Secret group ID that this shared secret belongs to.
- `name` (String) Name of the shared secret.
- `type` (String) Type of shared secret: `UsernamePassword`, `S3Credential`, `SymmetricKey`, or `Blob`.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `expiration_date` (String) Optional expiration date (e.g. `2026-03-31`).
- `username` (String) Username (for UsernamePassword type).
- `password` (String, Sensitive) Password (for UsernamePassword type).
- `access_key_id` (String) AWS access key ID (for S3Credential type).
- `secret_access_key` (String, Sensitive) AWS secret access key (for S3Credential type).
- `key` (String, Sensitive) Base64-encoded symmetric key (for SymmetricKey type).
- `content` (String, Sensitive) Secret content string (for Blob type).

### Read-Only

- `id` (String) Unique identifier of the shared secret.

## Import

An existing `anypoint_secret_group_shared_secret` can be imported using its composite ID: `organization_id/environment_id/secret_group_id/shared_secret_id`.

- `organization_id` — UUID of the organization
- `environment_id` — UUID of the environment
- `secret_group_id` — UUID of the secret group
- `shared_secret_id` — UUID of the shared secret

~> **Note:** Sensitive fields (passwords, secrets) are write-only and will not be populated after import. Set them manually to avoid drift.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_secret_group_shared_secret.imported
  id = "<organization_id>/<environment_id>/<secret_group_id>/<shared_secret_id>"
}

resource "anypoint_secret_group_shared_secret" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  secret_group_id = "<secret_group_id>"
  name            = "<shared_secret_name>"
  type            = "UsernamePassword"
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
terraform import anypoint_secret_group_shared_secret.imported <organization_id>/<environment_id>/<secret_group_id>/<shared_secret_id>
```
