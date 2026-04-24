###############################################################################
# Complete Sub-Organization with Private Space Flow
# --------------------------------------------------
# This example demonstrates the complete flow:
#   1. Create a new sub-organization within Salesforce org
#   2. Create 2 environments in the sub-organization
#   3. Assign connected app scopes to existing connected app
#   4. Create a private space in one environment
#   5. Create a private network in the private space
#
# Prerequisites:
#   - Parent organization ID (Salesforce org)
#   - Owner user ID (existing user in parent org)
#   - Connected app with client_id: e5a776d9862a4f2d8f61ba8450803908
#   - User authentication (username/password) for connected app scope assignment
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

###############################################################################
# Provider Configuration with Aliases
# --------------------------------------------------
# Two provider instances for different credential sets:
#   1. "admin" - Admin credentials with user authentication (username/password)
#                Used for privileged operations like org creation, scope assignment
#   2. "normal_user" - Normal user credentials with connected app authentication
#                      Used for standard operations like environment management
#
# Environment variables alternative:
#   Admin: TF_VAR_anypoint_admin_client_id, TF_VAR_anypoint_admin_client_secret,
#          TF_VAR_anypoint_admin_username, TF_VAR_anypoint_admin_password
#   Normal: TF_VAR_anypoint_normal_client_id, TF_VAR_anypoint_normal_client_secret
###############################################################################

# Admin provider - Used for privileged operations
provider "anypoint" {
  alias         = "admin"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
  auth_type     = "user"
}

# Normal user provider - Used for standard operations
# Note: Only configure if you have normal user credentials
# Otherwise, comment this out and use provider.admin for all resources
provider "anypoint" {
  alias         = "normal_user"
  client_id     = var.anypoint_normal_client_id
  client_secret = var.anypoint_normal_client_secret
  base_url      = var.anypoint_base_url
  auth_type     = "connected_app"
}

###############################################################################
# Step 1 – Create Sub-Organization
# Uses admin provider as this requires elevated privileges
###############################################################################

resource "anypoint_organization" "sub_org" {
  provider = anypoint.admin

  name                   = var.sub_org_name
  parent_organization_id = var.organization_id
  owner_id               = var.owner_user_id

  entitlements = {
    create_sub_orgs      = false
    create_environments  = true
    global_deployment    = false

    vcores_production = {
      assigned   = 1
      reassigned = 0
    }

    vcores_sandbox = {
      assigned   = 1
      reassigned = 0
    }

    vcores_design = {
      assigned   = 0.5
      reassigned = 0
    }

    static_ips = {
      assigned   = 0
      reassigned = 0
    }

    vpcs = {
      assigned   = 1
      reassigned = 0
    }

    vpns = {
      assigned   = 0
      reassigned = 0
    }

    network_connections = {
      assigned   = 1
      reassigned = 0
    }
    

    # hybrid = {
    #   enabled = true
    # }

    # runtime_fabric = true

    # flex_gateway = {
    #   enabled = true
    # }
  }

  lifecycle {
    # Ignore all changes to prevent updates after creation
    ignore_changes = all

    # Prevent accidental deletion
    prevent_destroy = true
  }
}

# resource "anypoint_organization" "sub_org_child_1" {
#   provider = anypoint.admin

#   name                   = "${var.sub_org_name}-child-1"
#   parent_organization_id = var.organization_id
#   owner_id               = var.owner_user_id

#   entitlements = {
#     create_sub_orgs      = false
#     create_environments  = true
#     global_deployment    = false

#     vcores_production = {
#       assigned   = 0.5
#       reassigned = 0
#     }

#     vcores_sandbox = {
#       assigned   = 0.5
#       reassigned = 0
#     }

#     vcores_design = {
#       assigned   = 0.5
#       reassigned = 0
#     }

#     static_ips = {
#       assigned   = 0
#       reassigned = 0
#     }

#     vpcs = {
#       assigned   = 1
#       reassigned = 0
#     }

#     vpns = {
#       assigned   = 0
#       reassigned = 0
#     }

#     network_connections = {
#       assigned   = 1
#       reassigned = 0
#     }

#     # Note: hybrid, runtime_fabric, flex_gateway are master-org-only flags.
#     # They are inherited by sub-orgs and cannot be set here.
#   }

#   lifecycle {
#     # Ignore all changes to prevent updates after creation
#     ignore_changes = all

#     # Prevent accidental deletion
#     prevent_destroy = true
#   }
# }


# resource "anypoint_organization" "sub_org_child_2" {
#   provider = anypoint.admin

#   name                   = "${var.sub_org_name}-child-2"
#   parent_organization_id = var.organization_id
#   owner_id               = var.owner_user_id

#   entitlements = {
#     create_sub_orgs      = false
#     create_environments  = true
#     global_deployment    = false

#     vcores_production = {
#       assigned   = 0.5
#       reassigned = 0
#     }

#     vcores_sandbox = {
#       assigned   = 0.5
#       reassigned = 0
#     }

#     vcores_design = {
#       assigned   = 0.5
#       reassigned = 0
#     }

#     static_ips = {
#       assigned   = 0
#       reassigned = 0
#     }

#     vpcs = {
#       assigned   = 1
#       reassigned = 0
#     }

#     vpns = {
#       assigned   = 0
#       reassigned = 0
#     }

#     network_connections = {
#       assigned   = 1
#       reassigned = 0
#     }

#     # Note: hybrid, runtime_fabric, flex_gateway are master-org-only flags.
#     # They are inherited by sub-orgs and cannot be set here.
#   }

#   lifecycle {
#     # Ignore all changes to prevent updates after creation
#     ignore_changes = all

#     # Prevent accidental deletion
#     prevent_destroy = true
#   }
# }

###############################################################################
# Step 2 – Create 2 Environments in Sub-Organization
# Uses admin provider for initial setup
###############################################################################

resource "anypoint_environment" "sandbox_suborg" {
  provider = anypoint.admin

  organization_id = anypoint_organization.sub_org.id
  name            = "${anypoint_organization.sub_org.name}-sandbox"
  type            = "sandbox"
  is_production   = false

  depends_on = [anypoint_organization.sub_org]
}


resource "anypoint_environment" "production_suborg" {
  provider = anypoint.admin

  organization_id = anypoint_organization.sub_org.id
  name            = "${anypoint_organization.sub_org.name}-production"
  type            = "production"
  is_production   = true

  depends_on = [anypoint_organization.sub_org]
}

###############################################################################
# Step 3 – Assign Connected App Scopes
# Uses admin provider as this requires user authentication (username/password)
# This could be assigning scopes to the normal_user's connected app
###############################################################################

resource "anypoint_connected_app_scopes" "app_scopes" {
  provider = anypoint.admin

  connected_app_id = var.connected_app_client_id

  scopes = [
    # CloudHub 2.0 access
    {
      scope = "admin:cloudhub"
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    },
    # Runtime Fabrics management
    {
      scope = "manage:runtime_fabrics"
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    },
    # CloudHub Networking management
    {
      scope = "manage:cloudhub_networking"
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    },
    # Environment creation
    {
      scope = "create:environment"
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    },
    # API Configuration
    {
      scope = "manage:api_configuration"
      context_params = {
        org = anypoint_organization.sub_org.id
        envId = anypoint_environment.sandbox_suborg.id
      }
    },
    # APIs management
    {
      scope = "manage:apis"
      context_params = {
        org = anypoint_organization.sub_org.id
        envId = anypoint_environment.sandbox_suborg.id
      }
    },
    # API Groups management
    {
      scope = "manage:api_groups"
      context_params = {
        org = anypoint_organization.sub_org.id
      }
    },

    # API Policies management
    {
      scope = "manage:api_policies"
      context_params = {
        org = anypoint_organization.sub_org.id
        envId = anypoint_environment.sandbox_suborg.id
      }
    },
    # Secret Groups management
    {
      scope = "manage:secret_groups"
      context_params = {
        org = anypoint_organization.sub_org.id
        envId = anypoint_environment.sandbox_suborg.id
      }
    },
    # Secrets management
    {
      scope = "manage:secrets"
      context_params = {
        org = anypoint_organization.sub_org.id
        envId = anypoint_environment.sandbox_suborg.id
      }
    },
    {
      scope = "manage:apis"
      context_params = {
        org = var.organization_id
        envId = "448ec638-4283-40e3-ba3a-d1db2b63e02d"
      }
    }
  ]

  depends_on = [
    anypoint_organization.sub_org,
    anypoint_environment.production_suborg,
    anypoint_environment.sandbox_suborg,
    anypoint_connected_app_scopes.app_scopes
  ]
}

###############################################################################
# Step 4 – Create Private Space in Production Environment
# Uses normal_user provider to demonstrate normal user can create private spaces
# (after admin has granted appropriate scopes)
# NOTE: If normal user credentials are not configured, change to: provider = anypoint.admin
###############################################################################

resource "anypoint_private_space" "sandbox_space" {
  provider = anypoint.normal_user

  organization_id = anypoint_organization.sub_org.id
  name            = "${anypoint_organization.sub_org.name}-sandbox-space"  
  region          = var.private_space_region
  enable_egress   = true
  enable_iam_role = false

  depends_on = [
    anypoint_environment.sandbox_suborg,
    anypoint_connected_app_scopes.app_scopes
  ]
  
  lifecycle {
    ignore_changes = all
    prevent_destroy = true
  }
}

###############################################################################
# Step 5 – Create Private Network in Private Space
# Uses normal_user provider to demonstrate normal user can create private networks
# NOTE: If normal user credentials are not configured, change to: provider = anypoint.admin
###############################################################################

resource "anypoint_private_network" "sandbox_network" {
  provider = anypoint.normal_user

  organization_id  = anypoint_organization.sub_org.id
  private_space_id = anypoint_private_space.sandbox_space.id  
  region           = var.private_space_region
  cidr_block       = var.network_cidr_block

  reserved_cidrs = var.network_reserved_cidrs

  depends_on = [anypoint_private_space.sandbox_space]
}

###############################################################################
# Step 6 – Create Private Space Association
# Uses normal_user provider to demonstrate normal user can create private space associations
# NOTE: If normal user credentials are not configured, change to: provider = anypoint.admin
###############################################################################

resource "anypoint_private_space_association" "sandbox_space_association" {
  provider = anypoint.normal_user
  organization_id = anypoint_organization.sub_org.id
  private_space_id = anypoint_private_space.sandbox_space.id
  associations = [
    {
      organization_id = "all"
      environment     = "all"
    }
  ]
}

###############################################################################
# Outputs
###############################################################################

output "sub_organization" {
  description = "Sub-organization details"
  value = {
    id         = anypoint_organization.sub_org.id
    name       = anypoint_organization.sub_org.name
    client_id  = anypoint_organization.sub_org.client_id
    created_at = anypoint_organization.sub_org.created_at
  }
}

output "environments" {
  description = "Created environments"
  value = {
    sandbox = {
      id            = anypoint_environment.sandbox_suborg.id
      name          = anypoint_environment.sandbox_suborg.name
      client_id     = anypoint_environment.sandbox_suborg.client_id
      arc_namespace = anypoint_environment.sandbox_suborg.arc_namespace
    }
  }
}

output "connected_app_scopes" {
  description = "Connected app scopes assignment ID"
  value       = anypoint_connected_app_scopes.app_scopes.id
}

output "private_space" {
  description = "Private space details"
  value = {
    id                 = anypoint_private_space.sandbox_space.id
    name               = anypoint_private_space.sandbox_space.name
    region             = anypoint_private_space.sandbox_space.region
    status             = anypoint_private_space.sandbox_space.status
    organization_id    = anypoint_private_space.sandbox_space.organization_id
    deployment_count   = anypoint_private_space.sandbox_space.mule_app_deployment_count
  }
}

output "private_network" {
  description = "Private network details"
  value = {
    id                   = anypoint_private_network.sandbox_network.id
    name                 = anypoint_private_network.sandbox_network.name
    cidr_block           = anypoint_private_network.sandbox_network.cidr_block
    inbound_static_ips   = anypoint_private_network.sandbox_network.inbound_static_ips
    outbound_static_ips  = anypoint_private_network.sandbox_network.outbound_static_ips
    dns_target           = anypoint_private_network.sandbox_network.dns_target
  }
}

###############################################################################
# Next Steps
###############################################################################

# 1. Deploy applications to the private space using the environment IDs
#
# 2. Configure VPN connections to the private network:
#    resource "anypoint_vpn_connection" "site_to_site" { ... }
#
# 3. Set up API instances and policies using the environment IDs
#
# 4. Create additional users and assign them to appropriate role groups
