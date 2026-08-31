variable "anypoint_client_id" {
  description = "Anypoint Platform connected-app client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform connected-app client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "org_id" {
  description = "The organization id. Also used as the asset group id."
  type        = string
  default     = "<org_id>"
}

# Path to a Mule connector/plugin JAR produced by your Mule build. No jar ships with
# these examples. The jar MUST contain META-INF/mule-artifact/mule-artifact.json or the
# publish fails with 400 INVALID_ASSET_METADATA "Could not find mule-artifact file
# inside jar file".
variable "connector_jar_path" {
  description = "Local path to a mule-plugin JAR (e.g. target/my-connector-mule-plugin.jar)."
  type        = string
  default     = ""
}
