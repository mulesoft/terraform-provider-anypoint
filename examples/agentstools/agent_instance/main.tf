# ###############################################################################
# # Anypoint Agent Instance Example
# # ================================
# # This example demonstrates creating an Agent instance in API Manager,
# # similar to creating an API instance but for AI agents.
# #
# # Usage:
# #   terraform init
# #   terraform plan
# #   terraform apply
# ###############################################################################

# terraform {
#   required_providers {
#     anypoint = {
#       source = "mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# ###############################################################################
# # Agent Instance
# # ---------------
# # Creates an agent instance with routing to a backend agent service
# ###############################################################################

# resource "anypoint_agent_instance" "customer_support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   technology      = "flexGateway"
#   instance_label  = "customer-support-agent"

#   # Exchange asset specification for the agent
#   spec = {
#     asset_id = var.agent_asset_id
#     group_id = var.organization_id
#     version  = var.agent_asset_version
#   }

#   # Endpoint configuration
#   endpoint = {
#     deployment_type = "HY"
#     base_path       = "agent/support"
#   }

#   # Omni Gateway deployment
#   gateway_id = var.gateway_id

#   # Backend agent service URI
#   upstream_uri = "http://agent-service.internal:8080"
# }

# ###############################################################################
# # Agent Instance with Production URL
# # ------------------------------------
# # Creates an agent instance for a production sales agent.
# # Note: Unlike API instances, agent instances always route to a single
# # upstream with 100% weight. Multi-upstream routing is not supported for agents.
# ###############################################################################

# resource "anypoint_agent_instance" "sales_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   technology      = "flexGateway"
#   instance_label  = "sales-agent"

#   spec = {
#     asset_id = var.agent_asset_id
#     group_id = var.organization_id
#     version  = var.agent_asset_version
#   }

#   endpoint = {
#     deployment_type = "HY"
#     base_path       = "agent/sales"
#   }

#   gateway_id = var.gateway_id

#   # Single upstream - agents always have one upstream with 100% weight
#   upstream_uri = "http://sales-agent.internal:8080"
# }

# ###############################################################################
# # LLM Gateway Policies
# # ----------------------
# # Apply LLM-specific policies to the customer support agent instance using
# # typed policy resources. These policies handle routing, guardrails, and
# # protocol translation for LLM traffic.
# ###############################################################################

# # ─── 1. Semantic Routing (HuggingFace) ───────────────────────
# # Routes requests to different LLM providers based on semantic
# # topic matching using HuggingFace embeddings.
# resource "anypoint_api_policy_semantic_routing_policy_huggingface" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     huggingface_url     = "http://huggingface.com"
#     huggingface_api_key = "your-hf-api-key"
#     threshold           = 0.2
#     timeout             = 10000
#     routes = [
#       {
#         provider = "openai"
#         model    = "gpt-4"
#         topics = [
#           { name = "Code", embeddings = "{code-embeddings}" },
#           { name = "Sales", embeddings = "{sales-embeddings}" }
#         ]
#       },
#       {
#         provider = "gemini"
#         model    = "gemini-pro"
#         topics = [
#           { name = "Support", embeddings = "{support-embeddings}" }
#         ]
#       }
#     ]
#     fallback_route = {
#       provider = "openai"
#       model    = "gpt-3.5-turbo"
#     }
#   }
# }

# # ─── 2. LLM Proxy Core Policy ────────────────────────────────
# # Header-based routing to different LLM vendors with model targeting.
# resource "anypoint_api_policy_llm_proxy_core_policy" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     header_name = "x-routing-header"
#     vendor_header_mapping = [
#       {
#         vendor      = "Openai"
#         headerValue = "openai"
#         targetModel = "gpt-4"
#       },
#       {
#         vendor      = "Gemini"
#         headerValue = "gemini"
#         targetModel = "gemini-pro"
#       }
#     ]
#   }
# }

# # ─── 3. LLM GW Core Policy ──────────────────────────────────
# # Gateway-level core routing for LLM traffic with vendor mapping.
# resource "anypoint_api_policy_llm_gw_core_policy" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     header_name = "x-routing-header"
#     vendor_header_mapping = [
#       {
#         vendor      = "Openai"
#         headerValue = "openai"
#         targetModel = "gpt-4"
#       },
#       {
#         vendor      = "Gemini"
#         headerValue = "gemini"
#         targetModel = "gemini-pro"
#       }
#     ]
#   }
# }

# # ─── 4. LLM Proxy Core ──────────────────────────────────────
# # Enables LLM proxy protocol support on the gateway.
# resource "anypoint_api_policy_llm_proxy_core" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {}
# }

# # ─── 5. Model-Based Routing ─────────────────────────────────
# # Routes to LLM vendors based on model name with fallback.
# resource "anypoint_api_policy_model_based_routing" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     supported_vendors = [
#       { vendor = "Openai", targetModel = "gpt-4" },
#       { vendor = "Gemini", targetModel = "gemini-pro" }
#     ]
#     fallback = {
#       provider = "gemini"
#       model    = "gemini-pro"
#     }
#   }
# }

# # ─── 6. Semantic Prompt Guard (OpenAI) ──────────────────────
# # Guards against disallowed topics using OpenAI embeddings
# # to detect and block prompts that match denied categories.
# resource "anypoint_api_policy_semantic_prompt_guard_policy_openai" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     openai_url             = "http://openai.com"
#     openai_api_key         = "your-openai-api-key"
#     openai_embedding_model = "text-embedding-ada-002"
#     threshold              = 0.2
#     timeout                = 10000
#     deny_topics = [
#       { name = "Politics", embeddings = "{politics-embeddings}" },
#       { name = "Violence", embeddings = "{violence-embeddings}" }
#     ]
#   }
# }

# ###############################################################################
# # A2A (Agent-to-Agent) Policies
# # ------------------------------
# # Apply A2A-specific policies to the customer support agent instance.
# # These policies handle PII detection, agent card publishing, schema
# # validation, and token-based rate limiting for agent-to-agent communication.
# ###############################################################################

# # ─── 7. A2A PII Detector ────────────────────────────────────
# resource "anypoint_api_policy_a2a_pii_detector" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     entities = ["Email", "US SSN", "Credit Card", "Phone Number"]
#     action   = "Reject"
#   }
# }

# # ─── 8. A2A Agent Card ──────────────────────────────────────
# resource "anypoint_api_policy_a2a_agent_card" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     consumer_url   = "http://www.example.com"
#     card_path      = "/.well-known/agent.json"
#     file_name      = "agent-card.json"
#     file_mime_type = "application/json"
#     file_source    = "Base64"
#     content        = "eyJuYW1lIjoiQ3VzdG9tZXIgU3VwcG9ydCBBZ2VudCIsInZlcnNpb24iOiIxLjAuMCIsImRlc2NyaXB0aW9uIjoiQW4gQTJBIGFnZW50IGZvciBjdXN0b21lciBzdXBwb3J0IiwiY2FwYWJpbGl0aWVzIjpbInRleHQiLCJmaWxlIl19"
#   }
# }

# # ─── 9. A2A Schema Validation ───────────────────────────────
# resource "anypoint_api_policy_a2a_schema_validation" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {}
# }

# # ─── 10. A2A Token Rate Limit ───────────────────────────────
# resource "anypoint_api_policy_a2a_token_rate_limit" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     maximum_tokens               = 100
#     time_period_in_milliseconds  = 10000
#     key_selector                 = "#[attributes.headers['ClientId']]"
#   }
# }

# # ─── 11. A2A Prompt Decorator ───────────────────────────────
# resource "anypoint_api_policy_a2a_prompt_decorator" "support_agent" {
#   organization_id = var.organization_id
#   environment_id  = var.environment_id
#   api_instance_id = anypoint_agent_instance.customer_support_agent.id

#   configuration = {
#     text_decorators = [
#       {
#         role = "system"
#         text = "#[\"You are a helpful customer support assistant. Always be polite and concise.\"]"
#       }
#     ]
#     file_decorators = [
#       {
#         fileName = "#[\"context.txt\"]"
#         fileType = "Base64"
#         file     = "#[\"Q29tcGFueSBwb2xpY3k6IEFsd2F5cyBlc2NhbGF0ZSBiaWxsaW5nIGlzc3Vlcw==\"]"
#       }
#     ]
#   }
# }
