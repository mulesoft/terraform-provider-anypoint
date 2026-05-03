---
page_title: "anypoint_secret_group_keystore Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a keystore within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats.
---

# anypoint_secret_group_keystore (Resource)

Manages a keystore within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats. Use `filebase64()` to read binary files (JKS/PKCS12) or `file()` for PEM text files.

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
  passphrase           = var.jks_passphrase
  alias                = "myalias"
}
```

## Schema

### Required

- `environment_id` (String) Environment ID.
- `secret_group_id` (String) Secret group ID that this keystore belongs to.
- `name` (String) Name of the keystore.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `type` (String) Keystore format: `PEM`, `JKS`, `PKCS12`, or `JCEKS`. Defaults to `PEM`.
- `certificate_base64` (String, Sensitive) Base64-encoded certificate content. For PEM files use `base64encode(file("cert.pem"))`, or for binary formats use `filebase64("cert.der")`.
- `key_base64` (String, Sensitive) Base64-encoded private key content. For PEM keys use `base64encode(file("key.pem"))`, or for binary keys use `filebase64("key.der")`. Required for PEM type.
- `keystore_file_base64` (String, Sensitive) Base64-encoded keystore file content. Use `filebase64("keystore.jks")` or `filebase64("keystore.p12")`. Required for JKS, PKCS12, and JCEKS types.
- `passphrase` (String, Sensitive) Passphrase for the keystore or encrypted PEM key.
- `alias` (String) Alias of the entry within the keystore. Used for JKS, PKCS12, and JCEKS types.
- `ca_path_base64` (String, Sensitive) Base64-encoded CA certificate chain (truststore). Optional for all types.

### Read-Only

- `id` (String) Unique identifier of the keystore.
- `expiration_date` (String) Expiration date of the certificate in the keystore.
- `algorithm` (String) Signature algorithm of the certificate.

## Import

Import is supported using the following syntax:

```shell
terraform import anypoint_secret_group_keystore.example organization_id/environment_id/secret_group_id/keystore_id
```
