#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
  printf '%s\n' 'Flow Desktop packaging requires Linux x86_64.' >&2
  exit 1
fi

version=v0.1.0-beta4 # renovate: datasource=github-releases depName=Flow-Tube/Flow-Desktop
release_version="${version#v}"
# Tauri encodes the prerelease separator as '+' in RPM metadata.
rpm_version="${release_version/-/+}"
filename="Flow_${release_version}_linux_x86_64.rpm"
url="https://github.com/Flow-Tube/Flow-Desktop/releases/download/${version}/${filename}"
checksum=cc5336fac2512df358de9a7513e13f5289c0643ad0964d48a48c061ff01217de
# Pinned updater key from the v0.1.0-beta2 release's tauri.conf.json.
public_key=RWQpaRLd6yhk18G30wcoSsrlLXi8bjGYFcC4gMCiYs/Ms5xKMe3P7qpR
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

curl --fail --show-error --silent --location --retry 3 \
  --proto '=https' --proto-redir '=https' "$url" -o "$temporary/$filename"
printf '%s  %s\n' "$checksum" "$temporary/$filename" | sha256sum -c -
curl --fail --show-error --silent --location --retry 3 \
  --proto '=https' --proto-redir '=https' "$url.sig" -o "$temporary/signature.base64"
base64 --decode < "$temporary/signature.base64" > "$temporary/signature"
minisign -Vm "$temporary/$filename" -x "$temporary/signature" -P "$public_key"

expected_identity="flow-beta:${rpm_version}:1:x86_64"
package_identity=$(rpm -qp --qf '%{NAME}:%{VERSION}:%{RELEASE}:%{ARCH}' "$temporary/$filename")
if [[ "$package_identity" != "$expected_identity" ]]; then
  printf 'Unexpected Flow RPM identity: expected %s, got %s\n' "$expected_identity" "$package_identity" >&2
  exit 1
fi
# Dependencies use the inherited repositories and their existing signature policy.
dnf5 install --assumeyes --setopt=install_weak_deps=False "$temporary/$filename"
expected_identity="${rpm_version}:1:x86_64"
installed_identity=$(rpm -q --qf '%{VERSION}:%{RELEASE}:%{ARCH}' flow-beta)
if [[ "$installed_identity" != "$expected_identity" ]]; then
  printf 'Unexpected installed Flow RPM identity: expected %s, got %s\n' "$expected_identity" "$installed_identity" >&2
  exit 1
fi
if [[ ! -x /usr/bin/flow-desktop-beta || ! -f '/usr/share/applications/Flow Beta.desktop' ]]; then
  printf '%s\n' 'Flow Desktop executable or desktop entry is missing.' >&2
  exit 1
fi

if ! GST_REGISTRY="$temporary/gstreamer-registry.bin" \
  gst-inspect-1.0 avdec_h264 > "$temporary/h264.log" 2>&1; then
  cat "$temporary/h264.log" >&2
  printf '%s\n' 'Flow Desktop requires an available H.264 decoder. No repositories were added or enabled.' >&2
  exit 1
fi
