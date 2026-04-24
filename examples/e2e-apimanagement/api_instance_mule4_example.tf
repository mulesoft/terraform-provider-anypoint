# # Example: API Instance with Mule4 Technology + Policies
# #
# # This example demonstrates how to create an API instance using Mule4 technology
# # and apply the full set of Mule4-compatible policies.
# #
# # Key differences from FlexGateway:
# # - technology = "mule4"
# # - endpoint.uri is used (direct implementation URL) instead of endpoint.base_path
# # - No gateway_id or deployment block needed
# # - No routing or TLS context (managed by the Mule runtime)
# # - Mule4-exclusive policies (XML Threat Protection, Mule OAuth Provider, etc.) are available

# resource "anypoint_api_instance" "mule4_api" {
#   environment_id  = var.environment_id
#   technology      = "mule4"
#   instance_label  = "mule4-api-instance"
#   approval_method = null  # or "manual" / "automatic"

#   spec = {
#     asset_id = var.api_asset_id
#     group_id = var.organization_id
#     version  = var.api_asset_version
#   }

#   endpoint = {
#     deployment_type  = "HY"      # HY (Hybrid), CH (CloudHub), or RF (Runtime Fabric)
#     type             = "http"
#     uri              = "http://www.google.com"
#     response_timeout = 30000
#   }
# }

# ###############################################################################
# # Local references for mule4 policies
# ###############################################################################
# locals {
#   mule4_org_id = var.organization_id
#   mule4_env_id = var.environment_id
#   mule4_api_id = anypoint_api_instance.mule4_api.id
# }

# ###############################################################################
# # Policies – applied to the Mule4 API instance
# ###############################################################################

# # ─── 1. Rate Limiting ────────────────────────────────────────────────────────
# resource "anypoint_api_policy_rate_limiting" "mule4_rate_limiting" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "rate-limit-100rpm"
#   order           = 1

#   configuration = {
#     key_selector = "#[attributes.queryParams['identifier']]"
#     rate_limits = [
#       {
#         maximum_requests            = 100
#         time_period_in_milliseconds = 60000
#       }
#     ]
#     expose_headers = true
#     clusterizable  = true
#   }
# }

# # ─── 2. Spike Control ────────────────────────────────────────────────────────
# resource "anypoint_api_policy_spike_control" "mule4_spike_control" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "spike-1rps"
#   order           = 2

#   configuration = {
#     maximum_requests            = 1
#     time_period_in_milliseconds = 1000
#     delay_time_in_millis        = 1000
#     delay_attempts              = 1
#     queuing_limit               = 5
#     expose_headers              = true
#   }
# }

# # ─── 3. Rate Limiting: SLA-based ─────────────────────────────────────────────
# resource "anypoint_api_policy_rate_limiting_sla_based" "mule4_rate_limiting_sla" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "sla-rate-limit"
#   order           = 3
#   disabled        = true

#   configuration = {
#     client_id_expression     = "#[attributes.headers['client_id']]"
#     client_secret_expression = "#[attributes.headers['client_secret']]"
#     expose_headers           = true
#     clusterizable            = true
#   }
# }

# # ─── 4. Client ID Enforcement ────────────────────────────────────────────────
# resource "anypoint_api_policy_client_id_enforcement" "mule4_client_id_enforcement" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "client-id-check"
#   order           = 4

#   configuration = {
#     credentials_origin_has_http_basic_authentication_header = "customExpression"
#     client_id_expression     = "#[attributes.headers['client_id']]"
#     client_secret_expression = "#[attributes.headers['client_secret']]"
#   }
# }

# # ─── 5. JWT Validation ───────────────────────────────────────────────────────
# resource "anypoint_api_policy_jwt_validation" "mule4_jwt_validation" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "jwt-rsa"
#   order           = 5
#   disabled        = true

#   configuration = {
#     jwt_origin                      = "httpBearerAuthenticationHeader"
#     signing_method                  = "rsa"
#     signing_key_length              = 256
#     jwt_key_origin                  = "text"
#     text_key                        = "your-(256|384|512)-bit-secret"
#     custom_key_expression           = "#[authentication.properties['key_to_your_public_pem_certificate']]"
#     jwks_url                        = "http://your-jwks-service.example:80/base/path"
#     jwks_service_time_to_live       = 60
#     jwks_service_connection_timeout = 10000
#     skip_client_id_validation       = false
#     client_id_expression            = "#[vars.claimSet.client_id]"
#     jwt_expression                  = "#[attributes.headers['jwt']]"
#     validate_aud_claim              = true
#     mandatory_aud_claim             = true
#     supported_audiences             = "aud.example.com"
#     mandatory_exp_claim             = true
#     mandatory_nbf_claim             = true
#     validate_custom_claim           = true
#     claims_to_headers               = []
#     mandatory_custom_claims         = []
#     non_mandatory_custom_claims     = []
#   }
# }

# # ─── 6. IP Allowlist ─────────────────────────────────────────────────────────
# resource "anypoint_api_policy_ip_allowlist" "mule4_ip_allowlist" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "allow-private-subnets"
#   order           = 6

#   configuration = {
#     ip_expression = "#[attributes.headers['x-forwarded-for']]"
#     ips           = ["192.168.0.1/16", "10.0.0.1"]
#   }
# }

# # ─── 7. IP Blocklist ─────────────────────────────────────────────────────────
# resource "anypoint_api_policy_ip_blocklist" "mule4_ip_blocklist" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "block-bad-actors"
#   order           = 7

#   configuration = {
#     ip_expression = "#[attributes.headers['x-forwarded-for']]"
#     ips           = ["108.1.12.12", "109.2.2.2"]
#   }
# }

# # ─── 8. JSON Threat Protection ───────────────────────────────────────────────
# resource "anypoint_api_policy_json_threat_protection" "mule4_json_threat_protection" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "json-limits"
#   order           = 8

#   configuration = {
#     max_container_depth          = -1
#     max_string_value_length      = -1
#     max_object_entry_name_length = -1
#     max_object_entry_count       = -1
#     max_array_element_count      = -1
#   }
# }

# # ─── 9. XML Threat Protection ────────────────────────────────────────────────
# # NOTE: xml-threat-protection is supported on Mule4 only (not FlexGateway).
# resource "anypoint_api_policy_xml_threat_protection" "mule4_xml_threat_protection" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "xml-threat"
#   order           = 9

#   configuration = {
#     max_node_depth                  = -1
#     max_attribute_count_per_element = -1
#     max_child_count                 = -1
#     max_text_length                 = -1
#     max_attribute_length            = -1
#     max_comment_length              = -1
#   }
# }

# # ─── 10. Cross-Origin Resource Sharing (CORS) ────────────────────────────────
# resource "anypoint_api_policy_cors" "mule4_cors" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "cors-public"
#   order           = 10

#   configuration = {
#     public_resource     = true
#     support_credentials = false
#     origin_groups       = []
#   }
# }

# # ─── 11. Message Logging ─────────────────────────────────────────────────────
# resource "anypoint_api_policy_message_logging" "mule4_message_logging" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "request-logger"
#   order           = 11

#   configuration = {
#     logging_configuration = [
#       {
#         item_name = "Default configuration -1"
#         item_data = {
#           message        = "#[attributes.headers['id']]"
#           conditional    = "#[attributes.headers['id']==1]"
#           category       = "log1"
#           level          = "INFO"
#           first_section  = true
#           second_section = true
#         }
#       }
#     ]
#   }
# }

# # ─── 12. Header Injection ────────────────────────────────────────────────────
# resource "anypoint_api_policy_header_injection" "mule4_header_injection" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "inject-headers"
#   order           = 12

#   configuration = {
#     inbound_headers = [
#       { key = "header-1", value = "value-1" },
#       { key = "header-2", value = "value-2" }
#     ]
#     outbound_headers = [
#       { key = "header-3", value = "value-3" },
#       { key = "header-4", value = "value-4" }
#     ]
#   }
# }

# # ─── 13. Header Removal ──────────────────────────────────────────────────────
# resource "anypoint_api_policy_header_removal" "mule4_header_removal" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "strip-internal-headers"
#   order           = 13

#   configuration = {
#     inbound_headers  = ["X-Internal-Debug", "X-Temp-Token"]
#     outbound_headers = ["Server", "X-Powered-By"]
#   }
# }

# # ─── 14. Basic Authentication: Simple ────────────────────────────────────────
# resource "anypoint_api_policy_http_basic_authentication" "mule4_http_basic_auth" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "basic-auth"
#   order           = 14
#   disabled        = true

#   configuration = {
#     username = "admin"
#     password = "admin"
#   }
# }

# # ─── 15. Basic Authentication: LDAP ──────────────────────────────────────────
# resource "anypoint_api_policy_ldap_authentication" "mule4_ldap_auth" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "ldap-auth"
#   order           = 15
#   disabled        = true

#   configuration = {
#     ldap_server_url           = "ldap-server.com:9090"
#     ldap_server_user_dn       = "ldapuserdn"
#     ldap_server_user_password = "Admin"
#     ldap_search_base          = "ou=People,dc=acme,dc=org"
#     ldap_search_filter        = "(uid={0})"
#     ldap_search_in_subtree    = true
#   }
# }

# # ─── 16. HTTP Caching ────────────────────────────────────────────────────────
# resource "anypoint_api_policy_http_caching" "mule4_http_caching" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "cache-600s"
#   order           = 16

#   configuration = {
#     http_caching_key       = "#[attributes.requestPath]"
#     max_cache_entries      = 10000
#     ttl                    = 600
#     distributed            = true
#     persist_cache          = true
#     use_http_cache_headers = true
#     invalidation_header    = "invalidation-header"
#     request_expression     = "#[attributes.method == 'GET' or attributes.method == 'HEAD']"
#     response_expression    = "#[[200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501] contains attributes.statusCode]"
#   }
# }

# # ─── 17. OAuth 2.0 Access Token Enforcement (Mule OAuth Provider) ────────────
# resource "anypoint_api_policy_external_oauth2_access_token_enforcement" "mule4_mule_oauth_provider" {
#   organization_id = local.mule4_org_id
#   environment_id  = local.mule4_env_id
#   api_instance_id = local.mule4_api_id
#   label           = "mule-oauth-provider"
#   order           = 17
#   disabled        = true
# configuration = {
#   token_url                = "https://oauth.example.com/token"
#   scope_validation_criteria = "AND"
#   secure_trust_store       = true
#   expose_headers           = true
#   skip_client_id_validation = false
#   authentication_timeout   = 10000
# }
# }

# ###############################################################################
# # Mule4-specific OAuth Token Enforcement Policies
# # These use the generic anypoint_api_policy resource (policy_type shortcut) as
# # they are Mule4-runtime policies not available on Flex Gateway.
# ###############################################################################

# # ─── 18. OpenAM OAuth 2.0 Token Enforcement ──────────────────────────────────
# # resource "anypoint_api_policy" "mule4_openam_oauth2" {
# #   organization_id = local.mule4_org_id
# #   environment_id  = local.mule4_env_id
# #   api_instance_id = local.mule4_api_id
# #   group_id        = "68ef9520-24e9-4cf2-b2f5-620025690913"
# #   asset_id        = "openam-access-token-enforcement"
# #   asset_version   = "1.1.4"
# #   label           = "openam-oauth2"
# #   order           = 18
# #   disabled        = true

# #   configuration_data = jsonencode({
# #     openamTokenIntrospectionEndpoint = "https://openam.example.com/oauth2/introspect"
# #     openamAuthorizationHeader        = "Basic YWRtaW46YWRtaW4="
# #     scopes                           = "read"
# #     expireIn                         = 3600
# #     maxCacheSize                     = 1000
# #   })
# # }

# # # ─── 19. OpenID Connect OAuth 2.0 Access Token Enforcement ───────────────────
# # resource "anypoint_api_policy" "mule4_openid_connect" {
# #   organization_id = local.mule4_org_id
# #   environment_id  = local.mule4_env_id
# #   api_instance_id = local.mule4_api_id
# #   group_id        = "68ef9520-24e9-4cf2-b2f5-620025690913"
# #   asset_id        = "openid-connect-access-token-enforcement"
# #   asset_version   = "1.1.4"
# #   label           = "openid-connect"
# #   order           = 19
# #   disabled        = true

# #   configuration_data = jsonencode({
# #     introspectionEndpoint = "https://idp.example.com/oauth2/introspect"
# #     clientId              = "my-client-id"
# #     clientSecret          = "my-client-secret"
# #     scopes                = "openid profile"
# #     maxCacheSize          = 1000
# #     expireIn              = 3600
# #   })
# # }

# # # ─── 20. PingFederate OAuth 2.0 Token Enforcement ────────────────────────────
# # resource "anypoint_api_policy" "mule4_pingfederate_oauth2" {
# #   organization_id = local.mule4_org_id
# #   environment_id  = local.mule4_env_id
# #   api_instance_id = local.mule4_api_id
# #   group_id        = "68ef9520-24e9-4cf2-b2f5-620025690913"
# #   asset_id        = "pingfederate-access-token-enforcement"
# #   asset_version   = "1.1.4"
# #   label           = "pingfederate-oauth2"
# #   order           = 20
# #   disabled        = true

# #   configuration_data = jsonencode({
# #     tokenIntrospectionEndpoint = "https://pingfederate.example.com/as/introspect.oauth2"
# #     clientId                   = "my-pf-client-id"
# #     clientSecret               = "my-pf-client-secret"
# #     scopes                     = "read"
# #     maxCacheSize               = 1000
# #     expireIn                   = 3600
# #   })
# # }

# ###############################################################################
# # Outputs
# ###############################################################################

# output "mule4_api_instance_id" {
#   value       = anypoint_api_instance.mule4_api.id
#   description = "The ID of the created Mule4 API instance"
# }

# output "mule4_api_autodiscovery_name" {
#   value       = anypoint_api_instance.mule4_api.product_version
#   description = "Use this value in your Mule4 application's autodiscovery configuration"
# }
