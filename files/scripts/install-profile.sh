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
export CARGO_HOME="$temporary/cargo"
export PUPPETEER_SKIP_DOWNLOAD=true
mkdir -p "$HOME"

CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath \
  -o "$temporary/hephaestus-profile" "$assets/installer/main.go"
install -m755 "$temporary/hephaestus-profile" /usr/bin/hephaestus-profile

(
  cd "$assets/build-assets"
  CGO_ENABLED=0 GO111MODULE=off GOTOOLCHAIN=local go build -trimpath \
    -o "$temporary/hephaestus-assets" .
)
"$temporary/hephaestus-assets" sources
"$temporary/hephaestus-assets" binaries
ln -s /usr/libexec/hephaestus/lua-language-server/bin/lua-language-server \
  /usr/libexec/hephaestus/bin/lua-language-server
make -C /usr/libexec/hephaestus/haskell-language-server install \
  PREFIX=/usr/libexec/hephaestus/hls \
  BINDIR=/usr/libexec/hephaestus/bin
rm -r -- /usr/libexec/hephaestus/haskell-language-server

cargo install --locked --git https://github.com/oxalica/nil \
  --rev dbbb5c24f9215476992aa1bd305f8e5398e89654 --root /usr/libexec/hephaestus nil
cargo install --locked --git https://github.com/nerdypepper/statix \
  --rev a5292554eddc5aebae8b66a0ebe37a4aa5720442 --root /usr/libexec/hephaestus statix

cue version
marksman --version
markdown-oxide --version
tofu-ls version
lua-language-server --version
haskell-language-server-wrapper --version
nil --version
statix --version
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

node "$assets/node/node_modules/yarn/bin/yarn.js" \
  --cwd "$plugins/markdown-preview.nvim/app" install \
  --frozen-lockfile --production --ignore-scripts --non-interactive

cargo build --locked --release --manifest-path "$plugins/cord.nvim/Cargo.toml" \
  --target-dir "$temporary/cord-target"
install -Dm755 "$temporary/cord-target/release/cord" /usr/libexec/hephaestus/cord/bin/cord

tree-sitter --version
expected_tree_sitter_version=$("$temporary/hephaestus-assets" binary-version tree-sitter)
test "$(tree-sitter --version | cut -d ' ' -f 2)" = "$expected_tree_sitter_version"
nvim --headless -u NONE -l "$assets/build-parsers.lua"

(
  cd "$assets/go-tools"
  GOTOOLCHAIN=local go build -mod=readonly -trimpath \
    -o /usr/libexec/hephaestus/bin/goimports golang.org/x/tools/cmd/goimports
)
bash "$assets/python/install.sh" /usr/libexec/hephaestus/python
ln -s /usr/libexec/hephaestus/python/bin/yamlfix /usr/libexec/hephaestus/bin/yamlfix

install -Dm644 "$plugins/bat/themes/Catppuccin Mocha.tmTheme" \
  "$assets/profile/config/bat/themes/CatppuccinMocha.tmTheme"
BAT_CACHE_PATH="$assets/bat" bat cache --build --source "$assets/profile/config/bat"
zsh -n "$assets/profile/zshrc"
HEPHAESTUS_REPO=/tmp/bluebuild nvim --headless -u "$assets/profile/config/nvim/init.lua" \
  '+lua assert(vim.g.mapleader == " " and vim.o.shiftwidth == 2)' '+qa!'
nvim --headless -u "$assets/profile/config/nvim/init.lua" -l "$assets/check-language-servers.lua"
