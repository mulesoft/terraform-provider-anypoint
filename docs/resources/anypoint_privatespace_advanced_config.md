---
page_title: "anypoint_privatespace_advanced_config Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages advanced configuration for an Anypoint Private Space.
---

# anypoint_privatespace_advanced_config (Resource)

Manages advanced configuration for an Anypoint Private Space.

-> **Authentication:** This resource calls the **CloudHub 2.0 private-space (Runtime Manager) control-plane API**. A `client_credentials` Connected App works — grant it the **Cloudhub Organization Admin** (`admin:cloudhub`) scope (plus **Manage Runtime Fabrics** for some operations). A Connected App missing these scopes is rejected with `HTTP 401`/`403` before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

## Example Usage

```terraform
resource "anypoint_privatespace_advanced_config" "example" {
  private_space_id = var.private_space_id

  ingress_configuration = {
    read_response_timeout = "600"
    protocol              = "https-redirect"

    logs = {
      port_log_level = "INFO"
      filters        = []
    }

    deployment = {
      status              = "APPLIED"
      last_seen_timestamp = 1753719215000
    }
  }

  enable_iam_role = true
}
```

## Schema

### Required

- `private_space_id` (String) The ID of the private space to configure.

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not provided, the organization ID will be inferred from the connected app credentials.
- `ingress_configuration` (Block) Ingress configuration for the private space. See [below for nested schema](#nestedschema--ingress_configuration).
- `enable_iam_role` (Boolean) Whether to enable IAM role for the private space. Defaults to `false`.

### Read-Only

- `id` (String) The unique identifier of the advanced configuration.

<a id="nestedschema--ingress_configuration"></a>
### Nested Schema for `ingress_configuration`

Optional:

- `read_response_timeout` (String) Read response timeout in seconds. Defaults to `"300"`.
- `protocol` (String) Protocol for ingress configuration. Defaults to `"https-redirect"`.
- `logs` (Block) Logs configuration for ingress. See [below for nested schema](#nestedschema--ingress_configuration--logs).
- `deployment` (Block) Deployment configuration for ingress. See [below for nested schema](#nestedschema--ingress_configuration--deployment).

<a id="nestedschema--ingress_configuration--logs"></a>
### Nested Schema for `ingress_configuration.logs`

Optional:

- `port_log_level` (String) Port log level. Defaults to `"ERROR"`.
- `filters` (Block List) List of log filters. Defaults to `[]`. See [below for nested schema](#nestedschema--ingress_configuration--logs--filters).

<a id="nestedschema--ingress_configuration--logs--filters"></a>
### Nested Schema for `ingress_configuration.logs.filters`

Required:

- `ip` (String) IP address for the filter.
- `level` (String) Log level for the filter.

<a id="nestedschema--ingress_configuration--deployment"></a>
### Nested Schema for `ingress_configuration.deployment`

Optional:

- `status` (String) Deployment status. Defaults to `"APPLIED"`.
- `last_seen_timestamp` (Number) Last seen timestamp. Defaults to `1753719215000`.

## Import

An existing private space advanced configuration can be imported using its private space ID (UUID).

Two import ID formats are supported:

| Format | When to use |
|--------|-------------|
| `<private_space_id>` | Private space belongs to the root organization |
| `<org_id>/<private_space_id>` | Private space belongs to a Business Group (sub-org) |

### Using an import block (Terraform ≥ 1.5 — recommended)

**Root org:**

```terraform
import {
  to = anypoint_privatespace_advanced_config.imported
  id = "<private_space_id>"
}

resource "anypoint_privatespace_advanced_config" "imported" {
  private_space_id = "<private_space_id>"
}
```

**Sub-org (Business Group):**

```terraform
import {
  to = anypoint_privatespace_advanced_config.imported
  id = "<org_id>/<private_space_id>"
}

resource "anypoint_privatespace_advanced_config" "imported" {
  private_space_id = "<private_space_id>"
  organization_id  = "<org_id>"
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
terraform import anypoint_privatespace_advanced_config.imported <private_space_id>

# Sub-org:
terraform import anypoint_privatespace_advanced_config.imported <org_id>/<private_space_id>
```
