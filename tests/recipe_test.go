package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGitCredentialHelperPackaging(t *testing.T) {
	module := readRecipeModule(t, "dnf")
	if !slices.Contains(module.Install.Packages, "git-credential-libsecret") {
		t.Fatal("image must install the Git Secret Service credential helper")
	}
	script := readFile(t, filepath.Join(repository(t), "files/scripts/install-profile.sh"))
	start := strings.Index(script, "rpm -q git-credential-libsecret\n")
	end := strings.Index(script, "CGO_ENABLED=0")
	if start < 0 || end <= start {
		t.Fatal("image build must check credential helper packaging before building the profile")
	}
	check := script[start:end]
	for _, scenario := range []struct {
		name       string
		installed  bool
		helper     bool
		executable bool
		want       bool
	}{
		{"installed", true, true, true, true},
		{"missing RPM", false, true, true, false},
		{"missing executable", true, false, false, false},
		{"non-executable helper", true, true, false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			directory := t.TempDir()
			status := "0"
			if !scenario.installed {
				status = "1"
			}
			writeFile(t, filepath.Join(directory, "rpm"), "#!/bin/sh\nexit "+status+"\n", 0o755)
			writeFile(t, filepath.Join(directory, "git"), "#!/bin/sh\nprintf '%s\\n' \"$HELPER_DIRECTORY\"\n", 0o755)
			if scenario.helper {
				mode := os.FileMode(0o644)
				if scenario.executable {
					mode = 0o755
				}
				writeFile(t, filepath.Join(directory, "git-credential-libsecret"),
					"#!/bin/sh\nexit 99\n", mode)
			}
			command := exec.Command("bash", "-euc", check)
			command.Env = append(os.Environ(), "PATH="+directory, "HELPER_DIRECTORY="+directory)
			output, ok := runCommand(t, command)
			if ok != scenario.want {
				t.Fatalf("packaging check success=%v, want %v: %s", ok, scenario.want, output)
			}
			if scenario.installed && !scenario.want &&
				!strings.Contains(output, "Git libsecret credential helper is missing or not executable") {
				t.Fatalf("missing helper diagnostic: %s", output)
			}
		})
	}
}

type recipeModule struct {
	Type    string `yaml:"type"`
	Source  string `yaml:"source"`
	Install struct {
		Packages []string `yaml:"packages"`
	} `yaml:"install"`
	Configurations []struct {
		Scope string `yaml:"scope"`
		Repo  struct {
			Name string `yaml:"name"`
			URL  string `yaml:"url"`
		} `yaml:"repo"`
		Install []string `yaml:"install"`
	} `yaml:"configurations"`
}

func readRecipeModule(t *testing.T, moduleType string) recipeModule {
	t.Helper()
	content := readFile(t, filepath.Join(repository(t), "recipes/recipe.yml"))
	module, err := parseRecipeModule(content, moduleType)
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func parseRecipeModule(content, moduleType string) (recipeModule, error) {
	var recipe struct {
		Modules []recipeModule `yaml:"modules"`
	}
	if err := yaml.Unmarshal([]byte(content), &recipe); err != nil {
		return recipeModule{}, fmt.Errorf("parse recipe: %w", err)
	}
	for _, module := range recipe.Modules {
		if module.Type == moduleType {
			return module, nil
		}
	}
	return recipeModule{}, fmt.Errorf("recipe is missing module %q", moduleType)
}

func (module recipeModule) installsSystemFlatpak(application string) bool {
	for _, configuration := range module.Configurations {
		flathub := (configuration.Repo.Name == "" || configuration.Repo.Name == "flathub") &&
			(configuration.Repo.URL == "" || configuration.Repo.URL == "https://dl.flathub.org/repo/flathub.flatpakrepo")
		if configuration.Scope == "system" && flathub &&
			slices.Contains(configuration.Install, application) {
			return true
		}
	}
	return false
}

func TestSystemFlatpaks(t *testing.T) {
	module := readRecipeModule(t, "default-flatpaks@v2")
	if !strings.HasPrefix(module.Source, "ghcr.io/blue-build/modules/default-flatpaks:v2@sha256:") {
		t.Fatal("Flatpak module source must match the v2 configuration")
	}
	var systemApps []string
	for _, configuration := range module.Configurations {
		if configuration.Scope == "system" {
			systemApps = append(systemApps, configuration.Install...)
		}
	}
	if !slices.Equal(systemApps, []string{"com.nextcloud.desktopclient.nextcloud"}) {
		t.Fatalf("unexpected system Flatpaks: %v", systemApps)
	}
	for _, application := range []string{
		"com.nextcloud.desktopclient.nextcloud",
	} {
		t.Run(application, func(t *testing.T) {
			if !module.installsSystemFlatpak(application) {
				t.Fatalf("%s must be installed system-wide from Flathub", application)
			}
		})
	}
}

func TestRecipeAssertionsIgnoreFormatting(t *testing.T) {
	const block = `
modules:
  - type: default-flatpaks@v2
    configurations:
      - scope: system
        repo:
          name: flathub
        install:
          - org.example.DesktopApp
`
	const flow = `modules: [{type: default-flatpaks@v2, configurations: [{scope: system, repo: {name: flathub}, install: [org.example.DesktopApp]}]}]`
	const indented = `
modules:
    - type: default-flatpaks@v2
      configurations:
          - scope: system
            repo:
                name: flathub
            install:
                - org.example.DesktopApp
`
	for _, scenario := range []struct {
		name    string
		content string
		want    bool
	}{
		{"block", block, true},
		{"different indentation", indented, true},
		{"flow", flow, true},
		{"comment", strings.ReplaceAll(block, "org.example.DesktopApp", "org.example.DesktopApp # desktop app"), true},
		{"default remote", strings.ReplaceAll(flow, "repo: {name: flathub}, ", ""), true},
		{"user scope", strings.ReplaceAll(flow, "scope: system", "scope: user"), false},
		{"other remote", strings.ReplaceAll(flow, "name: flathub", "name: other"), false},
		{"other remote URL", strings.ReplaceAll(flow, "name: flathub", "name: flathub, url: https://example.org/repo.flatpakrepo"), false},
		{"missing package", strings.ReplaceAll(flow, "[org.example.DesktopApp]", "[]"), false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			module, err := parseRecipeModule(scenario.content, "default-flatpaks@v2")
			if err != nil {
				t.Fatal(err)
			}
			if got := module.installsSystemFlatpak("org.example.DesktopApp"); got != scenario.want {
				t.Fatalf("system Flatpak assertion = %t, want %t", got, scenario.want)
			}
		})
	}
	for _, content := range []string{"modules: [", "modules: []"} {
		if _, err := parseRecipeModule(content, "default-flatpaks@v2"); err == nil {
			t.Errorf("invalid or missing module accepted: %s", content)
		}
	}
}
