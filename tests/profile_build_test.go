package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoRepositoryPythonScripts(t *testing.T) {
	command := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.py", "*.pyw", "*.pyi")
	command.Dir = repository(t)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list repository Python files: %v\n%s", err, output)
	}
	for _, name := range strings.Split(string(output), "\x00") {
		if name == "" {
			continue
		}
		_, err := os.Lstat(filepath.Join(command.Dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("repository Python file remains: %s", name)
	}
}

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
	if strings.Contains(script, ".py") {
		t.Fatal("profile build still invokes a repository Python script")
	}
	buildPosition := strings.Index(script, `-o "$temporary/hephaestus-assets" .`)
	sourcesPosition := strings.Index(script, `"$temporary/hephaestus-assets" sources`)
	installPosition := strings.Index(script, `"$temporary/hephaestus-assets" binaries`)
	versionPosition := strings.Index(script, "tree-sitter --version")
	parserPosition := strings.Index(script, `nvim --headless -u NONE -l "$assets/build-parsers.lua"`)
	if installPosition < 0 || versionPosition < 0 || parserPosition < 0 {
		t.Fatal("binary installation, version check or parser build is missing")
	}
	if installPosition >= versionPosition || versionPosition >= parserPosition {
		t.Fatal("locked Tree-sitter must be installed and checked before building parsers")
	}
	if buildPosition < 0 || sourcesPosition <= buildPosition || installPosition <= sourcesPosition {
		t.Fatal("Go asset installer must be built before fetching sources and binaries")
	}
}
