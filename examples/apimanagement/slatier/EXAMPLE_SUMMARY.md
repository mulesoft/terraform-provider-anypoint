# SLA Tier Example - Summary

**Created:** March 31, 2026
**Location:** `examples/apimanagement/slatier/`

## Overview

A comprehensive example demonstrating SLA (Service Level Agreement) tier management for API instances in Anypoint Platform.

## Files Created

### 1. `main.tf` (429 lines)
Complete Terraform configuration with 8 different SLA tier examples:

| Tier | Request Limits | Auto-Approve | Status | Use Case |
|------|----------------|--------------|--------|----------|
| **Gold** | 999,999,999/min (unlimited) | Yes | Active | Premium customers |
| **Silver** | 1k/min, 50k/hr, 1M/day | Yes | Active | Enterprise customers |
| **Bronze** | 100/min, 5k/hr | Yes | Active | Regular customers |
| **Trial** | 10/min, 500/hr, 5k/day | No | Active | Trial users |
| **Developer** | 2/sec, 50/min, 1k/hr | Yes | Active | Development/testing |
| **Partner** | 500/min, 25k/hr, 500k/day | No | Active | Integration partners |
| **Free** | 5/min, 1k/day | Yes | Active | Public API access |
| **Legacy** | 20/min | No | Deprecated | Phasing out |

### 2. `variables.tf`
Standard provider and resource configuration variables:
- Provider credentials (with defaults from e2e)
- Organization ID (default: `542cc7e3-2143-40ce-90e9-cf69da9b4da6`)
- Environment ID (required)
- API Instance ID (required)

### 3. `README.md` (370 lines)
Comprehensive documentation including:
- Overview of SLA tiers
- Detailed description of all 8 tiers
- Prerequisites and setup instructions
- Configuration options and time units
- Common use cases with examples
- Integration with rate limiting policies
- Best practices
- Troubleshooting guide
- Cleanup instructions

### 4. `terraform.tfvars.example`
Example variable file for quick setup with commented defaults.

### 5. `EXAMPLE_SUMMARY.md` (This file)
Summary of the example creation.

## Key Features

### Multiple Time Windows
Examples demonstrate combining different time periods:
- **Second-level** - Prevent burst traffic
- **Minute-level** - Standard rate limiting
- **Hour-level** - Hourly quotas
- **Day-level** - Daily usage limits

### Different Approval Models
- **Auto-approve** - Free/Trial tiers for immediate access
- **Manual approval** - Paid/Partner tiers for controlled access

### Tier Status Management
- **ACTIVE** - Available for new subscriptions
- **DEPRECATED** - Legacy tiers being phased out

### Real-World Use Cases
1. **Simple tiered access** - Free vs Pro
2. **Burst protection** - Multi-window rate limiting
3. **Partner integration** - Internal vs external partners
4. **Trial management** - Limited access with approval

## Resource Configuration

### Basic Structure
```hcl
resource "anypoint_api_instance_sla_tier" "name" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Tier Name"
  description = "Description"

  limits {
    time_period      = 1
    time_unit        = "MINUTE"  # SECOND, MINUTE, HOUR, DAY
    maximum_requests = 100
  }

  auto_approve = true        # true or false
  status       = "ACTIVE"    # ACTIVE or DEPRECATED
}
```

### Time Units Supported
- `SECOND` - Per-second limits
- `MINUTE` - Per-minute limits (most common)
- `HOUR` - Hourly quotas
- `DAY` - Daily limits

## Outputs Provided

### Individual Tier IDs
- `gold_tier_id`
- `silver_tier_id`
- `bronze_tier_id`
- `trial_tier_id`
- `developer_tier_id`
- `partner_tier_id`
- `free_tier_id`

### Aggregate Outputs
- `all_tier_ids` - Map of all tier IDs
- `all_tier_names` - Map of all tier names

## Integration Points

### With API Policies
SLA tiers work with the `rate-limiting-sla-based` policy:

```hcl
resource "anypoint_api_policy" "sla_rate_limit" {
  asset_id      = "rate-limiting-sla-based"
  asset_version = "1.3.1"

  configuration_data = jsonencode({
    clientIdExpression     = "#[attributes.headers['client_id']]"
    clientSecretExpression = "#[attributes.headers['client_secret']]"
    # ... configuration
  })
}
```

### With Client Applications
Once tiers are created, API consumers can:
1. Request access to the API
2. Select their desired tier
3. Get approved (auto or manual)
4. Receive client credentials
5. Make API calls within tier limits

## Best Practices Included

1. **Conservative Start** - Begin with lower limits
2. **Multiple Time Windows** - Comprehensive rate control
3. **Deprecation Strategy** - Use DEPRECATED status
4. **Manual Approval** - For paid/premium tiers
5. **Meaningful Naming** - Clear tier hierarchy
6. **Documentation** - Detailed descriptions
7. **Monitoring** - Track actual vs allowed usage
8. **Versioning** - Use version suffixes for major changes

## Usage Instructions

### Quick Start
```bash
# 1. Copy and edit variables
cp terraform.tfvars.example terraform.tfvars
# Edit: Set environment_id and api_instance_id

# 2. Initialize
terraform init

# 3. Review plan
terraform plan

# 4. Apply configuration
terraform apply

# 5. View outputs
terraform output
```

### Required Information
Before running, you need:
- **Environment ID** - Where your API is deployed
- **API Instance ID** - Numeric ID of your API instance

Finding API Instance ID:
```bash
# Using Anypoint CLI
anypoint-cli api-manager api list \
  --organizationId=<org-id> \
  --environmentId=<env-id>

# Or from UI URL:
# .../apis/12345678 <- This is your API Instance ID
```

## Example Tier Configurations

### Unlimited Access Tier
```hcl
limits {
  time_period      = 1
  time_unit        = "MINUTE"
  maximum_requests = 999999999  # Effectively unlimited
}
```

### Burst Protection Tier
```hcl
# Prevent burst attacks
limits {
  time_period      = 1
  time_unit        = "SECOND"
  maximum_requests = 10
}

# Daily quota
limits {
  time_period      = 1
  time_unit        = "DAY"
  maximum_requests = 50000
}
```

### Trial Tier
```hcl
limits {
  time_period      = 1
  time_unit        = "MINUTE"
  maximum_requests = 10
}

auto_approve = false  # Requires approval
status       = "ACTIVE"
```

## Troubleshooting Covered

### Common Issues
1. **API Instance Not Found** - Verify ID and access
2. **Tier Name Conflicts** - Import or rename existing
3. **Invalid Time Units** - Use uppercase values
4. **Subscription Impact** - Warning about tier deletion

### Solutions Provided
- Verification commands
- Import instructions
- Proper value formats
- Impact warnings

## Testing Recommendations

### Before Production
1. Test with developer tier first
2. Verify rate limits work correctly
3. Test approval workflows
4. Monitor actual usage patterns
5. Adjust limits based on data

### In Production
1. Start conservative with limits
2. Monitor API analytics
3. Review subscription requests
4. Adjust tiers based on usage
5. Communicate changes to consumers

## Additional Resources Linked

- Anypoint API Manager Documentation
- SLA Tiers Overview
- Rate Limiting Policies
- Terraform Provider Documentation
- MuleSoft Community Support

## Updates to Other Files

### `examples/README.md`
Added API Management section with:
- SLA Tier resource listed
- Use cases for API management
- Link to apimanagement directory

### Directory Structure
```
examples/apimanagement/
├── api_instance/
├── managed_flexgateway/
├── policies/
└── slatier/          ← NEW
    ├── main.tf
    ├── variables.tf
    ├── README.md
    ├── terraform.tfvars.example
    └── EXAMPLE_SUMMARY.md
```

## Success Criteria

✅ **Complete Example** - 8 diverse tier configurations
✅ **Comprehensive Documentation** - 370-line README
✅ **Standard Defaults** - Matches e2e configuration
✅ **Real-World Use Cases** - Production-ready patterns
✅ **Best Practices** - Industry-standard recommendations
✅ **Troubleshooting** - Common issues and solutions
✅ **Integration Examples** - Policy and client app integration
✅ **Updated Index** - examples/README.md includes new section

## Future Enhancements

Potential additions to this example:
1. Alert configuration for tier limit breaches
2. Client application subscription examples
3. Tier migration strategies
4. Analytics and monitoring setup
5. Automated tier adjustment based on usage

---

**Example Status:** ✅ Complete and Production-Ready
