terraform {
  required_providers {
    anypoint = {
      source  = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# --------------------------------------------------------------------------
# Basic Managed Omni Gateway with minimal configuration
# --------------------------------------------------------------------------
# resource "anypoint_managed_omnigateway" "basic" {
#   name           = "my-basic-gateway"
#   environment_id = var.environment_id
#   target_id      = var.target_id
#   # runtime_version is auto-resolved to the latest for the default channel (lts)
# }

resource "anypoint_managed_omnigateway" "gw" {
  environment_id = var.environment_id # local.sandbox_env_id  # from remote state — not hardcoded
  target_id    = var.target_id
  name      = "test-gw-23"
  size      = "small"
  release_channel = "lts"
  logging = {
    level    = "info"
    forward_logs = true
  }
  tracing = {
    enabled  = true
    sampling = 10
    labels = [
      {
        type          = "environment"
        name          = "env-label"
        default_value = "v1"
        key_name      = "MY_ENV_VAR"
      },
      {
        type          = "requestHeader"
        name          = "header-label"
        default_value = "v2"
        key_name      = "X-Request-ID"
      },
      {
        type          = "literal"
        name          = "static-label"
        default_value = "my-service"
      }
    ]
  }
}

# # --------------------------------------------------------------------------
# # Fully configured Managed Omni Gateway with explicit version
# # --------------------------------------------------------------------------
# resource "anypoint_managed_omnigateway" "complete" {
#   name            = "my-production-gateway"
#   environment_id  = var.environment_id
#   target_id       = var.target_id  
#   release_channel = "lts"
#   size            = "small"

#   # public_url and internal_url are auto-computed from the target's domain when omitted.
#   # Set public_url to override with your own domain (e.g. when using a custom TLS context).
#   ingress = {
#     forward_ssl_session = true
#     last_mile_security  = true
#   }

#   properties = {
#     upstream_response_timeout = 30
#     connection_idle_timeout   = 120
#   }

#   logging = {
#     level        = "info"
#     forward_logs = true
#   }

#   tracing = {
#     enabled  = false
#     sampling = 1
#     labels   = []
#   }
# }

# --------------------------------------------------------------------------
# Variables
# --------------------------------------------------------------------------
variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App client ID"
  type        = string
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "environment_id" {
  description = "The environment ID where the gateway will be deployed"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "target_id" {
  description = "The private space / target ID for the gateway deployment"
  type        = string
  default     = "8b5e6c2b-f8f3-4555-b7be-6e75f18dd04e"
}

# # --------------------------------------------------------------------------
# # Outputs
# # --------------------------------------------------------------------------
# # output "basic_gateway_id" {
# #   description = "The ID of the basic managed Omni Gateway"
# #   value       = anypoint_managed_omni_gateway.basic.id
# # }

# # output "basic_gateway_status" {
# #   description = "The status of the basic managed Omni Gateway"
# #   value       = anypoint_managed_omni_gateway.basic.status
# # }

# output "complete_gateway_id" {
#   description = "The ID of the fully configured managed Omni Gateway"
#   value       = anypoint_managed_omni_gateway.complete.id
# }

# output "complete_gateway_public_url" {
#   description = "The public URL of the fully configured gateway"
#   value       = anypoint_managed_omnigateway.complete.ingress.public_url
# }

# output "complete_gateway_internal_url" {
#   description = "The internal URL of the fully configured gateway"
#   value       = anypoint_managed_omnigateway.complete.ingress.internal_url
# }

# output "complete_gateway_status" {
#   description = "The current status of the fully configured gateway"
#   value       = anypoint_managed_omnigateway.complete.status
# }

# # --------------------------------------------------------------------------
# # Datasource — look up an existing gateway by ID
# # --------------------------------------------------------------------------
# variable "gateway_id" {
#   description = "The ID of an existing managed Omni Gateway to look up"
#   type        = string
#   default     = ""
# }

# data "anypoint_managed_omnigateway" "existing" {
#   id             = var.gateway_id
#   environment_id = var.environment_id
# }

# output "existing_gateway_name" {
#   description = "Name of the looked-up gateway"
#   value       = data.anypoint_managed_omnigateway.existing.name
# }

# output "existing_gateway_status" {
#   description = "Current status of the looked-up gateway"
#   value       = data.anypoint_managed_omnigateway.existing.status
# }

# output "existing_gateway_public_url" {
#   description = "Primary public URL of the looked-up gateway"
#   value       = data.anypoint_managed_omnigateway.existing.ingress.public_url
# }

# output "existing_gateway_internal_urls" {
#   description = "All internal URLs of the looked-up gateway"
#   value       = data.anypoint_managed_omnigateway.existing.ingress.internal_urls
# }

# output "existing_gateway_port_config" {
#   description = "Ingress/egress port configuration of the looked-up gateway"
#   value       = data.anypoint_managed_omnigateway.existing.port_configuration
# }

# output "existing_gateway_runtime_version" {
#   description = "Runtime version of the looked-up gateway"
#   value       = data.anypoint_managed_omnigateway.existing.runtime_version
# }
