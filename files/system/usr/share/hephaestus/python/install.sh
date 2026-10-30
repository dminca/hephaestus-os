#!/usr/bin/env bash
set -euo pipefail

directory=$(cd "$(dirname "$0")" && pwd)
expected=$(cat "$directory/.python-version")
actual=$(python3 --version | cut -d ' ' -f 2 | cut -d . -f 1,2)
if [[ "$actual" != "$expected" ]]; then
  echo "Python $actual does not match the locked graph's Python $expected" >&2
  exit 1
fi

python3 -m venv "$1"
"$1/bin/pip" install --disable-pip-version-check \
  --require-hashes -r "$directory/requirements.txt"
