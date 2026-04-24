# Firewall Rules Example

This example demonstrates how to use the `anypoint_firewall_rules` resource and `anypoint_firewallrules` data source to manage and retrieve firewall rules for an Anypoint Private Space.

## Overview

The `anypoint_firewall_rules` resource allows you to:
- Configure inbound and outbound firewall rules for a private space
- Manage the complete list of managed firewall rules using the PATCH API
- Define rules with CIDR blocks, protocols, port ranges, and types

The `anypoint_firewallrules` data source allows you to:
- Retrieve existing managed firewall rules from a private space
- Use existing rules as input for other resources
- Filter and analyze current firewall rule configurations

## Usage

1. Copy `terraform.tfvars.example` to `terraform.tfvars`:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. Edit `terraform.tfvars` with your actual values:
   ```hcl
   anypoint_client_id     = "your-actual-client-id"
   anypoint_client_secret = "your-actual-client-secret"
   anypoint_base_url      = "https://anypoint.mulesoft.com"
   private_space_id       = "your-actual-private-space-id"
   organization_id        = "your-actual-organization-id"  # Optional
   ```

3. Plan the changes:
   ```bash
   terraform plan
   ```

4. Apply the configuration:
   ```bash
   terraform apply
   ```

## Configuration

### Resource Configuration

The example configures firewall rules similar to the payload structure provided:

```hcl
resource "anypoint_firewall_rules" "example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id  # Optional: specify target organization

  rules = [
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 0
      to_port    = 65535
      type       = "outbound"
    },
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    },
    # ... more rules
  ]
}
```

### Data Source Configuration

The example also shows how to retrieve existing firewall rules:

```hcl
data "anypoint_firewallrules" "existing" {
  private_space_id = var.private_space_id
}

# Use the data source to filter rules
output "inbound_rules" {
  value = [
    for rule in data.anypoint_firewallrules.existing.rules : rule
    if rule.type == "inbound"
  ]
}

output "https_rules" {
  value = [
    for rule in data.anypoint_firewallrules.existing.rules : rule
    if rule.from_port == 443 && rule.to_port == 443
  ]
}
```

## Firewall Rule Parameters

Each rule in the `rules` list supports the following parameters:

- `cidr_block` (required): The CIDR block for the firewall rule (e.g., "0.0.0.0/0", "10.0.0.0/16", "local-private-network")
- `protocol` (required): The protocol for the firewall rule (tcp, udp, icmp)
- `from_port` (required): The starting port for the firewall rule
- `to_port` (required): The ending port for the firewall rule
- `type` (required): The type of the firewall rule (inbound, outbound)

## Organization ID Parameter

- `organization_id` (optional): The ID of the target organization where the private space is located. If not specified, uses the organization from provider credentials. This enables multi-organization management scenarios.

### Multi-Organization Usage

```hcl
resource "anypoint_firewall_rules" "cross_org_example" {
  private_space_id = "your-private-space-id"
  organization_id  = "target-organization-id"  # Different from provider's organization
  
  rules = [
    {
      cidr_block = "0.0.0.0/0"
      protocol   = "tcp"
      from_port  = 443
      to_port    = 443
      type       = "inbound"
    }
  ]
}
```

## Important Notes

1. **Complete Rule Management**: The resource manages the complete list of managed firewall rules. When you apply changes, it replaces the entire list with the configuration specified in Terraform.

2. **Private Space Dependency**: You must have an existing private space to configure firewall rules. The `private_space_id` parameter is required.

3. **API Endpoint**: The resource uses the PATCH API endpoint:
   ```
   https://anypoint.mulesoft.com/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}
   ```

4. **Resource ID**: The resource ID is the same as the private space ID since firewall rules are part of the private space configuration.

5. **Data Source Usage**: The `anypoint_firewallrules` data source retrieves only the `managedFirewallRules` from the API response, filtering out other private space information for focused firewall rule management.

## Importing Existing Firewall Rules

You can import existing firewall rules using the private space ID:

```bash
terraform import anypoint_firewall_rules.example your-private-space-id
```

After importing, run `terraform plan` to see the current state and adjust your configuration accordingly.

## Outputs

The example provides the following outputs:

- `firewall_rules_id`: The ID of the firewall rules resource
- `firewall_rules_private_space_id`: The private space ID
- `firewall_rules_count`: The number of configured firewall rules 