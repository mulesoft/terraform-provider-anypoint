# ============================================================================
# Managing MULTIPLE VERSIONS of one Exchange asset with `for_each`
# ============================================================================
#
# The Exchange UI's "Add version" button publishes an additional version of an
# asset side-by-side with the existing ones. There is no separate "asset version"
# resource in this provider — an `anypoint_exchange_asset` block already IS one
# GAV (group / asset / version). To manage N versions the way the UI does, drive
# ONE resource block with `for_each` over a map of versions:
#
#   * ADD a key           -> publishes a NEW version (additive, like "Add version").
#   * REMOVE a key         -> hard-deletes THAT version only (the others are untouched).
#   * EDIT a key's `version` string in place -> forces replacement of that entry
#     (see the safety notes below — this is the destructive path).
#
# This mirrors the platform's real data model, which has THREE scopes (all three
# were live-verified against production — see
# .agents/artifacts/version-scoping-live-proof.md):
#
#   1. VERSION scope   (independent per version): file_path, classifier,
#      api_version, status, tags, pages, terms_and_conditions, categories,
#      custom_fields. Each map entry may set these freely and independently.
#
#   2. GROUP scope     (SHARED across ALL versions of the asset): name,
#      description, contact_name, contact_email, manager. The platform physically
#      stores ONE value per asset, not per version. >>> These MUST be identical in
#      every map entry. <<< If they differ, the platform silently keeps the
#      existing group value and drops the others (proven: publishing v2 with a
#      different `description` left v1's description in place). Factor them into
#      `locals` and reference them from every entry so they can never drift apart.
#
#   3. VERSION-GROUP scope (shared within a MAJOR version): external `instances`.
#      Instances live at the version-group (major) level, so all versions that
#      share a major (e.g. 1.0.0 and 1.1.0) share one instance set. Define
#      instances on ONE entry per major (or keep them identical within a major)
#      to avoid a tug-of-war between entries.
# ----------------------------------------------------------------------------

# GROUP-scoped fields — declared ONCE and reused by every version entry so they
# are guaranteed identical (caveat #2 above). Change these and EVERY version's
# group metadata updates together (a single PATCH against the group endpoint).
locals {
  petstore_group = {
    name          = "TF Demo Petstore API (multi-version)"
    description   = "Petstore API published by Terraform, managed across versions with for_each."
    contact_name  = "Platform Team"
    contact_email = "platform@example.com"
  }

  # The versions map. The KEY is a stable, human-meaningful handle (NOT the
  # version number) so you can bump `version` without changing the key — that
  # keeps Terraform tracking the same instance instead of destroying+recreating
  # under a new key. Everything inside each value is VERSION-scoped and may differ
  # freely between versions (file, tags, status, docs, ...).
  petstore_versions = {
    v1 = {
      version   = "1.0.0"
      file_path = "test-assets/petstore.json"
      status    = "published"
      tags      = ["terraform", "petstore", "v1", "stable"]
    }
    v2 = {
      version   = "2.0.0"
      file_path = "test-assets/petstore-v2.json" # a genuinely different spec (adds /vaccinations)
      status    = "published"
      tags      = ["terraform", "petstore", "v2", "adds-vaccinations"]
    }
  }
}

resource "anypoint_exchange_asset" "petstore" {
  for_each = local.petstore_versions

  organization_id = var.org_id
  group_id        = var.org_id
  asset_id        = "tf-demo-petstore-multiversion"
  type            = "rest-api"
  classifier      = "oas"

  # ---- VERSION-scoped: pulled from each map entry, independent per version ----
  version   = each.value.version
  file_path = "${path.module}/${each.value.file_path}"
  main_file = basename(each.value.file_path)
  status    = each.value.status
  tags      = each.value.tags

  # ---- GROUP-scoped: identical across every entry (see caveat #2) ----
  name          = local.petstore_group.name
  description   = local.petstore_group.description
  contact_name  = local.petstore_group.contact_name
  contact_email = local.petstore_group.contact_email

  # ---- Recreate safety for the destructive path ----
  # `version` is replacement-forcing, so EDITING a version string in place (e.g.
  # "2.0.0" -> "2.0.1" on the v2 entry) destroys the old version and republishes.
  # create_before_destroy publishes the new GAV BEFORE hard-deleting the old one,
  # so a failed publish can't leave you with neither. This is safe for a version
  # bump because the new version is a distinct GAV that can coexist with the old.
  #
  # The #68 `status` OneOf validator is the complementary plan-time net: a typo
  # like status = "Published" is caught at PLAN time, before any destroy runs.
  #
  # PREFER adding/removing MAP KEYS (purely additive / version-scoped delete) over
  # editing a version string in place whenever you can — that avoids the
  # destroy-then-recreate entirely.
  lifecycle {
    create_before_destroy = true
  }
}

# Convenience outputs: a map of version-handle => composite GAV id, and the
# reverse (version number => id), so downstream config can reference any version.
output "petstore_version_ids" {
  description = "Map of version handle (v1/v2) to composite GAV id"
  value       = { for k, a in anypoint_exchange_asset.petstore : k => a.id }
}

output "petstore_ids_by_version" {
  description = "Map of version number (1.0.0/2.0.0) to composite GAV id"
  value       = { for k, a in anypoint_exchange_asset.petstore : a.version => a.id }
}
