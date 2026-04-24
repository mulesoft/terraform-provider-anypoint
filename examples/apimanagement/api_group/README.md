# API Group Example

This example demonstrates how to create and manage **API Groups** in Anypoint API Manager using Terraform.

## Overview

An API Group bundles multiple API instances under a shared, versioned contract. This lets API consumers subscribe to a set of related APIs with a single client application, instead of subscribing to each API individually.

Typical use cases:
- Grouping microservices that are always consumed together (e.g. Orders + Inventory + Shipping)
- Offering a versioned "product API" built from several backend APIs
- Providing a unified subscription entry point across multiple environments

## What This Example Creates

### 1. `anypoint_api_group.payments` — Simple single-version group
- One version (`v1`) with API instances from the primary environment

### 2. `anypoint_api_group.orders` — Multi-version, multi-environment group
- Version `v1` — sandbox instances
- Version `v2` — sandbox instances **and** staging instances (demonstrates cross-environment versioning)

## Prerequisites

1. An Anypoint Platform account with **API Manager** access
2. A Connected App with the `API Manager` permission scope
3. At least one already-deployed API instance — note its numeric ID from the API Manager URL

## Usage

### Step 1: Set Variables

Create a `terraform.tfvars` file:

```hcl
organization_id  = "your-org-id"
environment_id   = "your-sandbox-env-id"
api_instance_ids = [12345678, 87654321]
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

## Resource Reference

```hcl
resource "anypoint_api_group" "example" {
  organization_id = var.organization_id
  name            = "My API Group"

  versions = [
    {
      name = "v1"
      instances = [
        {
          environment_id       = var.environment_id
          group_instance_label = ""           # optional label
          api_instances        = [12345678]   # numeric API instance IDs
        }
      ]
    }
  ]
}
```

### Key Attributes

| Attribute | Description |
|---|---|
| `name` | Display name of the API Group |
| `versions[].name` | Version label (e.g. `v1`, `v2`) |
| `versions[].instances[].environment_id` | Environment that owns the API instances |
| `versions[].instances[].api_instances` | List of numeric API instance IDs |
| `versions[].instances[].group_instance_label` | Optional label for the instance set |

### Computed Attributes

| Attribute | Description |
|---|---|
| `id` | Numeric group ID assigned by the platform |
| `versions[].id` | Numeric version ID assigned by the platform |

## Import

```bash
# By group ID only
terraform import anypoint_api_group.example 565156

# By org ID and group ID
terraform import anypoint_api_group.example <org-id>/565156
```

## Outputs

| Output | Description |
|---|---|
| `payments_group_id` | Numeric ID of the Payments API Group |
| `orders_group_id` | Numeric ID of the Orders API Group |

## Cleanup

```bash
terraform destroy
```

> **Note:** Deleting an API group removes all consumer subscriptions associated with it.

## Additional Resources

- [API Groups – Anypoint Platform Docs](https://docs.mulesoft.com/api-manager/latest/api-groups-landing-page)
- [Terraform Anypoint Provider](https://registry.terraform.io/providers/mulesoft/anypoint)
