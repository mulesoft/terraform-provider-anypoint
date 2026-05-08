terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "0.1.0"
    }
  }

  # Tie this config to a Terraform Cloud / Enterprise workspace. Credentials
  # live as Sensitive Environment Variables on that workspace and are never
  # present in HCL, .tfvars, or any committed file.
  cloud {
    organization = "example-org"

    workspaces {
      name = "anypoint-platform"
    }
  }
}

# No client_id / client_secret / username / password block needed.
#
# The workspace must define these as Sensitive *Environment Variables*:
#
#   ANYPOINT_CLIENT_ID
#   ANYPOINT_CLIENT_SECRET
#   ANYPOINT_USERNAME          (or ANYPOINT_ADMIN_USERNAME)
#   ANYPOINT_PASSWORD          (or ANYPOINT_ADMIN_PASSWORD)
#   ANYPOINT_BASE_URL
#
# The provider reads them at plan/apply time.
provider "anypoint" {
  alias = "admin"
}

# -----------------------------------------------------------------------------
# Smoke test — replace with real resources in a production config.
# -----------------------------------------------------------------------------

data "anypoint_organization" "master" {
  provider = anypoint.admin
  id       = var.master_organization_id
}

output "master_org_name" {
  value = data.anypoint_organization.master.name
}
