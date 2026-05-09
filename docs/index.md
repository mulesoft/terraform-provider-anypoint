---
page_title: "Overview"
description: |-
  The Anypoint provider lets you manage MuleSoft Anypoint Platform resources — API instances, API policies, environments, organizations, gateways, secrets, and more — using Terraform.
---

# Anypoint Provider

The `anypoint` provider lets you manage [MuleSoft Anypoint Platform](https://anypoint.mulesoft.com) resources using Terraform — API instances, policies, environments, organizations, CloudHub 2.0 private spaces, Omni Gateway deployments, secrets, and Agent / MCP tools.

## Example Usage

```hcl
terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 0.0.4"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = "https://anypoint.mulesoft.com"
}
```

## Authentication

The provider authenticates against Anypoint Platform using a [Connected App](https://docs.mulesoft.com/access-management/connected-apps-overview) with the `client_credentials` grant type. Create a Connected App in your root org with the scopes required for the resources you intend to manage, then provide its `client_id` / `client_secret` to the provider.

## Schema

### Required

- `client_id` (String) – Connected App client ID.
- `client_secret` (String, Sensitive) – Connected App client secret.

### Optional

- `base_url` (String) – Anypoint Platform base URL. Defaults to `https://anypoint.mulesoft.com`. Override for EU (`https://eu1.anypoint.mulesoft.com`), GovCloud, or staging environments.

## Resources by Category

All resources provided by `terraform-provider-anypoint`, grouped by subcategory.

### Access Management

| Resource | Description |
|----------|-------------|
| [anypoint_connected_app_scopes](resources/anypoint_connected_app_scopes.md) | Manages scopes for an Anypoint Connected Application using user authentication. |
| [anypoint_environment](resources/anypoint_environment.md) | Manages an Anypoint Platform environment. |
| [anypoint_organization](resources/anypoint_organization.md) | Creates and manages an Anypoint Platform organization. |
| [anypoint_team](resources/anypoint_team.md) | Manages an Anypoint Platform team. |

### API Management

| Resource | Description |
|----------|-------------|
| [anypoint_api_instance](resources/anypoint_api_instance.md) | Manages an API instance in Anypoint API Manager. |
| [anypoint_api_instance_sla_tier](resources/anypoint_api_instance_sla_tier.md) | Manages an SLA tier for an API instance in Anypoint API Manager. |
| [anypoint_api_policy](resources/anypoint_api_policy.md) | Manages a policy applied to an API instance. Supports both known policies (via `policy_type`) and custom policies (via `group_id` + `asset_id`). |
| [anypoint_managed_omni_gateway](resources/anypoint_managed_omni_gateway.md) | Manages a CloudHub 2.0 Managed Omni Gateway instance in Anypoint Platform. |

### Agents & Tools

| Resource | Description |
|----------|-------------|
| [anypoint_agent_instance](resources/anypoint_agent_instance.md) | Manages an Agent instance in Anypoint API Manager. An Agent instance represents an Agent specification deployed to an Omni Gateway target with routing rules and upstream backends. |
| [anypoint_mcp_server](resources/anypoint_mcp_server.md) | Manages an MCP server in Anypoint API Manager. An MCP server represents an MCP server specification deployed to an Omni Gateway target. |

### CloudHub 2.0

| Resource | Description |
|----------|-------------|
| [anypoint_private_space_config](resources/anypoint_private_space_config.md) | Manages an Anypoint Private Space together with its network configuration and firewall rules as a single resource. |
| [anypoint_private_space_association](resources/anypoint_private_space_association.md) | Creates and manages associations between a CloudHub 2.0 private space and environments. |
| [anypoint_private_space_upgrade](resources/anypoint_private_space_upgrade.md) | Schedules an upgrade for a CloudHub 2.0 private space. |
| [anypoint_privatespace_advanced_config](resources/anypoint_privatespace_advanced_config.md) | Manages advanced configuration for an Anypoint Private Space. |
| [anypoint_tls_context](resources/anypoint_tls_context.md) | Manages a CloudHub 2.0 TLS Context with support for both PEM and JKS keystores. |
| [anypoint_vpn_connection](resources/anypoint_vpn_connection.md) | Creates a VPN connection in a CloudHub 2.0 private space. |

### Secrets Management

| Resource | Description |
|----------|-------------|
| [anypoint_secret_group](resources/anypoint_secret_group.md) | Manages a secret group in Anypoint Secrets Manager. |
| [anypoint_secret_group_certificate](resources/anypoint_secret_group_certificate.md) | Manages a certificate within a secret group. Supports PEM, JKS, PKCS12, and JCEKS formats. |
| [anypoint_secret_group_certificate_pinset](resources/anypoint_secret_group_certificate_pinset.md) | Manages a certificate pinset within a secret group for certificate pinning validation. |
| [anypoint_secret_group_keystore](resources/anypoint_secret_group_keystore.md) | Manages a keystore within a secret group. Supports PEM, JKS, PKCS12, and JCEKS formats. |
| [anypoint_secret_group_shared_secret](resources/anypoint_secret_group_shared_secret.md) | Manages a shared secret within a secret group. Supports UsernamePassword, S3Credential, SymmetricKey, and Blob types. |
| [anypoint_secret_group_tls_context](resources/anypoint_secret_group_tls_context.md) | Manages an Omni Gateway TLS context within a secret group in Anypoint Secrets Manager. |
| [anypoint_secret_group_truststore](resources/anypoint_secret_group_truststore.md) | Manages a truststore within a secret group. Supports PEM, JKS, PKCS12, and JCEKS formats. |

### API Policies — Inbound · Security & Authentication

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_access_block](resources/anypoint_api_policy_access_block.md) | Manages an Access Block policy on an Anypoint API instance. |
| [anypoint_api_policy_client_id_enforcement](resources/anypoint_api_policy_client_id_enforcement.md) | Manages a Client ID Enforcement policy on an Anypoint API instance. |
| [anypoint_api_policy_cors](resources/anypoint_api_policy_cors.md) | Manages a CORS policy on an Anypoint API instance. |
| [anypoint_api_policy_external_oauth2_access_token_enforcement](resources/anypoint_api_policy_external_oauth2_access_token_enforcement.md) | Manages an External OAuth 2.0 Access Token Enforcement policy on an Anypoint API instance. (mule4 only) |
| [anypoint_api_policy_http_basic_authentication](resources/anypoint_api_policy_http_basic_authentication.md) | Manages an HTTP Basic Authentication policy on an Anypoint API instance. |
| [anypoint_api_policy_ip_allowlist](resources/anypoint_api_policy_ip_allowlist.md) | Manages an IP Allowlist policy on an Anypoint API instance. |
| [anypoint_api_policy_ip_blocklist](resources/anypoint_api_policy_ip_blocklist.md) | Manages an IP Blocklist policy on an Anypoint API instance. |
| [anypoint_api_policy_jwt_validation](resources/anypoint_api_policy_jwt_validation.md) | Manages a JWT Validation policy on an Anypoint API instance. |
| [anypoint_api_policy_ldap_authentication](resources/anypoint_api_policy_ldap_authentication.md) | Manages an LDAP Authentication policy on an Anypoint API instance. |
| [anypoint_api_policy_oauth2_token_introspection](resources/anypoint_api_policy_oauth2_token_introspection.md) | Manages an OAuth 2.0 Token Introspection policy on an Anypoint API instance. |

### API Policies — Inbound · Traffic Management

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_health_check](resources/anypoint_api_policy_health_check.md) | Manages a Health Check policy on an Anypoint API instance. |
| [anypoint_api_policy_http_caching](resources/anypoint_api_policy_http_caching.md) | Manages an HTTP Caching policy on an Anypoint API instance. |
| [anypoint_api_policy_rate_limiting](resources/anypoint_api_policy_rate_limiting.md) | Manages a Rate Limiting policy on an Anypoint API instance. |
| [anypoint_api_policy_rate_limiting_sla_based](resources/anypoint_api_policy_rate_limiting_sla_based.md) | Manages a Rate Limiting SLA-Based policy on an Anypoint API instance. |
| [anypoint_api_policy_response_timeout](resources/anypoint_api_policy_response_timeout.md) | Manages a Response Timeout policy on an Anypoint API instance. |
| [anypoint_api_policy_spike_control](resources/anypoint_api_policy_spike_control.md) | Manages a Spike Control policy on an Anypoint API instance. |
| [anypoint_api_policy_spec_validation](resources/anypoint_api_policy_spec_validation.md) | Manages a Spec Validation policy on an Anypoint API instance. |
| [anypoint_api_policy_sse_logging](resources/anypoint_api_policy_sse_logging.md) | Manages an SSE Logging policy on an Anypoint API instance. |
| [anypoint_api_policy_stream_idle_timeout](resources/anypoint_api_policy_stream_idle_timeout.md) | Manages a Stream Idle Timeout policy on an Anypoint API instance. |

### API Policies — Inbound · Threat Protection

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_injection_protection](resources/anypoint_api_policy_injection_protection.md) | Manages an Injection Protection policy on an Anypoint API instance. |
| [anypoint_api_policy_json_threat_protection](resources/anypoint_api_policy_json_threat_protection.md) | Manages a JSON Threat Protection policy on an Anypoint API instance. |
| [anypoint_api_policy_xml_threat_protection](resources/anypoint_api_policy_xml_threat_protection.md) | Manages an XML Threat Protection policy on an Anypoint API instance. (mule4 only) |

### API Policies — Inbound · Transformation & Logging

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_body_transformation](resources/anypoint_api_policy_body_transformation.md) | Manages a Body Transformation policy on an Anypoint API instance. |
| [anypoint_api_policy_dataweave_body_transformation](resources/anypoint_api_policy_dataweave_body_transformation.md) | Manages a DataWeave Body Transformation policy on an Anypoint API instance. |
| [anypoint_api_policy_dataweave_headers_transformation](resources/anypoint_api_policy_dataweave_headers_transformation.md) | Manages a DataWeave Headers Transformation policy on an Anypoint API instance. |
| [anypoint_api_policy_dataweave_request_filter](resources/anypoint_api_policy_dataweave_request_filter.md) | Manages a DataWeave Request Filter policy on an Anypoint API instance. |
| [anypoint_api_policy_header_injection](resources/anypoint_api_policy_header_injection.md) | Manages a Header Injection policy on an Anypoint API instance. |
| [anypoint_api_policy_header_removal](resources/anypoint_api_policy_header_removal.md) | Manages a Header Removal policy on an Anypoint API instance. |
| [anypoint_api_policy_header_transformation](resources/anypoint_api_policy_header_transformation.md) | Manages a Header Transformation policy on an Anypoint API instance. |
| [anypoint_api_policy_message_logging](resources/anypoint_api_policy_message_logging.md) | Manages a Message Logging policy on an Anypoint API instance. |
| [anypoint_api_policy_script_evaluation_transformation](resources/anypoint_api_policy_script_evaluation_transformation.md) | Manages a Script Evaluation Transformation policy on an Anypoint API instance. |
| [anypoint_api_policy_tracing](resources/anypoint_api_policy_tracing.md) | Manages a Tracing policy on an Anypoint API instance. |

### API Policies — Inbound · Observability & Extensions

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_agent_connection_telemetry](resources/anypoint_api_policy_agent_connection_telemetry.md) | Manages an Agent Connection Telemetry policy on an Anypoint API instance. |
| [anypoint_api_policy_native_ext_authz](resources/anypoint_api_policy_native_ext_authz.md) | Manages a Native External Authorization policy on an Anypoint API instance. |
| [anypoint_api_policy_native_ext_proc](resources/anypoint_api_policy_native_ext_proc.md) | Manages a Native External Processing policy on an Anypoint API instance. |

### API Policies — Outbound

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_circuit_breaker](resources/anypoint_api_policy_circuit_breaker.md) | Manages a Circuit Breaker policy on an Anypoint API instance. |
| [anypoint_api_policy_credential_injection_basic_auth](resources/anypoint_api_policy_credential_injection_basic_auth.md) | Manages a Credential Injection (Basic Auth) policy on an Anypoint API instance. |
| [anypoint_api_policy_credential_injection_oauth2](resources/anypoint_api_policy_credential_injection_oauth2.md) | Manages a Credential Injection (OAuth2) policy on an Anypoint API instance. |
| [anypoint_api_policy_credential_injection_oauth2_obo](resources/anypoint_api_policy_credential_injection_oauth2_obo.md) | Manages a Credential Injection (OAuth2 On-Behalf-Of) policy on an Anypoint API instance. |
| [anypoint_api_policy_idle_timeout](resources/anypoint_api_policy_idle_timeout.md) | Manages an Idle Timeout policy on an Anypoint API instance. |
| [anypoint_api_policy_intask_authentication_policy](resources/anypoint_api_policy_intask_authentication_policy.md) | Manages an InTask Authentication policy on an Anypoint API instance. |
| [anypoint_api_policy_intask_authorization_code_policy](resources/anypoint_api_policy_intask_authorization_code_policy.md) | Manages an InTask Authorization Code policy on an Anypoint API instance. |
| [anypoint_api_policy_message_logging_outbound](resources/anypoint_api_policy_message_logging_outbound.md) | Manages a Message Logging (Outbound) policy on an Anypoint API instance. |
| [anypoint_api_policy_native_aws_lambda](resources/anypoint_api_policy_native_aws_lambda.md) | Manages a Native AWS Lambda policy on an Anypoint API instance. |

### API Policies — MCP (Model Context Protocol)

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_mcp_access_control](resources/anypoint_api_policy_mcp_access_control.md) | Manages an MCP Access Control policy on an Anypoint API instance. |
| [anypoint_api_policy_mcp_global_access_policy](resources/anypoint_api_policy_mcp_global_access_policy.md) | Manages an MCP Global Access Policy on an Anypoint API instance. |
| [anypoint_api_policy_mcp_pii_detector](resources/anypoint_api_policy_mcp_pii_detector.md) | Manages an MCP PII Detector policy on an Anypoint API instance. |
| [anypoint_api_policy_mcp_schema_validation](resources/anypoint_api_policy_mcp_schema_validation.md) | Manages an MCP Schema Validation policy on an Anypoint API instance. |
| [anypoint_api_policy_mcp_support](resources/anypoint_api_policy_mcp_support.md) | Manages an MCP Support policy on an Anypoint API instance. |
| [anypoint_api_policy_mcp_tool_mapping](resources/anypoint_api_policy_mcp_tool_mapping.md) | Manages an MCP Tool Mapping policy on an Anypoint API instance. |
| [anypoint_api_policy_mcp_transcoding_router](resources/anypoint_api_policy_mcp_transcoding_router.md) | Manages an MCP Transcoding Router policy on an Anypoint API instance. |

### API Policies — LLM Gateway

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_bedrock_llm_provider_policy](resources/anypoint_api_policy_bedrock_llm_provider_policy.md) | Manages a Bedrock LLM Provider policy on an Anypoint API instance. (outbound) |
| [anypoint_api_policy_gemini_llm_provider_policy](resources/anypoint_api_policy_gemini_llm_provider_policy.md) | Manages a Gemini LLM Provider policy on an Anypoint API instance. (outbound) |
| [anypoint_api_policy_gemini_transcoding_policy](resources/anypoint_api_policy_gemini_transcoding_policy.md) | Manages a Gemini Transcoding policy on an Anypoint API instance. (outbound) |
| [anypoint_api_policy_llm_gw_core_policy](resources/anypoint_api_policy_llm_gw_core_policy.md) | Manages an LLM Gateway Core Policy on an Anypoint API instance. |
| [anypoint_api_policy_llm_proxy_core](resources/anypoint_api_policy_llm_proxy_core.md) | Manages an LLM Proxy Core policy on an Anypoint API instance. |
| [anypoint_api_policy_llm_proxy_core_policy](resources/anypoint_api_policy_llm_proxy_core_policy.md) | Manages an LLM Proxy Core Policy on an Anypoint API instance. |
| [anypoint_api_policy_model_based_routing](resources/anypoint_api_policy_model_based_routing.md) | Manages a Model-Based Routing policy on an Anypoint API instance. |
| [anypoint_api_policy_openai_transcoding_policy](resources/anypoint_api_policy_openai_transcoding_policy.md) | Manages an OpenAI Transcoding policy on an Anypoint API instance. (outbound) |
| [anypoint_api_policy_semantic_prompt_guard_policy_openai](resources/anypoint_api_policy_semantic_prompt_guard_policy_openai.md) | Manages a Semantic Prompt Guard (OpenAI) policy on an Anypoint API instance. |
| [anypoint_api_policy_semantic_routing_policy_huggingface](resources/anypoint_api_policy_semantic_routing_policy_huggingface.md) | Manages a Semantic Routing (HuggingFace) policy on an Anypoint API instance. |

### API Policies — A2A (Agent-to-Agent)

| Resource | Description |
|----------|-------------|
| [anypoint_api_policy_a2a_agent_card](resources/anypoint_api_policy_a2a_agent_card.md) | Manages an A2A Agent Card policy on an Anypoint API instance. |
| [anypoint_api_policy_a2a_pii_detector](resources/anypoint_api_policy_a2a_pii_detector.md) | Manages an A2A PII Detector policy on an Anypoint API instance. |
| [anypoint_api_policy_a2a_prompt_decorator](resources/anypoint_api_policy_a2a_prompt_decorator.md) | Manages an A2A Prompt Decorator policy on an Anypoint API instance. |
| [anypoint_api_policy_a2a_schema_validation](resources/anypoint_api_policy_a2a_schema_validation.md) | Manages an A2A Schema Validation policy on an Anypoint API instance. |
| [anypoint_api_policy_a2a_token_rate_limit](resources/anypoint_api_policy_a2a_token_rate_limit.md) | Manages an A2A Token Rate Limit policy on an Anypoint API instance. |
