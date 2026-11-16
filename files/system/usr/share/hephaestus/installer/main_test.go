package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func profileFixture(t *testing.T) (source, home string) {
	t.Helper()
	directory := t.TempDir()
	source = filepath.Join(directory, "source")
	home = filepath.Join(directory, "home")
	for _, path := range []string{filepath.Join(source, "config/nvim"), filepath.Join(source, "config/tmux"), filepath.Join(home, ".config")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"zshrc", "config/nvim/init.lua", "config/tmux/tmux.conf"} {
		if err := os.WriteFile(filepath.Join(source, path), []byte("profile\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return source, home
}

func TestPrivateIdentityRestoration(t *testing.T) {
	source, home := profileFixture(t)
	config := filepath.Join(home, `custom config "quoted"`)
	t.Setenv("XDG_CONFIG_HOME", config)
	identity, err := parseIdentity([]byte(`{"name":"Example \"Name\"","email":"example@example.invalid"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := activateProfileWithIdentity(source, false, identity); err != nil {
		t.Fatal(err)
	}
	if err := activateProfileWithIdentity(source, false, identity); err != nil {
		t.Fatal(err)
	}
	for path, mode := range map[string]os.FileMode{
		filepath.Join(config, "hephaestus-private"):              0o700,
		filepath.Join(config, "hephaestus-private/git-identity"): 0o600,
	} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("permissions for %s = %v (%v)", path, info, err)
		}
	}
	command := exec.Command("git", "config", "--file", filepath.Join(home, ".gitconfig"), "--includes", "--get", "user.name")
	output, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != `Example "Name"` {
		t.Fatalf("effective Git identity = %q (%v)", output, err)
	}
	changed, err := parseIdentity([]byte(`{"name":"New Name","email":"new@example.invalid"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := activateProfileWithIdentity(source, false, changed); err == nil {
		t.Fatal("changed private identity must require --replace")
	}
	if err := activateProfileWithIdentity(source, true, changed); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(filepath.Join(config, "hephaestus-private/git-identity.hephaestus-backup.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("private backups = %v (%v)", backups, err)
	}
	saved, err := os.ReadFile(backups[0])
	if err != nil || string(saved) != string(identity) {
		t.Fatal("private backup changed")
	}
	info, err := os.Stat(backups[0])
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatal("private backup permissions are unsafe")
	}
}

func TestNeovimConfigIsWritableAndPreservesLock(t *testing.T) {
	for _, previous := range []string{"fresh", "managed-link", "personal-link", "file"} {
		t.Run(previous, func(t *testing.T) {
			source, home := profileFixture(t)
			destination := filepath.Join(home, ".config/nvim")
			switch previous {
			case "managed-link", "personal-link":
				target := filepath.Join(source, "config/nvim")
				if previous == "personal-link" {
					target = t.TempDir()
				}
				if err := os.Symlink(target, destination); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(destination, []byte("personal"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if previous != "fresh" {
				if err := activateProfile(source, false); err == nil {
					t.Fatal("migration must require --replace before backing up existing config")
				}
			}
			if err := activateProfile(source, true); err != nil {
				t.Fatal(err)
			}
			info, err := os.Lstat(destination)
			if err != nil || !info.IsDir() {
				t.Fatalf("Neovim config must be a writable directory: %v (%v)", info, err)
			}
			lock := filepath.Join(destination, "nvim-pack-lock.json")
			if err := os.WriteFile(lock, []byte("user-lock"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := activateProfile(source, false); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(lock)
			if err != nil || string(content) != "user-lock" {
				t.Fatal("activation changed the user lock")
			}
			target, err := os.Readlink(filepath.Join(destination, "init.lua"))
			want, resolveErr := filepath.EvalSymlinks(filepath.Join(source, "config/nvim/init.lua"))
			if err != nil || resolveErr != nil || target != want {
				t.Fatalf("init.lua link = %q (%v)", target, err)
			}
			if previous != "fresh" {
				backups, err := filepath.Glob(destination + ".hephaestus-backup.*")
				if err != nil || len(backups) != 1 {
					t.Fatalf("migration backup = %v (%v)", backups, err)
				}
			}
		})
	}
}

func TestIdentityValidation(t *testing.T) {
	for _, payload := range []string{
		`{}`, `null`, `[]`, `{"name":"","email":"x"}`,
		`{"name":"x","email":null}`, `{"name":2,"email":"x"}`,
		`{"name":"x\ninjected","email":"x"}`,
		`{"name":"x","email":"x","token":"sensitive"}`,
		`{"name":"x","email":"x"} {}`,
	} {
		if _, err := parseIdentity([]byte(payload)); err == nil {
			t.Errorf("invalid payload accepted: %s", payload)
		}
	}
}

func TestPrivateParentSymlinkRejected(t *testing.T) {
	source, home := profileFixture(t)
	private := filepath.Join(home, ".config/hephaestus-private")
	outside := t.TempDir()
	if err := os.Symlink(outside, private); err != nil {
		t.Fatal(err)
	}
	if err := activateProfileWithIdentity(source, true, []byte("identity")); err == nil {
		t.Fatal("private symlink accepted")
	}
	if _, err := os.Lstat(filepath.Join(home, ".zshrc")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed preflight mutated profile")
	}
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

func TestPrivatePermissionsRejected(t *testing.T) {
	for _, unsafe := range []string{"directory", "file"} {
		t.Run(unsafe, func(t *testing.T) {
			source, home := profileFixture(t)
			private := filepath.Join(home, ".config/hephaestus-private")
			mode := os.FileMode(0o700)
			if unsafe == "directory" {
				mode = 0o755
			}
			if err := os.Mkdir(private, mode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(private, mode); err != nil {
				t.Fatal(err)
			}
			if unsafe == "file" {
				if err := os.WriteFile(filepath.Join(private, "git-identity"), []byte("private"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := activateProfileWithIdentity(source, true, []byte("new")); err == nil {
				t.Fatal("unsafe permissions accepted")
			}
			if _, err := os.Lstat(filepath.Join(home, ".zshrc")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("unsafe permissions changed profile")
			}
		})
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
