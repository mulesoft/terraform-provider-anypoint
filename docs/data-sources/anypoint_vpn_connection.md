---
page_title: "anypoint_vpn_connection Data Source - terraform-provider-anypoint"
subcategory: "CloudHub 2.0"
description: |-
  Fetches information about a CloudHub 2.0 VPN connection.
---

# anypoint_vpn_connection (Data Source)

Fetches information about a CloudHub 2.0 VPN connection.

## Example Usage

```terraform
data "anypoint_vpn_connection" "example" {
  private_space_id = var.private_space_id
  connection_id    = var.vpn_connection_id
}

output "vpn_connection_name" {
  value = data.anypoint_vpn_connection.example.name
}

output "vpn_configurations" {
  value = data.anypoint_vpn_connection.example.vpns
}
```

## Schema

### Required

- `connection_id` (String) The ID of the VPN connection to look up.
- `private_space_id` (String) The private space ID where the VPN connection is located.

### Optional

- `organization_id` (String) The organization ID where the private space is located. If not specified, uses the organization from provider credentials.

### Read-Only

- `id` (String) The unique identifier for the VPN connection.
- `name` (String) The name of the VPN connection.
- `vpns` (List of Object) List of VPN configurations within this connection. (see [below for nested schema](#nestedatt--vpns))

<a id="nestedatt--vpns"></a>
### Nested Schema for `vpns`

Read-Only:

- `connection_id` (String) The connection ID.
- `connection_name` (String) The connection name.
- `local_asn` (String) Local Autonomous System Number.
- `name` (String) The name of the VPN.
- `remote_asn` (String) Remote Autonomous System Number.
- `remote_ip_address` (String) Remote IP address for the VPN.
- `static_routes` (List of String) List of static routes.
- `vpn_connection_status` (String) The status of the VPN connection.
- `vpn_id` (String) The VPN ID.
- `vpn_tunnels` (List of Object) List of VPN tunnels. (see [below for nested schema](#nestedobjatt--vpns--vpn_tunnels))

<a id="nestedobjatt--vpns--vpn_tunnels"></a>
### Nested Schema for `vpns.vpn_tunnels`

Read-Only:

- `is_logs_enabled` (Boolean) Whether logging is enabled for this tunnel.
- `psk` (String, Sensitive) Pre-shared key for the VPN tunnel.
- `ptp_cidr` (String) Point-to-point CIDR block.
- `startup_action` (String) Startup action for the tunnel.
