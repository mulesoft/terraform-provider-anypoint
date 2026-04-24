variable "anypoint_client_id" {
  description = "Anypoint Platform client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
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
#         "organizationId": "542cc7e3-2143-40ce-90e9-cf69da9b4da6",
#       },
#       {
#         "id": "c0c9f7f5-57bb-4333-82d7-dbdcab912234",
#         "name": "Sandbox",
#         "organizationId": "542cc7e3-2143-40ce-90e9-cf69da9b4da6",
#       }
#     ],
