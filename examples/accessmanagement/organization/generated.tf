# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "a02fab4f-4695-4325-882e-f326d1cef704"
resource "anypoint_organization" "imported_org" {
  provider = anypoint.admin
  entitlements = {
    create_environments = true
    create_sub_orgs     = false
    design_center = {
      api    = true
      mozart = false
    }
    omni_gateway = null
    gateways = {
      assigned = 0
    }
    global_deployment = false
    hybrid = {
      enabled = true
    }
    load_balancer = {
      assigned = 0
    }
    managed_gateway_large = {
      assigned = 0
    }
    managed_gateway_small = {
      assigned = 0
    }
    mq_messages = {
      add_on = 0
      base   = 0
    }
    mq_requests = {
      add_on = 0
      base   = 0
    }
    network_connections = {
      assigned   = 0
      reassigned = 0
    }
    runtime_fabric = true
    service_mesh   = null
    vcores_design = {
      assigned   = 0
      reassigned = 0
    }
    vcores_production = {
      assigned   = 0
      reassigned = 0
    }
    vcores_sandbox = {
      assigned   = 0
      reassigned = 0
    }
    vpcs = {
      assigned   = 0
      reassigned = 0
    }
    worker_logging_override = {
      enabled = false
    }
  }
  name                   = "terraform-suborg-example-renamed"
  owner_id               = "f7f43384-b33e-470c-ad4c-285aa0c01212"
  parent_organization_id = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}
