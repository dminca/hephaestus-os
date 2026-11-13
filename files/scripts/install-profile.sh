#!/usr/bin/env bash
set -euo pipefail

assets=/usr/share/hephaestus
export PATH="/usr/libexec/hephaestus/bin:$PATH"
plugins="$assets/nvim/plugins"
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT
export HOME="$temporary/home"
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_CACHE_HOME="$HOME/.cache"
export XDG_DATA_HOME="$HOME/.local/share"
export XDG_STATE_HOME="$HOME/.local/state"
export PUPPETEER_SKIP_DOWNLOAD=true
mkdir -p "$HOME"

nvim --headless -u NONE '+lua if vim.fn.has("nvim-0.12") ~= 1 then io.stderr:write("Hephaestus requires Neovim 0.12+ with vim.pack\n"); vim.cmd("cquit") end' +qa

rpm -q git-credential-libsecret
credential_helper="$(git --exec-path)/git-credential-libsecret"
if [[ ! -x "$credential_helper" ]]; then
  printf 'Git libsecret credential helper is missing or not executable: %s\n' "$credential_helper" >&2
  exit 1
fi

CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath \
  -o "$temporary/hephaestus-profile" "$assets/installer/main.go"
install -m755 "$temporary/hephaestus-profile" /usr/bin/hephaestus-profile

kwallet-query --help >/dev/null

(
  cd "$assets/build-assets"
  CGO_ENABLED=0 GO111MODULE=off GOTOOLCHAIN=local go build -trimpath \
    -o "$temporary/hephaestus-assets" .
)
"$temporary/hephaestus-assets" sources
"$temporary/hephaestus-assets" binaries
cue version
witr --version
hunk --version

(
  cd "$assets/node"
  npm ci --omit=dev --no-audit --no-fund
)
node_binaries=$(node "$assets/node/check-tools.cjs" --list)
for binary in $node_binaries; do
  ln -s "$assets/node/node_modules/.bin/$binary" "/usr/libexec/hephaestus/bin/$binary"
done
install -m 0755 "$assets/mmdc" /usr/libexec/hephaestus/bin/mmdc

install -Dm644 "$plugins/bat/themes/Catppuccin Mocha.tmTheme" \
  "$assets/profile/config/bat/themes/CatppuccinMocha.tmTheme"
BAT_CACHE_PATH="$assets/bat" bat cache --build --source "$assets/profile/config/bat"
zsh -n "$assets/profile/zshrc"
