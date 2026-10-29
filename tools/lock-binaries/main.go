package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dminca/hephaestus-os/internal/lockfile"
)

type artifactSpec struct {
	repository string
	version    string
	filename   string
	binaries   []string
	license    string
	bundle     string
	stripRoot  bool
}

var artifacts = []artifactSpec{
	{
		repository: "cue-lang/cue", version: "v0.17.1",
		filename: "cue_v0.17.1_linux_amd64.tar.gz",
		binaries: []string{"cue"}, license: "Apache-2.0",
	},
	{
		repository: "artempyanykh/marksman", version: "2026-02-08",
		filename: "marksman-linux-x64",
		binaries: []string{"marksman"}, license: "MIT",
	},
	{
		repository: "Feel-ix-343/markdown-oxide", version: "v0.25.12",
		filename: "markdown-oxide-v0.25.12-x86_64-unknown-linux-gnu",
		binaries: []string{"markdown-oxide"}, license: "Apache-2.0",
	},
	{
		repository: "opentofu/tofu-ls", version: "v0.5.3",
		filename: "tofu-ls_Linux_x86_64.tar.gz",
		binaries: []string{"tofu-ls"}, license: "MPL-2.0",
	},
	{
		repository: "LuaLS/lua-language-server", version: "3.19.2",
		filename: "lua-language-server-3.19.2-linux-x64.tar.gz",
		binaries: []string{"lua-language-server"}, license: "MIT",
		bundle: "lua-language-server",
	},
	{
		repository: "haskell/haskell-language-server", version: "2.15.0.0",
		filename: "haskell-language-server-2.15.0.0-x86_64-linux-fedora40.tar.xz",
		binaries: []string{"haskell-language-server-wrapper"}, license: "Apache-2.0",
		bundle: "haskell-language-server", stripRoot: true,
	},
	{
		repository: "tree-sitter/tree-sitter", version: "v0.26.1",
		filename: "tree-sitter-linux-x64.gz",
		binaries: []string{"tree-sitter"}, license: "MIT",
	},
	{
		repository: "sxyazi/yazi", version: "v26.9.1",
		filename: "yazi-x86_64-unknown-linux-musl.zip",
		binaries: []string{"yazi", "ya"}, license: "MIT",
	},
	{
		repository: "JohnnyMorganz/StyLua", version: "v2.6.0",
		filename: "stylua-linux-x86_64-musl.zip",
		binaries: []string{"stylua"}, license: "MPL-2.0",
	},
	{
		repository: "anordal/shellharden", version: "v4.3.2",
		filename: "shellharden-x86_64-unknown-linux-musl.tar.gz",
		binaries: []string{"shellharden"}, license: "MPL-2.0",
	},
	{
		repository: "tamasfe/taplo", version: "0.10.0",
		filename: "taplo-linux-x86_64.gz",
		binaries: []string{"taplo"}, license: "MIT",
	},
	{
		repository: "tekumara/typos-lsp", version: "v0.1.57",
		filename: "typos-lsp-v0.1.57-x86_64-unknown-linux-musl.tar.gz",
		binaries: []string{"typos-lsp"}, license: "MIT",
	},
	{
		repository: "Myriad-Dreamin/tinymist", version: "v0.15.10",
		filename: "tinymist-x86_64-unknown-linux-musl.tar.gz",
		binaries: []string{"tinymist"}, license: "Apache-2.0",
	},
	{
		repository: "typstyle-rs/typstyle", version: "v0.15.2",
		filename: "typstyle-x86_64-unknown-linux-musl",
		binaries: []string{"typstyle"}, license: "Apache-2.0",
	},
	{
		repository: "vi/websocat", version: "v1.14.1",
		filename: "websocat.x86_64-unknown-linux-musl",
		binaries: []string{"websocat"}, license: "MIT",
	},
	{
		repository: "simoarpe/ziggity", version: "v0.54.0",
		filename: "ziggity-v0.54.0-x86_64-linux-musl.tar.gz",
		binaries: []string{"ziggity"}, license: "MIT",
	},
	{
		repository: "pranshuparmar/witr", version: "v0.3.4",
		filename: "witr-linux-amd64",
		binaries: []string{"witr"}, license: "MIT",
	},
	{
		repository: "modem-dev/hunk", version: "v0.24.1",
		filename: "hunkdiff-linux-x64.tar.gz",
		binaries: []string{"hunk"}, license: "MIT",
	},
}

type binaryPin struct {
	Repository string   `json:"repository"`
	Version    string   `json:"version"`
	URL        string   `json:"url"`
	SHA256     string   `json:"sha256"`
	Binaries   []string `json:"binaries"`
	License    string   `json:"license"`
	Bundle     string   `json:"bundle,omitempty"`
	StripRoot  bool     `json:"strip_root,omitempty"`
}

type binaryLock struct {
	Architecture string      `json:"architecture"`
	Artifacts    []binaryPin `json:"artifacts"`
}

func main() {
	if err := updateLock(); err != nil {
		fmt.Fprintln(os.Stderr, "Binary locking failed:", err)
		os.Exit(1)
	}
}

func updateLock() error {
	root, err := lockfile.RepositoryRoot()
	if err != nil {
		return err
	}
	pins := make([]binaryPin, 0, len(artifacts))
	for _, spec := range artifacts {
		var release struct {
			Assets []struct {
				Name   string `json:"name"`
				URL    string `json:"browser_download_url"`
				Digest string `json:"digest"`
			} `json:"assets"`
		}
		endpoint := "repos/" + spec.repository + "/releases/tags/" + spec.version
		if err := lockfile.GitHubJSON(endpoint, &release); err != nil {
			return err
		}
		found := false
		for _, asset := range release.Assets {
			if asset.Name != spec.filename {
				continue
			}
			if !strings.HasPrefix(asset.URL, "https://") {
				return fmt.Errorf("release asset %s/%s is missing a valid HTTPS download URL", spec.repository, spec.filename)
			}
			checksum, err := lockfile.DownloadChecksum(asset.URL)
			if err != nil {
				return err
			}
			if asset.Digest != "" && asset.Digest != "sha256:"+checksum {
				return fmt.Errorf("publisher checksum mismatch: %s/%s", spec.repository, spec.filename)
			}
			pins = append(pins, binaryPin{
				Repository: spec.repository, Version: spec.version, URL: asset.URL,
				SHA256: checksum, Binaries: spec.binaries, License: spec.license,
				Bundle: spec.bundle, StripRoot: spec.stripRoot,
			})
			fmt.Printf("Locked %s: %s\n", spec.filename, checksum)
			found = true
			break
		}
		if !found {
			return fmt.Errorf("release %s is missing asset %s", endpoint, spec.filename)
		}
	}
	path := filepath.Join(root, "files/system/usr/share/hephaestus/binaries.lock.json")
	return lockfile.WriteJSON(path, binaryLock{Architecture: "x86_64", Artifacts: pins})
}
