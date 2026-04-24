# Mule4 Support in API Instance Resource

## Overview

The `anypoint_api_instance` resource now supports **both FlexGateway and Mule4 technologies** through a unified interface. The resource automatically handles the different endpoint configurations required by each technology.

## Changes Made

### 1. Enhanced Terraform Schema

The `endpoint` block now supports both FlexGateway and Mule4 patterns:

- **`base_path`** (Optional): For FlexGateway - constructs proxy URI as `http://0.0.0.0:8081/<base_path>`
- **`uri`** (Optional): For Mule4 - direct implementation URI (e.g., `http://www.google.com`)
- **`ssl_context_id`** (Optional): For FlexGateway TLS context (format: `secretGroupId/tlsContextId`)

These fields are **mutually exclusive** based on the `technology` value.

### 2. Updated Resource Model

```go
type EndpointModel struct {
    DeploymentType   types.String `tfsdk:"deployment_type"`
    Type             types.String `tfsdk:"type"`
    BasePath         types.String `tfsdk:"base_path"`        // FlexGateway
    URI              types.String `tfsdk:"uri"`              // Mule4
    ConsumerEndpoint types.String `tfsdk:"consumer_endpoint"`
    ResponseTimeout  types.Int64  `tfsdk:"response_timeout"`
    SSLContextID     types.String `tfsdk:"ssl_context_id"`   // FlexGateway
}
```

### 3. Technology-Specific Logic

#### Create & Update Operations (`expandCreateRequest`, `expandUpdateRequest`)

**For FlexGateway (`technology = "flexGateway"` or empty):**
```go
- Uses base_path → constructs proxyURI
- Sets TLSContexts with inbound context
- consumer_endpoint maps to endpoint.URI
```

**For Mule4 (`technology = "mule4"`):**
```go
- Uses direct uri
- Sets muleVersion4OrAbove = true
- Sets isCloudHub, proxyUri, referencesUserDomain to null
- No TLS contexts (managed by Mule runtime)
```

#### Read Operation (`flattenInstance`)

**For FlexGateway:**
```go
- Extracts base_path from proxyURI
- Reconstructs ssl_context_id from TLSContexts
- endpoint.URI → consumer_endpoint
```

**For Mule4:**
```go
- endpoint.URI → uri field
- Sets base_path and ssl_context_id to null
```

## Usage Examples

### FlexGateway API Instance (Existing Pattern)

```hcl
resource "anypoint_api_instance" "flex_api" {
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "flex-api"
  approval_method = "manual"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = "my-api"
    ssl_context_id  = "${anypoint_secret_group.main.id}/${anypoint_flex_tls_context.flex.id}"
  }

  gateway_id = anypoint_managed_flexgateway.main.id

  routing = [
    {
      label = "primary-route"
      upstreams = [
        {
          weight = 100
          uri    = "http://backend.example.com"
          label  = "backend"
        }
      ]
    }
  ]
}
```

### Mule4 API Instance (New Pattern)

```hcl
resource "anypoint_api_instance" "mule4_api" {
  environment_id  = var.environment_id
  technology      = "mule4"
  instance_label  = "mule4-api"
  approval_method = null

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    deployment_type = "HY"  # HY, CH (CloudHub), or RF (Runtime Fabric)
    type            = "http"
    uri             = "http://www.google.com"  # Direct implementation URI
    response_timeout = 30000
  }

  # Note: No gateway_id, routing, or TLS context for Mule4
}
```

## API Request Payloads

### FlexGateway Request
```json
{
  "technology": "flexGateway",
  "spec": {
    "assetId": "my-api",
    "groupId": "org-id",
    "version": "1.0.0"
  },
  "endpoint": {
    "deploymentType": "HY",
    "type": "http",
    "proxyUri": "http://0.0.0.0:8081/my-api",
    "tlsContexts": {
      "inbound": {
        "secretGroupId": "sg-123",
        "tlsContextId": "tls-456"
      }
    }
  },
  "deployment": {
    "targetId": "gateway-uuid",
    "type": "HY",
    "expectedStatus": "deployed"
  },
  "routing": [...]
}
```

### Mule4 Request
```json
{
  "technology": "mule4",
  "approvalMethod": null,
  "providerId": null,
  "endpointUri": null,
  "spec": {
    "assetId": "api-test",
    "groupId": "542cc7e3-2143-40ce-90e9-cf69da9b4da6",
    "version": "1.0.0"
  },
  "endpoint": {
    "muleVersion4OrAbove": true,
    "uri": "http://www.google.com",
    "type": "http",
    "isCloudHub": null,
    "proxyUri": null,
    "referencesUserDomain": null,
    "responseTimeout": null,
    "deploymentType": "HY"
  }
}
```

## API Response

The Mule4 API returns the autodiscovery instance name needed for Mule applications:

```json
{
  "id": 4650143,
  "autodiscoveryInstanceName": "v1:4650143",
  "endpoint": {
    "deploymentType": "HY",
    "muleVersion4OrAbove": true,
    "uri": "http://www.google.com",
    "type": "http"
  }
}
```

## Key Differences

| Feature | FlexGateway | Mule4 |
|---------|-------------|-------|
| **Endpoint** | `base_path` → `proxyURI` | Direct `uri` |
| **Gateway** | Requires `gateway_id` | Not applicable |
| **Routing** | Weighted upstreams | Handled by Mule app |
| **TLS** | FlexGateway TLS context | Mulesoft runtime config |
| **Deployment** | Target gateway details | Environment only |
| **Autodiscovery** | Not used | Use `autodiscoveryInstanceName` |

## Testing

Run the example configuration:

```bash
cd examples/comprehensive-e2e
terraform init
terraform plan -var-file=terraform.tfvars
terraform apply -var-file=terraform.tfvars
```

## Validation

The resource will validate that:
1. For `technology="flexGateway"`: `base_path` is provided (not `uri`)
2. For `technology="mule4"`: `uri` is provided (not `base_path`)
3. TLS context is only used with FlexGateway
4. Routing is only used with FlexGateway

## Backward Compatibility

✅ All existing FlexGateway configurations continue to work without changes.

The default `technology` value remains `"flexGateway"` for backward compatibility.
