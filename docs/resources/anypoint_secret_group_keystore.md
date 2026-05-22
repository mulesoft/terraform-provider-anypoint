---
page_title: "anypoint_secret_group_keystore Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a keystore within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats.
---

# anypoint_secret_group_keystore (Resource)

Manages a keystore within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats. Use `filebase64()` to read binary files (JKS/PKCS12) or `base64encode(file(...))` for PEM text files.

~> **Delete behaviour:** The Anypoint Secrets Manager API does not expose individual DELETE endpoints for sub-resources. `terraform destroy` removes this resource from Terraform state only — the keystore is deleted on the Platform when the parent `anypoint_secret_group` is destroyed.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

### PEM Keystore

```terraform
resource "anypoint_secret_group_keystore" "pem" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-pem-keystore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/certs/key.pem"))
}
```

### PEM Keystore with CA Chain

```terraform
resource "anypoint_secret_group_keystore" "pem_with_ca" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-pem-with-truststore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/certs/key.pem"))
  ca_path_base64     = base64encode(file("${path.module}/certs/truststore.pem"))
}
```

### JKS Keystore

```terraform
resource "anypoint_secret_group_keystore" "jks" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-jks-keystore"
  type            = "JKS"

  keystore_file_base64 = filebase64("${path.module}/certs/keystore.jks")
  store_passphrase     = var.jks_store_passphrase
  key_passphrase       = var.jks_key_passphrase
  alias                = "myalias"
}
```

## Schema

### Required

- `environment_id` (String) Environment ID. Changing this forces a new resource.
- `secret_group_id` (String) Secret group ID that this keystore belongs to. Changing this forces a new resource.
- `name` (String) Name of the keystore.

### Optional

- `organization_id` (String) The organization ID. If not provided, inferred from the connected app credentials.
- `type` (String) Keystore format: `PEM`, `JKS`, `PKCS12`, or `JCEKS`. Defaults to `PEM`. Changing this forces a new resource.
- `certificate_base64` (String, Sensitive) Base64-encoded certificate content. For PEM: `base64encode(file("cert.pem"))`. For binary DER: `filebase64("cert.der")`. Used for PEM type.
- `key_base64` (String, Sensitive) Base64-encoded private key content. For PEM: `base64encode(file("key.pem"))`. Required for PEM type.
- `keystore_file_base64` (String, Sensitive) Base64-encoded keystore file. Use `filebase64("keystore.jks")` or `filebase64("keystore.p12")`. Required for JKS, PKCS12, and JCEKS types.
- `store_passphrase` (String, Sensitive) Store-level passphrase (`storePassphrase`). Required for JKS, PKCS12, and JCEKS types.
- `key_passphrase` (String, Sensitive) Private-key entry passphrase (`keyPassphrase`). Required for JKS, PKCS12, and JCEKS types. Optional for PEM encrypted keys.
- `alias` (String) Entry alias within the keystore. Used for JKS, PKCS12, and JCEKS types.
- `ca_path_base64` (String, Sensitive) Base64-encoded CA certificate chain (appended as truststore). Optional for all types.

### Read-Only

- `id` (String) Unique identifier of the keystore.
- `expiration_date` (String) Expiration date of the certificate in the keystore.
- `algorithm` (String) Signature algorithm of the certificate.

## Import

An existing `anypoint_secret_group_keystore` can be imported using its composite ID: `organization_id/environment_id/secret_group_id/keystore_id`.

- `organization_id` — UUID of the organization
- `environment_id` — UUID of the environment
- `secret_group_id` — UUID of the secret group
- `keystore_id` — UUID of the keystore

~> **Note:** File content fields (`certificate_base64`, `key_base64`, etc.) are write-only and will not be populated after import. Set them manually to avoid drift.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_secret_group_keystore.imported
  id = "<organization_id>/<environment_id>/<secret_group_id>/<keystore_id>"
}

resource "anypoint_secret_group_keystore" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  secret_group_id = "<secret_group_id>"
  name            = "<keystore_name>"
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
terraform import anypoint_secret_group_keystore.imported <organization_id>/<environment_id>/<secret_group_id>/<keystore_id>
```
