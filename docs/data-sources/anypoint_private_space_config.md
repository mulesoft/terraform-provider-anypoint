---
page_title: "anypoint_private_space_config Data Source - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Retrieves configuration information for a CloudHub 2.0 private space.
---

# anypoint_private_space_config (Data Source)

Retrieves configuration information for a CloudHub 2.0 private space, including network and firewall settings.

## Example Usage

```terraform
data "anypoint_private_space_config" "example" {
  id              = var.private_space_id
  organization_id = var.organization_id
}

output "private_space_status" {
  value = data.anypoint_private_space_config.example.status
}

output "network_region" {
  value = data.anypoint_private_space_config.example.network.region
}
```

## Schema

### Required

- `id` (String) The unique identifier of the private space to look up.

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not specified, uses the organization from provider credentials.

### Read-Only

- `days_left_for_relaxed_quota` (Number) The number of days left for relaxed quota.
- `firewall_rules` (List of Object) Firewall rules configured for the private space. (see [below for nested schema](#nestedatt--firewall_rules))
- `mule_app_deployment_count` (Number) The number of Mule apps deployed in the private space.
- `name` (String) The name of the private space.
- `network` (Object) Network configuration for the private space. (see [below for nested schema](#nestedatt--network))
- `root_organization_id` (String) The root organization ID of the private space.
- `status` (String) The current status of the private space.
- `vpc_migration_in_progress` (Boolean) Whether a VPC migration is in progress.

<a id="nestedatt--firewall_rules"></a>
### Nested Schema for `firewall_rules`

Read-Only:

- `cidr_block` (String) The CIDR block for the firewall rule.
- `from_port` (Number) The starting port for the firewall rule.
- `protocol` (String) The protocol for the firewall rule (tcp, udp, icmp).
- `to_port` (Number) The ending port for the firewall rule.
- `type` (String) The type of the firewall rule (inbound, outbound).

<a id="nestedatt--network"></a>
### Nested Schema for `network`

Read-Only:

- `cidr_block` (String) The CIDR block for the private network.
- `dns_target` (String) The DNS target for the private network.
- `inbound_internal_static_ips` (List of String) Inbound internal static IPs assigned to the private network.
- `inbound_static_ips` (List of String) Inbound static IPs assigned to the private network.
- `outbound_static_ips` (List of String) Outbound static IPs assigned to the private network.
- `region` (String) The AWS region for the private network.
- `reserved_cidrs` (List of String) Reserved CIDR blocks for the private network.
