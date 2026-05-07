# Managed Omni Gateway Example

This example demonstrates how to deploy and configure Managed Omni Gateways in Anypoint Platform using Terraform.

## Overview

Managed Omni Gateway is a lightweight, high-performance API gateway that runs as a container or Kubernetes service. In managed mode, the gateway is controlled and monitored from Anypoint Platform.

Key features:
- Deploy to CloudHub 2.0 private spaces, Kubernetes, or Docker
- Centralized configuration and monitoring from Anypoint Platform
- Support for all Anypoint API policies
- Auto-scaling and high availability
- Built-in observability with logs, metrics, and tracing

## What This Example Creates

This example creates **2 Managed Omni Gateway instances**:

### 1. **Basic Gateway** - Minimal configuration
- Name: `my-basic-gateway`
- Runtime version: Auto-resolved to latest LTS
- Release channel: LTS (default)
- Size: Default
- Minimal settings for quick deployment

### 2. **Production Gateway** - Full configuration
- Name: `my-production-gateway`
- Runtime version: `1.9.9` (explicit)
- Release channel: LTS
- Size: small
- **Ingress Configuration**:
  - Forward SSL session: Enabled
  - Last mile security: Enabled
- **Properties**:
  - Upstream response timeout: 30 seconds
  - Connection idle timeout: 120 seconds
- **Logging**:
  - Level: INFO
  - Forward logs: Enabled
- **Tracing**: Disabled

## Prerequisites

Before running this example, you need:

1. **Anypoint Platform Account** with Runtime Manager permissions
2. **Connected App Credentials** (Client ID and Secret)
3. **Environment ID** - Where the gateway will be deployed
4. **Target ID** - CloudHub 2.0 private space or Kubernetes target

### Finding Your Target ID

```bash
# For CloudHub 2.0 private spaces
anypoint-cli runtime-mgr cloudhub2 private-space list

# For Kubernetes/Runtime Fabric
anypoint-cli runtime-mgr runtime-fabric list
```

## Configuration

### Managed Omni Gateway Resource Structure

```hcl
resource "anypoint_managed_omni_gateway" "example" {
  name            = "my-gateway"
  environment_id  = var.environment_id
  target_id       = var.target_id
  runtime_version = "1.9.9"
  release_channel = "lts"  # or "stable", "edge"
  size            = "small"  # or "large"

  ingress = {
    forward_ssl_session = true
    last_mile_security  = true
  }

  properties = {
    upstream_response_timeout = 30
    connection_idle_timeout   = 120
  }

  logging = {
    level        = "info"  # debug, info, warn, error
    forward_logs = true
  }

  tracing = {
    enabled = false
  }
}
```

## Key Configuration Options

### Release Channels
- **lts** - Long-term support (recommended for production)
- **stable** - Regular releases with new features
- **edge** - Latest features (may be less stable)

### Gateway Sizes

| Size | Use Case |
|------|----------|
| **small** | Development, light traffic |
| **large** | Production workloads |

### Ingress Configuration

#### forward_ssl_session
When `true`, forwards SSL session information to the upstream service:
- SSL certificate details
- Cipher information
- Client certificate (for mTLS)

#### last_mile_security
When `true`, enforces TLS for the last mile between gateway and upstream:
- Gateway re-encrypts traffic to upstream
- Ensures end-to-end encryption

### Properties

#### upstream_response_timeout
Maximum time (seconds) to wait for upstream response:
- Default: 30 seconds
- Range: 1-300 seconds
- Times out long-running requests

#### connection_idle_timeout
Maximum time (seconds) to keep idle connections open:
- Default: 60 seconds
- Range: 1-3600 seconds
- Helps manage connection pools

### Logging Levels
- **debug** - Detailed debugging information
- **info** - General informational messages (recommended)
- **warn** - Warning messages only
- **error** - Error messages only

## Usage

### Step 1: Set Required Variables

Create a `terraform.tfvars` file:

```hcl
anypoint_client_id     = "your-client-id"
anypoint_client_secret = "your-client-secret"
anypoint_base_url      = "https://anypoint.mulesoft.com"

environment_id = "your-env-id"
target_id      = "your-target-id"  # Private space or K8s target
```

### Step 2: Initialize Terraform

```bash
terraform init
```

### Step 3: Review the Plan

```bash
terraform plan
```

### Step 4: Deploy the Gateways

```bash
terraform apply
```

### Step 5: Verify Deployment

```bash
# View gateway IDs and status
terraform output

# Check in Runtime Manager UI
# Navigate to: Runtime Manager → Omni Gateway
```

## Common Use Cases

### Use Case 1: Development Gateway

Quick setup for development and testing:

```hcl
resource "anypoint_managed_omni_gateway" "dev" {
  name           = "dev-gateway"
  environment_id = var.dev_environment_id
  target_id      = var.dev_target_id
  # runtime_version auto-resolved
  # minimal configuration
}
```

### Use Case 2: Production Gateway with Monitoring

Full observability configuration:

```hcl
resource "anypoint_managed_omni_gateway" "prod" {
  name            = "prod-gateway"
  environment_id  = var.prod_environment_id
  target_id       = var.prod_target_id
  runtime_version = "1.9.9"
  release_channel = "lts"
  size            = "large"

  ingress = {
    forward_ssl_session = true
    last_mile_security  = true
  }

  properties = {
    upstream_response_timeout = 60
    connection_idle_timeout   = 300
  }

  logging = {
    level        = "info"
    forward_logs = true
  }

  tracing = {
    enabled = true
  }
}
```

### Use Case 3: High-Availability Setup

Multiple gateways for load balancing:

```hcl
resource "anypoint_managed_omni_gateway" "gateway" {
  count = 3  # Deploy 3 instances

  name            = "prod-gateway-${count.index + 1}"
  environment_id  = var.environment_id
  target_id       = var.target_id
  runtime_version = "1.9.9"
  size            = "large"

  ingress = {
    forward_ssl_session = true
    last_mile_security  = true
  }

  logging = {
    level        = "info"
    forward_logs = true
  }
}
```

## Integration with API Instances

### Deploy API to Gateway

```hcl
# Create gateway
resource "anypoint_managed_omni_gateway" "main" {
  name           = "api-gateway"
  environment_id = var.environment_id
  target_id      = var.target_id
}

# Deploy API instance to gateway
resource "anypoint_api_instance" "api" {
  environment_id = var.environment_id
  technology     = "flexGateway"
  gateway_id     = anypoint_managed_omni_gateway.main.id

  spec = {
    asset_id = "customer-api"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = "customers"
  }

  routing = [
    {
      label = "default"
      rules = {
        methods = "GET|POST|PUT|DELETE"
      }
      upstreams = [
        {
          weight = 100
          uri    = "http://customer-service.internal:8080"
          label  = "Customer Service"
        }
      ]
    }
  ]
}
```

## Outputs

This example provides:

```bash
# Gateway IDs
terraform output basic_gateway_id
terraform output complete_gateway_id

# Gateway status
terraform output basic_gateway_status
terraform output complete_gateway_status

# Gateway URLs (auto-computed)
terraform output complete_gateway_public_urls
terraform output complete_gateway_internal_url
```

## Best Practices

1. **Use LTS Channel** - For production stability
2. **Pin Runtime Version** - Explicit versions prevent unexpected updates
3. **Enable Logging** - Forward logs for troubleshooting and monitoring
4. **Size Appropriately** - Start with small, scale to large based on traffic
5. **Enable Last Mile Security** - Ensure end-to-end encryption
6. **Set Appropriate Timeouts** - Balance responsiveness and long operations
7. **Deploy Multiple Instances** - For high availability
8. **Monitor Gateway Health** - Use Anypoint Monitoring dashboards

## Monitoring and Observability

### View Gateway Metrics

Navigate to: **Runtime Manager → Omni Gateway → Your Gateway → Monitoring**

Metrics available:
- Request rate and latency
- Error rates
- CPU and memory usage
- Network throughput
- Active connections

### Access Gateway Logs

```bash
# Via Anypoint CLI
anypoint-cli runtime-mgr cloudhub2 omnigateway logs <gateway-id>

# Or in UI: Runtime Manager → Omni Gateway → Logs
```

### Enable Tracing

```hcl
tracing = {
  enabled = true
}
```

Traces appear in Anypoint Monitoring for request flow analysis.

## Lifecycle Management

### Update Gateway Configuration

Modify configuration and apply:
```bash
terraform apply
```

**Note:** Some changes may require gateway restart.

### Upgrade Runtime Version

```hcl
resource "anypoint_managed_omni_gateway" "main" {
  runtime_version = "1.10.0"  # Updated version
  # ... other configuration
}
```

```bash
terraform apply
```

### Scale Gateway

Change size:
```hcl
size = "large"  # Was "small"
```

### Remove Gateway

```bash
terraform destroy -target=anypoint_managed_omni_gateway.basic
```

**Warning:** Ensure no API instances are deployed to the gateway first.

## Troubleshooting

### Error: Target Not Found

```
Error: Target with ID 'xxx' not found
```

**Solution:** Verify target exists and you have access:
```bash
anypoint-cli runtime-mgr cloudhub2 private-space list
```

### Gateway Not Starting

**Common Causes:**
- Insufficient resources in target
- Network connectivity issues
- Invalid configuration

**Check logs:**
```bash
anypoint-cli runtime-mgr cloudhub2 omnigateway logs <gateway-id>
```

### Gateway Status: Disconnected

**Solution:**
1. Check target health
2. Verify network connectivity
3. Check gateway logs
4. Restart gateway if needed

### High Memory Usage

**Solutions:**
- Increase gateway size
- Reduce connection idle timeout
- Check for memory leaks in policies
- Review API traffic patterns

## Additional Resources

- [Omni Gateway Documentation](https://docs.mulesoft.com/gateway/)
- [Omni Gateway Installation](https://docs.mulesoft.com/gateway/omni-install)
- [Gateway Sizing Guide](https://docs.mulesoft.com/gateway/omni-architecture)
- [Gateway Configuration Reference](https://docs.mulesoft.com/gateway/omni-conn-reg-run)

## Cleanup

To remove all created gateways:

```bash
terraform destroy
```

**Warning:** Ensure no API instances are deployed before destroying gateways.
