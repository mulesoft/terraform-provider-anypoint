# Retrieve a single API policy by ID
data "anypoint_api_policy" "rate_limit" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  policy_id       = var.policy_id
  organization_id = var.organization_id
}

# Output policy details
output "policy_template" {
  value = data.anypoint_api_policy.rate_limit.policy_template_id
}

output "policy_configuration" {
  value = data.anypoint_api_policy.rate_limit.configuration_json
}

output "policy_disabled" {
  value = data.anypoint_api_policy.rate_limit.disabled
}

# List all policies for an API instance
data "anypoint_api_policies" "all" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  organization_id = var.organization_id
}

# Output all policies
output "all_policies" {
  value = data.anypoint_api_policies.all.policies
}

# Output policy count
output "policy_count" {
  value = length(data.anypoint_api_policies.all.policies)
}
