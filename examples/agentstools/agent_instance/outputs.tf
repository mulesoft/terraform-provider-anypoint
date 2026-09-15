###############################################################################
# Outputs
###############################################################################

output "agent_instance_id" {
  description = "Numeric ID of the Agent instance"
  value       = anypoint_agent_instance.bedrock.id
}

output "agent_instance_status" {
  description = "Current status of the Agent instance"
  value       = anypoint_agent_instance.bedrock.status
}

output "agent_instance_asset_id" {
  description = "Exchange asset ID computed from the Agent response"
  value       = anypoint_agent_instance.bedrock.asset_id
}
