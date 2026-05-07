# API Management Examples

This directory contains examples for managing Anypoint Platform API Management resources using Terraform.

## Available Examples

### [API Instance](./api_instance/)
- **Resource**: `anypoint_api_instance`
- **Description**: Deploy and configure API instances in API Manager for OmniGateway, Mule 4, or HTTP-managed deployments
- **API**: `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis`
- **Use Cases**:
  - Deploy OmniGateway-backed API instances with routing rules
  - Deploy Mule 4 runtime API instances with endpoint configuration
  - Configure approval methods (auto/manual), SLA tiers, and instance labels

---

### [API Instances Datasource](./api_instances/)
- **Data Source**: `anypoint_api_instances`
- **Description**: List all API instances registered in API Manager for a given environment
- **Use Cases**:
  - Enumerate all API instances and their technology/status
  - Filter instances by technology (`flexGateway`, `mule4`, `http`)
  - Look up a specific instance ID by `instance_label` to reference from another config

---

### [Managed Omni Gateway](./managed_omni_gateway/)
- **Resource**: `anypoint_managed_omni_gateway`
- **Data Source**: `anypoint_managed_omni_gateway`
- **Description**: Deploy and manage Omni Gateway instances in managed mode on a private space; also look up an existing gateway by ID
- **API**: `/gatewaymanager/api/v1/organizations/{orgId}/environments/{envId}/gateways`
- **Use Cases**:
  - Create a gateway with auto-derived ingress URLs (from target domain)
  - Override `public_url` / `internal_url` with a custom domain
  - Configure ingress (SSL session forwarding, last-mile security), properties, logging, and tracing
  - Read full gateway detail (status, port configuration, runtime version) via the datasource

---

### [Managed Omni Gateways Datasource](./managed_omni_gateways/)
- **Data Source**: `anypoint_managed_omni_gateways`
- **Description**: List all managed Omni Gateways in an environment
- **API**: `/gatewaymanager/api/v1/organizations/{orgId}/environments/{envId}/gateways`
- **Use Cases**:
  - Enumerate all gateways with their ID, name, target, and status
  - Look up a specific gateway ID by name to reference from an `anypoint_api_instance`

---

### [API Policies](./policies/)
- **Resource**: `anypoint_api_policy`
- **Description**: Apply and configure API policies for security, rate limiting, and traffic management
- **API**: `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/policies`
- **Use Cases**:
  - Apply security policies (JWT Validation, OAuth 2.0, Basic Auth, Client ID Enforcement)
  - Configure rate limiting, SLA-based rate limiting, and spike control
  - Add header injection/removal, CORS, and message logging policies
  - Control policy ordering

---

### [SLA Tiers](./slatier/)
- **Resource**: `anypoint_api_instance_sla_tier`
- **Description**: Define and manage SLA tiers with rate limits for API consumers
- **API**: `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/tiers`
- **Use Cases**:
  - Create tiered API access levels (Free / Pro / Enterprise or Bronze / Silver / Gold)
  - Define rate limits across multiple time windows (second, minute, hour, day)
  - Configure auto-approval vs manual approval per tier
  - Manage tier deprecation

---

### [API Group](./api_group/)
- **Resource**: `anypoint_api_group`
- **Description**: Create and manage API Groups — logical collections of API instances that expose a unified endpoint
- **API**: `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apiGroups`
- **Use Cases**:
  - Bundle multiple API instances behind a single group endpoint
  - Define group-level routing rules across multiple API versions

---

### [API Group SLA Tiers](./api_group_sla_tier/)
- **Resource**: `anypoint_api_group_sla_tier`
- **Description**: Manage SLA tiers scoped to an API Group
- **API**: `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apiGroups/{groupId}/tiers`
- **Use Cases**:
  - Apply Bronze / Silver / Gold access tiers to an API Group
  - Define per-tier rate limits and approval policies for group-level consumers

---

## Common Setup

All examples require:

1. **Provider Configuration**:
   ```hcl
   terraform {
     required_providers {
       anypoint = {
         source  = "sf.com/mulesoft/anypoint"
         version = "0.1.0"
       }
     }
   }

   provider "anypoint" {
     client_id     = var.anypoint_client_id
     client_secret = var.anypoint_client_secret
     base_url      = var.anypoint_base_url
   }
   ```

2. **Authentication**: Connected App credentials with `manage:apis` scope for the target environments
3. **Base URL**: `https://anypoint.mulesoft.com` (production) or `https://stgx.anypoint.mulesoft.com` (staging)
4. **Environment**: All resources require an existing `environment_id`

## Quick Start

```bash
cd examples/apimanagement/<example-folder>
terraform init
terraform plan
terraform apply
```

## Resource Dependency Map

```
anypoint_managed_omni_gateway  (deploy gateway on private space)
└── anypoint_api_instance      (deploy API on that gateway)
    ├── anypoint_api_policy    (apply security / traffic policies)
    └── anypoint_api_instance_sla_tier  (define consumer access tiers)

anypoint_api_group             (bundle API instances)
└── anypoint_api_group_sla_tier

Datasources (read-only, no dependencies):
  anypoint_managed_omni_gateways  → look up gateway ID by name
  anypoint_managed_omni_gateway   → look up full gateway detail by ID
  anypoint_api_instances         → look up API instance ID by label
```

## Cross-Config Reference Pattern

When the gateway and API instance live in separate Terraform configs, use the datasource to look up the gateway ID:

```hcl
# In api_instance config — reference a gateway created elsewhere
data "anypoint_managed_omni_gateways" "all" {
  environment_id = var.environment_id
}

locals {
  gateway_id = one([
    for gw in data.anypoint_managed_omni_gateways.all.gateways :
    gw.id if gw.name == "my-gateway-name"
  ])
}

resource "anypoint_api_instance" "api" {
  environment_id = var.environment_id
  gateway_id     = local.gateway_id
  technology     = "omniGateway"
  # ...
}
```

## API Documentation

- [Anypoint API Manager Documentation](https://docs.mulesoft.com/api-manager/)
- [API Policy Reference](https://docs.mulesoft.com/api-manager/2.x/policies)
- [SLA Tiers Overview](https://docs.mulesoft.com/api-manager/2.x/manage-client-apps-latest-task)
- [Omni Gateway Documentation](https://docs.mulesoft.com/gateway/)
