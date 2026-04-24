###############################################################################
# Outputs
###############################################################################

output "customer_support_agent_id" {
  description = "Customer support agent instance ID"
  value       = anypoint_agent_instance.customer_support_agent.id
}

output "customer_support_agent_status" {
  description = "Customer support agent instance status"
  value       = anypoint_agent_instance.customer_support_agent.status
}

output "sales_agent_id" {
  description = "Sales agent instance ID"
  value       = anypoint_agent_instance.sales_agent.id
}

output "sales_agent_status" {
  description = "Sales agent instance status"
  value       = anypoint_agent_instance.sales_agent.status
}
