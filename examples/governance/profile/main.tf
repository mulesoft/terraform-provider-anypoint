terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
    }
  }
}

variable "organization_id" {
  description = "Anypoint Organization ID"
  type        = string
}

variable "mulesoft_ruleset_group_id" {
  description = "Exchange group ID for MuleSoft-provided rulesets"
  type        = string
  default     = "68ef9520-24e9-4cf2-b2f5-620025690913"
}

# ─── Basic governance profile with two rulesets ──────────────────────────────
resource "anypoint_api_governance_profile" "best_practices" {
  organization_id = var.organization_id
  name            = "API Best Practices"
  description     = "Enforce best practices across all HTTP APIs"
  filter          = "scope:http-api"

  rulesets = [
    {
      group_id = var.mulesoft_ruleset_group_id
      asset_id = "api-catalog-information-best-practices"
      version  = "latest"
    },
    {
      group_id = var.mulesoft_ruleset_group_id
      asset_id = "api-documentation-best-practices"
      version  = "latest"
    }
  ]

  notification_config = {
    enabled = true
    notifications = [
      {
        enabled   = true
        condition = "OnFailure"
        recipients = [
          {
            contact_type      = "Publisher"
            notification_type = "Email"
            value             = ""
            label             = ""
          }
        ]
      }
    ]
  }
}

# ─── Security-focused profile ────────────────────────────────────────────────
resource "anypoint_api_governance_profile" "security" {
  organization_id = var.organization_id
  name            = "Security Standards"
  description     = "Enforce security best practices"
  filter          = "scope:http-api"

  rulesets = [
    {
      group_id = var.mulesoft_ruleset_group_id
      asset_id = "authentication-security-best-practices"
      version  = "latest"
    },
    {
      group_id = var.mulesoft_ruleset_group_id
      asset_id = "https-enforcement"
      version  = "latest"
    }
  ]

  notification_config = {
    enabled = true
    notifications = [
      {
        enabled   = true
        condition = "OnFailure"
        recipients = [
          {
            contact_type      = "Publisher"
            notification_type = "Email"
            value             = ""
            label             = ""
          }
        ]
      }
    ]
  }
}

# ─── Outputs ─────────────────────────────────────────────────────────────────
output "best_practices_profile_id" {
  value = anypoint_api_governance_profile.best_practices.id
}

output "security_profile_id" {
  value = anypoint_api_governance_profile.security.id
}
