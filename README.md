# hephaestus-os &nbsp; [![bluebuild build badge](https://github.com/dminca/hephaestus-os/actions/workflows/build.yml/badge.svg)](https://github.com/dminca/hephaestus-os/actions/workflows/build.yml)

See the [BlueBuild docs](https://blue-build.org/how-to/setup/) for quick setup instructions for setting up your own repository based on this template.

After setup, it is recommended you update this README to describe your custom image.

## Installation

> [!WARNING]  
> [This is an experimental feature](https://www.fedoraproject.org/wiki/Changes/OstreeNativeContainerStable), try at your own discretion.

To rebase an existing atomic Fedora installation to the latest build:

- First rebase to the unsigned image, to get the proper signing keys and policies installed:
  ```
  rpm-ostree rebase ostree-unverified-registry:ghcr.io/dminca/hephaestus-os:latest
  ```
- Reboot to complete the rebase:
  ```
  systemctl reboot
  ```
- Then rebase to the signed image, like so:
  ```
  rpm-ostree rebase ostree-image-signed:docker://ghcr.io/dminca/hephaestus-os:latest
  ```
- Reboot again to complete the installation
  ```
  systemctl reboot
  ```

The published `latest` tag points to the latest successful build. The recipe
also follows Secureblue's `latest` base, which can move to a new Fedora major.
Editor dependencies are installed in your user account, not during the image
build, so a parser or language-server failure does not block OS updates.

## Hunk

The image includes [Hunk](https://github.com/modem-dev/hunk) v0.24.1 as a
checksum-verified standalone executable at `/usr/libexec/hephaestus/bin/hunk`.
The managed Zsh profile includes this directory in PATH. Run `hunk diff` to
review changes. Update its pin in `tools/lock-binaries/main.go`, regenerate
`binaries.lock.json` with `go run ./tools/lock-binaries`, and rebuild the image;
do not use `hunk update` to modify the immutable image installation.

## Desktop preferences

On the first Plasma login after deploying the image, a native Plasma update script
enables seconds on existing digital clock widgets and maps Caps Lock to Escape.
Other widgets, keyboard layouts, and unrelated XKB options are preserved.
The keyboard change may require one additional logout/login to take effect.
These are saved per-user preferences, applied once rather than forced on every
login; they can subsequently be changed in KDE settings.

An additional Plasma update enables key repeat with a **233 ms delay** and
**17 repeats/s**. Log out and back in again if the active session has not picked
up the new values.

Nextcloud (`com.nextcloud.desktopclient.nextcloud`) is installed system-wide
from `flathub-verified`, filtered with `--subset=verified` before installation.
Nextcloud is a verified Flathub application. The recipe keeps the installation
system-wide and no longer adds an unrestricted user Flathub remote; Secureblue's
user remote policy is left alone. Existing unrestricted remotes are not removed.
Nextcloud starts
in the background at desktop login. After its initial Flatpak installation
completes, log out and back in to start it, or open Nextcloud manually. Account
setup and sync folders remain per-user. An existing user autostart override can
disable the system autostart entry.

## Updating pinned binaries

The `artifacts` catalog in `tools/lock-binaries/main.go` is the master list of
binary versions, release asset filenames, executable names, and licenses.
To upgrade a binary, edit its catalog entry, then regenerate the download URLs
and verified SHA256 checksums:

```bash
go run ./tools/lock-binaries
```

This requires authenticated `gh` access and `curl`. Review and commit both the
catalog change and the generated
`files/system/usr/share/hephaestus/binaries.lock.json`; do not edit the lock
independently. An offline test detects a lock that no longer matches the catalog.
Tests and builds consume the generated metadata without separate release constants.
Builds verify system-tool downloads against the locked checksums.

## Neovim: first launch and maintenance

This is a Secureblue/Fedora Atomic configuration, not a NixOS configuration.
The image supplies **Neovim 0.12+**, Git, compilers and runtime prerequisites;
`vim.pack` manages plugins and native `vim.lsp` manages language-server
activation. `nvim-lspconfig` supplies server definitions, not a second LSP
client. The design follows the native APIs used by
[craftzdog/dotfiles](https://github.com/craftzdog/dotfiles/tree/master/dot_config/nvim),
while preserving Hephaestus options, mappings and plugin settings.

After deploying the new image, run `hephaestus-profile`. When migrating from
the old whole-directory Neovim symlink (or conflicting personal files), run
`hephaestus-profile --replace` instead: it keeps timestamped backups before
activation. `~/.config/nvim` is now a writable directory containing image-backed
`init.lua` and `lua` symlinks, leaving room for the native user-owned lockfile.
`XDG_CONFIG_HOME` is respected.

Launch `nvim` with an internet connection. Missing plugins install synchronously
before their setup functions run. Mason installs missing language servers and
formatters asynchronously through `mason-tool-installer.nvim`; all entries
have prerequisite conditions. For example, Go tools need `go`, Node packages
need `node` and `npm`, `nil` needs Cargo/Rust/C, and Haskell needs GHCup and GHC.
Missing prerequisites produce warnings rather than failed blind installations.
Nix-language editing support does not make the host NixOS: downloaded tools use
normal platform defaults, without NixOS targets or compatibility wrappers.

Treesitter compiles parsers locally and asynchronously using image-supplied
tools. LSP clients are enabled only when the executable and prerequisites are
available, including after Mason completes installation. Rust attachment stays
owned by rustaceanvim. There are no network upgrades on every launch: Renovate
proposes plugin pin changes; Mason tools and parsers have explicit update commands.

| Task | Neovim command |
|---|---|
| Install missing plugins | Restart `nvim` |
| Inspect installed plugins | `:lua vim.print(vim.pack.get())` |
| Apply the image's reviewed plugin pins | `:lua vim.pack.update()`; review the buffer and `:write` to confirm, then restart |
| Return plugins to the user lock | `:lua vim.pack.update(nil, { offline = true, target = "lockfile" })`; review and `:write` |
| Inspect language servers and formatters | `:Mason` |
| Retry missing configured tools | `:MasonToolsInstall` |
| Deliberately update configured tools | `:MasonToolsUpdate` |
| Install a specific language server | `:MasonInstall lua-language-server` |
| Update installed parsers | `:TSUpdate` |
| Retry a missing parser | `:TSInstall lua` (replace `lua` with its language) |
| Inspect installation failures | `:messages`, `:MasonLog`, `:checkhealth` |

Plugins live in `stdpath("data")/site/pack/core/opt`, Mason tools in
`stdpath("data")/mason`, and parsers in `stdpath("data")/site`.
Neovim generates `stdpath("config")/nvim-pack-lock.json`, normally
`~/.config/nvim/nvim-pack-lock.json`. Subsequent launches, profile activation
and image updates preserve that native lock. The old `data/lazy` tree and
`state/lazy-lock.json` are no longer used and are not deleted automatically.
Save a copy of the native lock before updating if you need to roll back editor
revisions; an OS rollback alone does not roll back user dependencies.

Markdown preview, Typst preview, CodeSnap, and Cord use their plugin-managed
installers rather than custom image builds. `PackChanged` runs CodeSnap's
`make` and Markdown Preview's installer on install/update and refreshes parsers
on Treesitter updates. To retry a CodeSnap build, run `make` in its plugin
directory; for Markdown Preview, run `:call mkdp#util#install_sync()`.
Kulala's HTTP parser setup waits until Mason's `tree-sitter` executable is
available, then configures automatically; opening an HTTP buffer before that
shows a warning instead of attempting a broken parser build.
YAML formatting uses the
YAML language server through Conform's existing LSP formatting preference;
there is no separate Python formatter environment. Rust language-server
attachment remains owned by rustaceanvim, not a second Mason LSP client.

Some upstream installers still need local compilers or additional prerequisites.
In particular, Mason's Haskell language server installer requires
[GHCup](https://www.haskell.org/ghcup/); install GHCup in your user account first,
then run `:MasonInstall haskell-language-server`. Installation failures are
visible in the managers and can be retried without rebuilding the OS.

## Dependency ownership and compatibility

Exact versions make downloads predictable; they do not prove that dependencies
work together. Keep each dependency graph in its native manifest and lock, and
commit both together. Image builds consume system-tool locks; they must not
repair them. Editor updates are managed separately in the user account.

| Dependency | Edit | Generated/consumed metadata |
|---|---|---|
| System Mermaid CLI | `files/system/usr/share/hephaestus/node/package.json` | Adjacent `package-lock.json` |
| Renovate itself | `.github/renovate/package.json` | Adjacent `package-lock.json` |
| `fd` | `files/scripts/install-fd.sh` and the pinned BlueBuild Brew module | Installed from Homebrew during image build, smoke-checked, and copied into `/usr/libexec/hephaestus/bin` |
| Neovim plugins | Native specs/pins in `profile/config/nvim/lua/hephaestus/packages.lua`; settings in `plugins.lua` | User-owned `~/.config/nvim/nvim-pack-lock.json` |
| Editor servers and formatters | Mason registry names and prerequisite conditions in `tools.lua`; native LSP settings in `languages.lua` | User-owned Mason installation |
| Treesitter parsers | Language list in `profile/config/nvim/lua/hephaestus/treesitter.lua` | User-owned parser libraries |
| Release binaries | `tools/lock-binaries/main.go` | `binaries.lock.json`, regenerated by `go run ./tools/lock-binaries` |
| BlueBuild tooling | `recipes/recipe.yml` and the workflow's `cli_version` | Explicit CLI version, module and Nushell image digests |

Assets live under `files/system/usr/share/hephaestus`. Custom tools are installed
under `/usr/libexec/hephaestus`, without overwriting distribution executables.
The image retains system tools including CUE, Yazi, Websocat, Ziggity, Witr,
Hunk, `fd`, and Mermaid CLI, plus the Bat theme. `sources.lock.json` now covers
only that theme; editor plugin archives and tool builds are no longer part of
the image dependency graph.

The system preview tool requires Node `^24.11.0`; Renovate's locked graph requires
`^24.15.0` because its native dependencies have a higher minimum than Renovate's
own declaration. Both accept npm 11 and declare the tested npm version in
`packageManager`. CI provisions that version explicitly; the image uses Fedora's
npm 11. Project `.npmrc` files enforce engines and peer dependencies. Run npm
from the project directory so that its configuration is loaded.

### Adding or updating a tool

For a system npm tool, select a reviewed exact version and update its lock using
the project's declared Node/npm versions:

```bash
cd files/system/usr/share/hephaestus/node
npm install --save-exact --package-lock-only --ignore-scripts PACKAGE@VERSION
PUPPETEER_SKIP_DOWNLOAD=true npm ci --omit=dev --no-audit --no-fund
```

Replace `PACKAGE@VERSION` with the actual package and version. If it exposes a
new executable, add it to `node/check-tools.cjs`; the image validates and links
that list and verify the tool's real startup behavior. For Renovate, use the
same commands in `.github/renovate` instead. For editor tools, add the Mason
package to the corresponding plugin configuration and test it from Neovim;
do not add a second image installer.

Renovate tracks native plugin commit pins with `git-refs` and release pins with
`github-releases`. Harpoon tracks `harpoon2`; other revision pins follow the
upstream default branch. The existing LeetCode and Treesitter compatibility
holds remain excluded from automatic PRs, and Blink requires dashboard approval.
After deploying a reviewed plugin update, use `vim.pack.update()` and restart.
Existing plugin checkouts are not silently changed just because the image changed.
Mason tools are intentionally unpinned and updated with `:MasonToolsUpdate`;
Renovate cannot update mutable user installations or Treesitter-generated parsers.

Native Renovate managers own system npm and repository Go metadata.
Custom managers cover binary releases, Flow Desktop, and BlueBuild
pins. Extend the appropriate manager when adding a new dependency surface.
Never bypass incompatibilities with `--force` or `--legacy-peer-deps`.
For default-branch Git digest tracking, omit the custom manager's
`currentValueTemplate`; an explicit `HEAD` is treated as a named branch/tag by
Renovate's `git-refs` digest lookup and cannot resolve the default branch.

The Renovate job logs at debug level and propagates repository errors to CI.
If its own upgrade fails with npm `ERESOLVE`, update the exact Renovate version
in `.github/renovate/package.json` and regenerate the adjacent lock with
`npm install --package-lock-only --ignore-scripts` from that directory.
Keep the existing lock as the starting point: Renovate's automated update can
remove its package entry while retaining incompatible old transitive peers.
Commit the manifest and regenerated lock together; keep strict peer checks enabled.

### Where to put dependency tests

| Behavior | Test location |
|---|---|
| Node/npm engine enforcement | `tests/dependencies/node-runtime.test.cjs` |
| Manifest/lock synchronization and runtime minima | `tests/dependencies/npm-install.test.cjs` |
| Real Renovate rule resolution, extraction, artifact failure status | `tests/dependencies/renovate-policy.test.cjs` |
| System preview executable presence and startup | `tests/dependencies/preview-tools.test.cjs` |
| User bootstrap, asynchronous parsers, Mason LSP ownership | `tests/neovim_bootstrap_test.go`, `tests/parsers_test.go`, `tests/language_servers_test.go` |
| Real editor startup and preserved settings | `tests/check-neovim.lua` |
| Binary catalog and lock generator | `tests/lock_tools_test.go`, `tools/lock-binaries/main_test.go`, and asset-helper tests |

Use plain JavaScript with Node's built-in runner for npm/Renovate behavior and
Go's `testing` package for other tooling. Write one clear supported case and one
clear rejected case for each new constraint. Use temporary minimal fixtures,
descriptive names, direct assertions, and the expected diagnostic; unrelated
network failures must not count as successful rejection tests.

```bash
go test -count=1 ./...
node --test tests/renovate-config.test.cjs tests/dependencies/*.test.cjs
.github/renovate/node_modules/.bin/renovate-config-validator --strict renovate.json5
```

Install both locked npm projects first. Neovim fixture tests require Neovim
locally and skip when it is unavailable. For a real user-editor smoke check,
use isolated HOME/XDG directories with Neovim 0.12 or newer:

```bash
HEPHAESTUS_REPO="$PWD" nvim --headless -u NONE -l "$PWD/tests/check-neovim.lua"
```

This starts plugin installation if needed and requires network access on a
fresh account. Wait for installation to finish and retry after any reported
tool error. It is not part of the image build. CI installs stable Neovim
to run the Lua fixture tests rather than skipping them. The image build
checks the Fedora Neovim RPM for 0.12+ without provisioning any plugins.
CI checks the repository,
remaining system dependencies, and Renovate policy before building the image;
it no longer duplicates editor tool provisioning or runs an eight-cell Node
matrix.

Fedora RPMs, Secureblue's base, and Flatpaks intentionally receive rolling
updates. These locks do not promise bit-for-bit image reproduction or offline
availability. Inspect Renovate's artifact errors if a PR changes a manifest
without its lock: that PR is not installable, even if extraction succeeded.

## ISO

If build on Fedora Atomic, you can generate an offline ISO with the instructions available [here](https://blue-build.org/how-to/generate-iso/#_top). These ISOs cannot unfortunately be distributed on GitHub for free due to large sizes, so for public projects something else has to be used for hosting.

## Verification

These images are signed with [Sigstore](https://www.sigstore.dev/)'s [cosign](https://github.com/sigstore/cosign). You can verify the signature by downloading the `cosign.pub` file from this repo and running the following command:

```bash
cosign verify --key cosign.pub ghcr.io/dminca/hephaestus-os
```
