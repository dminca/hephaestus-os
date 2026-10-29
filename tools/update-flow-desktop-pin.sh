#!/usr/bin/env bash
set -euo pipefail

script=files/scripts/install-flow-desktop.sh
mapfile -t versions < <(sed -nE 's/^version=(v[0-9][A-Za-z0-9.+-]*)[[:space:]]+# renovate: datasource=github-releases depName=Flow-Tube\/Flow-Desktop$/\1/p' "$script")
if [[ ${#versions[@]} -ne 1 ]]; then
  printf '%s\n' 'Expected one Renovate-managed Flow Desktop version' >&2
  exit 1
fi
version=${versions[0]}
mapfile -t keys < <(sed -nE 's/^public_key=([A-Za-z0-9+/=]+)$/\1/p' "$script")
if [[ ${#keys[@]} -ne 1 ]]; then
  printf '%s\n' 'Expected one pinned Flow Desktop updater key' >&2
  exit 1
fi

rpm_version=${version#v}
filename="Flow_${rpm_version}_linux_x86_64.rpm"
url="https://github.com/Flow-Tube/Flow-Desktop/releases/download/${version}/${filename}"
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

curl --fail --show-error --silent --location --retry 3 \
  --proto '=https' --proto-redir '=https' "$url" -o "$temporary/$filename"
curl --fail --show-error --silent --location --retry 3 \
  --proto '=https' --proto-redir '=https' "$url.sig" -o "$temporary/signature.base64"
base64 --decode < "$temporary/signature.base64" > "$temporary/signature"
minisign -Vm "$temporary/$filename" -x "$temporary/signature" -P "${keys[0]}"
checksum=$(sha256sum "$temporary/$filename" | awk '{print $1}')

python3 - "$script" "$checksum" <<'PY'
import os
import re
import stat
import sys
import tempfile
from pathlib import Path

path = Path(sys.argv[1])
checksum = sys.argv[2]
if not re.fullmatch(r"[a-f0-9]{64}", checksum):
    raise SystemExit("invalid SHA256 checksum")
content = path.read_text()
pattern = re.compile(r"(?m)^checksum=[a-f0-9]{64}$")
updated, count = pattern.subn(f"checksum={checksum}", content)
if count != 1:
    raise SystemExit("expected exactly one pinned Flow Desktop checksum")
descriptor, temporary = tempfile.mkstemp(dir=path.parent, prefix=".flow-pin-")
try:
    with os.fdopen(descriptor, "w") as output:
        output.write(updated)
    os.chmod(temporary, stat.S_IMODE(path.stat().st_mode))
    os.replace(temporary, path)
finally:
    if os.path.exists(temporary):
        os.unlink(temporary)
PY
