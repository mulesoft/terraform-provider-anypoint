###############################################################################
# Complete Agent Tools Example
# ==============================
# This example demonstrates a complete AI agents setup with:
#   - Multiple MCP servers providing tool access
#   - Agent instances that consume those MCP servers
#   - Proper routing and deployment configuration
#
# Architecture:
#   Agent Instance → Omni Gateway → MCP Servers → Backend Systems
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply
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

###############################################################################
# MCP Servers - Tool Providers
# ------------------------------
# Create MCP servers that expose various enterprise tools to AI agents
###############################################################################

# Atlassian MCP Server - Jira & Confluence access
resource "anypoint_mcp_server" "atlassian" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "atlassian-mcp"

  spec = {
    asset_id = "atlassian-mcp-server"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "mcp/atlassian"
  }

  gateway_id   = var.gateway_id
  upstream_uri = "http://atlassian-mcp.internal:8080"
}

# Salesforce MCP Server - CRM data access
resource "anypoint_mcp_server" "salesforce" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "salesforce-mcp"

  spec = {
    asset_id = "salesforce-mcp-server"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "mcp/salesforce"
  }

  gateway_id   = var.gateway_id
  upstream_uri = "http://salesforce-mcp.internal:8080"
}

# Database MCP Server - Analytics & reporting
resource "anypoint_mcp_server" "database" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "database-mcp"

  spec = {
    asset_id = "database-mcp-server"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "mcp/database"
  }

  gateway_id   = var.gateway_id
  upstream_uri = "http://database-mcp.internal:8080"
}

###############################################################################
# Agent Instances - AI Agents
# -----------------------------
# Create agent instances that can access the MCP servers above
###############################################################################

# Customer Support Agent - Handles customer inquiries
resource "anypoint_agent_instance" "customer_support" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "customer-support-agent"

  spec = {
    asset_id = "customer-support-agent"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "agent/support"
  }

  gateway_id   = var.gateway_id
  upstream_uri = "http://support-agent.internal:8080"

  depends_on = [
    anypoint_mcp_server.atlassian,
    anypoint_mcp_server.salesforce
  ]
}

# Sales Agent - Provides sales assistance with A/B testing
resource "anypoint_agent_instance" "sales" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "sales-agent"

  spec = {
    asset_id = "sales-agent"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "agent/sales"
  }

  gateway_id = var.gateway_id

  # A/B test: 80% stable model, 20% new model
  routing = [
    {
      label = "Sales Agent A/B Test"
      upstreams = [
        {
          weight = 80
          uri    = "http://sales-agent-stable.internal:8080"
          label  = "Stable Model"
        },
        {
          weight = 20
          uri    = "http://sales-agent-new.internal:8080"
          label  = "New Model"
        }
      ]
    }
  ]

  depends_on = [
    anypoint_mcp_server.salesforce,
    anypoint_mcp_server.database
  ]
}

# Analytics Agent - Generates reports and insights
resource "anypoint_agent_instance" "analytics" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  technology      = "flexGateway"
  instance_label  = "analytics-agent"

  spec = {
    asset_id = "analytics-agent"
    group_id = var.organization_id
    version  = "1.0.0"
  }

  endpoint = {
    deployment_type = "HY"
    base_path       = "agent/analytics"
  }

  gateway_id   = var.gateway_id
  upstream_uri = "http://analytics-agent.internal:8080"

  depends_on = [
    anypoint_mcp_server.database,
    anypoint_mcp_server.salesforce
  ]
}

###############################################################################
# Data Sources - Query deployed agents and MCP servers
# -----------------------------------------------------
###############################################################################

data "anypoint_agent_instances" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id

  depends_on = [
    anypoint_agent_instance.customer_support,
    anypoint_agent_instance.sales,
    anypoint_agent_instance.analytics
  ]
}

data "anypoint_mcp_servers" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id

  depends_on = [
    anypoint_mcp_server.atlassian,
    anypoint_mcp_server.salesforce,
    anypoint_mcp_server.database
  ]
}
