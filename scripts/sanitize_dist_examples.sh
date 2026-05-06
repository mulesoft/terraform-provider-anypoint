#!/usr/bin/env bash
# sanitize_dist_examples.sh — Scrub hard-coded credentials, test-account
# identifiers, and ad-hoc "your-X-here" placeholders out of a copied
# examples tree inside a distribution package. Operates in-place.
#
# Usage:
#     ./scripts/sanitize_dist_examples.sh <examples-dir>
#
# This is intended to run against dist/packages/<platform>/examples/ after
# build_for_distribution.sh has copied examples but before they are tarred.
# The source examples/ tree is NEVER modified by this script. Portable to
# bash 3.2 (macOS default).

set -euo pipefail

TARGET_DIR="${1:-}"
if [ -z "$TARGET_DIR" ] || [ ! -d "$TARGET_DIR" ]; then
    echo "usage: $0 <examples-dir>" >&2
    exit 2
fi

# Known hard-coded values observed in the source examples/ tree. Scrubbed to
# a single canonical placeholder.
PLACEHOLDER='<add-your-value-here>'

PERL_SCRIPT='
BEGIN {
    our @literals = (
        q{0de4EA9E5bae4651B599a2071bFDD4E1},
        q{a66da37ba83d4c599264347952d4d533},
        q{e5a776d9862a4f2d8f61ba8450803908},
        q{0a5E1fbfc1154D9885c32842171F7490},
        q{ankitsarda_anypointstgx},
        q{Dreamz@007},
        q{542cc7e3-2143-40ce-90e9-cf69da9b4da6},
        q{a02fab4f-4695-4325-882e-f326d1cef704},
        q{f7f43384-b33e-470c-ad4c-285aa0c01212},
        q{your-actual-client-id-here},
        q{your-actual-client-secret-here},
        q{your-admin-client-secret-here},
        q{your-client-secret-here},
        q{your-client-secret},
        q{your-organization-id-here},
        q{your-username},
        q{your-password},
        q{your-admin-username},
        q{your-admin-password},
        q{your.admin@email.com},
        q{org-client-id},
        q{env-client-id},
        q{new-org-uuid},
    );
    our $re = do {
        my $pat = join q{|}, map { quotemeta $_ } @literals;
        qr/$pat/;
    };
    our $placeholder = q{<add-your-value-here>};
}
s/$main::re/$main::placeholder/g;
s/\bmy-second-env-renamed\b/my-second-env/g;
'

# Only touch text files where these values could reasonably appear.
# Binary blobs, archives, state files, lock files, etc. stay untouched.
FILES=()
while IFS= read -r -d '' f; do
    FILES+=("$f")
done < <(
    find "$TARGET_DIR" -type f \
        \( -name '*.tf' \
           -o -name '*.tfvars.example' \
           -o -name '*.sh' \
           -o -name '*.bat' \
           -o -name '*.md' \
           -o -name '*.txt' \
           -o -name '*.env.example' \
           -o -name '*.yml' \
           -o -name '*.yaml' \
           -o -name '*.json.example' \
           -o -name 'README*' \
        \) -print0
)

if [ "${#FILES[@]}" -eq 0 ]; then
    echo "sanitize: no candidate files under $TARGET_DIR" >&2
    exit 0
fi

echo "sanitize: scrubbing ${#FILES[@]} files under $TARGET_DIR"

perl -i -pe "$PERL_SCRIPT" "${FILES[@]}"

# Sanity check: fail loudly if any known secret survived.
if grep -RIlE \
    '0de4EA9E5bae4651B599a2071bFDD4E1|a66da37ba83d4c599264347952d4d533|e5a776d9862a4f2d8f61ba8450803908|0a5E1fbfc1154D9885c32842171F7490|ankitsarda_anypointstgx|Dreamz@007' \
    "$TARGET_DIR" >/dev/null 2>&1; then
    echo "sanitize: FAILED — secrets survived:" >&2
    grep -RInE \
        '0de4EA9E5bae4651B599a2071bFDD4E1|a66da37ba83d4c599264347952d4d533|e5a776d9862a4f2d8f61ba8450803908|0a5E1fbfc1154D9885c32842171F7490|ankitsarda_anypointstgx|Dreamz@007' \
        "$TARGET_DIR" >&2 || true
    exit 1
fi

echo "sanitize: done"
