#!/usr/bin/env bash
set -euo pipefail

brew=/usr/share/homebrew/home/linuxbrew/.linuxbrew/bin/brew
staged_prefix=/usr/share/homebrew/home/linuxbrew/.linuxbrew
prefix_link=/home/linuxbrew/.linuxbrew
formula=/home/linuxbrew/.linuxbrew/opt/fd/bin/fd
destination=/usr/libexec/hephaestus/bin/fd
temporary=
created_linuxbrew_home=false

if [[ ! -x "$brew" ]]; then
  echo "BlueBuild did not stage Homebrew in /usr/share/homebrew" >&2
  exit 1
fi

if [[ -e "$prefix_link" || -L "$prefix_link" ]]; then
  echo "Refusing to overwrite an existing Homebrew prefix: $prefix_link" >&2
  exit 1
fi

cleanup() {
  rm -f "$prefix_link"
  if [[ -n "$temporary" ]]; then
    rm -rf "$temporary"
  fi
  if [[ "$created_linuxbrew_home" == true ]]; then
    rmdir /home/linuxbrew
  fi
}
trap cleanup EXIT

if [[ ! -d /home/linuxbrew ]]; then
  mkdir -p /home/linuxbrew
  created_linuxbrew_home=true
fi
ln -s "$staged_prefix" "$prefix_link"
temporary=$(mktemp -d)
chown 1000:1000 "$temporary"

setpriv --reuid=1000 --regid=1000 --clear-groups env \
  HOME="$temporary" \
  HOMEBREW_CACHE="$temporary/cache" \
  HOMEBREW_TEMP="$temporary/tmp" \
  HOMEBREW_NO_AUTO_UPDATE=1 \
  HOMEBREW_NO_INSTALL_CLEANUP=1 \
  HOMEBREW_NO_ANALYTICS=1 \
  "$brew" install fd

if [[ ! -x "$formula" ]]; then
  echo "Homebrew installed fd but its executable is missing: $formula" >&2
  exit 1
fi

install -Dm755 "$formula" "$destination"
"$destination" --version
