
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
  technology      = "flexGateway"
  instance_label  = "sparq-agt-01-bedrock"
  gateway_id      = var.gateway_id

  spec = {
    asset_id = var.agent_asset_id
    group_id = var.organization_id
    version  = var.agent_asset_version
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "/agt/bedrock"
    type            = "a2a"
  }

  upstream_uri = "https://bedrock-agent-runtime.us-east-1.amazonaws.com"
}