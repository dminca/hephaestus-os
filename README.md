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
CI rejects a base whose Fedora major differs from the parser-validation
container; update and validate that container deliberately when the base moves.

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

The Nextcloud desktop sync client (`com.nextcloud.desktopclient.nextcloud`) is
installed system-wide from Flathub and starts in the background at desktop login.
After its initial Flatpak installation completes, log out and back in to start it,
or open Nextcloud manually. Account setup and sync folders remain per-user.
An existing user autostart override can disable the system autostart entry.

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
Builds verify downloads against the locked checksums, and both image and parser
CI builds check the installed Tree-sitter version against the lock before
compiling parsers. Parser CI also builds and loads the configured parsers to
check compatibility; matching a version or checksum alone does not establish it.

## Dependency ownership and compatibility

Exact versions make downloads predictable; they do not prove that dependencies
work together. Keep each dependency graph in its native manifest and lock, and
commit both together. Image builds consume locks; they must not repair them.

| Dependency | Edit | Generated/consumed metadata |
|---|---|---|
| Editor npm tools | `files/system/usr/share/hephaestus/node/package.json` | Adjacent `package-lock.json` |
| Renovate itself | `.github/renovate/package.json` | Adjacent `package-lock.json` |
| Python yamlfix | `files/system/usr/share/hephaestus/python/requirements.in` | Adjacent hash-pinned `requirements.txt` |
| Go tools | `files/system/usr/share/hephaestus/go-tools/go.mod` | Adjacent `go.sum`; Go `tool` directive |
| Rust nil/statix | Full Git revisions in `files/scripts/install-profile.sh` | Upstream Cargo locks, enforced with `--locked` |
| Neovim plugins | `sources.json` and `profile/config/nvim/lua/hephaestus/plugins.lua` under assets | `sources.lock.json`, regenerated by `go run ./tools/lock-profile` |
| Release binaries | `tools/lock-binaries/main.go` | `binaries.lock.json`, regenerated by `go run ./tools/lock-binaries` |
| BlueBuild tooling | `recipes/recipe.yml` and the workflow's `cli_version` | Explicit CLI version, module and Nushell image digests |

Assets live under `files/system/usr/share/hephaestus`. Custom tools are installed
under `/usr/libexec/hephaestus`, without overwriting distribution executables.
Neovim already loads bundled local plugins; automatic plugin installation and
update checking are disabled. Keep external language servers in their native
tool graph rather than adding runtime downloads through a plugin manager.

The editor bundle requires Node `^24.11.0`; Renovate's locked graph requires
`^24.15.0` because its native dependencies have a higher minimum than Renovate's
own declaration. Both accept npm 11 and declare the tested npm version in
`packageManager`. CI provisions that version explicitly; the image uses Fedora's
npm 11. Project `.npmrc` files enforce engines and peer dependencies. Run npm
from the project directory so that its configuration is loaded.

TypeScript is intentionally held below 7 in `renovate.json5`: the pinned Svelte
server excludes TypeScript 7, and TypeScript 7 no longer provides the `tsserver`
executable required by this image. Removing that ceiling requires a reviewed
consumer migration, not merely a regenerated lockfile. TypeScript/Svelte updates
are grouped separately from unrelated editor tools and Renovate itself.

### Adding or updating a tool

For an editor npm tool, select a reviewed exact version and update its lock using
the project's declared Node/npm versions:

```bash
cd files/system/usr/share/hephaestus/node
npm install --save-exact --package-lock-only --ignore-scripts PACKAGE@VERSION
PUPPETEER_SKIP_DOWNLOAD=true npm ci --omit=dev --no-audit --no-fund
```

Replace `PACKAGE@VERSION` with the actual package and version. If it exposes a
new executable, add it to `node/check-tools.cjs`; the image validates and links
that list. Also wire its consumer, such as Neovim's `languages.lua`, and verify
its real startup behavior. For Renovate, use the same commands in
`.github/renovate` instead; never add Renovate to the editor bundle.

For Python, use the Python minor in `python/.python-version` and the lock
compiler version in `.github/python-tools.txt`:

```bash
cd files/system/usr/share/hephaestus/python
pip-compile --generate-hashes --allow-unsafe --strip-extras --no-emit-index-url --output-file=requirements.txt requirements.in
bash install.sh /tmp/hephaestus-python-check
```

Choose a fresh destination for the check environment. The installer rejects
another Python minor and installs only hash-pinned requirements. Renovate's
`pip-compile` manager maintains the source and output together.

For Go tools, use a local compiler satisfying the tool module's `go` directive:

```bash
cd files/system/usr/share/hephaestus/go-tools
GOTOOLCHAIN=local go get -tool golang.org/x/tools/cmd/goimports@VERSION
GOTOOLCHAIN=local go mod tidy
GOTOOLCHAIN=local go build -mod=readonly -o /tmp/goimports golang.org/x/tools/cmd/goimports
```

Native Renovate managers own npm, Python, and Go metadata. Custom managers cover
plugin commits, binary releases, Rust revisions, Flow Desktop, and BlueBuild
pins. Extend the appropriate manager when adding a new dependency surface.
Never bypass incompatibilities with `--force` or `--legacy-peer-deps`.

### Where to put dependency tests

| Behavior | Test location |
|---|---|
| Node/npm engine enforcement | `tests/dependencies/node-runtime.test.cjs` |
| Manifest/lock synchronization, peer conflicts, runtime minima | `tests/dependencies/npm-install.test.cjs` |
| Real Renovate rule resolution, extraction, artifact failure status | `tests/dependencies/renovate-policy.test.cjs` |
| Editor executable presence and startup | `tests/dependencies/editor-tools.test.cjs` |
| Python/Go/Rust installation and rejection paths | `tests/dependency_tools_test.go` (integration build tag) |
| Fedora alignment and Python source/lock agreement | `tests/dependency_policy_test.go` |
| Binary/source catalogs and lock generators | Existing `tests/lock_tools_test.go`, `tools/lock-*/main_test.go`, and asset-helper tests |

Use plain JavaScript with Node's built-in runner for npm/Renovate behavior and
Go's `testing` package for other tooling. Write one clear supported case and one
clear rejected case for each new constraint. Use temporary minimal fixtures,
descriptive names, direct assertions, and the expected diagnostic; unrelated
network failures must not count as successful rejection tests.

```bash
go test -count=1 ./...
node --test tests/renovate-config.test.cjs tests/dependencies/*.test.cjs
go test -count=1 -tags=integration ./tests -run 'TestPython|TestGoTool|TestRust' -v
.github/renovate/node_modules/.bin/renovate-config-validator --strict renovate.json5
```

Install both locked npm projects first. Integration tests require the target
Python, local Go compiler, and Cargo/rustc; they intentionally fail rather than
skip when prerequisites are missing. Root Go tests do not traverse the nested
tool module. CI runs native tool installation separately and checks actual Node
runtime boundaries before parser/image builds. The Fedora job checks Linux
editor executables; the final image still compiles parsers and validates Neovim
language servers. Keep these jobs blocking when configuring branch protection.

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
