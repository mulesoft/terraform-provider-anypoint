---
page_title: "anypoint_private_space_config Resource - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Manages an Anypoint Private Space together with its network configuration and firewall rules as a single resource.
---

# anypoint_private_space_config (Resource)

Manages an Anypoint Private Space together with its network configuration and firewall rules as a single composite resource. This resource combines private space creation, network provisioning, and firewall rule management into a unified workflow.

-> **Authentication:** This resource calls the **CloudHub 2.0 private-space (Runtime Manager) control-plane API**. A `client_credentials` Connected App works — grant it the **Cloudhub Organization Admin** (`admin:cloudhub`) scope (plus **Manage Runtime Fabrics** for some operations). A Connected App missing these scopes is rejected with `HTTP 401`/`403` before anything is created; the fix is to add the scopes (or use `auth_type = "user"` with a user that has the equivalent permissions). See [Authentication](../index.md#control-plane-resources-need-the-right-scopes-important).

## Example Usage

### Minimal (space only, no network)

```terraform
resource "anypoint_private_space_config" "example" {
  name = "my-private-space"
}
```

### Basic (space + network)

```terraform
resource "anypoint_private_space_config" "example" {
  name            = "my-private-space"
  organization_id = var.organization_id
  enable_egress   = true

  network {
    region     = "us-east-1"
    cidr_block = "10.0.0.0/22"
  }
}
```

### Full (space + network + firewall rules)

```terraform
resource "anypoint_private_space_config" "example" {
  name            = "my-private-space"
  organization_id = var.organization_id
  enable_egress   = true
  enable_iam_role = false

  network {
    region         = "us-east-1"
    cidr_block     = "10.0.0.0/22"
    reserved_cidrs = ["10.0.3.0/24"]
  }

  firewall_rules = [
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 30500
      to_port    = 32500
      type       = "inbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 0
      to_port    = 65535
      type       = "outbound"
    },
  ]
}

output "private_space_id" {
  value = anypoint_private_space_config.example.id
}

output "network_dns_target" {
  value = anypoint_private_space_config.example.network.dns_target
}

output "inbound_static_ips" {
  value = anypoint_private_space_config.example.network.inbound_static_ips
}
```

## Schema

### Required

- `name` (String) The name of the private space.

### Optional

- `organization_id` (String) The organization ID where the private space will be created. Defaults to the provider organization.
- `enable_egress` (Boolean) Whether to enable egress for the private space. Defaults to `false`.
- `enable_iam_role` (Boolean) Whether to enable IAM role for the private space. Defaults to `false`.
- `firewall_rules` (List of Object) Firewall rules for the private space. Omit to use platform-managed default rules. Each object has the following attributes:
  - `cidr_block` (String, Required) The CIDR block for the firewall rule.
  - `protocol` (String, Required) The protocol for the firewall rule (`tcp`, `udp`, `icmp`).
  - `from_port` (Number, Required) The starting port for the firewall rule.
  - `to_port` (Number, Required) The ending port for the firewall rule.
  - `type` (String, Required) The type of the firewall rule (`inbound` or `outbound`).
- `network` (Block) Network configuration for the private space. Omit to create the space without a network.
  - `region` (String, Optional) The AWS region for the private network. Forces replacement if changed.
  - `cidr_block` (String, Optional) The CIDR block for the private network. Forces replacement if changed.
  - `reserved_cidrs` (List of String, Optional) Reserved CIDR blocks for the private network.

### Read-Only

- `id` (String) The unique identifier for the private space.
- `status` (String) The current status of the private space (e.g., `Running`, `Provisioning`).
- `root_organization_id` (String) The root organization ID of the private space.
- `mule_app_deployment_count` (Number) The number of Mule apps currently deployed in the private space.
- `days_left_for_relaxed_quota` (Number) The number of days left for the relaxed deployment quota.
- `vpc_migration_in_progress` (Boolean) Whether a VPC migration is currently in progress.
- `managed_firewall_rules` (List of String) Platform-managed firewall rule identifiers.
- `global_space_status` (Map of String) Per-region global space status details.
- Within the `network` block:
  - `inbound_static_ips` (List of String) Inbound static IPs assigned to the private network.
  - `inbound_internal_static_ips` (List of String) Inbound internal static IPs assigned to the private network.
  - `outbound_static_ips` (List of String) Outbound static IPs assigned to the private network.
  - `dns_target` (String) The DNS target hostname for the private network.

## Import

An existing private space configuration can be imported using its private space ID (UUID).

Two import ID formats are supported:

| Format | When to use |
|--------|-------------|
| `<private_space_id>` | Private space belongs to the root organization |
| `<org_id>/<private_space_id>` | Private space belongs to a Business Group (sub-org) |

### Using an import block (Terraform ≥ 1.5 — recommended)

**Root org:**

```terraform
import {
  to = anypoint_private_space_config.imported
  id = "<private_space_id>"
}

resource "anypoint_private_space_config" "imported" {
  name = "<private_space_name>"
}
```

**Sub-org (Business Group):**

```terraform
import {
  to = anypoint_private_space_config.imported
  id = "<org_id>/<private_space_id>"
}

resource "anypoint_private_space_config" "imported" {
  name            = "<private_space_name>"
  organization_id = "<org_id>"
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

After import, run `terraform plan` to verify the state matches the actual configuration. The imported state will capture all network and firewall settings from the platform.

### Using the CLI (deprecated, Terraform < 1.5)

```shell
# Root org:
terraform import anypoint_private_space_config.imported <private_space_id>

# Sub-org:
terraform import anypoint_private_space_config.imported <org_id>/<private_space_id>
```
