terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# ---------------------------------------------------------------------------
# Self-managed (connected-mode) Flex/Omni Gateway
#
# Unlike a *managed* Omni Gateway (CloudHub 2.0), the platform does NOT run a
# runtime for you. This resource mints a registration token; you feed that token
# to the Flex runtime on your own host so it can self-register. The gateway
# object then appears on the platform and its computed fields populate on the
# next refresh/plan.
# ---------------------------------------------------------------------------
resource "anypoint_self_managed_gateway" "gw" {
  name           = "my-flex-gateway"
  environment_id = var.environment_id
  # organization_id is optional — omit to use the connected-app credentials org.
}

# ---------------------------------------------------------------------------
# The minted token is what you hand to flexctl on the runtime host. This is the
# same command the Anypoint UI shows on "Add Self-Managed Omni Gateway", with the
# token sourced from the Terraform output instead of pasted by hand:
#
#   docker run --entrypoint flexctl -u $UID \
#     -v "$(pwd)":/registration mulesoft/flex-gateway \
#     registration create \
#     --organization=<your-org-id> \
#     --token="$(terraform output -raw flex_registration_token)" \
#     --output-directory=/registration \
#     --mode=connected \
#     my-flex-gateway
#
# (--mode=connected is current; the older --connected / --connected=true flag is deprecated.)
#
# The token is a short-lived, one-shot enrollment secret and is NOT recoverable
# on import — capture it from the apply output promptly.
# ---------------------------------------------------------------------------
output "flex_registration_token" {
  description = "One-shot registration token to feed to the Flex runtime"
  value       = anypoint_self_managed_gateway.gw.registration_token
  sensitive   = true
}

output "gateway_id" {
  description = "Platform-assigned gateway ID (empty until the runtime registers)"
  value       = anypoint_self_managed_gateway.gw.gateway_id
}

output "gateway_status" {
  description = "Status of the registered gateway (empty until it registers)"
  value       = anypoint_self_managed_gateway.gw.status
}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------
variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App client ID"
  type        = string
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "environment_id" {
  description = "The environment ID the gateway registers into"
  type        = string
  default     = "<environment_id>"
}
