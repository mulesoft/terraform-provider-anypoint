###############################################################################
# Anypoint Terraform Provider – API Management Demo
# ===================================================
# This demo walks through a real-world API lifecycle:
#
#   Step 1 → Create a sub-organization with Sandbox & Production environments
#   Step 2 → Grant Connected App access, create Private Space & Omni Gateway
#   Step 3 → Create an API Instance with canary routing
#   Step 4 → Apply security policies (JWT, rate limiting, IP allowlist)
#   Step 5 → Define SLA tiers for consumer self-service
#   Step 6 → Promote the API from Sandbox → Production
#   Step 8 → Bundle APIs into an API Group for consumers
#
# Usage:
#   terraform init
#   terraform plan       ← preview what will be created
#   terraform apply      ← provision everything
#   terraform destroy    ← tear it all down
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
    }
  }
}

# Default provider – uses Connected App (client credentials) for API management
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Admin provider – uses user credentials for org/env/connected-app management
provider "anypoint" {
  alias         = "admin"
  auth_type     = "user"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = var.anypoint_base_url
}

###############################################################################
# Step 1 – Organization & Environments
# ---------------------------------------
# Stand up an entire business unit from scratch: a sub-organization
# with Sandbox and Production environments — all as code.
###############################################################################

resource "anypoint_organization" "commerce_bu" {
  provider = anypoint.admin

  name                   = "Commerce Business Unit"
  parent_organization_id = var.parent_organization_id
  owner_id               = var.owner_id

  entitlements = {
    create_sub_orgs     = false
    create_environments = true
    global_deployment   = false

    vcores_production   = { assigned = 1, reassigned = 0 }
    vcores_sandbox      = { assigned = 1, reassigned = 0 }
    vcores_design       = { assigned = 1, reassigned = 0 }
    vpcs                = { assigned = 1, reassigned = 0 }
    network_connections = { assigned = 1, reassigned = 0 }
    # static_ips and vpns are server-managed; not settable via Terraform.
  }
}

resource "anypoint_environment" "sandbox" {
  provider        = anypoint.admin
  organization_id = anypoint_organization.commerce_bu.id
  name            = "my-dev"
  type            = "sandbox"
  is_production   = false
}

resource "anypoint_environment" "production" {
  provider        = anypoint.admin
  organization_id = anypoint_organization.commerce_bu.id
  name            = "my-prod"
  type            = "production"
  is_production   = true
}

###############################################################################
# Step 2 – Connected App Permissions, Private Space & Omni Gateway
# ------------------------------------------------------------------
# Grant our Connected App fine-grained access to the new org and
# environments, provision a Private Space, and deploy a Omni Gateway.
###############################################################################

# 2a. Grant the Connected App scopes for the new org and environments
resource "anypoint_connected_app_scopes" "app_permissions" {
  provider         = anypoint.admin
  connected_app_id = var.anypoint_client_id

  scopes = [
    { scope = "admin:cloudhub",          context_params = { org = anypoint_organization.commerce_bu.id } },
    { scope = "manage:runtime_fabrics",  context_params = { org = anypoint_organization.commerce_bu.id } },
    { scope = "manage:cloudhub_networking",   context_params = { org = anypoint_organization.commerce_bu.id } },
    { scope = "create:environment",          context_params = { org = anypoint_organization.commerce_bu.id } },
    { scope = "manage:api_groups",       context_params = { org = anypoint_organization.commerce_bu.id} },
    { scope = "manage:apis",             context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
    { scope = "manage:api_policies",     context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
    { scope = "manage:api_configuration",context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
    { scope = "manage:secret_groups",    context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
    { scope = "manage:secrets",          context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
    { scope = "manage:api_groups",       context_params = { org = anypoint_organization.commerce_bu.id } },
    { scope = "manage:apis",             context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.production.id } },
    { scope = "manage:api_policies",     context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.production.id } },
    { scope = "manage:api_configuration",context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.production.id } },
    { scope = "manage:secret_groups",    context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
    { scope = "manage:secrets",          context_params = { org = anypoint_organization.commerce_bu.id, envId = anypoint_environment.sandbox.id } },
  ]
}

# 2b. Provision a Private Space with Network for workload isolation
resource "anypoint_private_space_config" "private_space" {
  organization_id = anypoint_organization.commerce_bu.id
  name            = "commerce-private-space"
  enable_egress   = true

  network {
    region     = var.region
    cidr_block = "10.0.0.0/16"
  }

  depends_on = [anypoint_connected_app_scopes.app_permissions]
}

###############################################################################
# Step 2d – Secrets Management (Secret Group, Keystore, Truststore, TLS Context)

resource "anypoint_secret_group" "main" {
  environment_id = var.environment_id
  name           = "commerce-secrets-group"
  downloadable   = false
  depends_on = [anypoint_private_space_config.private_space]
}

resource "anypoint_secret_group_keystore" "tls" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "commerce-tls-keystore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/../certs/key.pem"))
  depends_on = [anypoint_secret_group.main]
}

resource "anypoint_secret_group_truststore" "ca" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "commerce-ca-truststore"
  type            = "PEM"

  truststore_base64 = base64encode(file("${path.module}/../certs/truststore.pem"))
  depends_on = [anypoint_secret_group.main]
}

resource "anypoint_secret_group_tls_context" "omni" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "commerce-omni-tls-context"

  keystore_id   = anypoint_secret_group_keystore.tls.id
  truststore_id = anypoint_secret_group_truststore.ca.id

  alpn_protocols = ["h2", "http/1.1"]
  depends_on = [anypoint_secret_group_keystore.tls, anypoint_secret_group_truststore.ca]
}

# 2e. Deploy a Managed Omni Gateway into the Private Space
resource "anypoint_managed_omni_gateway" "commerce-gateway" {
  environment_id  = var.environment_id
  name            = "commerce-gateway"
  target_id       = var.target_id

  depends_on = [anypoint_secret_group_tls_context.omni]
}

###############################################################################
# Step 3 – API Instance with Single Upstream
# --------------------------------------------
# Deploy an API from Exchange to the Omni Gateway with a single backend.
# For simple deployments, use upstream_uri for a single backend service.
###############################################################################

resource "anypoint_api_instance" "orders_api" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "orders-api"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.parent_organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = "/orders/v1"
  }

  gateway_id = anypoint_managed_omni_gateway.commerce-gateway.id
  upstream_uri = "http://orders-api.internal:8080"

  depends_on = [anypoint_managed_omni_gateway.commerce-gateway]
}

###############################################################################
# API Instance with Canary Routing (Multi-Upstream)
# ---------------------------------------------------
# Unlike agent_instance and mcp_server (which only support single upstream),
# api_instance supports multiple upstreams with weighted routing for
# canary deployments, A/B testing, or gradual rollouts.
###############################################################################

resource "anypoint_api_instance" "inventory_api_canary" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "inventory-api-canary"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.parent_organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = "/inventory/v1"
  }

  gateway_id = anypoint_managed_omni_gateway.commerce-gateway.id

  # Multi-upstream routing: 90% stable, 10% canary
  # This pattern is ONLY available for api_instance, NOT for agent/mcp resources
  routing = [
    {
      label = "Canary Deployment"
      upstreams = [
        {
          weight = 90
          uri    = "http://inventory-stable.internal:8080"
          label  = "Stable Version"
        },
        {
          weight = 10
          uri    = "http://inventory-canary.internal:8080"
          label  = "Canary Version"
        }
      ]
    }
  ]

  depends_on = [anypoint_managed_omni_gateway.commerce-gateway]
}

###############################################################################
# Step 4 – Security Policies
# ----------------------------
# Layer on enterprise security: JWT validation, rate limiting, and IP
# allowlisting — all as code, version-controlled, and auditable.
###############################################################################

# 4a. JWT Validation – only accept tokens from our identity provider
resource "anypoint_api_policy_jwt_validation" "orders_jwt" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.orders_api.id

  label           = "jwt-rsa"
  order           = 1
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

  depends_on = [anypoint_api_instance.orders_api]
}

# 4b. Rate Limiting – protect backend from traffic spikes
resource "anypoint_api_policy_rate_limiting" "orders_rate_limit" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.orders_api.id

  configuration = {
    rate_limits = [{
      maximum_requests = 100
      time_period_in_milliseconds = 60000
    }]
    expose_headers = true
    clusterizable  = true
  }

  order = 2
}

# 4c. IP Allowlist – restrict to known corporate IP ranges
resource "anypoint_api_policy_ip_allowlist" "orders_ip_allow" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.orders_api.id

  configuration = {
    ip_expression = "#[attributes.headers['x-forwarded-for']]"
    ips           = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  }

  order = 3
}

###############################################################################
# Step 5 – SLA Tiers
# ---------------------
# Define tiered rate limits for consumer self-service.
# Partners sign up, pick a tier, and get auto-approved or manually approved.
###############################################################################

resource "anypoint_api_instance_sla_tier" "gold" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.orders_api.id

  name         = "Gold"
  description  = "Premium tier – 1000 req/min"
  auto_approve = false
  status       = "ACTIVE"

  limits = [{
    visible                     = true
    maximum_requests            = 1000
    time_period_in_milliseconds = 60000
  }]
}

resource "anypoint_api_instance_sla_tier" "silver" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.orders_api.id

  name         = "Silver"
  description  = "Standard tier – 200 req/min"
  auto_approve = true
  status       = "ACTIVE"

  limits = [{
    visible                     = true
    maximum_requests            = 200
    time_period_in_milliseconds = 60000
  }]
}

resource "anypoint_api_instance_sla_tier" "trial" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.orders_api.id

  name         = "Trial"
  description  = "Evaluation tier – 10 req/min, auto-approved"
  auto_approve = true
  status       = "ACTIVE"

  limits = [{
    visible                     = true
    maximum_requests            = 10
    time_period_in_milliseconds = 60000
  }]
}

###############################################################################
# Step 6 – API Group
# ---------------------
# Bundle the Orders API with a Payments API into a single group so
# consumers can subscribe to a unified product with a single contract.
###############################################################################

resource "anypoint_api_instance" "payments_api" {
  organization_id = var.parent_organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "payments-api"

  spec = {
    asset_id = var.api_asset_id
    group_id = var.parent_organization_id
    version  = var.api_asset_version
  }

  endpoint = {
    deployment_type = "HY"
    type            = "http"
    base_path       = "/payments/v1"
  }

  gateway_id   = anypoint_managed_omni_gateway.commerce-gateway.id
  upstream_uri = "http://payments.internal:8080"

  depends_on = [anypoint_managed_omni_gateway.commerce-gateway]
}


resource "anypoint_managed_omni_gateway" "commerce-suborg-gateway" {
  organization_id = anypoint_organization.commerce_bu.id
  environment_id  = anypoint_environment.sandbox.id
  name            = "commerce-suborg-gateway"
  target_id       = anypoint_private_space_config.private_space.id

  depends_on = [anypoint_private_space_config.private_space]
}
