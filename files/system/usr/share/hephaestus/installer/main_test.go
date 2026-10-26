package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func profileFixture(t *testing.T) (source, home string) {
	t.Helper()
	directory := t.TempDir()
	source = filepath.Join(directory, "source")
	home = filepath.Join(directory, "home")
	for _, path := range []string{filepath.Join(source, "config/nvim"), filepath.Join(home, ".config")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"zshrc", "config/nvim/init.lua"} {
		if err := os.WriteFile(filepath.Join(source, path), []byte("profile\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return source, home
}

func TestConflictingSymlinkIsPreserved(t *testing.T) {
	source, home := profileFixture(t)
	target := filepath.Join(home, "personal-zshrc")
	if err := os.WriteFile(target, []byte("personal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(home, ".zshrc")
	if err := os.Symlink(target, destination); err != nil {
		t.Fatal(err)
	}
	if err := activateProfile(source, false); err == nil {
		t.Fatal("expected conflicting symlink to be rejected")
	}
	if err := activateProfile(source, true); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(destination + ".hephaestus-backup.*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one backup, got %v (%v)", backups, err)
	}
	savedTarget, err := os.Readlink(backups[0])
	if err != nil || savedTarget != target {
		t.Fatalf("backup target = %q (%v), want %q", savedTarget, err, target)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "personal\n" {
		t.Fatalf("personal file changed: %q (%v)", content, err)
	}
}

func TestActivationFailureKeepsCompletedBackup(t *testing.T) {
	source, home := profileFixture(t)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(home, ".config")
	if err := os.Chmod(config, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(config, 0o755); err != nil {
			t.Error(err)
		}
	})
	// Privileged users can bypass directory permissions; do not report a false pass.
	probe := filepath.Join(config, "permission-probe")
	if err := os.WriteFile(probe, nil, 0o644); err == nil {
		t.Skip("current user bypasses directory write permissions")
	} else if !errors.Is(err, os.ErrPermission) {
		t.Fatal(err)
	}
	if err := activateProfile(source, true); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected explicit permission error, got %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(home, ".zshrc.hephaestus-backup.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("completed backup missing: %v (%v)", backups, err)
	}
	content, err := os.ReadFile(backups[0])
	if err != nil || string(content) != "original\n" {
		t.Fatalf("backup damaged: %q (%v)", content, err)
	}
	if _, err := os.Lstat(filepath.Join(config, "nvim")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed destination unexpectedly changed: %v", err)
	}
}
