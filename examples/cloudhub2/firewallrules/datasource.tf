# terraform {
#   required_providers {
#     anypoint = {
#       source = "sf.com/mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# Data source to retrieve existing firewall rules
# data "anypoint_firewallrules" "existing" {
#   private_space_id = "f7dcdb6c-017d-4989-8d87-28e8477412e0"
# }

# # Output the retrieved firewall rules
# output "existing_firewall_rules" {
#   description = "List of existing firewall rules"
#   value       = data.anypoint_firewallrules.existing.rules
# }

# output "existing_firewall_rules_count" {
#   description = "Number of existing firewall rules"
#   value       = length(data.anypoint_firewallrules.existing.rules)
# }

# output "existing_firewall_inbound_rules" {
#   description = "List of inbound firewall rules"
#   value = [
#     for rule in data.anypoint_firewallrules.existing.rules : rule
#     if rule.type == "inbound"
#   ]
# }

# output "existing_firewall_outbound_rules" {
#   description = "List of outbound firewall rules"
#   value = [
#     for rule in data.anypoint_firewallrules.existing.rules : rule
#     if rule.type == "outbound"
#   ]
# }

# output "existing_firewall_https_rules" {
#   description = "List of HTTPS (port 443) firewall rules"
#   value = [
#     for rule in data.anypoint_firewallrules.existing.rules : rule
#     if rule.from_port == 443 && rule.to_port == 443
#   ]
# } 