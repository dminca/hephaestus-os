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
}

var artifacts = []artifactSpec{
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
