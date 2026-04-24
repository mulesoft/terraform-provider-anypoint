# Anypoint Platform Configuration
variable "anypoint_admin_client_id" {
  description = "Anypoint Connected App Client ID (must support password grant)"
  type        = string
  sensitive   = true
  default     = "a66da37ba83d4c599264347952d4d533"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "0de4EA9E5bae4651B599a2071bFDD4E1"
}

variable "anypoint_admin_username" {
  description = "Anypoint Platform Username"
  type        = string
  sensitive   = true
  default     = "ankitsarda_anypointstgx"
}

variable "anypoint_admin_password" {
  description = "Anypoint Platform Password"
  type        = string
  sensitive   = true
  default     = "Dreamz@007"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "username" {
  description = "Username for the user"
  type        = string
  default     = "demo_user"
  
  validation {
    condition     = length(var.username) >= 3
    error_message = "Username must be at least 3 characters long."
  }
}

variable "first_name" {
  description = "First name of the user"
  type        = string
  default     = "Demo"
  
  validation {
    condition     = length(var.first_name) >= 1
    error_message = "First name cannot be empty."
  }
}

variable "last_name" {
  description = "Last name of the user"
  type        = string
  default     = "User"
  
  validation {
    condition     = length(var.last_name) >= 1
    error_message = "Last name cannot be empty."
  }
}

variable "email" {
  description = "Email address of the user"
  type        = string
  default     = "demo_user@example.com"
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.email))
    error_message = "Email must be a valid email address."
  }
}

variable "phone_number" {
  description = "Phone number of the user (optional)"
  type        = string
  default     = null
}

variable "password" {
  description = "Password for the user"
  type        = string
  sensitive   = true
  default     = "DemoUser@123"
  
  validation {
    condition     = length(var.password) >= 8
    error_message = "Password must be at least 8 characters long."
  }
} 

variable "mfa_verification_excluded" {
  description = "MFA verification excluded"
  type        = bool
  default     = false
}