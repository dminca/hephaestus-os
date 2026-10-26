package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncompleteReleaseMetadataPreservesLock(t *testing.T) {
	for name, response := range map[string]string{
		"MalformedJSON": "{",
		"MissingAssets": `{}`,
		"MissingURL":    `{"assets":[{"name":"tree-sitter-linux-x64.gz"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			assets := filepath.Join(root, "files/system/usr/share/hephaestus")
			if err := os.MkdirAll(assets, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(assets, "binaries.lock.json")
			for filename, content := range map[string]string{
				filepath.Join(root, "go.mod"): "module fixture\n",
				path:                          "previous reviewed lock\n",
			} {
				if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			binDirectory := t.TempDir()
			stub := "#!/bin/sh\nprintf '%s' \"$GITHUB_RESPONSE\"\n"
			if err := os.WriteFile(filepath.Join(binDirectory, "gh"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(root)
			t.Setenv("PATH", binDirectory)
			t.Setenv("GITHUB_RESPONSE", response)
			err := updateLock()
			if err == nil {
				t.Fatal("incomplete release metadata was accepted")
			}
			if strings.HasPrefix(err.Error(), "download ") {
				t.Fatalf("invalid metadata reached download step: %v", err)
			}
			if name == "MissingURL" && !strings.Contains(err.Error(), "missing a valid HTTPS download URL") {
				t.Fatalf("expected missing download URL error, got %v", err)
			}
			saved, err := os.ReadFile(path)
			if err != nil || string(saved) != "previous reviewed lock\n" {
				t.Fatalf("previous lock was modified: %q (%v)", saved, err)
			}
		})
	}
}
