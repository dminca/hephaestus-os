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

Launch `nvim` normally with an internet connection. Missing plugins, language
servers, formatters, and parsers install automatically in your user account.
Most plugins are Lua; Mason and plugin build hooks use upstream prebuilt
releases where supported. Treesitter still compiles native parsers locally,
asynchronously, using the compiler and tools supplied by the image.

Basic editing remains available while tools install. Language-specific features
become available as their dependencies finish; reopen a buffer if its language
server has not attached yet. There are no automatic plugin or tool upgrades on
every launch.

| Task | Neovim command |
|---|---|
| Inspect plugin installation; retry a failed build | `:Lazy`, then select the plugin and press `gb` |
| Install missing plugins | `:Lazy install` |
| Deliberately update plugins | `:Lazy update` |
| Return plugins to the user lock | `:Lazy restore` |
| Inspect language servers and formatters | `:Mason` |
| Retry missing configured tools | `:MasonToolsInstall` |
| Install a specific language server | `:MasonInstall lua-language-server` |
| Update installed parsers | `:TSUpdate` |
| Retry a missing parser | `:TSInstall lua` (replace `lua` with its language) |
| Inspect installation failures | `:messages`, `:MasonLog`, `:checkhealth` |

The image's read-only configuration is linked into `~/.config/nvim`.
Plugins live in `stdpath("data")/lazy`, Mason tools in
`stdpath("data")/mason`, and parsers in `stdpath("data")/site`.
The configuration seeds `stdpath("state")/lazy-lock.json` once from the checked-in
native lazy.nvim lock. Subsequent launches and image updates preserve your
user lock. These paths normally resolve under `~/.local/share/nvim` and
`~/.local/state/nvim`, respectively.

Markdown preview, Typst preview, CodeSnap, and Cord use their plugin-managed
installation hooks rather than custom image builds. YAML formatting uses the
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
| Neovim plugins | `profile/config/nvim/lua/hephaestus/plugins.lua` under assets | Native `profile/config/nvim/lazy-lock.json` seed; user-managed state lock |
| Editor servers and formatters | Mason lists in `plugins.lua` and `languages.lua` | User-owned Mason installation |
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

To deliberately refresh the shared plugin seed, test the plugins after
`:Lazy update`, then copy the native user lock into
`files/system/usr/share/hephaestus/profile/config/nvim/lazy-lock.json`.
Held plugin revisions remain explicit in `plugins.lua`. Existing users retain
their own lock rather than having updates forced by the new image.

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
use isolated HOME/XDG directories with Neovim 0.11 or newer:

```bash
HEPHAESTUS_REPO="$PWD" nvim --headless -u NONE -l "$PWD/tests/check-neovim.lua"
```

This starts plugin installation if needed and requires network access on a
fresh account. Wait for installation to finish and retry after any reported
tool error. It is not part of the image build. CI checks the repository,
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
