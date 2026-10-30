#!/usr/bin/env bash
set -euo pipefail

expected=$1
actual=${2%%.*}
if [[ "$actual" != "$expected" ]]; then
  echo "Base Fedora $2 differs from parser Fedora $expected" >&2
  exit 1
fi
