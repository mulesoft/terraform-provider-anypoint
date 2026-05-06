terraform {
  required_providers {
    anypoint = {
      source  = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# -----------------------------------------------------------------------------
# Step 1 — pull the Anypoint admin creds from AWS Secrets Manager.
#
# The secret is expected to be a JSON blob with four fields. Create it once,
# out of band, with `aws secretsmanager create-secret` (see README.md).
#
# NOTE: the decoded secret value lands in `terraform.tfstate` in plaintext.
# Use S3 backend with SSE-KMS, restrict bucket access to the apply role, and
# enable Terraform 1.7+ native state encryption for defense-in-depth.
# -----------------------------------------------------------------------------

data "aws_secretsmanager_secret_version" "anypoint_admin" {
  secret_id = var.anypoint_secret_id
}

locals {
  anypoint_admin = jsondecode(data.aws_secretsmanager_secret_version.anypoint_admin.secret_string)
}

# -----------------------------------------------------------------------------
# Step 2 — feed the creds into the provider. Nothing sensitive is written to
# HCL.
# -----------------------------------------------------------------------------

provider "anypoint" {
  alias         = "admin"
  client_id     = local.anypoint_admin.client_id
  client_secret = local.anypoint_admin.client_secret
  username      = local.anypoint_admin.username
  password      = local.anypoint_admin.password
  base_url      = var.anypoint_base_url
}

# -----------------------------------------------------------------------------
# Step 3 — verify the wiring with a cheap data-source lookup. Replace with
# your actual Anypoint resources in a real project.
# -----------------------------------------------------------------------------

data "anypoint_organization" "master" {
  provider = anypoint.admin
  id       = var.master_organization_id
}

output "master_org_name" {
  value = data.anypoint_organization.master.name
}
