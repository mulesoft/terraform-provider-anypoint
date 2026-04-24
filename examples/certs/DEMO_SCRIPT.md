# Anypoint Terraform Provider – Demo Script (5 min)

**Audience:** Senior Leadership & Product Managers  
**Focus:** End-to-end API Management — from org creation to API grouping  
**Setup:** Terminal open with `examples/demo/` directory, Anypoint Platform UI open in browser

---

## Opening – The Problem (30 seconds)

> "Today, managing APIs on Anypoint Platform is a manual, click-driven process. Teams create organizations and environments in the UI, provision private spaces, configure gateways, deploy API instances, apply policies one by one, and when it's time to promote to production — they repeat the same steps hoping nothing is missed.
>
> This doesn't scale. It's error-prone, unauditable, and impossible to replicate consistently.
>
> What we've built is a **Terraform provider for Anypoint Platform** — it lets teams manage the entire platform lifecycle as code, from org setup through API governance."

---

## Scene 1 – Organization & Environments (45 seconds)

> "Let me start from the very beginning — standing up an entire business unit."

**[Show `main.tf` in the editor — scroll to Scene 1]**

> "Here we create a **sub-organization** called 'Commerce Business Unit' under our root org. We allocate entitlements — 2 production vCores, 2 sandbox vCores, 1 VPC, network connections — all defined declaratively.
>
> Then we create two environments: **Sandbox** and **Production**. That's 3 resources. In the UI, this is multiple screens across different admin pages. Here it's 30 lines of code, version-controlled and repeatable."

---

## Scene 2 – Connected App Permissions, Private Space & Flex Gateway (60 seconds)

**[Scroll to Scene 2 in `main.tf`]**

> "Now that the org and environments exist, we need to **grant our Connected App access** to them. This is often the most tedious manual step — navigating to Access Management, finding the connected app, and clicking through scope checkboxes for each environment.
>
> Here, we define all the scopes declaratively — `manage:apis`, `manage:api_policies`, `manage:secret_groups` — scoped to specific org-and-environment combinations. If we add a new environment later, we add one line and re-apply.
>
> Next, we provision a **Private Space** in `us-east-1` for workload isolation. And inside that Private Space, we deploy a **Managed Flex Gateway** — our API gateway runtime. We configure LTS release channel, small sizing, SSL forwarding, and logging — all in code.
>
> So in Scene 1 and 2, we went from nothing to a fully provisioned infrastructure: org, two environments, permissions, a private space, and a running gateway. Zero clicks."

---

## Scene 3 – API Instance with Canary Routing (45 seconds)

**[Scroll to Scene 3]**

> "Now we deploy an API. In about 20 lines, we define an API instance — referencing the Orders API spec from Exchange, version 1.0. We deploy it to the Flex Gateway we just created.
>
> And here's the interesting part — **canary routing** with weighted traffic splitting: 90% to our stable backend, 10% to a canary release. This is the exact same thing you'd configure through 5-6 screens in the API Manager UI — but it's code, reviewable in a PR.
>
> Notice that `organization_id`, `environment_id`, and `gateway_id` all reference the resources we created above. Terraform wires the dependency graph automatically."

**[Run in terminal]:**
```bash
terraform plan
```

> "Terraform shows exactly what it will create — org, environments, permissions, private space, gateway, API instances, policies, tiers, alerts, promotion, API group. **19 resources total**, all from a single file."

---

## Scene 4 – Security Policies (45 seconds)

**[Scroll to Scene 4 in `main.tf`]**

> "Once the API instance is created, we layer on security.
>
> **JWT validation** — we configure the JWKS endpoint, required audience, and expiration checks. Unauthenticated requests are rejected at the gateway.
>
> **Rate limiting** — 100 requests per minute, with headers exposed so clients see their remaining quota.
>
> **IP allowlisting** — restrict to corporate IP ranges only.
>
> We support **43 policy types** in total — from CORS and message logging to OAuth2 token introspection and threat protection. Every one has a typed configuration block with full validation.
>
> The key point: these policies are **declarative and ordered**. If someone changes a policy in the UI, `terraform plan` will detect the drift."

---

## Scene 5 – SLA Tiers (20 seconds)

**[Scroll to Scene 5]**

> "Next, SLA tiers. Gold at 1000 req/min with manual approval, Silver at 200 with auto-approval, and Trial at 10 for evaluations. Consumers pick a tier through Exchange, and rate limits are enforced by the gateway. All defined as code."

---

## Scene 6 – Alerts (15 seconds)

**[Scroll to Scene 6]**

> "Operational alerts — a critical alert fires if we exceed 50 server errors in one minute. The team gets an email with full context. Defined alongside the API, not buried in a UI settings page."

---

## Scene 7 – Environment Promotion (45 seconds)

**[Scroll to Scene 7]**

> "This is my favorite part. **Environment promotion.**
>
> With a single resource block, we promote the Orders API from Sandbox to Production. We set `include_policies`, `include_tiers`, and `include_alerts` to `true` — so the JWT policy, rate limits, IP allowlist, SLA tiers, and alerts all come along.
>
> Today, this is a manual process where someone has to re-apply each policy in production. With Terraform, it's one `apply` command. Repeatable. Auditable in git."

---

## Scene 8 – API Groups (20 seconds)

**[Scroll to Scene 8]**

> "Finally, we create a Payments API and bundle both Orders and Payments into an **API Group** — 'Commerce APIs'. Consumers subscribe to one group, get access to both APIs with a single contract."

---

## Close – Why This Matters (30 seconds)

> "Let me summarize. We went from **zero to a fully governed API platform in one file**:
>
> - Created an organization and two environments
> - Granted fine-grained connected app permissions
> - Provisioned a Private Space and Flex Gateway
> - Deployed an API with canary routing
> - Applied security policies, SLA tiers, and alerts
> - Promoted everything to production
> - Grouped APIs for unified consumption
>
> **19 resources, one file, one command.** The provider covers 4 domains: API Management, Secrets Management, CloudHub 2.0, and Access Management.
>
> - **Consistency** — same config, same result, every time
> - **Auditability** — every change goes through git and PR review
> - **Speed** — what took an hour of clicking takes one `terraform apply`
> - **Drift detection** — `terraform plan` catches out-of-band changes
>
> Questions?"

---

## Demo Checklist

### Before the demo
- [ ] Run `terraform init` in `examples/demo/`
- [ ] Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in credentials
- [ ] Verify `terraform plan` completes without errors
- [ ] Have Anypoint Platform UI open to API Manager for the target environment
- [ ] Have the `main.tf` file open in an editor with syntax highlighting

### Demo flow
1. Editor: Show `main.tf` — walk through all 8 scenes
2. Terminal: Run `terraform plan` to show the execution plan (19 resources)
3. Terminal: Run `terraform apply -auto-approve` (optional — takes ~60s)
4. Browser: Show the new sub-org and environments in Access Management
5. Browser: Show the Private Space and Flex Gateway in Runtime Manager
6. Browser: Show the API instance in API Manager UI
7. Browser: Show policies, SLA tiers, and alerts applied in the UI

### Backup plan
If apply fails during the live demo, the `terraform plan` output is sufficient — it shows all 19 resources that would be created. Focus on the code walkthrough and the plan output.
