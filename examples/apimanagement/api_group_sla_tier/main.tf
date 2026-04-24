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

###############################################################################
# 1. Bronze Tier – Standard Access (auto-approved)
###############################################################################
resource "anypoint_api_group_sla_tier" "bronze" {
  organization_id  = var.organization_id
  environment_id   = var.environment_id
  group_instance_id = var.group_instance_id

  name        = "Bronze"
  description = "Standard tier for regular consumers"
  status      = "ACTIVE"
  auto_approve = true

  default_limits = [
    {
      maximum_requests            = 100
      time_period_in_milliseconds = 60000      # 1 minute
      visible                     = true
    },
    {
      maximum_requests            = 5000
      time_period_in_milliseconds = 3600000    # 1 hour
      visible                     = true
    }
  ]
}

###############################################################################
# 2. Silver Tier – High-volume Access
###############################################################################
resource "anypoint_api_group_sla_tier" "silver" {
  organization_id  = var.organization_id
  environment_id   = var.environment_id
  group_instance_id = var.group_instance_id

  name        = "Silver"
  description = "High-volume tier for enterprise consumers"
  status      = "ACTIVE"
  auto_approve = false

  default_limits = [
    {
      maximum_requests            = 500
      time_period_in_milliseconds = 60000      # 1 minute
      visible                     = true
    },
    {
      maximum_requests            = 25000
      time_period_in_milliseconds = 3600000    # 1 hour
      visible                     = true
    },
    {
      maximum_requests            = 500000
      time_period_in_milliseconds = 86400000   # 1 day
      visible                     = true
    }
  ]
}

###############################################################################
# 3. Gold Tier – Unlimited Access (requires manual approval)
###############################################################################
resource "anypoint_api_group_sla_tier" "gold" {
  organization_id  = var.organization_id
  environment_id   = var.environment_id
  group_instance_id = var.group_instance_id

  name        = "Gold"
  description = "Premium tier with effectively unlimited access"
  status      = "ACTIVE"
  auto_approve = false

  default_limits = [
    {
      maximum_requests            = 999999999
      time_period_in_milliseconds = 60000      # 1 minute – effectively unlimited
      visible                     = true
    }
  ]
}

###############################################################################
# 4. Trial Tier – Limited developer access (auto-approved)
###############################################################################
resource "anypoint_api_group_sla_tier" "trial" {
  organization_id  = var.organization_id
  environment_id   = var.environment_id
  group_instance_id = var.group_instance_id

  name        = "Trial"
  description = "Trial tier for developers evaluating the API group"
  status      = "ACTIVE"
  auto_approve = true

  default_limits = [
    {
      maximum_requests            = 10
      time_period_in_milliseconds = 60000      # 1 minute
      visible                     = true
    },
    {
      maximum_requests            = 200
      time_period_in_milliseconds = 3600000    # 1 hour
      visible                     = true
    }
  ]
}

###############################################################################
# Outputs
###############################################################################
output "bronze_tier_id" {
  description = "ID of the Bronze SLA tier."
  value       = anypoint_api_group_sla_tier.bronze.id
}

output "silver_tier_id" {
  description = "ID of the Silver SLA tier."
  value       = anypoint_api_group_sla_tier.silver.id
}

output "gold_tier_id" {
  description = "ID of the Gold SLA tier."
  value       = anypoint_api_group_sla_tier.gold.id
}

output "trial_tier_id" {
  description = "ID of the Trial SLA tier."
  value       = anypoint_api_group_sla_tier.trial.id
}

output "all_tier_ids" {
  description = "Map of tier name → tier ID."
  value = {
    bronze = anypoint_api_group_sla_tier.bronze.id
    silver = anypoint_api_group_sla_tier.silver.id
    gold   = anypoint_api_group_sla_tier.gold.id
    trial  = anypoint_api_group_sla_tier.trial.id
  }
}
