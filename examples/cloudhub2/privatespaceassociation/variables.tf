variable "anypoint_client_id" {
  description = "Anypoint Platform client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "private_space_id" {
  description = "The ID of the private space where associations will be created"
  type        = string
}

# "environments": [
#       {
#         "id": "a4d171b4-9ad4-41da-9d77-18a3ade0a93d",
#         "name": "Design",
#         "organizationId": "<org_id>",
#       },
#       {
#         "id": "<private_space_id>",
#         "name": "Sandbox",
#         "organizationId": "<org_id>",
#       }
#     ],
