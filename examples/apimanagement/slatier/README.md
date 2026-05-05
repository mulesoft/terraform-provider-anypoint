# SLA Tier Management Example

This example demonstrates how to create and manage SLA (Service Level Agreement) tiers for API instances in Anypoint Platform using Terraform.

## Overview

SLA tiers define rate limits and access controls for API consumers. They allow you to:
- Control API access based on subscription levels
- Implement tiered pricing models
- Protect backend systems from overuse
- Differentiate service levels for different customer segments

## What This Example Creates

This example creates **8 different SLA tiers** demonstrating various use cases:

### 1. **Gold Tier** - Unlimited Access
- Maximum requests: Effectively unlimited (999,999,999 per minute)
- Auto-approve: Yes
- **Use case:** Premium customers with unlimited API access

### 2. **Silver Tier** - High Volume
- 1,000 requests per minute
- 50,000 requests per hour
- 1,000,000 requests per day
- **Use case:** Standard enterprise customers

### 3. **Bronze Tier** - Standard Access
- 100 requests per minute
- 5,000 requests per hour
- Auto-approve: Yes
- **Use case:** Regular customers with moderate usage

### 4. **Trial Tier** - Limited Access
- 10 requests per minute
- 500 requests per hour
- 5,000 requests per day
- Auto-approve: No (requires manual approval)
- **Use case:** Trial users and prospects

### 5. **Developer Tier** - Testing/Development
- 2 requests per second
- 50 requests per minute
- 1,000 requests per hour
- **Use case:** Developers testing integrations

### 6. **Partner Tier** - Trusted Partners
- 500 requests per minute
- 25,000 requests per hour
- 500,000 requests per day
- Auto-approve: No (manual approval required)
- **Use case:** Integration partners and B2B connections

### 7. **Free Tier** - Public Access
- 5 requests per minute
- 1,000 requests per day
- **Use case:** Public APIs with basic free access

### 8. **Legacy Tier** - Deprecated
- 20 requests per minute
- Status: DEPRECATED
- **Use case:** Legacy tier being phased out

## Prerequisites

Before running this example, you need:

1. **Anypoint Platform Account** with appropriate permissions
2. **Connected App Credentials** (Client ID and Secret)
3. **Existing API Instance** - You must have an API instance already deployed

### Finding Your API Instance ID

```bash
# Using Anypoint CLI
anypoint-cli api-manager api list --organizationId=<org-id> --environmentId=<env-id>

# Or using the UI
# Navigate to: API Manager → API Administration → Select your API → Copy the ID from the URL
```

## Usage

### Step 1: Set Required Variables

Create a `terraform.tfvars` file:

```hcl
# Required - Must be provided
environment_id  = "your-environment-id"
api_instance_id = "12345678"  # Numeric API instance ID

# Optional - Override defaults if needed
organization_id = "your-organization-id"
```

### Step 2: Initialize Terraform

```bash
terraform init
```

### Step 3: Review the Plan

```bash
terraform plan
```

### Step 4: Apply the Configuration

```bash
terraform apply
```

### Step 5: Verify SLA Tiers

```bash
# View created tier IDs
terraform output

# View specific tier
terraform output gold_tier_id
```

## Configuration Options

### SLA Tier Resource Structure

```hcl
resource "anypoint_api_instance_sla_tier" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Tier Name"
  description = "Tier description"

  # Define multiple rate limit windows
  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 100
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 5000
      visible                     = true
    }
  ]

  auto_approve = true        # true or false
  status       = "ACTIVE"    # ACTIVE or DEPRECATED
}
```

### Time Period Conversions

Convert time periods to milliseconds:
- **1 second** = 1,000 ms
- **1 minute** = 60,000 ms
- **1 hour** = 3,600,000 ms
- **1 day** = 86,400,000 ms

### Limits Configuration

- `time_period_in_milliseconds` - Time window in milliseconds
- `maximum_requests` - Maximum requests allowed in the time window
- `visible` - Whether this limit is visible to API consumers

### Status Values

- `ACTIVE` - Tier is available for new subscriptions
- `DEPRECATED` - Tier is marked for removal, existing subscriptions remain but new ones cannot be created

### Auto-Approve

- `true` - API consumers are automatically approved when requesting access
- `false` - Requires manual approval by API administrator

## Common Use Cases

### Use Case 1: Simple Tiered Access

Create basic Free/Pro tiers:

```hcl
resource "anypoint_api_instance_sla_tier" "free" {
  name = "Free"

  limits = [
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 100
      visible                     = true
    }
  ]

  auto_approve = true
  status       = "ACTIVE"
}

resource "anypoint_api_instance_sla_tier" "pro" {
  name = "Pro"

  limits = [
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 10000
      visible                     = true
    }
  ]

  auto_approve = false  # Paid tier requires approval
  status       = "ACTIVE"
}
```

### Use Case 2: Burst Protection

Combine multiple time windows to prevent burst traffic:

```hcl
resource "anypoint_api_instance_sla_tier" "protected" {
  name = "Protected"

  limits = [
    {
      time_period_in_milliseconds = 1000  # 1 second - prevent burst
      maximum_requests            = 10
      visible                     = true
    },
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 500
      visible                     = true
    },
    {
      time_period_in_milliseconds = 86400000  # 1 day - daily quota
      maximum_requests            = 50000
      visible                     = true
    }
  ]
}
```

### Use Case 3: Partner Integration Tiers

Separate tiers for internal vs external partners:

```hcl
resource "anypoint_api_instance_sla_tier" "internal_partner" {
  name        = "Internal Partner"
  description = "For internal business units"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 1000
      visible                     = true
    }
  ]

  auto_approve = true  # Internal partners auto-approved
  status       = "ACTIVE"
}

resource "anypoint_api_instance_sla_tier" "external_partner" {
  name        = "External Partner"
  description = "For third-party integration partners"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 500
      visible                     = true
    }
  ]

  auto_approve = false  # External partners require approval
  status       = "ACTIVE"
}
```

## Managing SLA Tiers

### Updating a Tier

Modify the tier configuration and apply:

```bash
terraform apply
```

### Deprecating a Tier

Change the status to `DEPRECATED`:

```hcl
resource "anypoint_api_instance_sla_tier" "old_tier" {
  name   = "Old Tier"
  status = "DEPRECATED"  # Mark as deprecated
  # ... other configuration
}
```

### Removing a Tier

Remove the resource from your configuration and apply:

```bash
# Remove the resource block from main.tf
terraform apply
```

**Warning:** Removing a tier will impact any active subscriptions using that tier.

## Outputs

This example provides several outputs:

```bash
# Individual tier IDs
terraform output gold_tier_id
terraform output silver_tier_id

# All tier IDs
terraform output all_tier_ids

# All tier names
terraform output all_tier_names
```

## Integration with Rate Limiting Policies

SLA tiers work in conjunction with rate limiting policies:

1. **SLA Tiers** - Define subscription-based rate limits per consumer
2. **Rate Limiting Policies** - Apply global rate limits to all traffic

Example workflow:
```hcl
# Create SLA tiers (this example)
resource "anypoint_api_instance_sla_tier" "gold" {
  # ... tier configuration
}

# Apply SLA-based rate limiting policy
resource "anypoint_api_policy_rate_limiting_sla_based" "sla_rate_limit" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  configuration = {
    client_id_expression     = "#[attributes.headers['client_id']]"
    client_secret_expression = "#[attributes.headers['client_secret']]"
    expose_headers           = true
    clusterizable            = true
  }
}
```

## Best Practices

1. **Start Conservative** - Begin with lower limits and increase based on actual usage
2. **Use Multiple Time Windows** - Combine SECOND/MINUTE/HOUR/DAY limits for comprehensive control
3. **Plan for Deprecation** - Use the DEPRECATED status instead of deleting tiers immediately
4. **Manual Approval for Paid Tiers** - Set `auto_approve = false` for premium tiers
5. **Meaningful Names** - Use clear, descriptive names (Gold/Silver/Bronze or Free/Pro/Enterprise)
6. **Document Limits** - Include clear descriptions of what each tier provides
7. **Monitor Usage** - Track actual API usage against tier limits
8. **Version Tier Names** - If you need to change limits significantly, create a new tier (e.g., "Bronze-v2")

## Troubleshooting

### Error: API Instance Not Found

```
Error: API instance 12345678 not found in environment
```

**Solution:** Verify the API instance exists and you have the correct ID:
```bash
anypoint-cli api-manager api list --environmentId=<env-id>
```

### Error: Tier Name Already Exists

```
Error: SLA tier with name 'Gold' already exists
```

**Solution:** Either:
1. Import the existing tier: `terraform import anypoint_api_instance_sla_tier.gold <tier-id>`
2. Use a different name
3. Delete the existing tier in the UI first

### Update Returns HTTP 201

The Anypoint Platform returns HTTP 201 (not 200) on a successful SLA tier update. The provider handles this correctly — if you see a provider error claiming "unexpected status 201", ensure you are on the latest provider version.

## Additional Resources

- [Anypoint API Manager Documentation](https://docs.mulesoft.com/api-manager/)
- [SLA Tiers Overview](https://docs.mulesoft.com/api-manager/2.x/manage-client-apps-latest-task)
- [Rate Limiting Policies](https://docs.mulesoft.com/api-manager/2.x/rate-limiting-and-throttling)
- [Terraform Anypoint Provider Docs](https://registry.terraform.io/providers/mulesoft/anypoint)

## Cleanup

To remove all created SLA tiers:

```bash
terraform destroy
```

**Warning:** This will delete all SLA tiers and may impact active API subscriptions.

## Support

For issues or questions:
- Provider Issues: [GitHub Issues](https://github.com/mulesoft/terraform-provider-anypoint/issues)
- Anypoint Platform: [MuleSoft Support](https://help.mulesoft.com/)
