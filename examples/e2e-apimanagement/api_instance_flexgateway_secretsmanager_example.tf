###############################################################################
# Comprehensive End-to-End Example
# ---------------------------------
# Provisions the full Anypoint Platform stack from scratch:
#
#   1. Private Space          – isolated compute environment
#   2. Private Network        – VPC-like network inside the private space
#   3. VPN Connection         – site-to-site VPN tunnel to on-prem
#   4. Managed Flex Gateway   – Flex Gateway runtime in the private space
#   5. API Instance           – API proxy deployed to the Flex Gateway
#   6. API Policies (43)      – full inbound + outbound policy suite
#
# Usage:
#   cp terraform.tfvars.example terraform.tfvars   # fill in your values
#   terraform init
#   terraform plan
#   terraform apply
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

###############################################################################
# Step 1 – Private Space + Network (use anypoint_private_space_config)
###############################################################################

# resource "anypoint_private_space_config" "main" {
#   name            = var.private_space_name
#   organization_id = var.organization_id
#   enable_egress   = true
#
#   network {
#     region     = var.region
#     cidr_block = var.network_cidr_block
#   }
# }

###############################################################################
# Step 2 – VPN Connection
# Note: depends_on ensures the private space config is created before the VPN.
###############################################################################

# resource "anypoint_vpn_connection" "site_to_site" {
#   depends_on       = [anypoint_private_space_config.main]
#   private_space_id = anypoint_private_space_config.main.id
#   organization_id  = var.organization_id
#   name             = "${var.private_space_name}-vpn"

#   vpns = [
#     {
#       local_asn         = var.vpn_local_asn
#       remote_asn        = var.vpn_remote_asn
#       remote_ip_address = var.vpn_remote_ip
#       static_routes     = []

#       vpn_tunnels = [
#         {
#           psk            = var.vpn_tunnel_1_psk
#           ptp_cidr       = var.vpn_tunnel_1_ptp_cidr
#           startup_action = "start"
#         },
#         {
#           psk            = var.vpn_tunnel_2_psk
#           ptp_cidr       = var.vpn_tunnel_2_ptp_cidr
#           startup_action = "start"
#         }
#       ]
#     }
#   ]
# }

###############################################################################
# Step 3b – Secrets Management (Secret Group, Keystore, Truststore, TLS Context)
###############################################################################

resource "anypoint_secret_group" "main" {
  environment_id = var.environment_id
  name           = "secrets-group-terraform-example-1"
  downloadable   = false
}

resource "anypoint_secret_group_keystore" "tls" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-keystore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/../certs/key.pem"))
}

resource "anypoint_secret_group_truststore" "ca" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "ca-truststore"
  type            = "PEM"

  truststore_base64 = base64encode(file("${path.module}/../certs/truststore.pem"))
}

resource "anypoint_flex_tls_context" "flex" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "flex-tls-context"

  keystore_id   = anypoint_secret_group_keystore.tls.id
  truststore_id = anypoint_secret_group_truststore.ca.id

  alpn_protocols = ["h2", "http/1.1"]
}

###############################################################################
# Step 4 – Managed Flex Gateway
###############################################################################

resource "anypoint_managed_flexgateway" "main" {
  name            = "managed-flexgateway-example-1"
  environment_id  = var.environment_id
  target_id       = "675c4efb-d44e-44cd-ac6f-d5a1128e6236"

  # ingress = {
  #   forward_ssl_session = true
  #   last_mile_security  = true
  # }

  # properties = {
  #   upstream_response_timeout = 30
  #   connection_idle_timeout   = 120
  # }

  # logging = {
  #   level        = "info"
  #   forward_logs = true
  # }

  # tracing = {
  #   enabled = false
  # }

  depends_on = [anypoint_flex_tls_context.flex]
}

###############################################################################
# Step 5 – API Instance on the Flex Gateway
###############################################################################

resource "anypoint_api_instance" "main" {
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "main-api-1"
  approval_method = "manual"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = var.api_base_path
    ssl_context_id  = "${anypoint_secret_group.main.id}/${anypoint_flex_tls_context.flex.id}"
  }

  gateway_id = anypoint_managed_flexgateway.main.id

  routing = [
    {
      label = "read-traffic"
      rules = {
        methods = "GET"
      }
      upstreams = [
        {
          weight = var.upstream_primary_weight
          uri    = var.upstream_primary_uri
          label  = "primary"
        },
        {
          weight = 100 - var.upstream_primary_weight
          uri    = var.upstream_secondary_uri
          label  = "secondary"
        }
      ]
    },
    {
      label = "write-traffic"
      rules = {
        methods = "POST|PUT|PATCH|DELETE"
        path    = "/api/*"
      }
      upstreams = [
        {
          weight = 100
          uri    = var.upstream_primary_uri
          label  = "primary"
        }
      ]
    }
  ]

  depends_on = [anypoint_managed_flexgateway.main, anypoint_flex_tls_context.flex]
}

# resource "anypoint_api_instance" "main_4" {
#   environment_id  = var.environment_id
#   technology      = "flexGateway"
#   instance_label  = "main-api-6"
#   approval_method = "manual"

#   spec = {    
#     asset_id = var.api_asset_id
#     group_id = var.organization_id
#     version  = var.api_asset_version
#   }

#   endpoint = {
#     deployment_type = "HY"
#     type            = "http"
#     base_path       = var.api_base_path
#     // Expose port
#     // Rename this to tls_***
#     // Export host
#     ssl_context_id  = "${anypoint_secret_group.main.id}/${anypoint_flex_tls_context.flex.id}"
#   }

#     # upstream = {
#     #  uri = "http://echo.com"
#     #  label = "echo1"
#     #  tls_context_id = "${anypoint_secret_group.main.id}/${anypoint_flex_tls_context.flex.id}"
#     # }

#   gateway_id = anypoint_managed_flexgateway.main.id

#   routing = [
#     {
#       label = "read-traffic"
#       rules = {
#         methods = "GET"
#       }
#       upstreams = [
#         {
#           weight = var.upstream_primary_weight
#           uri    = "http://echo.com"
#           label  = "echo1"
#         },
#         {
#           weight = 100 - var.upstream_primary_weight
#           uri    = "http://echo2.com"
#           label  = "echo2"
#         }
#       ]
#     },
#     {
#       label = "write-traffic"
#       rules = {
#         methods = "POST|PUT|PATCH|DELETE"
#         path    = "/api/*"
#       }
#       upstreams = [
#         {
#           weight = 100 // default to 100, validation on total weightreight
#           uri    = var.upstream_primary_uri
#           label  = "primary"
#         }
#       ]
#     }
#   ]

#   depends_on = [anypoint_managed_flexgateway.main, anypoint_flex_tls_context.flex]
# }

###############################################################################
# Step 6 – API Policies (applied to the API instance)
###############################################################################

# locals block to reduce repetition
locals {
  org_id      = var.organization_id
  env_id      = var.environment_id
  api_id      = anypoint_api_instance.main.id

  # upstream_id is the routing-upstream UUID for outbound policies.
  # Retrieve it from the Anypoint API Manager UI or via the REST API after
  # the API instance has been created. It is NOT the same as the api_id.
  upstream_id = var.upstream_id
}

# ─── 1. Rate Limiting ───────────────────────────────────────────────────────
resource "anypoint_api_policy_rate_limiting" "rate_limiting" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "rate-limit-100rpm"
  order           = 1

  configuration = {
    key_selector   = "#[attributes.queryParams['identifier']]"
    rate_limits = [
      {
        maximum_requests            = 100
        time_period_in_milliseconds = 60000
      }
    ]
    expose_headers = true
    clusterizable  = true
  }
}

# ─── 2. Spike Control ───────────────────────────────────────────────────────
resource "anypoint_api_policy_spike_control" "spike_control" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "spike-1rps"
  order           = 2

  configuration = {
    maximum_requests            = 1
    time_period_in_milliseconds = 1000
    delay_time_in_millis        = 1000
    delay_attempts              = 1
    queuing_limit               = 5
    expose_headers              = true
  }
}

# ─── 3. Rate Limiting SLA Based ─────────────────────────────────────────────
resource "anypoint_api_policy_rate_limiting_sla_based" "rate_limiting_sla" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "sla-rate-limit"
  order           = 3
  disabled        = true

  configuration = {
    client_id_expression     = "#[attributes.headers['client_id']]"
    client_secret_expression = "#[attributes.headers['client_secret']]"
    expose_headers           = true
    clusterizable            = true
  }
}

# ─── 4. Client ID Enforcement ───────────────────────────────────────────────
resource "anypoint_api_policy_client_id_enforcement" "client_id_enforcement" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "client-id-check"
  order           = 4

  configuration = {
    credentials_origin_has_http_basic_authentication_header = "customExpression"
    client_id_expression     = "#[attributes.headers['client_id']]"
    client_secret_expression = "#[attributes.headers['client_secret']]"
  }
}

# ─── 5. JWT Validation ──────────────────────────────────────────────────────
resource "anypoint_api_policy_jwt_validation" "jwt_validation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "jwt-rsa"
  order           = 5
  disabled        = true

  configuration = {
    jwt_origin                      = "httpBearerAuthenticationHeader"
    signing_method                  = "rsa"
    signing_key_length              = 256
    jwt_key_origin                  = "text"
    text_key                        = "your-(256|384|512)-bit-secret"
    custom_key_expression           = "#[authentication.properties['key_to_your_public_pem_certificate']]"
    jwks_url                        = "http://your-jwks-service.example:80/base/path"
    jwks_service_time_to_live       = 60
    jwks_service_connection_timeout = 10000
    skip_client_id_validation       = false
    client_id_expression            = "#[vars.claimSet.client_id]"
    jwt_expression                  = "#[attributes.headers['jwt']]"
    validate_aud_claim              = true
    mandatory_aud_claim             = true
    supported_audiences             = "aud.example.com"
    mandatory_exp_claim             = true
    mandatory_nbf_claim             = true
    validate_custom_claim           = true
    claims_to_headers               = []
    mandatory_custom_claims         = []
    non_mandatory_custom_claims     = []
  }
}

# ─── 6. IP Allowlist ────────────────────────────────────────────────────────
resource "anypoint_api_policy_ip_allowlist" "ip_allowlist" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "allow-private-subnets"
  order           = 6

  configuration = {
    ip_expression = "#[attributes.headers['x-forwarded-for']]"
    ips           = ["192.168.0.1/16", "10.0.0.1"]
  }
}

# ─── 7. IP Blocklist ────────────────────────────────────────────────────────
resource "anypoint_api_policy_ip_blocklist" "ip_blocklist" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "block-bad-actors"
  order           = 7

  configuration = {
    ip_expression = "#[attributes.headers['x-forwarded-for']]"
    ips           = ["108.1.12.12", "109.2.2.2"]
  }
}

# ─── 8. JSON Threat Protection ──────────────────────────────────────────────
resource "anypoint_api_policy_json_threat_protection" "json_threat_protection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "json-limits"
  order           = 8

  configuration = {
    max_container_depth          = -1
    max_string_value_length      = -1
    max_object_entry_name_length = -1
    max_object_entry_count       = -1
    max_array_element_count      = -1
  }
}

# ─── 9. External Authorization ──────────────────────────────────────────────
resource "anypoint_api_policy_native_ext_authz" "ext_authz" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "ext-auth-http"
  order           = 9

  configuration = {
    uri                      = "http://auth.com"
    server_type              = "http"
    request_timeout          = 5000
    server_api_version       = "v3"
    include_peer_certificate = false
    allowed_headers          = ["allowed-head-1"]
    service_request_headers_to_add = [
      { key = "header-1", value = "xyz" },
      { key = "header-2", value = "value-2" }
    ]
    service_response_upstream_headers           = ["response-header1", "response-header2"]
    service_response_upstream_headers_to_append = ["response-header-to-add-1", "response-header-to-add-2"]
    service_response_client_headers             = ["Client-Header-1", "Client-Header-2"]
    service_response_client_headers_on_success  = ["Success-Header-1", "Success-Header-2"]
    path_prefix = "path"
  }
}

# ─── 10. External Processing ────────────────────────────────────────────────
resource "anypoint_api_policy_native_ext_proc" "ext_proc" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "ext-processor"
  order           = 10

  configuration = {
    uri                  = "h2://external.com"
    message_timeout      = 1000
    max_message_timeout  = 0
    failure_mode_allow   = true
    allow_mode_override  = true
    request_header_mode  = "send"
    response_header_mode = "send"
    request_body_mode    = "none"
    response_body_mode   = "none"
    request_trailer_mode  = "skip"
    response_trailer_mode = "skip"
  }
}

# ─── 11. SSE Logging ────────────────────────────────────────────────────────
resource "anypoint_api_policy_sse_logging" "sse_logging" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "sse-logs"
  order           = 11

  configuration = {
    logs = [
      { message = "#[payload]", category = "log-1", level = "INFO" },
      { message = "#[payload]", category = "log-2", level = "INFO" }
    ]
  }
}

# ─── 12. CORS ───────────────────────────────────────────────────────────────
resource "anypoint_api_policy_cors" "cors" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "cors-public"
  order           = 12

  configuration = {
    public_resource     = true
    support_credentials = false
    origin_groups       = []
  }
}

# ─── 13. Message Logging ────────────────────────────────────────────────────
resource "anypoint_api_policy_message_logging" "message_logging" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "request-logger"
  order           = 13

  configuration = {
    logging_configuration = [
      {
        item_name = "Default configuration -1"
        item_data = {
          message        = "#[attributes.headers['id']]"
          conditional    = "#[attributes.headers['id']==1]"
          category       = "log1"
          level          = "INFO"
          first_section  = true
          second_section = true
        }
      }
    ]
  }
}

# ─── 14. Header Injection ───────────────────────────────────────────────────
resource "anypoint_api_policy_header_injection" "header_injection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "inject-headers"
  order           = 14

  configuration = {
    inbound_headers = [
      { key = "header-1", value = "value-1" },
      { key = "header-2", value = "value-2" }
    ]
    outbound_headers = [
      { key = "header-3", value = "value-3" },
      { key = "header-4", value = "value-4" }
    ]
  }
}

# ─── 15. Header Removal ─────────────────────────────────────────────────────
resource "anypoint_api_policy_header_removal" "header_removal" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "strip-internal-headers"
  order           = 15

  configuration = {
    inbound_headers  = ["X-Internal-Debug", "X-Temp-Token"]
    outbound_headers = ["Server", "X-Powered-By"]
  }
}

# ─── 16. Basic Authentication - Simple ───────────────────────────────────────
resource "anypoint_api_policy_http_basic_authentication" "http_basic_auth" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "basic-auth"
  order           = 16
  disabled        = true

  configuration = {
    username = "admin"
    password = "admin"
  }
}

# ─── 17. Response Timeout ───────────────────────────────────────────────────
resource "anypoint_api_policy_response_timeout" "response_timeout" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "timeout-15s"
  order           = 18

  configuration = {
    timeout = 15
  }
}

# ─── 18. Stream Idle Timeout ────────────────────────────────────────────────
resource "anypoint_api_policy_stream_idle_timeout" "stream_idle_timeout" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "idle-60s"
  order           = 19

  configuration = {
    timeout = 60
  }
}

# ─── 19. Health Check ───────────────────────────────────────────────────────
resource "anypoint_api_policy_health_check" "health_check" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "health-200"
  order           = 20

  configuration = {
    endpoint    = "http://www.google.com"
    path        = "/status"
    status_code = "200"
  }
}

# ─── 20. HTTP Caching ───────────────────────────────────────────────────────
resource "anypoint_api_policy_http_caching" "http_caching" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "cache-600s"
  order           = 21

  configuration = {
    http_caching_key       = "#[attributes.requestPath]"
    max_cache_entries      = 10000
    ttl                    = 600
    distributed            = true
    persist_cache          = true
    use_http_cache_headers = true
    invalidation_header    = "invalidation-header"
    request_expression     = "#[attributes.method == 'GET' or attributes.method == 'HEAD']"
    response_expression    = "#[[200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501] contains attributes.statusCode]"
  }
}

# ─── 21. OAuth 2.0 Token Introspection ──────────────────────────────────────
resource "anypoint_api_policy_oauth2_token_introspection" "oauth2_introspection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "oauth2-introspect"
  order           = 22
  disabled        = true

  configuration = {
    introspection_url        = "http://www.google.com"
    authorization_value      = "Basic am9obkBleGFtcGxlLmNvbToxMjM0NTY="
    validated_token_ttl      = 600
    scope_validation_criteria = "AND"
    skip_client_id_validation = false
    consumer_by              = "client_id"
    expose_headers           = false
    max_cache_entries        = 10000
    authentication_timeout   = 10000
  }
}

# ─── 22. Access Block ───────────────────────────────────────────────────────
resource "anypoint_api_policy_access_block" "access_block" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "block-all"
  order           = 23
  disabled        = true

  configuration = {}
}

# ─── 23. Agent Connection Telemetry ─────────────────────────────────────────
resource "anypoint_api_policy_agent_connection_telemetry" "agent_connection_telemetry" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "agent-telemetry"
  order           = 24

  configuration = {
    source_agent_id = "#[attributes.headers['X-ANYPOINT-API-INSTANCE-ID']]"
  }
}

# ─── 24. Basic Authentication - LDAP ────────────────────────────────────────
resource "anypoint_api_policy_ldap_authentication" "ldap_auth" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "ldap-auth"
  order           = 24
  disabled        = true

  configuration = {
    ldap_server_url           = "ldap-server.com:9090"
    ldap_server_user_dn       = "ldapuserdn"
    ldap_server_user_password = "Admin"
    ldap_search_base          = "ou=People,dc=acme,dc=org"
    ldap_search_filter        = "(uid={0})"
    ldap_search_in_subtree    = true
  }
}

# ─── 25. Tracing ─────────────────────────────────────────────────────────────
resource "anypoint_api_policy_tracing" "tracing" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "tracing-sp1"
  order           = 25

  configuration = {
    sampling = {
      client  = 100
      random  = 100
      overall = 100
    }
    span_name = "sp-1"
    labels    = []
  }
}

# ─── 26. XML Threat Protection ──────────────────────────────────────────────
resource "anypoint_api_policy_xml_threat_protection" "xml_threat_protection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "xml-threat"
  order           = 26

  configuration = {
    max_node_depth                  = -1
    max_attribute_count_per_element = -1
    max_child_count                 = -1
    max_text_length                 = -1
    max_attribute_length            = -1
    max_comment_length              = -1
  }
}

# ─── 27. Injection Protection ───────────────────────────────────────────────
resource "anypoint_api_policy_injection_protection" "injection_protection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "injection-protect"
  order           = 27

  configuration = {
    built_in_protections = ["xss", "sql"]
    custom_protections = [
      {
        name  = "custom-rule-1"
        regex = "abcd*"
      }
    ]
    protect_path_and_query = true
    protect_headers        = true
    protect_body           = true
    reject_requests        = true
  }
}

# ─── 28. DataWeave Request Filter ───────────────────────────────────────────
resource "anypoint_api_policy_dataweave_request_filter" "dataweave_request_filter" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "dw-filter"
  order           = 28

  configuration = {
    script = <<-DW
      %dw 2.0
      output application/json
      ---
      if (attributes.headers['client-id'] != null)
        {
          'success': true
        }
      else
        {
          'success': false,
          'response': {
            'statusCode': 401,
            'body': 'Error: client-id header is required',
            'headers': {
                'www-authenticate': 'custom'
            }
          }
        }
    DW
    requires_payload = true
  }
}

# ─── 29. Body Transformation ────────────────────────────────────────────────
resource "anypoint_api_policy_body_transformation" "body_transformation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "body-transform"
  order           = 29

  configuration = {
    script       = "asdasdsdewre"
    request_flow = "onRequest"
  }
}

# ─── 30. Header Transformation ──────────────────────────────────────────────
resource "anypoint_api_policy_header_transformation" "header_transformation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "header-transform"
  order           = 30

  configuration = {
    script           = "asdadasdsa"
    requires_payload = true
    request_flow     = "onResponse"
  }
}

# ─── 31. DataWeave Body Transformation ──────────────────────────────────────
resource "anypoint_api_policy_dataweave_body_transformation" "dataweave_body_transformation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "dw-body-transform"
  order           = 31

  configuration = {
    script = <<-DW
      %dw 2.0
      output application/json
      ---
      payload
    DW
    request_flow = "onResponse"
  }
}

# ─── 32. DataWeave Headers Transformation ───────────────────────────────────
resource "anypoint_api_policy_dataweave_headers_transformation" "dataweave_headers_transformation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "dw-headers-transform"
  order           = 32

  configuration = {
    script = <<-DW
      %dw 2.0
      output application/json
      ---
      attributes.headers ++ {
        "client_id": payload.id,
        "client_secret": payload.secret
      }
    DW
    requires_payload = true
    request_flow     = "onRequest"
  }
}

# ─── 33. Script Evaluation Transformation ───────────────────────────────────
resource "anypoint_api_policy_script_evaluation_transformation" "script_evaluation_transformation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "script-eval-transform"
  order           = 33

  configuration = {
    script           = "abcde"
    requires_payload = true
    request_flow     = "onRequest"
  }
}

# ─── 34. Schema / Spec Validation ───────────────────────────────────────────
# resource "anypoint_api_policy_spec_validation" "spec_validation" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "spec-validate"
#   order           = 34

#   configuration = {
#     config              = "BASIC"
#     validation_criteria = "NONE"
#     notification_actions = [
#       {
#         notification_level = "WARN"
#         notification_type  = "LOGGER"
#       }
#     ]
#   }
# }

###############################################################################
# Step 6b – Outbound Policies
# Outbound policies apply to upstream (egress) traffic. They require an
# upstream_id referencing one of the routing upstreams defined on the API
# instance. Set var.upstream_id to the UUID shown in API Manager UI.
###############################################################################

# ─── 35. Message Logging Outbound ────────────────────────────────────────────
# resource "anypoint_api_policy_message_logging_outbound" "message_logging_outbound" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "outbound-logger"
#   order           = 1
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     logging_configuration = [
#       {
#         item_name = "Default configuration"
#         item_data = {
#           message        = "#[attributes.headers['id']]"
#           conditional    = "#[attributes.headers['id']==1]"
#           category       = "outbound-log1"
#           level          = "INFO"
#           first_section  = true
#           second_section = true
#         }
#       }
#     ]
#   }
# }

# ─── 36. In-Task Authorization Code Policy ───────────────────────────────────
# resource "anypoint_api_policy_intask_authorization_code_policy" "intask_authz" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "intask-authz-code"
#   order           = 2
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     secondary_auth_provider = "oauth2"
#     authorization_endpoint  = "https://auth.example.com/oauth2/authorize"
#     token_endpoint          = "https://auth.example.com/oauth2/token"
#     redirect_uri            = "https://www.google.com"
#     scopes                  = "read write"
#   }
# }

# ─── 37. Credential Injection – OAuth 2.0 ────────────────────────────────────
# resource "anypoint_api_policy_credential_injection_oauth2" "cred_inject_oauth2" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "cred-inject-oauth2"
#   order           = 3
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     oauth_service = "https://auth.example.com/oauth2/token"
#     client_id     = "my-service-client-id"
#     client_secret = var.oauth_client_secret
#     scope         = ["api:read"]
#   }
# }

# ─── 38. Credential Injection – Basic Auth ───────────────────────────────────
# resource "anypoint_api_policy_credential_injection_basic_auth" "cred_inject_basic" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "cred-inject-basic"
#   order           = 4
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     username = "upstream-svc-user"
#     password = var.upstream_basic_auth_password
#   }
# }

# ─── 39. Idle Timeout ────────────────────────────────────────────────────────
# resource "anypoint_api_policy_idle_timeout" "idle_timeout" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "upstream-idle-120s"
#   order           = 5
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     timeout = 120
#   }
# }

# ─── 40. Circuit Breaker ─────────────────────────────────────────────────────
# resource "anypoint_api_policy_circuit_breaker" "circuit_breaker" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "circuit-breaker"
#   order           = 6
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     thresholds = {
#       count        = 5
#       time         = 10
#       rtime        = 5
#       erp          = 50
#       mstime       = 30000
#     }
#   }
# }

# ─── 41. Native AWS Lambda ───────────────────────────────────────────────────
# resource "anypoint_api_policy_native_aws_lambda" "aws_lambda" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "aws-lambda-invoke"
#   order           = 7
#   upstream_ids    = [local.upstream_id]

#   configuration = {
#     arn              = "arn:aws:lambda:us-east-1:123456789012:function:my-lambda-function"
#     invocation_mode  = "REQUEST_RESPONSE"
#     authentication_mode = "CREDENTIALS"
#     credentials = {
#       access_key_id     = var.aws_access_key_id
#       secret_access_key = var.aws_secret_access_key
#     }
#   }
# }

###############################################################################
# Step 6c – SLA Tier
###############################################################################
resource "anypoint_api_instance_sla_tier" "tier1" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.main.id

  name         = "Tier1"
  description  = "SLA Tier1"
  auto_approve = false
  status       = "ACTIVE"

  limits = [
    {
      time_period_in_milliseconds = 60000
      maximum_requests            = 10
      visible                     = true
    },
    {
      time_period_in_milliseconds = 1000
      maximum_requests            = 5
      visible                     = true
    }
  ]
}

###############################################################################
# Step 7 – Alert
###############################################################################
resource "anypoint_api_instance_alert" "request_count" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.main.id

  name            = "alert-1"
  severity        = "warning"
  deployment_type = "HY"
  metric_type     = "api_request_count"

  condition = {
    operator  = "above"
    threshold = 99
    interval  = 5
  }

  notifications = [
    {
      type       = "email"
      recipients = [var.alert_email]
      subject    = "$${severity}: $${api} $${condition}"
      message    = <<-EOT
        Hello,
        You are receiving this alert because:
         The API $${api} has $${condition} of $${value} at $${timestamp}.
        The API has reached the threshold based on $${condition} is $${operator} $${threshold} for $${period}.

        Environment: $${environment}
        $${dashboardLink}
        $${apiLink}
      EOT
    }
  ]
}

###############################################################################
# Step 8 – API Instance Promotion
# Promotes the API instance (with its policies, SLA tiers, and alerts)
# from the current environment into a target environment.
###############################################################################

resource "anypoint_api_instance_promotion" "to_production" {
  organization_id = var.organization_id
  environment_id = var.target_environment_id
  source_api_id  = anypoint_api_instance.main.id

  include_alerts   = true
  include_policies = true
  include_tiers    = true

  depends_on = [anypoint_api_instance.main]
}
