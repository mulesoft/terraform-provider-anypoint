# Anypoint Platform Configuration
variable "anypoint_admin_client_id" {
  description = "Anypoint Platform Admin Connected App Client ID"
  type        = string
  default     = "<anypoint_admin_client_id>"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Platform Admin Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_admin_client_secret>"
}

variable "anypoint_admin_username" {
  description = "Anypoint Platform Admin Username (for scope assignment)"
  type        = string
  default     = "<admin_username_here>"
}

variable "anypoint_admin_password" {
  description = "Anypoint Platform Admin Password (for scope assignment)"
  type        = string
  sensitive   = true
  default     = "<admin_password_here>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Organization ID"
  type        = string
  default     = "<org_id>"
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