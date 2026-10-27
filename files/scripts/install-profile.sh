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
witr --version
hunk --version

npm ci --prefix "$assets/node" --omit=dev --no-audit --no-fund
for binary in bash-language-server typescript-language-server tsserver tsc \
  vscode-css-language-server vscode-eslint-language-server vscode-html-language-server \
  vscode-json-language-server tailwindcss-language-server svelteserver yaml-language-server \
  emmet-ls biome pyright pyright-langserver fixjson; do
  test -x "$assets/node/node_modules/.bin/$binary"
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
test "$(tree-sitter --version | cut -d ' ' -f 2)" = "0.26.1"
nvim --headless -u NONE -l "$assets/build-parsers.lua"

GOBIN=/usr/libexec/hephaestus/bin go install golang.org/x/tools/cmd/goimports@v0.31.0
python3 -m venv /usr/libexec/hephaestus/python
/usr/libexec/hephaestus/python/bin/pip install --disable-pip-version-check 'yamlfix==1.19.1'
ln -s /usr/libexec/hephaestus/python/bin/yamlfix /usr/libexec/hephaestus/bin/yamlfix

install -Dm644 "$plugins/bat/themes/Catppuccin Mocha.tmTheme" \
  "$assets/profile/config/bat/themes/CatppuccinMocha.tmTheme"
BAT_CACHE_PATH="$assets/bat" bat cache --build --source "$assets/profile/config/bat"
zsh -n "$assets/profile/zshrc"
HEPHAESTUS_REPO=/tmp/bluebuild nvim --headless -u "$assets/profile/config/nvim/init.lua" \
  '+lua assert(vim.g.mapleader == " " and vim.o.shiftwidth == 2)' '+qa!'
