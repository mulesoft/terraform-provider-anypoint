# Real-World Use Cases

This directory contains focused, real-world Terraform configurations that demonstrate how resources work together in a typical Anypoint Platform deployment scenario. Unlike the comprehensive end-to-end examples, each file here targets a specific practical pattern.

## Files

| File | Description |
|------|-------------|
| `secretsgroup_omnigateway.tf` | Provision a Secret Group (keystore + truststore + TLS context) and a Managed Omni Gateway with a custom public URL and ingress security settings |
| `api_instance.tf` | Deploy a OmniGateway-backed API instance with weighted read/write routing, referencing the gateway and TLS context from datasources |

> **Note:** These two files share state in the same Terraform root module. `api_instance.tf` depends on resources defined in `secretsgroup_omnigateway.tf` (specifically `anypoint_secret_group.main` and `anypoint_secret_group_tls_context.omni`).

---

## Scenario: Secure API Proxy with TLS and Weighted Routing

This configuration models the following real-world pattern:

```
[Secrets Management]
  anypoint_secret_group          "real-world-example-secrets"
  ├── anypoint_secret_group_keystore   "tls-keystore"     (PEM cert + key)
  ├── anypoint_secret_group_truststore "ca-truststore"    (PEM CA chain)
  └── anypoint_secret_group_tls_context        "omni-tls-context" (h2 + http/1.1)
          │
          ▼
[Managed Omni Gateway]
  anypoint_managed_omni_gateway   "real-world-example-gateway"
  └── ingress: custom public URL, SSL session forwarding, last-mile security
          │
          ▼  (looked up via datasource in api_instance.tf)
[API Instance]
  anypoint_api_instance          "payments-api"
  ├── routing: read  (GET)        → 90% primary + 10% secondary upstream
  └── routing: write (POST|PUT|PATCH|DELETE /api/*) → 100% primary upstream
```

### Key Patterns Demonstrated

**1. Datasource-based cross-resource references**

`api_instance.tf` does not hard-code the gateway ID or TLS context ID. Instead it uses datasources and `locals` to look them up by name at plan time:

```hcl
data "anypoint_managed_omni_gateways" "all" {
  environment_id = var.env_id
}

data "anypoint_secret_group_tls_contexts" "main" {
  environment_id  = var.env_id
  secret_group_id = anypoint_secret_group.main.id
}

locals {
  gateway_id = one([
    for gw in data.anypoint_managed_omni_gateways.all.gateways :
    gw.id if gw.name == "real-world-example-gateway"
  ])

  omni_tls_context_id = one([
    for tls in data.anypoint_secret_group_tls_contexts.main.tls_contexts :
    tls.id if tls.name == "omni-tls-context"
  ])
}
```

This pattern lets `api_instance.tf` remain reusable regardless of whether the gateway was created in the same config or already existed.

**2. Custom public URL on Omni Gateway**

The gateway overrides the auto-derived ingress URL with a user-supplied domain:

```hcl
resource "anypoint_managed_omni_gateway" "complete" {
  ingress = {
    public_url          = "https://example.mulesoft.com/"
    forward_ssl_session = true
    last_mile_security  = true
  }
}
```

Set `public_url` to `""` (or omit the `ingress` block entirely) to fall back to the auto-derived URL from the target domain.

**3. TLS context referenced from SSL Context ID**

The `ssl_context_id` for the API instance endpoint is constructed by combining the secret group ID with the TLS context ID:

```hcl
endpoint = {
  ssl_context_id = "${anypoint_secret_group.main.id}/${local.omni_tls_context_id}"
}
```

**4. Weighted traffic splitting**

Read traffic is split between a primary and a secondary upstream (e.g. canary or blue-green), while write traffic always goes to primary:

```hcl
routing = [
  {
    label = "read-traffic"
    rules = { methods = "GET" }
    upstreams = [
      { weight = var.upstream_primary_weight,       uri = var.upstream_primary_uri,   label = "primary" },
      { weight = 100 - var.upstream_primary_weight, uri = var.upstream_secondary_uri, label = "secondary" }
    ]
  },
  {
    label = "write-traffic"
    rules = { methods = "POST|PUT|PATCH|DELETE", path = "/api/*" }
    upstreams = [
      { weight = 100, uri = var.upstream_primary_uri, label = "primary" }
    ]
  }
]
```

---

## Prerequisites

- PEM certificate files must be present before `terraform apply`:
  - `../certs/cert.pem` — server certificate
  - `../certs/key.pem` — private key
  - `../certs/truststore.pem` — CA / trust chain

- A private space target must already exist. Supply its ID via `var.target_id`.

---

## Variables

| Variable | Defined in | Description | Default |
|----------|-----------|-------------|---------|
| `anypoint_client_id` | `secretsgroup_omnigateway.tf` | Connected App client ID | — |
| `anypoint_client_secret` | `secretsgroup_omnigateway.tf` | Connected App client secret | — |
| `anypoint_base_url` | `secretsgroup_omnigateway.tf` | Anypoint control-plane URL | stgx |
| `environment_id` | `secretsgroup_omnigateway.tf` | Environment for gateway + secrets | — |
| `target_id` | `secretsgroup_omnigateway.tf` | Private space / target ID for the gateway | — |
| `organization_id` | `api_instance.tf` | Organization for the API instance | — |
| `env_id` | `api_instance.tf` | Environment for the API instance (same as `environment_id`) | — |
| `api_asset_id` | `api_instance.tf` | Exchange asset ID for the API spec | `api-test` |
| `api_asset_version` | `api_instance.tf` | Exchange asset version | `1.0.0` |
| `upstream_primary_uri` | `api_instance.tf` | Primary backend URI | — |
| `upstream_secondary_uri` | `api_instance.tf` | Secondary backend URI (canary/blue-green) | — |
| `upstream_primary_weight` | `api_instance.tf` | Traffic % routed to primary (0–100) | `90` |

---

## Quick Start

```bash
# Ensure cert files exist
ls ../certs/cert.pem ../certs/key.pem ../certs/truststore.pem

terraform init
terraform plan
terraform apply
```

---

## Outputs (from `secretsgroup_omnigateway.tf`)

| Output | Description |
|--------|-------------|
| `complete_gateway_id` | ID of the created Omni Gateway |
| `complete_gateway_public_url` | Ingress public URL (custom or auto-derived) |
| `complete_gateway_internal_url` | Ingress internal URL |
| `complete_gateway_status` | Runtime status of the gateway |

---

## See Also

- [E2E API Management](../e2e-apimanagement/README.md) — full stack including policies, SLA tiers, alerts, and promotion
- [API Management Examples](../apimanagement/README.md) — individual resource examples
- [Secrets Management Examples](../secretsmanagement/README.md) — standalone secrets resource examples
