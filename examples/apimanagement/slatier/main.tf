terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# ─────────────────────────────────────────────────────────────
# 1. Gold Tier - Unlimited Access
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "gold" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  name        = "Gold"
  description = "Gold tier with unlimited API access for premium customers"

  # Unlimited access - effectively no rate limits
  limits = [
    {
      time_period_in_milliseconds = 60000      # 1 minute
      maximum_requests            = 999999999  # Effectively unlimited
      visible                     = true
    }
  ]

  auto_approve = true
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 2. Silver Tier - High Volume
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "silver" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Silver"
  description = "Silver tier with high volume limits for standard customers"

  # Multiple rate limit windows
  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 1000
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 50000
      visible                     = true
    },
    {
      time_period_in_milliseconds = 86400000  # 1 day
      maximum_requests            = 1000000
      visible                     = true
    }
  ]

  auto_approve = true
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 3. Bronze Tier - Standard Access
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "bronze" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Bronze"
  description = "Bronze tier with standard limits for regular customers"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 100
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 5000
      visible                     = true
    }
  ]

  auto_approve = true
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 4. Trial Tier - Limited Access with Manual Approval
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "trial" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Trial"
  description = "Trial tier with limited access requiring approval"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 10
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 500
      visible                     = true
    },
    {
      time_period_in_milliseconds = 86400000  # 1 day
      maximum_requests            = 5000
      visible                     = true
    }
  ]

  auto_approve = false  # Requires manual approval
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 5. Developer Tier - For Development/Testing
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "developer" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Developer"
  description = "Developer tier for testing and development purposes"

  limits = [
    {
      time_period_in_milliseconds = 1000  # 1 second
      maximum_requests            = 2
      visible                     = true
    },
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 50
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 1000
      visible                     = true
    }
  ]

  auto_approve = true
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 6. Partner Tier - For Trusted Partners
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "partner" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Partner"
  description = "Partner tier for trusted integration partners"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 500
      visible                     = true
    },
    {
      time_period_in_milliseconds = 3600000  # 1 hour
      maximum_requests            = 25000
      visible                     = true
    },
    {
      time_period_in_milliseconds = 86400000  # 1 day
      maximum_requests            = 500000
      visible                     = true
    }
  ]

  auto_approve = false  # Partners require manual approval
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 7. Free Tier - Public Access
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "free" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Free"
  description = "Free tier with basic access for public users"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 5
      visible                     = true
    },
    {
      time_period_in_milliseconds = 86400000  # 1 day
      maximum_requests            = 1000
      visible                     = true
    }
  ]

  auto_approve = true
  status       = "ACTIVE"
}

# ─────────────────────────────────────────────────────────────
# 8. Deprecated Tier - Marked for Removal
# ─────────────────────────────────────────────────────────────
resource "anypoint_api_instance_sla_tier" "deprecated" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id

  name        = "Legacy-v1"
  description = "Legacy tier - deprecated, use Bronze or higher"

  limits = [
    {
      time_period_in_milliseconds = 60000  # 1 minute
      maximum_requests            = 20
      visible                     = true
    }
  ]

  auto_approve = false
  status       = "DEPRECATED"
}

# ─────────────────────────────────────────────────────────────
# Outputs
# ─────────────────────────────────────────────────────────────
output "gold_tier_id" {
  description = "ID of the Gold tier"
  value       = anypoint_api_instance_sla_tier.gold.id
}

output "silver_tier_id" {
  description = "ID of the Silver tier"
  value       = anypoint_api_instance_sla_tier.silver.id
}

output "bronze_tier_id" {
  description = "ID of the Bronze tier"
  value       = anypoint_api_instance_sla_tier.bronze.id
}

output "trial_tier_id" {
  description = "ID of the Trial tier"
  value       = anypoint_api_instance_sla_tier.trial.id
}

output "developer_tier_id" {
  description = "ID of the Developer tier"
  value       = anypoint_api_instance_sla_tier.developer.id
}

output "partner_tier_id" {
  description = "ID of the Partner tier"
  value       = anypoint_api_instance_sla_tier.partner.id
}

output "free_tier_id" {
  description = "ID of the Free tier"
  value       = anypoint_api_instance_sla_tier.free.id
}

output "all_tier_ids" {
  description = "All created SLA tier IDs"
  value = {
    gold       = anypoint_api_instance_sla_tier.gold.id
    silver     = anypoint_api_instance_sla_tier.silver.id
    bronze     = anypoint_api_instance_sla_tier.bronze.id
    trial      = anypoint_api_instance_sla_tier.trial.id
    developer  = anypoint_api_instance_sla_tier.developer.id
    partner    = anypoint_api_instance_sla_tier.partner.id
    free       = anypoint_api_instance_sla_tier.free.id
    deprecated = anypoint_api_instance_sla_tier.deprecated.id
  }
}

output "all_tier_names" {
  description = "All created SLA tier names"
  value = {
    gold       = anypoint_api_instance_sla_tier.gold.name
    silver     = anypoint_api_instance_sla_tier.silver.name
    bronze     = anypoint_api_instance_sla_tier.bronze.name
    trial      = anypoint_api_instance_sla_tier.trial.name
    developer  = anypoint_api_instance_sla_tier.developer.name
    partner    = anypoint_api_instance_sla_tier.partner.name
    free       = anypoint_api_instance_sla_tier.free.name
    deprecated = anypoint_api_instance_sla_tier.deprecated.name
  }
}
