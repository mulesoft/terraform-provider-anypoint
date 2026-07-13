data "anypoint_managed_omni_gateways" "all" {
  organization_id = var.organization_id
  environment_id  = var.env_id
}

# Look up TLS contexts in the secret group created in main.tf
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

resource "anypoint_api_instance" "payments" {
  environment_id = var.env_id
  gateway_id     = local.gateway_id
  technology     = "omniGateway"
  instance_label = "payments-api"
  approval_method = "manual"
  spec = {
    asset_id = var.api_asset_id
    group_id = var.organization_id
    version  = var.api_asset_version
  }
  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = "payments"
    ssl_context_id  = "${anypoint_secret_group.main.id}/${local.omni_tls_context_id}"
  }
  
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
}

variable "organization_id" {
  description = "The organization ID where the API instance will be deployed"
  type        = string
  default     = "<org_id>"
}

variable "env_id" {
  description = "The environment ID where the API instance will be deployed"
  type        = string
  default     = "<private_space_id>"
}

# ── API Instance ────────────────────────────────────────────────────────────

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
  default     = "https://api-test:8081"
}

variable "upstream_primary_uri" {
  description = "Primary backend URI"
  type        = string
  default     = "http://backend-primary.internal:8080"
}

variable "upstream_secondary_uri" {
  description = "Secondary backend URI (for canary / blue-green)"
  type        = string
  default     = "http://backend-secondary.internal:8080"
}

variable "upstream_primary_weight" {
  description = "Traffic weight (0-100) sent to the primary upstream"
  type        = number
  default     = 90
}
