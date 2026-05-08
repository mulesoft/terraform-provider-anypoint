terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Example 1: Configure advanced settings using default organization (from provider credentials)
resource "anypoint_privatespace_advanced_config" "example" {
  private_space_id = var.private_space_id

  ingress_configuration = {
    read_response_timeout = "600"
    protocol              = "https-redirect"
    
    logs = {
      port_log_level = "INFO"
      filters = []
    }

    deployment = {
      status              = "APPLIED"
      last_seen_timestamp = 1753719215000
    }
  }

  enable_iam_role = true
}

# Example 2: Configure advanced settings in a specific organization
resource "anypoint_privatespace_advanced_config" "example_custom_org" {
  private_space_id = var.private_space_id_custom_org
  organization_id  = var.custom_organization_id  # Optional: specify different org

  ingress_configuration = {
    read_response_timeout = "800"
    protocol              = "https-redirect"
    
    logs = {
      port_log_level = "INFO"
      filters = []
    }

    deployment = {
      status              = "APPLIED"
      last_seen_timestamp = 1753719215000
    }
  }

  enable_iam_role = true
}

# Output the configuration
output "privatespace_advanced_config_id" {
  value = anypoint_privatespace_advanced_config.example.id
}

output "ingress_configuration" {
  value = anypoint_privatespace_advanced_config.example.ingress_configuration
}

output "enable_iam_role" {
  value = anypoint_privatespace_advanced_config.example.enable_iam_role
}