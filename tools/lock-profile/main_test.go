package main

import (
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
