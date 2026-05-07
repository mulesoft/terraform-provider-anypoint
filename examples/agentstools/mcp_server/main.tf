###############################################################################
# Anypoint MCP Server Example
# ============================
# This example demonstrates creating an MCP (Model Context Protocol) server
# instance in API Manager for exposing AI tools and resources to agents.
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply
###############################################################################

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

###############################################################################
# MCP Server - Atlassian Integration
# ------------------------------------
# Creates an MCP server that exposes Atlassian (Jira, Confluence) tools
# to AI agents via the Model Context Protocol
###############################################################################

resource "anypoint_mcp_server" "atlassian_mcp" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "atlassian-mcp-server"

  # Exchange asset specification for the MCP server
  spec = {
    asset_id = var.mcp_asset_id
    group_id = var.organization_id
    version  = var.mcp_asset_version
  }

  # Endpoint configuration - base_path is used to construct proxyUri
  endpoint = {
    deployment_type = "HY"
    base_path       = "mcp1"
  }

  # Omni Gateway deployment
  gateway_id = var.gateway_id

  # Backend MCP server implementation
  upstream_uri = "http://example.com"
}

###############################################################################
# MCP Server - Salesforce Integration
# -------------------------------------
# Creates an MCP server that provides Salesforce CRM access to agents
###############################################################################

resource "anypoint_mcp_server" "salesforce_mcp" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "salesforce-mcp-server"

  spec = {
    asset_id = "mcp-server"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "mcp2"
  }

  gateway_id = var.gateway_id

  upstream_uri = "http://salesforce-mcp.internal:8080"
}

###############################################################################
# MCP Server - Enterprise Tools
# -------------------------------
# MCP server exposing enterprise productivity tools.
# Note: Unlike API instances, MCP servers always route to a single
# upstream with 100% weight. Multi-upstream routing is not supported.
###############################################################################

resource "anypoint_mcp_server" "enterprise_tools_mcp" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "enterprise-tools-mcp"

  spec = {
    asset_id = "postman-mcp-server"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "mcp-tools"
  }

  gateway_id = var.gateway_id

  # Single upstream - MCP servers always have one upstream with 100% weight
  upstream_uri = "http://mcp-tools.internal:8080"
}

###############################################################################
# MCP Server Policies
# --------------------
# Apply MCP-specific policies to the Atlassian MCP server using typed
# policy resources. Each anypoint_api_policy_mcp_* resource provides
# native HCL configuration blocks with typed fields.
###############################################################################

# ─── 1. MCP PII Detector ─────────────────────────────────────
resource "anypoint_api_policy_mcp_pii_detector" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {
    entities = ["Email", "US SSN", "Credit Card", "Phone Number"]
  }
}

# ─── 2. MCP Schema Validation ────────────────────────────────
resource "anypoint_api_policy_mcp_schema_validation" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {
    validate_tool_schema = true
  }
}

# ─── 3. MCP Access Control ───────────────────────────────────
resource "anypoint_api_policy_mcp_access_control" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {
    rules = [
      "permit(principal,action,resource);",
      "permit(principal,action1,resource);"
    ]
    auth_type = "ClientId"
  }
}

# ─── 4. MCP Support ──────────────────────────────────────────
resource "anypoint_api_policy_mcp_support" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {}
}

# ─── 5. MCP Global Access Policy ─────────────────────────────
resource "anypoint_api_policy_mcp_global_access_policy" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {
    rules = [
      {
        rule       = "Allow"
        match_type = "literal"
        value      = "jira-search"
      },
      {
        rule       = "Block"
        match_type = "pattern"
        value      = "admin-*"
      }
    ]
  }
}

# ─── 6. MCP Tool Mapping ─────────────────────────────────────
resource "anypoint_api_policy_mcp_tool_mapping" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {
    tool_mappings = [
      {
        sourceToolName    = "jira_create_issue"
        mappedToolName    = "create_ticket"
        mappedDescription = "Create a new Jira ticket"
        mappingType       = "literal"
      },
      {
        sourceToolName    = "confluence_search"
        mappedToolName    = "search_docs"
        mappedDescription = "Search Confluence documentation"
        mappingType       = "regex"
      }
    ]
    log_mappings = true
  }
}

# ─── 7. MCP Transcoding Router ───────────────────────────────
resource "anypoint_api_policy_mcp_transcoding_router" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id

  configuration = {
    transcoding_path = "/mcp"
    routes = [
      {
        upstreamName = "jira-backend"
        tools        = ["jira_create_issue", "jira_search"]
        resources    = ["jira://projects"]
        prompts      = ["jira-summary"]
      },
      {
        upstreamName = "confluence-backend"
        tools        = ["confluence_search"]
        resources    = ["confluence://spaces"]
        prompts      = []
      }
    ]
  }
}

###############################################################################
# LLM Provider Outbound Policies
# --------------------------------
# Outbound policies for LLM provider transcoding and credential injection.
# These use the xapi/v1 outbound-policies endpoint and require upstream_ids.
###############################################################################

# ─── 8. Bedrock LLM Provider ────────────────────────────────
resource "anypoint_api_policy_bedrock_llm_provider_policy" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id
  upstream_ids    = [anypoint_mcp_server.atlassian_mcp.upstream_id]

  configuration = {
    aws_access_key_id     = "your-aws-access-key-id"
    aws_secret_access_key = "your-aws-secret-access-key"
    aws_session_token     = "your-aws-session-token"
    aws_region            = "us-east-1"
    service_name          = "bedrock"
    timeout               = 60000
  }
}

# ─── 9. Gemini LLM Provider ─────────────────────────────────
resource "anypoint_api_policy_gemini_llm_provider_policy" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id
  upstream_ids    = [anypoint_mcp_server.atlassian_mcp.upstream_id]

  configuration = {
    api_key = "your-gemini-api-key"
    timeout = 60000
    model_mapper = [
      { from = "generic-large", to = "gpt-4" },
      { from = "generic-small", to = "gpt-5" }
    ]
  }
}

# ─── 10. OpenAI Transcoding ─────────────────────────────────
resource "anypoint_api_policy_openai_transcoding_policy" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id
  upstream_ids    = [anypoint_mcp_server.atlassian_mcp.upstream_id]

  configuration = {
    api_key = "your-openai-api-key"
    timeout = 60000
    model_mapper = [
      { from = "alias-large", to = "gpt-4" },
      { from = "alias-small", to = "gpt-5" }
    ]
  }
}

# ─── 11. Gemini Transcoding ─────────────────────────────────
resource "anypoint_api_policy_gemini_transcoding_policy" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_server.atlassian_mcp.id
  upstream_ids    = [anypoint_mcp_server.atlassian_mcp.upstream_id]

  configuration = {}
}
