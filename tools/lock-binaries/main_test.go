package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dminca/hephaestus-os/internal/lockfile"
)

func checkCatalogLock(lock binaryLock) error {
	if lock.Architecture != "x86_64" || len(lock.Artifacts) != len(artifacts) {
		return fmt.Errorf("architecture or artifact count differs from the catalog")
	}
	for i, spec := range artifacts {
		pin := lock.Artifacts[i]
		expectedURL := "https://github.com/" + spec.repository + "/releases/download/" + spec.version + "/" + spec.filename
		if pin.Repository != spec.repository || pin.Version != spec.version || pin.URL != expectedURL ||
			pin.License != spec.license || !reflect.DeepEqual(pin.Binaries, spec.binaries) {
			return fmt.Errorf("artifact %d (%s) differs from the catalog", i, spec.repository)
		}
		digest, err := hex.DecodeString(pin.SHA256)
		if err != nil || len(digest) != sha256.Size {
			return fmt.Errorf("artifact %s requires a SHA256 checksum", spec.repository)
		}
	}
	return nil
}

func TestCommittedLockMatchesCatalog(t *testing.T) {
	root, err := lockfile.RepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "files/system/usr/share/hephaestus/binaries.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock binaryLock
	if err := json.Unmarshal(content, &lock); err != nil {
		t.Fatal(err)
	}
	if err := checkCatalogLock(lock); err != nil {
		t.Fatalf("%v; regenerate with go run ./tools/lock-binaries", err)
	}
	for name, mutate := range map[string]func(*binaryLock){
		"architecture": func(lock *binaryLock) { lock.Architecture = "other" },
		"count":        func(lock *binaryLock) { lock.Artifacts = lock.Artifacts[:len(lock.Artifacts)-1] },
		"repository":   func(lock *binaryLock) { lock.Artifacts[0].Repository = "other/tool" },
		"version":      func(lock *binaryLock) { lock.Artifacts[0].Version += "-stale" },
		"url":          func(lock *binaryLock) { lock.Artifacts[0].URL += "-stale" },
		"binaries":     func(lock *binaryLock) { lock.Artifacts[0].Binaries = []string{"other"} },
		"license":      func(lock *binaryLock) { lock.Artifacts[0].License = "other" },
		"checksum":     func(lock *binaryLock) { lock.Artifacts[0].SHA256 = "invalid" },
		"order":        func(lock *binaryLock) { lock.Artifacts[0], lock.Artifacts[1] = lock.Artifacts[1], lock.Artifacts[0] },
	} {
		t.Run(name, func(t *testing.T) {
			var changed binaryLock
			if err := json.Unmarshal(content, &changed); err != nil {
				t.Fatal(err)
			}
			mutate(&changed)
			if err := checkCatalogLock(changed); err == nil {
				t.Fatal("accepted inconsistent generated lock")
			}
		})
	}
}

func TestIncompleteReleaseMetadataPreservesLock(t *testing.T) {
	for name, response := range map[string]string{
		"MalformedJSON": "{",
		"MissingAssets": `{}`,
		"MissingURL":    `{"assets":[{"name":"cue_v0.17.1_linux_amd64.tar.gz"}]}`,
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
