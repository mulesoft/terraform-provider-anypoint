# Anypoint Platform Configuration
variable "anypoint_admin_client_id" {
  description = "Anypoint Platform Admin Connected App Client ID"
  type        = string
  default     = "a66da37ba83d4c599264347952d4d533"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Platform Admin Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "0de4EA9E5bae4651B599a2071bFDD4E1"
}

variable "anypoint_admin_username" {
  description = "Anypoint Platform Admin Username (for scope assignment)"
  type        = string
  default     = "ankitsarda_anypointstgx"
}

variable "anypoint_admin_password" {
  description = "Anypoint Platform Admin Password (for scope assignment)"
  type        = string
  sensitive   = true
  default     = "Dreamz@007"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Organization ID"
  type        = string
  default     = "a02fab4f-4695-4325-882e-f326d1cef704"
}

variable "environment_name" {
  description = "Name of the environment to create"
  type        = string
  default     = "my-second-env-renamed"
}

variable "environment_type" {
  description = "Type of the environment (design, sandbox, production)"
  type        = string
  default     = "sandbox"
  
  validation {
    condition     = contains(["design", "sandbox", "production"], var.environment_type)
    error_message = "Environment type must be one of: design, sandbox, production."
  }
}

variable "is_production" {
  description = "Whether this is a production environment"
  type        = bool
  default     = false
} 