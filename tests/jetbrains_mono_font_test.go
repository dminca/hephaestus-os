package tests

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestJetBrainsMonoNerdFont(t *testing.T) {
	root := repository(t)
	var recipe struct {
		Modules []struct {
			Type    string   `yaml:"type"`
			Scripts []string `yaml:"scripts"`
		} `yaml:"modules"`
	}
	if err := yaml.Unmarshal([]byte(readFile(t, filepath.Join(root, "recipes/recipe.yml"))), &recipe); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, module := range recipe.Modules {
		for _, script := range module.Scripts {
			found = found || module.Type == "script" && script == "install-jetbrains-mono-nerd-font.sh"
		}
	}
	if !found {
		t.Fatal("image recipe must install JetBrainsMono Nerd Font")
	}

	for _, scenario := range []struct {
		name        string
		corrupt     bool
		missingFont bool
		wantSuccess bool
	}{
		{name: "verified archive", wantSuccess: true},
		{name: "checksum mismatch", corrupt: true},
		{name: "missing regular font", missingFont: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			dir := t.TempDir()
			contents := filepath.Join(dir, "contents")
			if err := os.Mkdir(contents, 0o755); err != nil {
				t.Fatal(err)
			}
			font := "JetBrainsMonoNerdFont-Regular.ttf"
			if scenario.missingFont {
				font = "other.ttf"
			}
			writeFile(t, filepath.Join(contents, font), "font fixture", 0o644)
			writeFile(t, filepath.Join(contents, "OFL.txt"), "license fixture", 0o644)
			archive := filepath.Join(dir, "fixture.tar.xz")
			if output, err := exec.Command("tar", "-cJf", archive, "-C", contents, ".").CombinedOutput(); err != nil {
				t.Fatalf("create fixture: %v: %s", err, output)
			}
			data, err := os.ReadFile(archive)
			if err != nil {
				t.Fatal(err)
			}
			script := readFile(t, filepath.Join(root, "files/scripts/install-jetbrains-mono-nerd-font.sh"))
			if !scenario.corrupt {
				script = regexp.MustCompile(`checksum=[a-f0-9]{64}`).ReplaceAllString(script, fmt.Sprintf("checksum=%x", sha256.Sum256(data)))
			}
			script = strings.ReplaceAll(script, "/usr/share/fonts/jetbrains-mono-nerd-font", filepath.Join(dir, "fonts"))
			script = strings.ReplaceAll(script, "/usr/share/licenses/jetbrains-mono-nerd-font", filepath.Join(dir, "licenses"))
			writeFile(t, filepath.Join(dir, "curl"), "#!/bin/bash\nwhile [[ $# -gt 0 ]]; do if [[ \"$1\" == -o ]]; then cp \"$FIXTURE\" \"$2\"; exit; fi; shift; done\nexit 1\n", 0o755)
			writeFile(t, filepath.Join(dir, "fc-cache"), "#!/bin/bash\nprintf '%s\\n' \"$*\" > \"$CACHE_LOG\"\n", 0o755)
			command := exec.Command("bash", "-c", script)
			command.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "FIXTURE="+archive, "CACHE_LOG="+filepath.Join(dir, "cache.log"))
			output, success := runCommand(t, command)
			if success != scenario.wantSuccess {
				t.Fatalf("success=%t, want %t: %s", success, scenario.wantSuccess, output)
			}
			if !success {
				if _, err := os.Stat(filepath.Join(dir, "fonts")); !os.IsNotExist(err) {
					t.Fatal("invalid archive must not install fonts")
				}
				return
			}
			for _, name := range []string{"fonts/JetBrainsMonoNerdFont-Regular.ttf", "licenses/OFL.txt", "cache.log"} {
				if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
					t.Fatalf("missing %s: %v", name, err)
				}
			}
		})
	}
}
