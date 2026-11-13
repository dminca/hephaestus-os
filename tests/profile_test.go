package tests

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repository(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func runCommand(t *testing.T, command *exec.Cmd) (string, bool) {
	t.Helper()
	output, err := command.CombinedOutput()
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
	}
	return string(output), err == nil
}

func installProfile(t *testing.T, binary, home string, args ...string) (string, bool) {
	t.Helper()
	root := repository(t)
	profile := filepath.Join(root, "files/system/usr/share/hephaestus/profile")
	arguments := append([]string{"--source", profile}, args...)
	command := exec.Command(binary, arguments...)
	command.Env = append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"))
	return runCommand(t, command)
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be absent, got %v", path, err)
	}
}

func assertBackup(t *testing.T, pattern, child, content string) {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected one backup for %s, got %v (%v)", pattern, matches, err)
	}
	path := matches[0]
	if child != "" {
		path = filepath.Join(path, child)
	}
	if got := readFile(t, path); got != content {
		t.Fatalf("backup %s = %q, want %q", path, got, content)
	}
}

func TestProfile(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "hephaestus-profile")
	source := filepath.Join(repository(t), "files/system/usr/share/hephaestus/installer/main.go")
	build := exec.Command("go", "build", "-trimpath", "-o", binary, source)
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if output, ok := runCommand(t, build); !ok {
		t.Fatal(output)
	}

	t.Run("InstallAndRepeat", func(t *testing.T) {
		home := t.TempDir()
		for attempt := 0; attempt < 2; attempt++ {
			if output, ok := installProfile(t, binary, home); !ok {
				t.Fatal(output)
			}
		}
		for destination, origin := range map[string]string{
			".zshrc": "zshrc", ".config/nvim/init.lua": "config/nvim/init.lua",
			".config/nvim/lua": "config/nvim/lua",
			".tmux.conf":       "config/tmux/tmux.conf", ".config/tmux": "config/tmux",
		} {
			got, err := filepath.EvalSymlinks(filepath.Join(home, destination))
			if err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(repository(t), "files/system/usr/share/hephaestus/profile", origin)
			want, err := filepath.EvalSymlinks(source)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("%s resolves to %q, want %q", destination, got, want)
			}
		}
	})

	t.Run("WalletIdentityAndFailure", func(t *testing.T) {
		home := t.TempDir()
		bin := t.TempDir()
		writeFile(t, filepath.Join(bin, "kwallet-query"), `#!/bin/sh
test "$1" = --folder && test "$2" = Hephaestus &&
test "$3" = --read-password && test "$4" = git-identity &&
test "$5" = -- && test "$6" = kdewallet || exit 9
if [ "$WALLET_FAIL" = yes ]; then
  printf 'DO-NOT-LOG-SECRET\n' >&2
  exit 4
fi
if [ "$WALLET_INVALID" = yes ]; then
  printf '%s\n' '{"name":"DO-NOT-LOG-SECRET","email":null}'
  exit 0
fi
printf '%s\n' '{"name":"Demo Name","email":"demo@example.invalid"}'
`, 0o755)
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("DBUS_SESSION_BUS_ADDRESS", "test-only")
		for attempt := 0; attempt < 2; attempt++ {
			if output, ok := installProfile(t, binary, home, "--git-wallet", "kdewallet"); !ok {
				t.Fatal(output)
			}
		}
		t.Setenv("WALLET_FAIL", "yes")
		output, ok := installProfile(t, binary, home, "--replace", "--git-wallet", "kdewallet")
		if ok || strings.Contains(output, "DO-NOT-LOG-SECRET") ||
			!strings.Contains(output, "Wallet read failed") {
			t.Fatalf("unsafe or missing wallet failure diagnostic: %s", output)
		}
		identity := filepath.Join(home, ".config/hephaestus-private/git-identity")
		if !strings.Contains(readFile(t, identity), "demo@example.invalid") {
			t.Fatal("failed wallet retrieval changed prior identity")
		}
		fresh := t.TempDir()
		if output, ok := installProfile(t, binary, fresh, "--git-wallet", "kdewallet"); ok {
			t.Fatalf("failed wallet activated a fresh profile: %s", output)
		}
		assertMissing(t, filepath.Join(fresh, ".zshrc"))
		assertMissing(t, filepath.Join(fresh, ".gitconfig"))
		t.Setenv("WALLET_FAIL", "")
		t.Setenv("WALLET_INVALID", "yes")
		output, ok = installProfile(t, binary, fresh, "--git-wallet", "kdewallet")
		if ok || strings.Contains(output, "DO-NOT-LOG-SECRET") || !strings.Contains(output, "invalid email") {
			t.Fatalf("unsafe invalid payload handling: %s", output)
		}
		assertMissing(t, filepath.Join(fresh, ".gitconfig"))
		t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
		if output, ok := installProfile(t, binary, fresh, "--git-wallet", "kdewallet"); ok ||
			!strings.Contains(output, "D-Bus session bus") {
			t.Fatalf("missing bus not rejected: %s", output)
		}
	})

	t.Run("LegacyTmuxConflict", func(t *testing.T) {
		home := t.TempDir()
		writeFile(t, filepath.Join(home, ".tmux.conf"), "personal tmux\n", 0o644)
		if output, ok := installProfile(t, binary, home); ok || !strings.Contains(output, ".tmux.conf") {
			t.Fatalf("legacy tmux not detected: %s", output)
		}
		assertMissing(t, filepath.Join(home, ".zshrc"))
		if output, ok := installProfile(t, binary, home, "--replace"); !ok {
			t.Fatal(output)
		}
		assertBackup(t, filepath.Join(home, ".tmux.conf.hephaestus-backup.*"), "", "personal tmux\n")
	})

	t.Run("ConflictsDoNotPartiallyInstall", func(t *testing.T) {
		home := t.TempDir()
		path := filepath.Join(home, ".zshrc")
		const content = "existing configuration\n"
		writeFile(t, path, content, 0o644)
		output, ok := installProfile(t, binary, home)
		if ok || !strings.Contains(output, ".zshrc") {
			t.Fatalf("expected conflict, got success=%v: %s", ok, output)
		}
		if readFile(t, path) != content {
			t.Fatal("existing configuration changed")
		}
		assertMissing(t, filepath.Join(home, ".config/nvim"))
	})

	t.Run("ReplaceBacksUpFilesAndDirectories", func(t *testing.T) {
		home := t.TempDir()
		writeFile(t, filepath.Join(home, ".zshrc"), "existing configuration\n", 0o644)
		writeFile(t, filepath.Join(home, ".gitconfig"), "[alias]\n\tst = status\n", 0o644)
		mkdir(t, filepath.Join(home, ".config/nvim"))
		writeFile(t, filepath.Join(home, ".config/nvim/init.lua"), "-- existing editor\n", 0o644)
		if output, ok := installProfile(t, binary, home, "--replace"); !ok {
			t.Fatal(output)
		}
		assertBackup(t, filepath.Join(home, ".zshrc.hephaestus-backup.*"), "", "existing configuration\n")
		assertBackup(t, filepath.Join(home, ".gitconfig.hephaestus-backup.*"), "", "[alias]\n\tst = status\n")
		assertBackup(t, filepath.Join(home, ".config/nvim/init.lua.hephaestus-backup.*"), "", "-- existing editor\n")
	})

	t.Run("SymlinkedConfigParentIsRejected", func(t *testing.T) {
		root := t.TempDir()
		home, outside := filepath.Join(root, "home"), filepath.Join(root, "outside")
		mkdir(t, home)
		mkdir(t, outside)
		if err := os.Symlink(outside, filepath.Join(home, ".config")); err != nil {
			t.Fatal(err)
		}
		if output, ok := installProfile(t, binary, home, "--replace"); ok {
			t.Fatalf("symlinked config accepted: %s", output)
		}
		entries, err := os.ReadDir(outside)
		if err != nil || len(entries) != 0 {
			t.Fatalf("outside directory changed: %v (%v)", entries, err)
		}
		assertMissing(t, filepath.Join(home, ".zshrc"))
	})

	t.Run("MissingSourceIsRejected", func(t *testing.T) {
		source := t.TempDir()
		output, ok := installProfile(t, binary, t.TempDir(), "--source", source)
		if ok || !strings.Contains(output, "Incomplete profile") {
			t.Fatalf("expected incomplete profile error, got success=%v: %s", ok, output)
		}
	})

	t.Run("BrokenSymlinkIsBackedUp", func(t *testing.T) {
		home := t.TempDir()
		destination := filepath.Join(home, ".zshrc")
		missing := filepath.Join(home, "missing")
		if err := os.Symlink(missing, destination); err != nil {
			t.Fatal(err)
		}
		if output, ok := installProfile(t, binary, home); ok {
			t.Fatalf("broken symlink was not treated as a conflict: %s", output)
		}
		if output, ok := installProfile(t, binary, home, "--replace"); !ok {
			t.Fatal(output)
		}
		backups, err := filepath.Glob(destination + ".hephaestus-backup.*")
		if err != nil || len(backups) != 1 {
			t.Fatalf("expected one symlink backup, got %v (%v)", backups, err)
		}
		target, err := os.Readlink(backups[0])
		if err != nil || target != missing {
			t.Fatalf("backup symlink target = %q (%v), want %q", target, err, missing)
		}
	})

	t.Run("RelativeConfigDirectoryIsRejected", func(t *testing.T) {
		home := t.TempDir()
		command := exec.Command(binary, "--source", filepath.Join(repository(t), "files/system/usr/share/hephaestus/profile"))
		command.Env = append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME=relative")
		output, ok := runCommand(t, command)
		if ok || !strings.Contains(output, "XDG_CONFIG_HOME must be absolute") {
			t.Fatalf("expected absolute path error, got success=%v: %s", ok, output)
		}
		assertMissing(t, filepath.Join(home, ".zshrc"))
	})
}
