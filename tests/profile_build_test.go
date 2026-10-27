package tests

import (
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
