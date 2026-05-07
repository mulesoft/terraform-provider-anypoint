terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# ─────────────────────────────────────────────────────────────
# Variables
# ─────────────────────────────────────────────────────────────

variable "anypoint_client_id" {
  type      = string
  sensitive = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  type      = string
  sensitive = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  type    = string  
  default = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  type        = string
  description = "Organization ID"
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  type        = string
  description = "Environment ID where the API instance is deployed"
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "api_instance_id" {
  type        = string
  description = "Numeric ID of the API instance to apply policies to"
  default     = "4696123"
}

# ─────────────────────────────────────────────────────────────
# Local values for convenience
# ─────────────────────────────────────────────────────────────
locals {
  org_id = var.organization_id
  env_id = var.environment_id
  api_id = var.api_instance_id
}

# ═════════════════════════════════════════════════════════════
# SECURITY POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 1. Client ID Enforcement ────────────────────────────────
resource "anypoint_api_policy_client_id_enforcement" "client_id_enforcement" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "client-id-check"
  order           = 1

  configuration = {
    credentials_origin_has_http_basic_authentication_header = "customExpression"
    client_id_expression                                    = "#[attributes.headers['client_id']]"
    client_secret_expression                                = "#[attributes.headers['client_secret']]"
  }
}

# ─── 2. JWT Validation ───────────────────────────────────────
resource "anypoint_api_policy_jwt_validation" "jwt_validation" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "jwt-rsa"
  order           = 2
  disabled        = true # Enable when JWT infrastructure is ready

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

# ─── 3. IP Allowlist ─────────────────────────────────────────
resource "anypoint_api_policy_ip_allowlist" "ip_allowlist" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "allow-internal-networks"
  order           = 3

  configuration = {
    ip_expression = "#[attributes.headers['x-forwarded-for']]"
    ips           = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  }
}

# ─── 4. IP Blocklist ─────────────────────────────────────────
resource "anypoint_api_policy_ip_blocklist" "ip_blocklist" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "block-malicious-ips"
  order           = 4

  configuration = {
    ip_expression = "#[attributes.headers['x-forwarded-for']]"
    ips           = ["203.0.113.0/24", "198.51.100.0/24"]
  }
}

# ─── 5. Basic Authentication ─────────────────────────────────
resource "anypoint_api_policy_http_basic_authentication" "http_basic_auth" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "basic-auth"
  order           = 5
  disabled        = true

  configuration = {
    username = "admin"
    password = "changeme"
  }
}

# ═════════════════════════════════════════════════════════════
# RATE LIMITING POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 6. Rate Limiting ────────────────────────────────────────
resource "anypoint_api_policy_rate_limiting" "rate_limiting" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "rate-limit-100rpm"
  order           = 6

  configuration = {
    key_selector = "#[attributes.queryParams['identifier']]"
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

# ─── 7. Spike Control ────────────────────────────────────────
resource "anypoint_api_policy_spike_control" "spike_control" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "spike-1rps"
  order           = 7

  configuration = {
    maximum_requests            = 1
    time_period_in_milliseconds = 1000
    delay_time_in_millis        = 1000
    delay_attempts              = 1
    queuing_limit               = 5
    expose_headers              = true
  }
}

# ─── 8. Rate Limiting SLA Based ──────────────────────────────
resource "anypoint_api_policy_rate_limiting_sla_based" "rate_limiting_sla" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "sla-rate-limit"
  order           = 8
  disabled        = true

  configuration = {
    client_id_expression     = "#[attributes.headers['client_id']]"
    client_secret_expression = "#[attributes.headers['client_secret']]"
    expose_headers           = true
    clusterizable            = true
  }
}

# ═════════════════════════════════════════════════════════════
# TRAFFIC MANAGEMENT POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 9. CORS ─────────────────────────────────────────────────
resource "anypoint_api_policy_cors" "cors" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "cors-public"
  order           = 9

  configuration = {
    public_resource     = true
    support_credentials = false
    origin_groups       = []
  }
}

# ─── 10. Header Injection ────────────────────────────────────
resource "anypoint_api_policy_header_injection" "header_injection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "inject-headers"
  order           = 10

  configuration = {
    inbound_headers = [
      { key = "X-Request-ID", value = "#[uuid()]" },
      { key = "X-Request-Time", value = "#[now()]" }
    ]
    outbound_headers = [
      { key = "X-API-Version", value = "v1.0" },
      { key = "X-Response-Time", value = "#[now()]" }
    ]
  }
}

# ─── 11. Header Removal ──────────────────────────────────────
resource "anypoint_api_policy_header_removal" "header_removal" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "strip-internal-headers"
  order           = 11

  configuration = {
    inbound_headers  = ["X-Internal-Debug", "X-Temp-Token"]
    outbound_headers = ["Server", "X-Powered-By"]
  }
}

# ═════════════════════════════════════════════════════════════
# SECURITY & THREAT PROTECTION POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 12. JSON Threat Protection ──────────────────────────────
resource "anypoint_api_policy_json_threat_protection" "json_threat_protection" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "json-limits"
  order           = 12

  configuration = {
    max_container_depth          = 5
    max_string_value_length      = 1000
    max_object_entry_name_length = 100
    max_object_entry_count       = 100
    max_array_element_count      = 100
  }
}

# ─── 13. XML Threat Protection ───────────────────────────────
# NOTE: xml-threat-protection is a Mule 4 native policy and is NOT supported
# on Omni Gateway (the Anypoint API returns 400 "does not have an implementation
# for the API with technology(omniGateway)"). Apply this policy only to Mule 4
# API instances (technology = "mule4").
#
# resource "anypoint_api_policy_xml_threat_protection" "xml_threat_protection" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id   # must be a Mule 4 API instance
#   label           = "xml-threat"
#   order           = 13
#
#   configuration = {
#     max_node_depth                  = 10
#     max_attribute_count_per_element = 10
#     max_child_count                 = 100
#     max_text_length                 = 1000
#     max_attribute_length            = 100
#     max_comment_length              = 500
#   }
# }

# ═════════════════════════════════════════════════════════════
# MONITORING & LOGGING POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 14. Message Logging ─────────────────────────────────────
resource "anypoint_api_policy_message_logging" "message_logging" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "request-logger"
  order           = 14

  configuration = {
    logging_configuration = [
      {
        item_name = "Request Logging"
        item_data = {
          message        = "#[attributes.headers['request-id']]"
          conditional    = "#[attributes.method == 'POST']"
          category       = "api-requests"
          level          = "INFO"
          first_section  = true
          second_section = true
        }
      }
    ]
  }
}

# ─── 15. Response Timeout ────────────────────────────────────
resource "anypoint_api_policy_response_timeout" "response_timeout" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "timeout-30s"
  order           = 15

  configuration = {
    timeout = 30
  }
}

# ═════════════════════════════════════════════════════════════
# CACHING POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 16. HTTP Caching ────────────────────────────────────────
resource "anypoint_api_policy_http_caching" "http_caching" {
  organization_id = local.org_id
  environment_id  = local.env_id
  api_instance_id = local.api_id
  label           = "cache-600s"
  order           = 16

  configuration = {
    http_caching_key       = "#[attributes.requestPath ++ '?' ++ attributes.queryString]"
    max_cache_entries      = 10000
    ttl                    = 600
    distributed            = true
    persist_cache          = true
    use_http_cache_headers = true
    invalidation_header    = "X-Cache-Invalidate"
    request_expression     = "#[attributes.method == 'GET' or attributes.method == 'HEAD']"
    response_expression    = "#[[200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501] contains attributes.statusCode]"
  }
}

# ═════════════════════════════════════════════════════════════
# OUTBOUND POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 18. Message Logging (Outbound) ──────────────────────────
# Outbound policies require valid upstream_ids from the API instance's
# Omni Gateway routing configuration. Uncomment and set upstream_ids
# to a real upstream UUID before applying.
#
# resource "anypoint_api_policy_message_logging_outbound" "message_logging_outbound" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   order           = 18
#   upstream_ids    = ["<your-upstream-uuid>"]
#   configuration = {
#     logging_configuration = [
#       {
#         itemName = "Default configuration"
#         itemData = {
#           message       = "#[attributes.headers['id']]"
#           conditional   = "#[true]"
#           level         = "INFO"
#           firstSection  = true
#           secondSection = true
#         }
#       }
#     ]
#   }
# }

# ═════════════════════════════════════════════════════════════
# SCHEMA / SPEC VALIDATION POLICIES
# ═════════════════════════════════════════════════════════════

# ─── 17. Schema Validation ───────────────────────────────────
# Spec validation requires an API instance that has a REST or WSDL
# specification attached. Uncomment when using a spec-backed API.
#
# resource "anypoint_api_policy_spec_validation" "schema_validation" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_id
#   label           = "spec-validation"
#   order           = 17
#   configuration = {
#     block_operation          = true
#     strict_params_validation = true
#   }
# }

# ═════════════════════════════════════════════════════════════
# OUTPUTS
# ═════════════════════════════════════════════════════════════

output "policy_summary" {
  description = "Summary of all applied policies"
  value = {
    security = {
      client_id_enforcement = anypoint_api_policy_client_id_enforcement.client_id_enforcement.id
      jwt_validation        = anypoint_api_policy_jwt_validation.jwt_validation.id
      ip_allowlist          = anypoint_api_policy_ip_allowlist.ip_allowlist.id
      ip_blocklist          = anypoint_api_policy_ip_blocklist.ip_blocklist.id
      basic_auth            = anypoint_api_policy_http_basic_authentication.http_basic_auth.id
    }
    rate_limiting = {
      rate_limiting     = anypoint_api_policy_rate_limiting.rate_limiting.id
      spike_control     = anypoint_api_policy_spike_control.spike_control.id
      sla_rate_limiting = anypoint_api_policy_rate_limiting_sla_based.rate_limiting_sla.id
    }
    traffic_management = {
      cors              = anypoint_api_policy_cors.cors.id
      header_injection  = anypoint_api_policy_header_injection.header_injection.id
      header_removal    = anypoint_api_policy_header_removal.header_removal.id
    }
    monitoring = {
      message_logging   = anypoint_api_policy_message_logging.message_logging.id
      response_timeout  = anypoint_api_policy_response_timeout.response_timeout.id
    }
    caching = {
      http_caching = anypoint_api_policy_http_caching.http_caching.id
    }
    # spec_validation and message_logging_outbound are commented out above —
    # they require a spec-backed API and valid upstream_ids respectively.
  }
}
