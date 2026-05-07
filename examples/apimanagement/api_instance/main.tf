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

# --------------------------------------------------------------------------
# API Instance with weighted routing to multiple upstreams
# --------------------------------------------------------------------------
###############################################################################
# Example 1 – Minimal config (only required fields + upstream_uri shorthand)
# --------------------------------------------------------------------------
# Only needs: environment_id, gateway_id, spec, endpoint.base_path,
# and upstream_uri. Everything else uses defaults:
#   technology     = "flexGateway"
#   endpoint.type  = "http"
#   endpoint.deployment_type = "HY"
#   upstream weight = 100
#   routing = [{upstreams: [{weight: 100, uri: <upstream_uri>}]}]
###############################################################################

resource "anypoint_api_instance" "minimal" {
  environment_id = var.environment_id
  gateway_id     = var.gateway_id
  instance_label = "minimal-demo"
  upstream_uri   = "http://backend.internal:8080"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    base_path = "minimal"
  }  
}

###############################################################################
# Example 2 – Simple config with optional fields
# --------------------------------------------------------------------------
# Adds instance_label, consumer_endpoint, and approval_method on top of
# the minimal config. Still uses upstream_uri for simple routing.
###############################################################################

resource "anypoint_api_instance" "simple_with_options" {
  environment_id    = var.environment_id
  gateway_id        = var.gateway_id
  instance_label    = "orders-api-demo"
  approval_method   = "manual"
  consumer_endpoint = "https://api.example.com/orders"
  upstream_uri      = "http://backend.internal:8080"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    base_path = "simpleWithOptions"
  }

}

###############################################################################
# Example 3 – Single route with single upstream (using routing block)
# --------------------------------------------------------------------------
# Uses the full routing block instead of upstream_uri. Useful when you need
# a route label or upstream label but still have a single backend.
# Weight defaults to 100 so it can be omitted for a single upstream.
###############################################################################

resource "anypoint_api_instance" "single_route" {
  environment_id = var.environment_id
  gateway_id     = var.gateway_id
  instance_label = "single-route-demo"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    base_path = "singleRoute"
  }

  routing = [
    {
      label = "default"
      upstreams = [
        {
          uri   = "http://backend.internal:8080"
          label = "primary"
        }
      ]
    }
  ]

}

###############################################################################
# Example 4 – Weighted multi-upstream routing (canary / blue-green)
# --------------------------------------------------------------------------
# Two upstreams with weights that must sum to 100.
# Validation ensures the total is correct at plan time.
###############################################################################

resource "anypoint_api_instance" "weighted_routing" {
  environment_id = var.environment_id
  gateway_id     = var.gateway_id
  instance_label = "weighted-routing-demo"
  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    base_path = "weightedRouting"
  }

  routing = [
    {
      label = "canary"
      upstreams = [
        {
          weight = 90
          uri    = "http://backend-stable.internal:8080"
          label  = "stable"
        },
        {
          weight = 10
          uri    = "http://backend-canary.internal:8080"
          label  = "canary"
        }
      ]
    }
  ]

}

###############################################################################
# Example 5 – Full config: multi-route, rules, TLS upstreams, consumer endpoint
# --------------------------------------------------------------------------
# Advanced routing with multiple routes, method/path rules, upstream TLS
# contexts, and a consumer-facing endpoint. This is the most complete
# configuration showing all available features.
###############################################################################

# resource "anypoint_api_instance" "main_5" {
#   environment_id    = var.environment_id
#   instance_label    = "fullConfig"
#   approval_method   = "manual"
#   consumer_endpoint = "https://www.consumerendpoint.com"
#   gateway_id        = var.gateway_id

#   spec = {
#     asset_id = var.api_asset_id
#     group_id = var.organization_id
#     version  = var.api_asset_version
#   }

#   endpoint = {
#     base_path = "basePath1"
#   }

#   routing = [
#     {
#       label = "read-traffic"
#       rules = {
#         methods = "GET"
#       }
#       upstreams = [
#         {
#           weight         = 90
#           uri            = "https://echo.com"
#           label          = "echo1"
#           tls_context_id = "fb1ba718-9b11-456d-8e39-bcc5d0b365f6/84126c23-b0ce-45fe-b38a-3410c8467d5b"
#         },
#         {
#           weight         = 10
#           uri            = "https://echo2.com"
#           label          = "echo2"
#           tls_context_id = "fb1ba718-9b11-456d-8e39-bcc5d0b365f6/84126c23-b0ce-45fe-b38a-3410c8467d5b"
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
#           uri   = "https://echo.com"
#           label = "primary"
#         }
#       ]
#     }
#   ]

# }

# --------------------------------------------------------------------------
# Variables
# --------------------------------------------------------------------------
variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App client ID"
  type        = string
  default = "e5a776d9862a4f2d8f61ba8450803908"
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

variable "organization_id" {
  description = "The Anypoint organization (group) ID for Exchange assets"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "The environment ID where the API instances will be created"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "gateway_id" {
  description = "Base name for the Managed Omni Gateway (will have '-gw' appended)"
  type        = string
  default     = "b123b2eb-35aa-454c-9750-dff9e2d218c9"
}

variable "api_asset_id" {
  description = "Exchange asset ID for the API specification"
  type        = string
  default     = "api-test"
}

variable "api_asset_version" {
  description = "Exchange asset version"
  type        = string
  default     = "1.0.0"
}

variable "api_base_path" {
  description = "Base path for the API instance (appended to http://0.0.0.0:8081/)"
  type        = string
  default     = "basePath"
}

# --------------------------------------------------------------------------
# Outputs
# --------------------------------------------------------------------------
output "minimal_api_id" {
  description = "The numeric ID of the minimal API instance"
  value       = anypoint_api_instance.minimal.id
}

output "minimal_api_status" {
  description = "The status of the minimal API instance"
  value       = anypoint_api_instance.minimal.status
}

output "simple_with_options_api_id" {
  description = "The numeric ID of the simple API instance"
  value       = anypoint_api_instance.simple_with_options.id
}

output "simple_with_options_api_status" {
  description = "The status of the simple API instance"
  value       = anypoint_api_instance.simple_with_options.status
}

output "single_route_api_id" {
  description = "The numeric ID of the single route API instance"
  value       = anypoint_api_instance.single_route.id
}

output "single_route_api_status" {
  description = "The status of the single route API instance"
  value       = anypoint_api_instance.single_route.status
}

output "weighted_routing_api_id" {
  description = "The numeric ID of the weighted routing API instance"
  value       = anypoint_api_instance.weighted_routing.id
}

output "weighted_routing_api_status" {
  description = "The status of the weighted routing API instance"
  value       = anypoint_api_instance.weighted_routing.status
}

# output "full_config_api_id" {
#   description = "The numeric ID of the full config API instance"
#   value       = anypoint_api_instance.main_5.id
# }

# output "full_config_api_status" {
#   description = "The status of the full config API instance"
#   value       = anypoint_api_instance.main_5.status
# }