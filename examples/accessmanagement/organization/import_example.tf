# -----------------------------------------------------------------------------
# Importing an existing Anypoint organization into Terraform
# -----------------------------------------------------------------------------
#
# The resource block below is commented out by default so `terraform plan` in
# this example directory stays clean. To adopt an existing organization into
# state follow these three steps:
#
#   1. Uncomment the `resource "anypoint_organization" "imported_org"` block
#      and edit `name`, `parent_organization_id`, and `owner_id` to match the
#      actual values on the server. You can look these up in the Anypoint UI
#      or with `curl` against the Access Management API.
#
#   2. Run `terraform import` with the organization's UUID:
#
#        terraform import anypoint_organization.imported_org <organization-uuid>
#
#      e.g.
#
#        terraform import anypoint_organization.imported_org 00000000-0000-0000-0000-000000000000
#
#   3. Run `terraform plan`. If your HCL (step 1) matches the server, the plan
#      is empty — you now own the org in Terraform. If `parent_organization_id`
#      doesn't match, UPDATE THE HCL, don't apply: that attribute has
#      RequiresReplace and would otherwise destroy+recreate the resource.
#
# The first refresh hydrates every Read-Only / Optional attribute (entitlements,
# subscription, environments, timestamps, client_id, etc.) from the Anypoint
# API. `parent_organization_id` is derived from the tail of the server-returned
# ancestor chain on first refresh; see docs/resources/organization.md for the
# full contract.

# resource "anypoint_organization" "imported_org" {
#   provider = anypoint.admin
#
#   name                   = "My Existing Organization"
#   parent_organization_id = "<add-your-value-here>"
#   owner_id               = "<add-your-value-here>"
#
#   # Entitlements are Optional+Computed — omitting the block is equivalent to
#   # `entitlements = {}` and lets state reflect whatever the server reports.
#   # Add overrides only for entitlements you actually want to manage.
# }
#
# output "imported_org_id" {
#   value = anypoint_organization.imported_org.id
# }
#
# output "imported_org_entitlements" {
#   value = anypoint_organization.imported_org.entitlements
# }
