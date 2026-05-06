# End-to-End API Management Examples

This directory contains comprehensive end-to-end examples that provision a complete Anypoint Platform API management stack using Terraform — from infrastructure through to runtime policies.

## Files

| File | Description |
|------|-------------|
| `api_instance_flexgateway_secretsmanager_example.tf` | Full FlexGateway stack: Secrets Management → Flex Gateway → API Instance → 33+ inbound policies → SLA Tier |
| `api_instance_mule4_example.tf` | Mule4 runtime stack: API Instance + 17 inbound policies (including Mule4-exclusive ones) |
| `variables.tf` | All input variables shared across both examples |
| `outputs.tf` | Outputs for the FlexGateway example (gateway URLs, API instance ID, policy IDs, etc.) |
| `terraform.tfvars.example` | Template for your credentials and IDs — copy to `terraform.tfvars` |
| `MULE4_SUPPORT.md` | Technical notes on Mule4 vs FlexGateway schema differences |

---

## Example 1: FlexGateway + Secrets Manager (`api_instance_flexgateway_secretsmanager_example.tf`)

A complete provisioning flow across 7 steps.

### Provisioning Steps

```
Step 1  (commented)   anypoint_private_space
Step 2  (commented)   anypoint_private_network
Step 3  (commented)   anypoint_vpn_connection
Step 3b (active)      Secrets Management: secret group → keystore → truststore → TLS context
Step 4  (active)      anypoint_managed_flexgateway
Step 5  (active)      anypoint_api_instance  (technology = "flexGateway", weighted routing)
Step 6  (active)      33 inbound API policies
Step 6b (commented)   7 outbound API policies (require upstream_id)
Step 6c (active)      anypoint_api_instance_sla_tier
```

Steps 1–3 (private space, network, VPN) are commented out so you can run the example against an existing environment without creating infrastructure. Uncomment them for a fully greenfield deployment.

### Resources Created

#### Secrets Management
| Resource | Name in example |
|----------|----------------|
| `anypoint_secret_group` | `main` |
| `anypoint_secret_group_keystore` | `tls` (PEM type) |
| `anypoint_secret_group_truststore` | `ca` (PEM type) |
| `anypoint_flex_tls_context` | `flex` (ALPN: h2 + http/1.1) |

#### Gateway & API
| Resource | Description |
|----------|-------------|
| `anypoint_managed_flexgateway` | Flex Gateway on private space target |
| `anypoint_api_instance` | FlexGateway-backed proxy with TLS context and weighted read/write routing |

#### Inbound Policies (33 total, applied in order)

| # | Resource | Label | Notes |
|---|----------|-------|-------|
| 1 | `anypoint_api_policy_rate_limiting` | `rate-limit-100rpm` | 100 req/min per identifier |
| 2 | `anypoint_api_policy_spike_control` | `spike-1rps` | 1 req/s, queue 5 |
| 3 | `anypoint_api_policy_rate_limiting_sla_based` | `sla-rate-limit` | **disabled** |
| 4 | `anypoint_api_policy_client_id_enforcement` | `client-id-check` | Custom header expressions |
| 5 | `anypoint_api_policy_jwt_validation` | `jwt-rsa` | RSA, JWKS, audience + expiry validation — **disabled** |
| 6 | `anypoint_api_policy_ip_allowlist` | `allow-private-subnets` | Private subnets via x-forwarded-for |
| 7 | `anypoint_api_policy_ip_blocklist` | `block-bad-actors` | |
| 8 | `anypoint_api_policy_json_threat_protection` | `json-limits` | |
| 9 | `anypoint_api_policy_native_ext_authz` | `ext-auth-http` | External authorization server |
| 10 | `anypoint_api_policy_native_ext_proc` | `ext-processor` | gRPC external processor |
| 11 | `anypoint_api_policy_sse_logging` | `sse-logs` | Structured SSE logs |
| 12 | `anypoint_api_policy_cors` | `cors-public` | Open CORS (public resource) |
| 13 | `anypoint_api_policy_message_logging` | `request-logger` | Conditional header logging |
| 14 | `anypoint_api_policy_header_injection` | `inject-headers` | Inbound + outbound header injection |
| 15 | `anypoint_api_policy_header_removal` | `strip-internal-headers` | Remove debug/internal headers |
| 16 | `anypoint_api_policy_http_basic_authentication` | `basic-auth` | **disabled** |
| 17 | `anypoint_api_policy_response_timeout` | `timeout-15s` | 15s upstream response timeout |
| 18 | `anypoint_api_policy_stream_idle_timeout` | `idle-60s` | 60s stream idle timeout |
| 19 | `anypoint_api_policy_health_check` | `health-200` | Periodic health probe |
| 20 | `anypoint_api_policy_http_caching` | `cache-600s` | Distributed 600s cache |
| 21 | `anypoint_api_policy_oauth2_token_introspection` | `oauth2-introspect` | **disabled** |
| 22 | `anypoint_api_policy_access_block` | `block-all` | **disabled** |
| 23 | `anypoint_api_policy_agent_connection_telemetry` | `agent-telemetry` | |
| 24 | `anypoint_api_policy_ldap_authentication` | `ldap-auth` | **disabled** |
| 25 | `anypoint_api_policy_tracing` | `tracing-sp1` | 100% sampling |
| 26 | `anypoint_api_policy_xml_threat_protection` | `xml-threat` | |
| 27 | `anypoint_api_policy_injection_protection` | `injection-protect` | XSS + SQL + custom rules |
| 28 | `anypoint_api_policy_dataweave_request_filter` | `dw-filter` | DataWeave client-id gate |
| 29 | `anypoint_api_policy_body_transformation` | `body-transform` | |
| 30 | `anypoint_api_policy_header_transformation` | `header-transform` | |
| 31 | `anypoint_api_policy_dataweave_body_transformation` | `dw-body-transform` | DataWeave payload passthrough |
| 32 | `anypoint_api_policy_dataweave_headers_transformation` | `dw-headers-transform` | Inject client_id/secret from payload |
| 33 | `anypoint_api_policy_script_evaluation_transformation` | `script-eval-transform` | |

#### Outbound Policies (7, commented out — require `upstream_id`)

| # | Resource | Notes |
|---|----------|-------|
| 34 | `anypoint_api_policy_message_logging_outbound` | Log upstream responses |
| 35 | `anypoint_api_policy_intask_authorization_code_policy` | OAuth 2.0 auth code outbound |
| 36 | `anypoint_api_policy_credential_injection_oauth2` | Inject OAuth2 token to upstream |
| 37 | `anypoint_api_policy_credential_injection_basic_auth` | Inject Basic Auth to upstream |
| 38 | `anypoint_api_policy_idle_timeout` | Upstream idle connection timeout |
| 39 | `anypoint_api_policy_circuit_breaker` | Count-based circuit breaker |
| 40 | `anypoint_api_policy_native_aws_lambda` | Invoke AWS Lambda per request |

To enable outbound policies, set `var.upstream_id` to the routing upstream UUID for the API instance. Retrieve it from the API Manager UI or via `GET /xapi/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/upstreams`. For `anypoint_mcp_server`, the `upstream_id` computed attribute gives the server-assigned upstream ID directly — reference it as `anypoint_mcp_server.example.upstream_id`.

#### Other Resources
| Resource | Description |
|----------|-------------|
| `anypoint_api_instance_sla_tier` | `Tier1` — 10 req/min + 5 req/s, manual approval |

---

## Example 2: Mule4 Runtime (`api_instance_mule4_example.tf`)

Demonstrates a `technology = "mule4"` API instance with a direct implementation URI and the full set of Mule4-compatible policies.

### Key Differences from FlexGateway

| Aspect | FlexGateway | Mule4 |
|--------|-------------|-------|
| `endpoint.base_path` | Required | Not used |
| `endpoint.uri` | Not used | Required (implementation URL) |
| `endpoint.ssl_context_id` | Supported | Not supported |
| `gateway_id` | Required | Not needed |
| `routing` block | Required | Not needed |
| Autodiscovery | Not needed | `product_version` output used |

### Policies Applied

| # | Resource | Notes |
|---|----------|-------|
| 1 | `anypoint_api_policy_rate_limiting` | |
| 2 | `anypoint_api_policy_spike_control` | |
| 3 | `anypoint_api_policy_rate_limiting_sla_based` | **disabled** |
| 4 | `anypoint_api_policy_client_id_enforcement` | |
| 5 | `anypoint_api_policy_jwt_validation` | **disabled** |
| 6 | `anypoint_api_policy_ip_allowlist` | |
| 7 | `anypoint_api_policy_ip_blocklist` | |
| 8 | `anypoint_api_policy_json_threat_protection` | |
| 9 | `anypoint_api_policy_xml_threat_protection` | Mule4-only policy |
| 10 | `anypoint_api_policy_cors` | |
| 11 | `anypoint_api_policy_message_logging` | |
| 12 | `anypoint_api_policy_header_injection` | |
| 13 | `anypoint_api_policy_header_removal` | |
| 14 | `anypoint_api_policy_http_basic_authentication` | **disabled** |
| 15 | `anypoint_api_policy_ldap_authentication` | **disabled** |
| 16 | `anypoint_api_policy_http_caching` | |
| 17 | `anypoint_api_policy_external_oauth2_access_token_enforcement` | Mule OAuth Provider — **disabled** |

Additional Mule4-specific OAuth policies are commented out (OpenAM, OpenID Connect, PingFederate token enforcement). These use the `anypoint_api_policy` generic resource as they are not available on FlexGateway.

---

## Quick Start

```bash
# 1. Clone / navigate to this directory
cd examples/e2e-apimanagement

# 2. Set up credentials
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your real values

# 3. Place TLS certificate files (for FlexGateway example)
# The example expects PEM files at:
#   ../certs/cert.pem        – server certificate
#   ../certs/key.pem         – private key
#   ../certs/truststore.pem  – CA / trust chain

# 4. Run
terraform init
terraform plan
terraform apply
```

## Required Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `anypoint_client_id` | Connected App client ID | Yes |
| `anypoint_client_secret` | Connected App client secret | Yes |
| `anypoint_base_url` | Anypoint control-plane URL | Yes |
| `organization_id` | Anypoint organization ID | Yes |
| `environment_id` | Source environment ID | Yes |
| `api_asset_id` | Exchange asset ID for the API spec | Yes |
| `api_asset_version` | Exchange asset version | Yes |
| `api_base_path` | API base path (FlexGateway) | Yes (FlexGateway) |
| `upstream_primary_uri` | Primary backend URI | Yes (FlexGateway) |
| `upstream_secondary_uri` | Secondary backend URI | Yes (FlexGateway) |
| `upstream_primary_weight` | Traffic weight for primary (0–100) | No (default: `90`) |
| `upstream_id` | Routing upstream UUID (for outbound policies) | No (default: `""`) |
| `alert_email` | Email address for alerts | No (default: `admin@example.com`) |

## Required Connected App Scopes

The Connected App must have the following scopes granted for the relevant environments:

| Scope | Required For |
|-------|-------------|
| `manage:apis` | Create/update API instances, apply policies |
| `manage:api_alerts` | Create API alerts |
| `read:secrets` | Read secrets in Secrets Manager |
| `manage:secrets` | Create/update secret groups, keystores, TLS contexts |
| `manage:gateways` | Create/update Flex Gateway instances |

## Outputs

After `terraform apply`, the following outputs are available:

| Output | Description |
|--------|-------------|
| `flex_gateway_id` | ID of the created Flex Gateway |
| `flex_gateway_status` | Gateway runtime status |
| `flex_gateway_public_url` | Public ingress URL (auto-derived or user-provided) |
| `flex_gateway_internal_url` | Internal ingress URL |
| `api_instance_id` | Numeric ID of the API instance in API Manager |
| `api_instance_status` | API instance status |
| `api_base_path` | Configured base path |
| `policy_ids` | Map of `policy_name → policy_id` for all 33 inbound policies |
| `sla_tier_id` | ID of the created SLA tier |
| `alert_id` | ID of the API alert |
| `promoted_api_instance_id` | ID of the promoted API instance in the target environment |
| `promoted_api_instance_status` | Status of the promoted instance |

## Architecture

```
[Secrets Manager]
  anypoint_secret_group
  └── anypoint_secret_group_keystore  (TLS cert)
  └── anypoint_secret_group_truststore (CA chain)
  └── anypoint_flex_tls_context        (TLS policy)
          │
          ▼
[Managed Flex Gateway]
  anypoint_managed_flexgateway
  └── ingress: TLS context ID → public_url + internal_url
          │
          ▼
[API Instance]
  anypoint_api_instance  (technology=flexGateway)
  ├── routing: read traffic  → 90% primary + 10% secondary upstream
  └── routing: write traffic → 100% primary upstream
          │
          ▼
[Policies applied in order]
  1. Rate Limiting     2. Spike Control     3. SLA Rate Limiting
  4. Client ID Enf.   5. JWT Validation    6-7. IP Allow/Block
  8. JSON Threat       9. Ext AuthZ         10. Ext Processing
  11. SSE Logging     12. CORS             13. Message Logging
  14. Header Inject   15. Header Remove    16. Basic Auth
  17. Response Timeout 18. Stream Timeout  19. Health Check
  20. HTTP Caching    21. OAuth2 Introspect 22. Access Block
  23. Telemetry       24. LDAP Auth        25. Tracing
  26. XML Threat      27. Injection Protect 28. DW Request Filter
  29-33. Transformation policies
          │
          ▼
[Lifecycle]
  anypoint_api_instance_sla_tier    (Tier1: 10 req/min, manual approval)
```

## Troubleshooting

### `Error creating policy: 409 conflict (label=...)` on a re-apply

This used to happen because the Anypoint Platform's outbound endpoints are
asymmetric:

| Method | `xapi/v1/.../policies/outbound-policies/{id}` | `api/v1/.../policies/{id}` |
|--------|------------------------------------------------|------------------------------|
| POST   | creates (this is the only place outbound POST works) | n/a    |
| GET    | **404 Not Found** even for policies that exist | 200 OK              |
| PATCH  | **404 Not Found**                              | 200 OK                      |
| DELETE | **404 Not Found**                              | 204 No Content              |
| LIST (no id) | **405 Method Not Allowed** (`Allow: POST`) | 200 OK (returns inbound + outbound) |

The provider previously routed Read/Update/Delete for outbound policies
through the xapi/v1 path, so on every `terraform refresh` (which runs
before `apply`), Read got a 404 and silently called
`resp.State.RemoveResource()`. The next apply then tried to recreate the
policy → 409, because the policy was very much still on the server.

The fix: only `CREATE` for outbound policies uses the dedicated
`xapi/v1/.../outbound-policies` endpoint. Every other CRUD operation goes
through the universal `api/v1/.../policies/{id}` path, which the platform
serves correctly for both inbound and outbound entries.

If you still see a 409 (e.g. you ran an older provider build and ended up
with an orphaned policy), the provider now lists policies via
`api/v1/.../policies`, finds the orphan by asset coordinates + label, and
adopts it into state automatically. If multiple policies match the same
asset+label, recovery refuses to guess; resolve with `terraform import`:

```bash
terraform import anypoint_api_policy_message_logging_outbound.message_logging_outbound \
  ORG_ID/ENV_ID/API_ID/POLICY_ID
```

### `(known after apply)` shown for `order` on every plan

The `order` attribute is computed for outbound policies (the server assigns
it; clients cannot send it). The provider uses `UseStateForUnknown` so
re-plans don't show spurious diffs for `order` when nothing else changed.

## See Also

- [API Management Examples](../apimanagement/README.md) — individual resource examples
- [Secrets Management Examples](../secretsmanagement/) — standalone secrets resource examples
- [Real-World Use Cases](../real-world-use-cases/) — smaller focused scenarios
- [MULE4_SUPPORT.md](./MULE4_SUPPORT.md) — technical notes on Mule4 vs FlexGateway API differences
