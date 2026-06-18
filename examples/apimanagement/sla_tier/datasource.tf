# Retrieve a single SLA tier by ID
data "anypoint_api_instance_sla_tier" "gold" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  tier_id         = var.sla_tier_id
  organization_id = var.organization_id
}

# Output SLA tier details
output "tier_name" {
  value = data.anypoint_api_instance_sla_tier.gold.name
}

output "tier_auto_approve" {
  value = data.anypoint_api_instance_sla_tier.gold.auto_approve
}

output "tier_limits" {
  value = data.anypoint_api_instance_sla_tier.gold.limits
}

# List all SLA tiers for an API instance
data "anypoint_api_instance_sla_tiers" "all" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  organization_id = var.organization_id
}

# Output all SLA tiers
output "all_tiers" {
  value = data.anypoint_api_instance_sla_tiers.all.tiers
}

# Output active tiers
output "active_tiers" {
  value = [
    for tier in data.anypoint_api_instance_sla_tiers.all.tiers :
    tier if tier.status == "ACTIVE"
  ]
}
