terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "0.1.0"
    }
    vault = {
      source  = "hashicorp/vault"
      version = ">= 4.0"
    }
  }
}

# The vault provider reads VAULT_ADDR and VAULT_TOKEN (or AppRole /
# OIDC / Kubernetes auth) from the environment by default. No credentials
# live in HCL.
provider "vault" {}

# -----------------------------------------------------------------------------
# Step 1 — pull the Anypoint admin creds from a KV v2 secret.
#
# Create it once, out of band, with `vault kv put` (see README.md).
#
# NOTE: the decoded secret value lands in `terraform.tfstate` in plaintext.
# Use a remote backend with encryption at rest and restrict read access.
# -----------------------------------------------------------------------------

data "vault_kv_secret_v2" "anypoint_admin" {
  mount = var.vault_mount
  name  = var.vault_path
}

# -----------------------------------------------------------------------------
# Step 2 — feed the creds into the provider.
# -----------------------------------------------------------------------------

provider "anypoint" {
  alias         = "admin"
  client_id     = data.vault_kv_secret_v2.anypoint_admin.data["client_id"]
  client_secret = data.vault_kv_secret_v2.anypoint_admin.data["client_secret"]
  username      = data.vault_kv_secret_v2.anypoint_admin.data["username"]
  password      = data.vault_kv_secret_v2.anypoint_admin.data["password"]
  base_url      = var.anypoint_base_url
}

# -----------------------------------------------------------------------------
# Step 3 — verify the wiring with a cheap data-source lookup.
# -----------------------------------------------------------------------------

data "anypoint_organization" "master" {
  provider = anypoint.admin
  id       = var.master_organization_id
}

output "master_org_name" {
  value = data.anypoint_organization.master.name
}
