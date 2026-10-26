package tests

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileBuildUsesLockedTreeSitter(t *testing.T) {
	root := repository(t)
	var lock struct {
		Artifacts []struct {
			Repository string   `json:"repository"`
			Version    string   `json:"version"`
			URL        string   `json:"url"`
			SHA256     string   `json:"sha256"`
			Binaries   []string `json:"binaries"`
		} `json:"artifacts"`
	}
	path := filepath.Join(root, "files/system/usr/share/hephaestus/binaries.lock.json")
	if err := json.Unmarshal([]byte(readFile(t, path)), &lock); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, artifact := range lock.Artifacts {
		if artifact.Repository != "tree-sitter/tree-sitter" {
			continue
		}
		found = true
		if artifact.Version != "v0.26.1" {
			t.Fatalf("unexpected Tree-sitter version: %s", artifact.Version)
		}
		const url = "https://github.com/tree-sitter/tree-sitter/releases/download/v0.26.1/tree-sitter-linux-x64.gz"
		if artifact.URL != url {
			t.Fatalf("unexpected Tree-sitter release asset: %s", artifact.URL)
		}
		const checksum = "d74182fffbf441f247371ad6af77d99b10ac9f788d7e068e4b44992d9d6d7c26"
		if artifact.SHA256 != checksum {
			t.Fatal("Tree-sitter checksum differs from the reviewed upstream asset")
		}
		if len(artifact.Binaries) != 1 || artifact.Binaries[0] != "tree-sitter" {
			t.Fatalf("unexpected executable names: %v", artifact.Binaries)
		}
	}
	if !found {
		t.Fatal("Tree-sitter is missing from the binary lock")
	}
	script := readFile(t, filepath.Join(root, "files/scripts/install-profile.sh"))
	if strings.Contains(script, "cargo install --locked tree-sitter-cli") {
		t.Fatal("build still depends on the unavailable crates.io release")
	}
	installPosition := strings.Index(script, `python3 "$assets/install-binaries.py"`)
	versionPosition := strings.Index(script, "tree-sitter --version")
	parserPosition := strings.Index(script, `nvim --headless -u NONE -l "$assets/build-parsers.lua"`)
	if installPosition < 0 || versionPosition < 0 || parserPosition < 0 {
		t.Fatal("binary installation, version check or parser build is missing")
	}
	if installPosition >= versionPosition || versionPosition >= parserPosition {
		t.Fatal("locked Tree-sitter must be installed and checked before building parsers")
	}
}
