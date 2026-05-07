# API Instance Example

This example demonstrates how to create and manage API instances in Anypoint Platform using Terraform.

## Overview

API instances represent deployed API proxies in API Manager. They connect API specifications from Exchange to runtime implementations and enable you to:
- Deploy APIs to Omni Gateway or other targets
- Configure routing rules with weighted load balancing
- Define consumer endpoints
- Apply policies and SLA tiers
- Manage API lifecycle

## What This Example Creates

This example creates **2 API instances**:

### 1. **MyHealth API** - Complex routing with weighted upstreams
- Technology: Omni Gateway
- Instance Label: `myhealth-api`
- Approval Method: Manual
- Base Path: `/myhealth`
- **Routing**:
  - **Read Traffic** (GET requests):
    - 50% → Backend 1
    - 50% → Backend 2
  - **Write Traffic** (POST/PUT/PATCH/DELETE to `/api/*`):
    - 100% → Write Backend

### 2. **Simple API** - Basic routing configuration
- Technology: Omni Gateway
- Instance Label: `simple-api`
- Approval Method: Automatic
- Base Path: `/simple`
- Single backend with default routing

## Prerequisites

Before running this example, you need:

1. **Anypoint Platform Account** with API Manager permissions
2. **Connected App Credentials** (Client ID and Secret)
3. **Existing Managed Omni Gateway** - Get the gateway ID
4. **API Specification in Exchange** - Asset ID and version
5. **Environment ID** - Where the API will be deployed

## Configuration

### API Instance Resource Structure

```hcl
resource "anypoint_api_instance" "example" {
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "my-api"
  approval_method = "manual"  # or "automatic"

  spec = {
    asset_id = "my-api-spec"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"  # Hybrid
    type            = "http"
    base_path       = "myapi"
  }

  gateway_id = var.gateway_id

  routing = [
    {
      label = "traffic-route"
      rules = {
        methods = "GET|POST"
        path    = "/api/*"
      }
      upstreams = [
        {
          weight = 100
          uri    = "http://backend.example.com"
          label  = "Backend"
        }
      ]
    }
  ]
}
```

## Key Configuration Options

### Technology
- `flexGateway` - Omni Gateway runtime
- `mule4` - Mule 4 runtime
- `mule3` - Mule 3 runtime

### Approval Method
- `manual` - API consumer requests require approval
- `automatic` - API consumer requests are auto-approved

### Deployment Type
- `HY` - Hybrid deployment
- `CH` - CloudHub deployment
- `RTF` - Runtime Fabric deployment

### Routing Rules

#### Methods
Pipe-separated HTTP methods:
```hcl
methods = "GET|POST|PUT|DELETE"
```

#### Path Patterns
```hcl
path = "/api/*"       # Wildcard
path = "/users/{id}"  # Path parameter
```

#### Weighted Upstreams
Distribute traffic across multiple backends:
```hcl
upstreams = [
  { weight = 70, uri = "http://primary.example.com", label = "Primary" },
  { weight = 30, uri = "http://secondary.example.com", label = "Secondary" }
]
```

## Usage

### Step 1: Set Required Variables

Create a `terraform.tfvars` file:

```hcl
anypoint_client_id     = "your-client-id"
anypoint_client_secret = "your-client-secret"
anypoint_base_url      = "https://anypoint.mulesoft.com"

organization_id = "your-org-id"
environment_id  = "your-env-id"
gateway_id      = "your-gateway-id"
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

### Step 5: Verify API Instances

```bash
# View created API instance IDs
terraform output

# Check in API Manager UI
# Navigate to: API Manager → API Administration → Your APIs
```

## Common Use Cases

### Use Case 1: Blue-Green Deployment

```hcl
resource "anypoint_api_instance" "api" {
  # ... basic configuration

  routing = [
    {
      label = "blue-green"
      rules = {
        methods = "GET|POST|PUT|DELETE"
      }
      upstreams = [
        {
          weight = 90  # 90% to blue (current)
          uri    = "http://blue.example.com"
          label  = "Blue"
        },
        {
          weight = 10  # 10% to green (canary)
          uri    = "http://green.example.com"
          label  = "Green"
        }
      ]
    }
  ]
}
```

### Use Case 2: Path-Based Routing

```hcl
resource "anypoint_api_instance" "api" {
  # ... basic configuration

  routing = [
    {
      label = "read-operations"
      rules = {
        methods = "GET"
        path    = "/api/v1/*"
      }
      upstreams = [
        {
          weight = 100
          uri    = "http://read-replica.example.com"
          label  = "Read Replica"
        }
      ]
    },
    {
      label = "write-operations"
      rules = {
        methods = "POST|PUT|DELETE"
        path    = "/api/v1/*"
      }
      upstreams = [
        {
          weight = 100
          uri    = "http://primary.example.com"
          label  = "Primary"
        }
      ]
    }
  ]
}
```

### Use Case 3: Multi-Region Routing

```hcl
resource "anypoint_api_instance" "api" {
  # ... basic configuration

  routing = [
    {
      label = "multi-region"
      rules = {
        methods = "GET|POST|PUT|DELETE"
      }
      upstreams = [
        {
          weight = 50
          uri    = "http://us-east.example.com"
          label  = "US East"
        },
        {
          weight = 30
          uri    = "http://us-west.example.com"
          label  = "US West"
        },
        {
          weight = 20
          uri    = "http://eu-central.example.com"
          label  = "EU Central"
        }
      ]
    }
  ]
}
```

## Integration with Other Resources

### Apply Policies

```hcl
resource "anypoint_api_policy_rate_limiting" "rate_limit" {
  api_instance_id = anypoint_api_instance.myhealth_api.id

  configuration = {
    key_selector = "#[attributes.queryParams['client_id']]"
    rate_limits = [
      {
        maximum_requests            = 100
        time_period_in_milliseconds = 60000
      }
    ]
  }
}
```

### Create SLA Tiers

```hcl
resource "anypoint_api_instance_sla_tier" "gold" {
  api_instance_id = anypoint_api_instance.myhealth_api.id

  name = "Gold"
  limits = [
    {
      time_period_in_milliseconds = 60000
      maximum_requests            = 10000
      visible                     = true
    }
  ]
}
```

## Outputs

This example provides:

```bash
# View API IDs
terraform output myhealth_api_id
terraform output simple_api_id

# View API status
terraform output myhealth_api_status
```

## Best Practices

1. **Use Meaningful Labels** - Instance labels help identify APIs in monitoring
2. **Start with Manual Approval** - Control who accesses your API initially
3. **Weighted Routing** - Enables gradual rollouts and canary deployments
4. **Path-Based Rules** - Route different operations to optimized backends
5. **Monitor Upstream Health** - Ensure backend services are available
6. **Version Your APIs** - Use semantic versioning in Exchange assets
7. **Test Routing Rules** - Verify traffic distribution before production

## Troubleshooting

### Error: Gateway Not Found

```
Error: Gateway with ID 'xxx' not found
```

**Solution:** Verify the gateway exists and is deployed:
```bash
# Use Anypoint CLI
anypoint-cli runtime-mgr cloudhub2 omnigateway list

# Or check in UI: Runtime Manager → Omni Gateway
```

### Error: Asset Not Found in Exchange

```
Error: Asset 'my-api-spec' not found in Exchange
```

**Solution:** Verify the asset exists:
```bash
anypoint-cli exchange asset list --organization-id=<org-id>
```

### Routing Not Working

**Common Issues:**
- Check upstream URIs are accessible from the gateway
- Verify path patterns match actual request paths
- Ensure HTTP methods match the routing rules
- Check gateway logs for routing errors

## Lifecycle Management

### Update Routing Weights

Modify weights and apply:
```bash
terraform apply
```

### Add New Upstream

Add to upstreams array and apply:
```hcl
upstreams = [
  { weight = 60, uri = "http://backend1.example.com", label = "Backend 1" },
  { weight = 30, uri = "http://backend2.example.com", label = "Backend 2" },
  { weight = 10, uri = "http://backend3.example.com", label = "Backend 3" }  # New
]
```

### Remove API Instance

```bash
terraform destroy -target=anypoint_api_instance.myhealth_api
```

**Warning:** This will remove all associated policies, SLA tiers, and contracts.

## Additional Resources

- [Anypoint API Manager Documentation](https://docs.mulesoft.com/api-manager/)
- [Omni Gateway Documentation](https://docs.mulesoft.com/gateway/)
- [API Routing Strategies](https://docs.mulesoft.com/api-manager/2.x/configure-multiple-credential-providers)
- [API Lifecycle Management](https://docs.mulesoft.com/api-manager/2.x/api-lifecycle-concept)

## Cleanup

To remove all created resources:

```bash
terraform destroy
```

This will remove both API instances and their configurations.
