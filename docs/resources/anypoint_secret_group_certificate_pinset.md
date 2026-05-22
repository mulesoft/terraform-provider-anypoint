---
page_title: "anypoint_secret_group_certificate_pinset Resource - terraform-provider-anypoint"
subcategory: "Secrets Management"
description: |-
  Manages a certificate pinset within a secret group in Anypoint Secrets Manager. A certificate pinset is used for certificate pinning validation.
---

# anypoint_secret_group_certificate_pinset (Resource)

Manages a certificate pinset within a secret group in Anypoint Secrets Manager. A certificate pinset is used for certificate pinning validation.

~> **Delete behaviour:** The Anypoint Secrets Manager API does not expose individual DELETE endpoints for sub-resources. `terraform destroy` removes this resource from Terraform state only — the certificate pinset is deleted on the Platform when the parent `anypoint_secret_group` is destroyed.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_secret_group_certificate_pinset" "example" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "my-cert-pinset"

  certificate_pinset_base64 = base64encode(file("${path.module}/certs/cert.pem"))
}
```

## Schema

### Required

- `environment_id` (String) Environment ID.
- `secret_group_id` (String) Secret group ID that this certificate pinset belongs to.
- `name` (String) Name of the certificate pinset.
- `certificate_pinset_base64` (String, Sensitive) Base64-encoded certificate file for pinning. For PEM: `base64encode(file("cert.pem"))`.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.

### Read-Only

- `id` (String) Unique identifier of the certificate pinset.
- `expiration_date` (String) Expiration date of the pinned certificate.
- `algorithm` (String) Signature algorithm of the pinned certificate.

## Import

An existing `anypoint_secret_group_certificate_pinset` can be imported using its composite ID: `organization_id/environment_id/secret_group_id/certificate_pinset_id`.

- `organization_id` — UUID of the organization
- `environment_id` — UUID of the environment
- `secret_group_id` — UUID of the secret group
- `certificate_pinset_id` — UUID of the certificate pinset

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_secret_group_certificate_pinset.imported
  id = "<organization_id>/<environment_id>/<secret_group_id>/<certificate_pinset_id>"
}

resource "anypoint_secret_group_certificate_pinset" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  secret_group_id = "<secret_group_id>"
  name            = "<pinset_name>"
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
terraform import anypoint_secret_group_certificate_pinset.imported <organization_id>/<environment_id>/<secret_group_id>/<certificate_pinset_id>
```
