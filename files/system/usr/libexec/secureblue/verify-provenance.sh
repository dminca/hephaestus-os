#!/usr/bin/bash

# SPDX-FileCopyrightText: Copyright 2025-2026 The Secureblue Authors
# SPDX-FileCopyrightText: Copyright 2026 The Hephaestus OS Authors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

export HOME="${HOME:-~}"

if [[ -f /usr/bin/rpm-ostree ]]; then
    image_ref=$(/usr/bin/rpm-ostree status --booted --json | /usr/bin/jq -cr '.deployments[0]."container-image-reference"')
    image_ref=${image_ref#*:docker://}
else
    image_ref=$(/usr/bin/jq -r '.status.booted.image.image.image' /run/bootc/status.json)
fi

case "${image_ref}" in
    ghcr.io/dminca/hephaestus-os:latest)
        source_uri='github.com/dminca/hephaestus-os'
        source_branch='main'
        ;;
    ghcr.io/secureblue/*)
        source_uri='github.com/secureblue/secureblue'
        image_tag="${image_ref##*:}"
        case "${image_tag}" in
            latest|latest-uki|uki)
                source_branch='live'
                ;;
            br-*)
                source_branch="${image_tag%-uki}"
                source_branch="${source_branch#br-}"
                source_branch="${source_branch%-*}"
                ;;
            *)
                echo "WARNING: Unknown Secureblue image tag '${image_tag}'; unable to check provenance." >&2
                exit 1
                ;;
        esac
        ;;
    *)
        echo "WARNING: Unknown image reference '${image_ref}'; unable to check provenance." >&2
        exit 1
        ;;
esac

echo "Verifying build provenance for ${image_ref}..."
full_ref=$(crane digest --full-ref "${image_ref}")
echo "Image reference: ${full_ref}"

slsa-verifier verify-image \
    --source-uri "${source_uri}" \
    --source-branch "${source_branch}" \
    "${full_ref}"
