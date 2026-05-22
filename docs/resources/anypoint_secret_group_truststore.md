---
page_title: "anypoint_secret_group_truststore Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a truststore within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats.
---

# anypoint_secret_group_truststore (Resource)

Manages a truststore within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats. Use `base64encode(file(...))` for PEM text files or `filebase64(...)` for binary JKS/PKCS12 files.

~> **Delete behaviour:** The Anypoint Secrets Manager API does not expose individual DELETE endpoints for sub-resources. `terraform destroy` removes this resource from Terraform state only — the truststore is deleted on the Platform when the parent `anypoint_secret_group` is destroyed.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

### PEM Truststore

```terraform
resource "anypoint_secret_group_truststore" "pem" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "ca-truststore"
  type            = "PEM"

  truststore_base64 = base64encode(file("${path.module}/certs/truststore.pem"))
}
```

### JKS Truststore

```terraform
resource "anypoint_secret_group_truststore" "jks" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "ca-truststore-jks"
  type            = "JKS"

  truststore_base64 = filebase64("${path.module}/certs/truststore.jks")
  passphrase        = var.jks_passphrase
}
```

## Schema

### Required

- `environment_id` (String) Environment ID.
- `secret_group_id` (String) Secret group ID that this truststore belongs to.
- `name` (String) Name of the truststore.
- `truststore_base64` (String, Sensitive) Base64-encoded truststore file content. For PEM: `base64encode(file("truststore.pem"))`. For JKS/PKCS12: `filebase64("truststore.jks")`.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `type` (String) Truststore format: `PEM`, `JKS`, `PKCS12`, or `JCEKS`. Defaults to `PEM`.
- `passphrase` (String, Sensitive) Passphrase for the truststore. Required for JKS, PKCS12, and JCEKS formats.

### Read-Only

- `id` (String) Unique identifier of the truststore.
- `expiration_date` (String) Expiration date of the certificate in the truststore.
- `algorithm` (String) Signature algorithm of the certificate.

## Import

An existing `anypoint_secret_group_truststore` can be imported using its composite ID: `organization_id/environment_id/secret_group_id/truststore_id`.

- `organization_id` — UUID of the organization
- `environment_id` — UUID of the environment
- `secret_group_id` — UUID of the secret group
- `truststore_id` — UUID of the truststore

~> **Note:** `truststore_base64` is a write-only field and will not be populated after import. Set it manually to avoid drift on the next plan.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_secret_group_truststore.imported
  id = "<organization_id>/<environment_id>/<secret_group_id>/<truststore_id>"
}

resource "anypoint_secret_group_truststore" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  secret_group_id = "<secret_group_id>"
  name            = "<truststore_name>"
  type            = "PEM"
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
terraform import anypoint_secret_group_truststore.imported <organization_id>/<environment_id>/<secret_group_id>/<truststore_id>
```
