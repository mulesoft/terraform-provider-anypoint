###############################################################################
# Outputs
###############################################################################

# ── Private Space Config ─────────────────────────────────────────────────────

# output "private_space_id" {
#   description = "ID of the private space"
#   value       = anypoint_private_space_config.main.id
# }

# output "private_space_status" {
#   description = "Status of the private space"
#   value       = anypoint_private_space_config.main.status
# }

# output "network_region" {
#   description = "Region of the private network"
#   value       = anypoint_private_space_config.main.network.region
# }

# # ── VPN Connection ──────────────────────────────────────────────────────────

# output "vpn_connection_id" {
#   description = "ID of the VPN connection"
#   value       = anypoint_vpn_connection.site_to_site.id
# }

# output "vpn_connection_name" {
#   description = "Name of the VPN connection"
#   value       = anypoint_vpn_connection.site_to_site.name
# }

# ── Flex Gateway ────────────────────────────────────────────────────────────

output "flex_gateway_id" {
  description = "ID of the Flex Gateway"
  value       = anypoint_managed_flexgateway.main.id
}

output "flex_gateway_status" {
  description = "Status of the Flex Gateway"
  value       = anypoint_managed_flexgateway.main.status
}

output "flex_gateway_public_url" {
  description = "Public ingress URL (auto-derived from target domain, or user-provided)"
  value       = anypoint_managed_flexgateway.main.ingress.public_url
}

output "flex_gateway_internal_url" {
  description = "Internal ingress URL (auto-derived from target domain)"
  value       = anypoint_managed_flexgateway.main.ingress.internal_url
}

# ── API Instance ────────────────────────────────────────────────────────────

output "api_instance_id" {
  description = "Numeric ID of the API instance in API Manager"
  value       = anypoint_api_instance.main.id
}

output "api_instance_status" {
  description = "Status of the API instance"
  value       = anypoint_api_instance.main.status
}

output "api_base_path" {
  description = "Base path of the API instance"
  value       = var.api_base_path
}

# ── Policy IDs ──────────────────────────────────────────────────────────────

output "policy_ids" {
  description = "Map of policy name → policy ID"
  value = {
    rate_limiting              = anypoint_api_policy_rate_limiting.rate_limiting.id
    spike_control              = anypoint_api_policy_spike_control.spike_control.id
    rate_limiting_sla          = anypoint_api_policy_rate_limiting_sla_based.rate_limiting_sla.id
    client_id_enforcement      = anypoint_api_policy_client_id_enforcement.client_id_enforcement.id
    jwt_validation             = anypoint_api_policy_jwt_validation.jwt_validation.id
    ip_allowlist               = anypoint_api_policy_ip_allowlist.ip_allowlist.id
    ip_blocklist               = anypoint_api_policy_ip_blocklist.ip_blocklist.id
    json_threat_protection     = anypoint_api_policy_json_threat_protection.json_threat_protection.id
    ext_authz                  = anypoint_api_policy_native_ext_authz.ext_authz.id
    ext_proc                   = anypoint_api_policy_native_ext_proc.ext_proc.id
    sse_logging                = anypoint_api_policy_sse_logging.sse_logging.id
    cors                       = anypoint_api_policy_cors.cors.id
    message_logging            = anypoint_api_policy_message_logging.message_logging.id
    header_injection           = anypoint_api_policy_header_injection.header_injection.id
    header_removal             = anypoint_api_policy_header_removal.header_removal.id
    http_basic_auth            = anypoint_api_policy_http_basic_authentication.http_basic_auth.id
    response_timeout           = anypoint_api_policy_response_timeout.response_timeout.id
    stream_idle_timeout        = anypoint_api_policy_stream_idle_timeout.stream_idle_timeout.id
    health_check               = anypoint_api_policy_health_check.health_check.id
    http_caching               = anypoint_api_policy_http_caching.http_caching.id
    oauth2_introspection       = anypoint_api_policy_oauth2_token_introspection.oauth2_introspection.id
    access_block               = anypoint_api_policy_access_block.access_block.id
    agent_connection_telemetry = anypoint_api_policy_agent_connection_telemetry.agent_connection_telemetry.id
    ldap_auth                  = anypoint_api_policy_ldap_authentication.ldap_auth.id
    tracing                    = anypoint_api_policy_tracing.tracing.id
    xml_threat_protection      = anypoint_api_policy_xml_threat_protection.xml_threat_protection.id
    injection_protection       = anypoint_api_policy_injection_protection.injection_protection.id
    dataweave_request_filter          = anypoint_api_policy_dataweave_request_filter.dataweave_request_filter.id
    body_transformation               = anypoint_api_policy_body_transformation.body_transformation.id
    header_transformation             = anypoint_api_policy_header_transformation.header_transformation.id
    dataweave_body_transformation     = anypoint_api_policy_dataweave_body_transformation.dataweave_body_transformation.id
    dataweave_headers_transformation  = anypoint_api_policy_dataweave_headers_transformation.dataweave_headers_transformation.id
    script_evaluation_transformation  = anypoint_api_policy_script_evaluation_transformation.script_evaluation_transformation.id
  }
}

# ── SLA Tier ─────────────────────────────────────────────────────────────────

output "sla_tier_id" {
  description = "ID of the created SLA tier"
  value       = anypoint_api_instance_sla_tier.tier1.id
}

# ── Alert ────────────────────────────────────────────────────────────────────

output "alert_id" {
  description = "ID of the created API alert"
  value       = anypoint_api_instance_alert.request_count.id
}

# ── Promotion ─────────────────────────────────────────────────────────────────

output "promoted_api_instance_id" {
  description = "ID of the promoted API instance in the target environment"
  value       = anypoint_api_instance_promotion.to_production.id
}

output "promoted_api_instance_status" {
  description = "Status of the promoted API instance"
  value       = anypoint_api_instance_promotion.to_production.status
}
