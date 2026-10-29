package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidExistingLockIsNotReplaced(t *testing.T) {
	cases := []string{"MalformedJSON", "MissingSources", "MissingRevision", "MissingURL", "MissingChecksum"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			pin := sourcePin{
				Revision: strings.Repeat("a", 40),
				URL:      "https://example.org/archive",
				SHA256:   strings.Repeat("a", 64),
			}
			switch name {
			case "MissingRevision":
				pin.Revision = ""
			case "MissingURL":
				pin.URL = ""
			case "MissingChecksum":
				pin.SHA256 = ""
			}
			lock := sourceLock{Sources: map[string]sourcePin{"folke/lazy.nvim": pin}}
			if name == "MissingSources" {
				lock.Sources = nil
			}
			encoded, err := json.Marshal(lock)
			if err != nil {
				t.Fatal(err)
			}
			content := string(encoded)
			if name == "MalformedJSON" {
				content = "{"
			}
			root := t.TempDir()
			assets := filepath.Join(root, "files/system/usr/share/hephaestus")
			config := filepath.Join(assets, "profile/config/nvim/lua/hephaestus")
			if err := os.MkdirAll(config, 0o755); err != nil {
				t.Fatal(err)
			}
			for path, value := range map[string]string{
				filepath.Join(root, "go.mod"):              "module fixture\n",
				filepath.Join(config, "plugins.lua"):       "",
				filepath.Join(assets, "sources.lock.json"): content,
			} {
				if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			manifest, err := json.Marshal(sourceManifest{Sources: []sourceSelection{
				{Repository: "folke/lazy.nvim", Ref: "stable", Revision: strings.Repeat("a", 40), Type: "branch"},
				{Repository: "catppuccin/bat", Ref: "6810349b28055dce54076712fc05fc68da4b8ec0", Revision: "6810349b28055dce54076712fc05fc68da4b8ec0", Type: "held"},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(assets, "sources.json"), manifest, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			t.Setenv("PATH", t.TempDir())
			err = updateLock()
			if err == nil || !strings.Contains(err.Error(), "source lock") {
				t.Fatalf("expected source lock validation error, got %v", err)
			}
			saved, err := os.ReadFile(filepath.Join(assets, "sources.lock.json"))
			if err != nil || string(saved) != content {
				t.Fatalf("invalid lock was modified: %q (%v)", saved, err)
			}
		})
	}
}

func TestCommittedSourceManifestMatchesLock(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(root, "files/system/usr/share/hephaestus")
	var manifest sourceManifest
	if err := json.Unmarshal([]byte(readFileForTest(t, filepath.Join(assets, "sources.json"))), &manifest); err != nil {
		t.Fatal(err)
	}
	var lock sourceLock
	if err := json.Unmarshal([]byte(readFileForTest(t, filepath.Join(assets, "sources.lock.json"))), &lock); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Sources) != 64 || len(lock.Sources) != len(manifest.Sources) {
		t.Fatalf("source manifest/lock counts = %d/%d, want 64/64", len(manifest.Sources), len(manifest.Sources))
	}
	seen := make(map[string]bool, len(manifest.Sources))
	for _, selection := range manifest.Sources {
		if seen[selection.Repository] {
			t.Errorf("duplicate source manifest entry %s", selection.Repository)
			continue
		}
		seen[selection.Repository] = true
		pin, exists := lock.Sources[selection.Repository]
		if !exists {
			t.Errorf("missing locked source %s", selection.Repository)
			continue
		}
		digest, err := hex.DecodeString(pin.SHA256)
		if err != nil || len(digest) != sha256.Size {
			t.Errorf("invalid checksum for %s", selection.Repository)
		}
		switch selection.Type {
		case "branch", "held":
			if selection.Revision != pin.Revision {
				t.Errorf("source %s revision = %s, lock has %s", selection.Repository, selection.Revision, pin.Revision)
			}
		case "release":
			if len(pin.Revision) != 40 {
				t.Errorf("release source %s has invalid resolved commit", selection.Repository)
			}
		default:
			t.Errorf("source %s has unknown policy %q", selection.Repository, selection.Type)
		}
		wantURL := "https://codeload.github.com/" + selection.Repository + "/tar.gz/" + pin.Revision
		if pin.URL != wantURL {
			t.Errorf("source %s URL = %s, want %s", selection.Repository, pin.URL, wantURL)
		}
	}
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
