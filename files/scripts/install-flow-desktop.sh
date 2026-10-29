#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
  printf '%s\n' 'Flow Desktop packaging requires Linux x86_64.' >&2
  exit 1
fi

version=v0.1.0-beta2 # renovate: datasource=github-releases depName=Flow-Tube/Flow-Desktop
rpm_version="${version#v}"
filename="Flow_${rpm_version}_linux_x86_64.rpm"
url="https://github.com/Flow-Tube/Flow-Desktop/releases/download/${version}/${filename}"
checksum=7a2b2ab7add117ac70652bee52b499360ed57bd36fa368cbad036af9f5bcdcee
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

test "$(rpm -qp --qf '%{NAME}:%{VERSION}:%{RELEASE}:%{ARCH}' "$temporary/$filename")" \
  = "flow-beta:${rpm_version}:1:x86_64"
# Dependencies use the inherited repositories and their existing signature policy.
dnf5 install --assumeyes --setopt=install_weak_deps=False "$temporary/$filename"
test "$(rpm -q --qf '%{VERSION}:%{RELEASE}:%{ARCH}' flow-beta)" = "${rpm_version}:1:x86_64"
test -x /usr/bin/flow
test -f '/usr/share/applications/Flow Beta.desktop'

if ! GST_REGISTRY="$temporary/gstreamer-registry.bin" \
  gst-inspect-1.0 avdec_h264 > "$temporary/h264.log" 2>&1; then
  cat "$temporary/h264.log" >&2
  printf '%s\n' 'Flow Desktop requires an available H.264 decoder. No repositories were added or enabled.' >&2
  exit 1
fi
