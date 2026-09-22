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

# ---------------------------------------------------------------------------
# Import an existing MCP bridge into Terraform state.
#
# Steps:
#   1. cp terraform.tfvars.example terraform.tfvars and fill in credentials,
#      organization_id, environment_id, and mcp_bridge_id.
#   2. Uncomment the import block below. Leave the resource block commented
#      if you will generate config:
#        terraform init
#        terraform plan -generate-config-out=generated.tf
#        terraform apply
#   3. Or uncomment both the import block and the resource stub, fill the
#      remaining placeholders, then: terraform init && terraform apply
#   4. terraform plan — description and input_schema come back null (they
#      live on the generated Exchange MCP asset). Adjust generated.tf if needed.
#
# Import ID format:
#   anypoint_mcp_bridge -> organization_id/environment_id/mcp_bridge_id
#
# The mcp_bridge_id is the numeric ID from Anypoint API Manager
# (visible in the URL when viewing the instance, e.g. "21058094").
#
# Import reconstructs source_apis (including tls_context_id and custom
# http_mapping). Instance settings are recovered.
# ---------------------------------------------------------------------------

# import {
#   to = anypoint_mcp_bridge.imported
#   id = "${var.organization_id}/${var.environment_id}/${var.mcp_bridge_id}"
# }

# resource "anypoint_mcp_bridge" "imported" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   gateway_id      = var.gateway_id
#
#   mcp_asset_name = "<mcp_asset_name>"
#   port           = 8081
#   base_path      = "<base_path>"
#
#   instance_label    = "<instance_label>"
#   approval_method   = "manual"
#   consumer_endpoint = "https://<ingress>/<base_path>"
#
#   source_apis = [
#     {
#       label        = "<source_api_label>"
#       upstream_uri = "https://<backend-host>"
#       asset_id     = "<source_rest_api_asset_id>"
#       version      = "1.0.0"
#       tools = [
#         { method = "GET", path = "/example/{id}" },
#       ]
#     },
#   ]
# }
