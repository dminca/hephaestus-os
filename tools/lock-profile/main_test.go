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

const fixtureHeldCommit = "6810349b28055dce54076712fc05fc68da4b8ec0"

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
			if err := os.MkdirAll(filepath.Join(assets, "nvim"), 0o755); err != nil {
				t.Fatal(err)
			}
			for path, value := range map[string]string{
				filepath.Join(root, "go.mod"):                "module fixture\n",
				filepath.Join(config, "plugins.lua"):         `{ "example/plugin" }`,
				filepath.Join(assets, "sources.lock.json"):   content,
				filepath.Join(assets, "nvim/lazy-lock.json"): string(mustMarshal(t, fixtureLazyLock(strings.Repeat("a", 40)))),
			} {
				if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
					t.Fatal(err)
				}
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

func TestCommittedLazyLockMatchesIntegrityLock(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(root, "files/system/usr/share/hephaestus")
	var pins lazyLock
	if err := json.Unmarshal([]byte(readFileForTest(t, filepath.Join(assets, "nvim/lazy-lock.json"))), &pins); err != nil {
		t.Fatal(err)
	}
	var lock sourceLock
	if err := json.Unmarshal([]byte(readFileForTest(t, filepath.Join(assets, "sources.lock.json"))), &lock); err != nil {
		t.Fatal(err)
	}
	if len(pins) != 64 || len(lock.Sources) != len(pins) {
		t.Fatalf("lazy lock/integrity lock counts = %d/%d, want 64/64", len(pins), len(pins))
	}
	seen := make(map[string]bool, len(pins))
	for name, plugin := range pins {
		if seen[plugin.Repository] {
			t.Errorf("duplicate lazy lock repository %s", plugin.Repository)
			continue
		}
		seen[plugin.Repository] = true
		pin, exists := lock.Sources[plugin.Repository]
		if !exists {
			t.Errorf("missing locked source %s", plugin.Repository)
			continue
		}
		digest, err := hex.DecodeString(pin.SHA256)
		if err != nil || len(digest) != sha256.Size {
			t.Errorf("invalid checksum for %s", plugin.Repository)
		}
		if plugin.Commit != pin.Revision {
			t.Errorf("lazy lock %s commit = %s, integrity lock has %s", name, plugin.Commit, pin.Revision)
		}
		wantURL := "https://codeload.github.com/" + plugin.Repository + "/tar.gz/" + pin.Revision
		if pin.URL != wantURL {
			t.Errorf("source %s URL = %s, want %s", plugin.Repository, pin.URL, wantURL)
		}
	}
}

func fixtureLazyLock(commit string) lazyLock {
	return lazyLock{
		"lazy.nvim": {
			Branch: "stable", Commit: commit, Repository: "folke/lazy.nvim", Type: "branch",
		},
		"catppuccin-bat": {
			Branch: fixtureHeldCommit, Commit: fixtureHeldCommit, Repository: "catppuccin/bat", Type: "held",
		},
		"plugin": {
			Branch: "HEAD", Commit: commit, Repository: "example/plugin", Type: "branch",
		},
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
