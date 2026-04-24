# API Group SLA Tier Example

This example demonstrates how to create and manage **SLA tiers for API Group instances** in Anypoint API Manager using Terraform.

## Overview

API Group SLA tiers work like regular API instance SLA tiers, but apply to an entire API Group rather than a single API instance. When a consumer subscribes to the API Group, they choose an SLA tier that governs the rate limits across all APIs bundled in the group.

Key differences from `anypoint_api_instance_sla_tier`:

| | `anypoint_api_instance_sla_tier` | `anypoint_api_group_sla_tier` |
|---|---|---|
| Scoped to | Single API instance | API Group instance |
| Rate-limit field | `limits` | `default_limits` |
| ID reference | `api_instance_id` | `group_instance_id` |
| API path | `.../apis/{id}/tiers` | `.../groupInstances/{id}/tiers` |

## What This Example Creates

| Tier | Rate Limits | Auto-Approve | Use Case |
|---|---|---|---|
| **Bronze** | 100/min · 5,000/hr | Yes | Regular consumers |
| **Silver** | 500/min · 25,000/hr · 500,000/day | No | Enterprise consumers |
| **Gold** | Effectively unlimited | No | Premium / SLA-backed consumers |
| **Trial** | 10/min · 200/hr | Yes | Developers evaluating the group |

## Prerequisites

1. An Anypoint Platform account with **API Manager** access
2. A Connected App with the `API Manager` permission scope
3. An existing API Group with at least one version instance — note its **group instance ID**

### Finding the Group Instance ID

The group instance ID appears in the URL when you open a group instance in API Manager:

```
https://<anypoint>/apimanager/.../groupInstances/565156/tiers
                                                  ^^^^^^
                                          this is the group_instance_id
```

Alternatively, you can read it from the `versions[*].id` attribute of an `anypoint_api_group` resource created with Terraform.

## Usage

### Step 1: Set Variables

Create a `terraform.tfvars` file:

```hcl
organization_id   = "your-org-id"
environment_id    = "your-env-id"
group_instance_id = "565156"   # numeric group instance ID
```

### Step 2: Initialize Terraform

```bash
terraform init
```

### Step 3: Review the Plan

```bash
terraform plan
```

### Step 4: Apply

```bash
terraform apply
```

### Step 5: Check Outputs

```bash
terraform output all_tier_ids
```

## Resource Reference

```hcl
resource "anypoint_api_group_sla_tier" "example" {
  organization_id   = var.organization_id
  environment_id    = var.environment_id
  group_instance_id = var.group_instance_id

  name        = "Bronze"
  description = "Standard tier"
  status      = "ACTIVE"
  auto_approve = true

  default_limits = [
    {
      maximum_requests            = 100
      time_period_in_milliseconds = 60000    # 1 minute
      visible                     = true
    },
    {
      maximum_requests            = 5000
      time_period_in_milliseconds = 3600000  # 1 hour
      visible                     = true
    }
  ]
}
```

### Key Attributes

| Attribute | Required | Description |
|---|---|---|
| `organization_id` | Optional | Defaults to provider org |
| `environment_id` | Required | Environment of the group instance |
| `group_instance_id` | Required | Numeric group instance ID (forces recreation if changed) |
| `name` | Required | Display name of the tier |
| `description` | Optional | Human-readable description |
| `status` | Optional | `ACTIVE` (default) or `INACTIVE` |
| `auto_approve` | Optional | `false` by default |
| `default_limits` | Required | One or more rate-limit windows |

### `default_limits` Fields

| Field | Description |
|---|---|
| `maximum_requests` | Max requests in the window |
| `time_period_in_milliseconds` | Window size in ms |
| `visible` | Whether limit is shown to consumers (default `true`) |

### Common Time Periods

| Period | Milliseconds |
|---|---|
| 1 second | 1,000 |
| 1 minute | 60,000 |
| 1 hour | 3,600,000 |
| 1 day | 86,400,000 |

## Import

```bash
terraform import anypoint_api_group_sla_tier.bronze \
  <org-id>/<env-id>/<group-instance-id>/<tier-id>
```

## Outputs

| Output | Description |
|---|---|
| `bronze_tier_id` | ID of the Bronze SLA tier |
| `silver_tier_id` | ID of the Silver SLA tier |
| `gold_tier_id` | ID of the Gold SLA tier |
| `trial_tier_id` | ID of the Trial SLA tier |
| `all_tier_ids` | Map of tier name → tier ID |

## End-to-End Example with API Group

Create the API Group and its SLA tiers together:

```hcl
resource "anypoint_api_group" "payments" {
  organization_id = var.organization_id
  name            = "Payments API Group"

  versions = [
    {
      name = "v1"
      instances = [
        {
          environment_id = var.environment_id
          api_instances  = [12345678, 87654321]
        }
      ]
    }
  ]
}

resource "anypoint_api_group_sla_tier" "bronze" {
  organization_id   = var.organization_id
  environment_id    = var.environment_id
  group_instance_id = anypoint_api_group.payments.versions[0].id   # version ID = group instance ID

  name         = "Bronze"
  auto_approve = true

  default_limits = [
    {
      maximum_requests            = 100
      time_period_in_milliseconds = 60000
      visible                     = true
    }
  ]
}
```

## Cleanup

```bash
terraform destroy
```

> **Warning:** Removing an SLA tier will affect any active consumer subscriptions using that tier.

## Additional Resources

- [API Groups – Anypoint Platform Docs](https://docs.mulesoft.com/api-manager/latest/api-groups-landing-page)
- [SLA Tiers Overview](https://docs.mulesoft.com/api-manager/2.x/manage-client-apps-latest-task)
- [Terraform Anypoint Provider](https://registry.terraform.io/providers/mulesoft/anypoint)
