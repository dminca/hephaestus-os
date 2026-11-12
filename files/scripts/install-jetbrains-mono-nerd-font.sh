#!/usr/bin/env bash
set -euo pipefail

version=v3.5.1
checksum=04d5e8f903693f9dd13e16f867e994834e681eb3c72c0d337a770dcda09010cf
url="https://github.com/ryanoasis/nerd-fonts/releases/download/${version}/JetBrainsMono.tar.xz"
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

curl --fail --show-error --silent --location --retry 3 \
  --proto '=https' --proto-redir '=https' "$url" -o "$temporary/font.tar.xz"
printf '%s  %s\n' "$checksum" "$temporary/font.tar.xz" | sha256sum -c -
tar -xJf "$temporary/font.tar.xz" -C "$temporary"
test -f "$temporary/JetBrainsMonoNerdFont-Regular.ttf"
install -d /usr/share/fonts/jetbrains-mono-nerd-font
install -m 0644 "$temporary"/*.ttf /usr/share/fonts/jetbrains-mono-nerd-font/
install -d /usr/share/licenses/jetbrains-mono-nerd-font
install -m 0644 "$temporary/OFL.txt" /usr/share/licenses/jetbrains-mono-nerd-font/OFL.txt
fc-cache -f /usr/share/fonts/jetbrains-mono-nerd-font
