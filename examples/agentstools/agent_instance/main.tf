###############################################################################
# Anypoint Agent Instance Example
# ================================
# Creates an Agent instance on an Omni Gateway, routing to a backend agent
# service. An Agent instance is similar to an API instance but purpose-built
# for AI agents.
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply
#
# Note: Unlike API instances, an Agent instance always routes to a single
# upstream (100% weight). Multi-upstream weighted routing is not supported for
# agents — use `upstream_uri` (not `routing`).
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

resource "anypoint_agent_instance" "bedrock" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "omniGateway"
  instance_label  = "sparq-agt-01-bedrock"
  gateway_id      = var.gateway_id

  # Exchange asset specification for the agent
  spec = {
    asset_id = var.agent_asset_id
    group_id = var.organization_id
    version  = var.agent_asset_version
  }

  # Endpoint configuration
  endpoint = {
    deployment_type = "HY"
    base_path       = "/agt/bedrock"
    type            = "a2a"
  }

  # Backend agent service URI (single upstream, 100% weight)
  upstream_uri = "https://bedrock-agent-runtime.us-east-1.amazonaws.com"
}
