---
page_title: "anypoint_secret_group_certificate Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a certificate within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats.
---

# anypoint_secret_group_certificate (Resource)

Manages a certificate within a secret group in Anypoint Secrets Manager. Supports PEM, JKS, PKCS12, and JCEKS formats.

~> **Delete behaviour:** The Anypoint Secrets Manager API does not expose individual DELETE endpoints for sub-resources. `terraform destroy` removes this resource from Terraform state only — the certificate is deleted on the Platform when the parent `anypoint_secret_group` is destroyed.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_secret_group_certificate" "example" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "my-certificate"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/certs/cert.pem"))
}
```

## Schema

### Required

- `environment_id` (String) Environment ID.
- `secret_group_id` (String) Secret group ID that this certificate belongs to.
- `name` (String) Name of the certificate.
- `certificate_base64` (String, Sensitive) Base64-encoded certificate file content. For PEM: `base64encode(file("cert.pem"))`. For binary: `filebase64("cert.der")`.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `type` (String) Certificate format: `PEM`, `JKS`, `PKCS12`, or `JCEKS`. Defaults to `PEM`.

### Read-Only

- `id` (String) Unique identifier of the certificate.
- `expiration_date` (String) Expiration date of the certificate.
- `algorithm` (String) Signature algorithm of the certificate.

## Import

An existing `anypoint_secret_group_certificate` can be imported using its composite ID: `organization_id/environment_id/secret_group_id/certificate_id`.

- `organization_id` — UUID of the organization
- `environment_id` — UUID of the environment
- `secret_group_id` — UUID of the secret group
- `certificate_id` — UUID of the certificate

~> **Note:** `certificate_base64` is a write-only field and will not be populated after import. Set it manually to avoid drift on the next plan.

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_secret_group_certificate.imported
  id = "<organization_id>/<environment_id>/<secret_group_id>/<certificate_id>"
}

resource "anypoint_secret_group_certificate" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  secret_group_id = "<secret_group_id>"
  name            = "<certificate_name>"
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
terraform import anypoint_secret_group_certificate.imported <organization_id>/<environment_id>/<secret_group_id>/<certificate_id>
```
