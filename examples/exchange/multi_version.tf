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
#   * ADD a key    -> publishes a NEW version (additive, like "Add version").
#   * REMOVE a key -> hard-deletes THAT version only (the others are untouched).
#   * EDIT a key's `version` in place -> forces replacement of that entry.
#
# The map is keyed by the VERSION NUMBER ("1.0.0", "1.0.1", "2.0.0"), so resource
# addresses read `anypoint_exchange_asset.petstore["1.0.0"]`. HCL map keys may be
# quoted strings, so dotted version numbers are fine.
#
# ----------------------------------------------------------------------------
# THREE scopes — the platform does NOT store every field per-version. There are
# three tiers:
#
#   1. VERSION scope (independent per GAV): file_path, classifier, api_version,
#      status. `status` has its own per-version endpoint, so deprecating 1.0.0
#      leaves 2.0.0 published — genuinely per-version.
#
#   2. VERSION-GROUP scope (SHARED within a MAJOR line): >>> tags/labels <<< and
#      external `instances`. Exchange stores ONE label set per major
#      (versionGroup), NOT per patch. So 1.0.0 and 1.0.1 ALWAYS share one label
#      set and the LAST publish wins — if two entries in the same major set
#      different tags they silently clobber each other, and the loser then drifts
#      on every `plan`. FIX: source tags from the MAJOR (see `petstore_majors`),
#      never from the individual patch.
#
#   3. GROUP scope (SHARED across ALL versions): name, description, contact_name,
#      contact_email, manager. ONE value per asset. These MUST be identical in
#      every entry, or the platform keeps the existing value and drops the rest.
#      Factored into `petstore_group` and referenced from every entry.
#
# Rule of thumb: only file_path / classifier / api_version / status may differ
# between two versions that SHARE a major. Everything else must be identical
# within a major, and name/description/contacts identical across the whole asset.
# ----------------------------------------------------------------------------

locals {
  # ---- GROUP scope: identical across the whole asset (all versions) ----
  petstore_group = {
    name          = "TF Demo Petstore API (multi-version)"
    description   = "Petstore API published by Terraform, managed across versions with for_each."
    contact_name  = "Platform Team"
    contact_email = "platform@example.com"
  }

  # ---- VERSION-GROUP scope: one entry per MAJOR line, shared by every patch ----
  # tags live HERE, not on the individual version, because Exchange stores labels
  # per versionGroup (major). Keying by major makes it IMPOSSIBLE for two patches
  # in the same major (e.g. 1.0.0 and 1.0.1) to disagree on tags.
  petstore_majors = {
    "1" = {
      tags = ["terraform", "petstore", "v1"]
    }
    "2" = {
      tags = ["terraform", "petstore", "v2", "adds-vaccinations"]
    }
  }

  # ---- VERSION scope: one entry per PUBLISHED VERSION (GAV) ----
  # The map KEY is the version number. `major` links each patch to its version
  # group above. Only genuinely per-version fields live here.
  petstore_versions = {
    "1.0.0" = {
      major     = "1"
      file_path = "test-assets/petstore.json"
      status    = "published"
      # api_version is REQUIRED at create for rest-api, soap-api, evented-api,
      # grpc-api and http-api. Omitting it makes the publish fail with
      # `400 MISSING_REQUIRED_PROPERTIES: apiVersion`, and the provider blocks it
      # at plan time. (graphql-api needs a spec FILE but not api_version.)
      # It is the human-facing API contract version (distinct from the GAV
      # `version`) and IS per-version.
      api_version = "v1"
    }
    "1.0.1" = {
      major       = "1" # same major as 1.0.0 -> shares the "1" tag set
      file_path   = "test-assets/petstore-v1_0_1.json"
      status      = "published"
      api_version = "v1"
    }
    "2.0.0" = {
      major       = "2"
      file_path   = "test-assets/petstore-v2.json" # a genuinely different spec (adds /vaccinations)
      status      = "published"
      api_version = "v2"
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

  # ---- VERSION-scoped: from each map entry, independent per version ----
  version     = each.key # the map key IS the version number
  file_path   = "${path.module}/${each.value.file_path}"
  main_file   = basename(each.value.file_path)
  status      = each.value.status
  api_version = each.value.api_version

  # ---- VERSION-GROUP-scoped: from the MAJOR, shared by every patch in it ----
  # Sourcing tags from the major (not the patch) guarantees 1.0.0 and 1.0.1 never
  # disagree, so there is no last-publish-wins clobber and no perpetual drift.
  tags = local.petstore_majors[each.value.major].tags

  # ---- GROUP-scoped: identical across every entry (scope #3) ----
  name          = local.petstore_group.name
  description   = local.petstore_group.description
  contact_name  = local.petstore_group.contact_name
  contact_email = local.petstore_group.contact_email

  # ---- Recreate safety for the destructive path ----
  # `version` is replacement-forcing, so EDITING a version string in place (e.g.
  # "2.0.0" -> "2.0.1") destroys the old version and republishes. PREFER
  # adding/removing MAP KEYS (purely additive / version-scoped delete) whenever
  # you can. create_before_destroy publishes the new GAV BEFORE hard-deleting the
  # old one, so a failed publish can't leave you with neither.
  lifecycle {
    create_before_destroy = true
  }
}

# Version number => composite GAV id, for downstream references.
output "petstore_ids_by_version" {
  description = "Map of version number (1.0.0/1.0.1/2.0.0) to composite GAV id"
  value       = { for k, a in anypoint_exchange_asset.petstore : k => a.id }
}
