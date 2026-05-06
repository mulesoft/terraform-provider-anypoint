variable "aws_region" {
  description = "AWS region where the Anypoint admin credentials secret lives."
  type        = string
  default     = "us-east-1"
}

variable "anypoint_secret_id" {
  description = <<EOT
Name or ARN of the AWS Secrets Manager secret holding the Anypoint admin
credentials. The secret value must be JSON with the following shape:

  {
    "client_id":     "…",
    "client_secret": "…",
    "username":      "…",
    "password":      "…"
  }
EOT
  type        = string
  default     = "anypoint/terraform-admin"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL."
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "master_organization_id" {
  description = "UUID of the master organization to look up as a smoke test."
  type        = string
  default     = "<add-your-value-here>"
}
