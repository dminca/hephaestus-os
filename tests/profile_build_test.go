package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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
	if !strings.Contains(script, `export PATH="/usr/libexec/hephaestus/bin:$PATH"`) {
		t.Fatal("image build must prefer pinned executables over distribution packages")
	}
	for name, content := range map[string]string{
		"image build": script,
		"parser CI":   readFile(t, filepath.Join(root, ".github/workflows/build.yml")),
	} {
		t.Run(name, func(t *testing.T) {
			lookup := strings.Index(content, "expected_tree_sitter_version=$(")
			check := strings.Index(content, `test "$(tree-sitter --version | cut -d ' ' -f 2)" = "$expected_tree_sitter_version"`)
			parser := strings.Index(content, "nvim --headless -u NONE -l")
			if lookup < 0 || check <= lookup || parser <= check {
				t.Fatal("lock-derived Tree-sitter version must be checked before parser compilation")
			}
			line := strings.SplitN(content[lookup:], "\n", 2)[0]
			if !strings.Contains(line, "binary-version") || !strings.Contains(line, "tree-sitter") {
				t.Fatal("expected Tree-sitter version must come from the asset helper")
			}
			if name == "parser CI" && !strings.Contains(line, `--lock "$HEPHAESTUS_ASSETS/binaries.lock.json"`) {
				t.Fatal("parser CI must read the repository binary lock")
			}
		})
	}
}

func TestProfileBuildInstallsHunk(t *testing.T) {
	root := repository(t)
	lock := committedBinaryLock(t)
	found := false
	for _, artifact := range lock.Artifacts {
		if artifact.Repository != "modem-dev/hunk" {
			continue
		}
		found = true
		if len(artifact.Binaries) != 1 || artifact.Binaries[0] != "hunk" {
			t.Fatal("Hunk release must provide the hunk executable")
		}
	}
	if !found {
		t.Fatal("Hunk is missing from the binary lock")
	}
	script := readFile(t, filepath.Join(root, "files/scripts/install-profile.sh"))
	install := strings.Index(script, `"$temporary/hephaestus-assets" binaries`)
	check := strings.Index(script, "hunk --version")
	if install < 0 || check <= install {
		t.Fatal("Hunk must be installed before its version is checked")
	}
}

func TestProfileBuildInstallsFdWithHomebrew(t *testing.T) {
	root := repository(t)
	module := readRecipeModule(t, "dnf")
	if slices.Contains(module.Install.Packages, "fd-find") {
		t.Fatal("fd must not be installed from the Fedora fd-find package")
	}
	brew := readRecipeModule(t, "brew")
	if !strings.HasPrefix(brew.Source, "ghcr.io/blue-build/modules/brew:latest@sha256:") {
		t.Fatal("recipe must use a pinned BlueBuild Brew module")
	}
	recipe := readFile(t, filepath.Join(root, "recipes/recipe.yml"))
	if !strings.Contains(recipe, "- install-fd.sh") {
		t.Fatal("recipe must run the Homebrew fd installer")
	}
	script := readFile(t, filepath.Join(root, "files/scripts/install-fd.sh"))
	for _, expected := range []string{
		"/usr/share/homebrew/home/linuxbrew/.linuxbrew/bin/brew",
		"setpriv --reuid=1000 --regid=1000",
		"install fd",
		"/home/linuxbrew/.linuxbrew/opt/fd/bin/fd",
		"/usr/libexec/hephaestus/bin/fd",
		`"$destination" --version`,
	} {
		if !strings.Contains(script, expected) {
			t.Errorf("Homebrew fd installer is missing %q", expected)
		}
	}
}

func TestHomebrewFdBuildEnvironment(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		fail         bool
		systemConfig bool
		transient    bool
	}{
		{"success", false, false, false},
		{"install failure", true, false, false},
		{"success preserves system policy", false, true, false},
		{"failure preserves system policy", true, true, false},
		{"transient metadata timeout", false, true, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			prefix := filepath.Join(root, "staged")
			link := filepath.Join(root, "home", ".linuxbrew")
			systemConfig := filepath.Join(root, "system-brew.env")
			destination := filepath.Join(root, "bin", "fd")
			record := filepath.Join(root, "home-path")
			commands := filepath.Join(root, "commands")
			for _, path := range []string{filepath.Join(prefix, "bin"), filepath.Join(prefix, "opt/fd/bin"), commands} {
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			write := func(path, content string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			const originalConfig = "HOMEBREW_SYSTEM_ENV_TAKES_PRIORITY=1\nHOMEBREW_CACHE=/home/linuxbrew/.cache/Homebrew\nHOMEBREW_TEMP=/var/tmp/homebrew\nHOMEBREW_NO_INSECURE_REDIRECT=1\nHOMEBREW_REQUIRE_TAP_TRUST=1\nHOMEBREW_NO_ANALYTICS=1\n"
			if scenario.systemConfig {
				write(systemConfig, originalConfig)
				if err := os.Chmod(systemConfig, 0o640); err != nil {
					t.Fatal(err)
				}
			}
			write(filepath.Join(commands, "chown"), "#!/bin/sh\nexit 0\n")
			write(filepath.Join(commands, "sleep"), "#!/bin/sh\nexit 0\n")
			write(filepath.Join(commands, "setpriv"), "#!/bin/sh\nshift 3\nexec \"$@\"\n")
			write(filepath.Join(commands, "install"), "#!/bin/sh\nmkdir -p \"$(dirname \"$3\")\"\ncp \"$2\" \"$3\"\n")
			write(filepath.Join(prefix, "bin/brew"), `#!/usr/bin/env bash
set -euo pipefail
printf '%s' "$HOME" > "$TEST_HOME_RECORD"
[[ "$USER" == linuxbrew && "$LOGNAME" == linuxbrew ]]
[[ "$XDG_CONFIG_HOME" == "$HOME/config" ]]
# Match Homebrew's system, user, then system-priority configuration loading.
[[ ! -f "$TEST_SYSTEM_CONFIG" ]] || source "$TEST_SYSTEM_CONFIG"
source "$XDG_CONFIG_HOME/homebrew/brew.env"
if [[ -n "${HOMEBREW_SYSTEM_ENV_TAKES_PRIORITY:-}" ]]; then
  source "$TEST_SYSTEM_CONFIG"
  [[ "$HOMEBREW_NO_INSECURE_REDIRECT" == 1 ]]
  [[ "$HOMEBREW_REQUIRE_TAP_TRUST" == 1 ]]
  [[ "$HOMEBREW_NO_ANALYTICS" == 1 ]]
fi
[[ "$HOMEBREW_CACHE" == "$HOME/cache" ]]
[[ "$HOMEBREW_LOGS" == "$HOME/logs" ]]
[[ "$HOMEBREW_TEMP" == "$HOME/tmp" ]]
for path in "$HOMEBREW_CACHE" "$HOMEBREW_LOGS" "$HOMEBREW_TEMP"; do
  [[ -d "$path" && -w "$path" ]]
done
mkdir -p "$HOMEBREW_CACHE/bootsnap"
[[ "$1 $2" == "install fd" ]]
printf 'attempt\n' >> "$TEST_ATTEMPTS"
if [[ "$TEST_TRANSIENT" == true && $(wc -l < "$TEST_ATTEMPTS") -eq 1 ]]; then
  echo 'metadata curl exited with 28' >&2
  exit 1
fi
[[ "$TEST_FAIL_INSTALL" != true ]] || exit 42
`)
			write(filepath.Join(prefix, "opt/fd/bin/fd"), "#!/bin/sh\n[ \"$1\" = --version ]\necho 'fd fixture'\n")
			script := readFile(t, filepath.Join(repository(t), "files/scripts/install-fd.sh"))
			script = strings.NewReplacer(
				"/usr/share/homebrew/home/linuxbrew/.linuxbrew", prefix,
				"/usr/libexec/hephaestus/bin/fd", destination,
				"/home/linuxbrew", filepath.Join(root, "home"),
				"/etc/homebrew/brew.env", systemConfig,
			).Replace(script)
			path := filepath.Join(root, "install-fd.sh")
			write(path, script)
			cmd := exec.Command("bash", path)
			cmd.Env = append(os.Environ(),
				"PATH="+commands+string(os.PathListSeparator)+os.Getenv("PATH"),
				"XDG_CACHE_HOME=/home/linuxbrew/.cache",
				"XDG_CONFIG_HOME=/root/.config",
				"TEST_SYSTEM_CONFIG="+systemConfig,
				"TEST_HOME_RECORD="+record,
				"TEST_FAIL_INSTALL="+strconv.FormatBool(scenario.fail),
				"TEST_TRANSIENT="+strconv.FormatBool(scenario.transient),
				"TEST_ATTEMPTS="+filepath.Join(root, "attempts"),
			)
			output, err := cmd.CombinedOutput()
			if scenario.fail {
				if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 {
					t.Fatalf("expected brew exit 42, got %v\n%s", err, output)
				}
			} else if err != nil || !strings.Contains(string(output), "fd fixture") {
				t.Fatalf("fd build smoke check failed: %v\n%s", err, output)
			}
			wantAttempts := 1
			if scenario.transient {
				wantAttempts = 2
			}
			if scenario.fail {
				wantAttempts = 3
				if !strings.Contains(string(output), "failed after 3 attempts (exit 42)") {
					t.Fatalf("missing terminal failure diagnostic: %s", output)
				}
				if _, err := os.Stat(destination); !os.IsNotExist(err) {
					t.Fatalf("failed install published fd: %v", err)
				}
			}
			if got := strings.Count(readFile(t, filepath.Join(root, "attempts")), "attempt\n"); got != wantAttempts {
				t.Fatalf("brew attempts = %d, want %d", got, wantAttempts)
			}
			home := readFile(t, record)
			for _, path := range []string{link, home} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Errorf("build left temporary path %s: %v", path, err)
				}
				if scenario.systemConfig {
					if got := readFile(t, systemConfig); got != originalConfig {
						t.Fatalf("original system policy changed: %q", got)
					}
					info, err := os.Stat(systemConfig)
					if err != nil || info.Mode().Perm() != 0o640 {
						t.Fatalf("original policy permissions changed: %v, %v", info, err)
					}
				} else if _, err := os.Stat(systemConfig); !os.IsNotExist(err) {
					t.Fatalf("installer created a system policy: %v", err)
				}
				if _, err := os.Lstat(filepath.Join(root, "home")); !os.IsNotExist(err) {
					t.Errorf("build left temporary home: %v", err)
				}
			}
		})
	}
}

func TestProfilesUseIsolatedExecutables(t *testing.T) {
	root := repository(t)
	profile := filepath.Join(root, "files/system/usr/share/hephaestus/profile")
	zsh := readFile(t, filepath.Join(profile, "zshrc"))
	if !strings.Contains(zsh, `"$GOPATH/bin" /usr/libexec/hephaestus/bin $path`) {
		t.Fatal("Zsh must find pinned tools before distribution executables")
	}
	init := readFile(t, filepath.Join(profile, "config/nvim/init.lua"))
	if !strings.Contains(init, `vim.env.PATH = "/usr/libexec/hephaestus/bin:"`) {
		t.Fatal("Neovim launched outside Zsh must also find pinned tools")
	}
	if !strings.Contains(init, `vim.treesitter.language.register("json", "jsonc")`) {
		t.Fatal("JSONC filetype must use the JSON parser")
	}
	plugins := readFile(t, filepath.Join(profile, "config/nvim/lua/hephaestus/plugins.lua"))
	for _, name := range []string{"tinymist", "websocat"} {
		if !strings.Contains(plugins, `"/usr/libexec/hephaestus/bin/`+name+`"`) {
			t.Fatalf("Typst preview must use isolated %s", name)
		}
	}

}

func TestCustomToolsDoNotOverwriteDistributionPackages(t *testing.T) {
	root := repository(t)
	script := readFile(t, filepath.Join(root, "files/scripts/install-profile.sh"))
	for _, obsolete := range []string{
		`"/usr/bin/$binary"`, "/usr/bin/mmdc", "GOBIN=/usr/bin", "/usr/bin/yamlfix",
	} {
		if strings.Contains(script, obsolete) {
			t.Errorf("custom tool installation still writes distribution path %s", obsolete)
		}
	}
}
