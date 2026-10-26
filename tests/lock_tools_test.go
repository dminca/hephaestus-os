package tests

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const lockToolStub = `#!/usr/bin/env bash
set -euo pipefail
command=$(basename "$0")
printf '%s %s\n' "$command" "$*" >> "$CALL_LOG"
case "$command" in
  gh)
    if [[ "${FAIL_API:-}" == 1 ]]; then
      printf 'GitHub request failed\n' >&2
      exit 1
    fi
    endpoint=$2
    cat "$API_FIXTURES/$endpoint"
    ;;
  curl)
    if [[ "${FAIL_DOWNLOAD:-}" == 1 ]]; then
      printf 'Download failed\n' >&2
      exit 1
    fi
    printf 'archive fixture'
    ;;
esac
`

const reviewedBatRevision = "6810349b28055dce54076712fc05fc68da4b8ec0"

type sourcePinFixture struct {
	Revision string `json:"revision"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

type lockScenario struct {
	name          string
	expectedError string
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mkdir(t, filepath.Dir(path))
	writeFile(t, path, string(content)+"\n", 0o644)
}

func prepareProfileLock(t *testing.T, assets, api, scenario, checksum string) string {
	t.Helper()
	config := filepath.Join(assets, "profile/config/nvim/lua/hephaestus/plugins.lua")
	mkdir(t, filepath.Dir(config))
	writeFile(t, config, `{ "example/plugin" }`, 0o644)
	revision := strings.Repeat("a", 40)
	sources := map[string]sourcePinFixture{
		"folke/lazy.nvim": {Revision: revision, URL: "https://example.org/lazy", SHA256: checksum},
		"catppuccin/bat":  {Revision: reviewedBatRevision, URL: "https://example.org/bat", SHA256: checksum},
	}
	endpoint := "repos/example/plugin/commits/HEAD"
	switch scenario {
	case "ReusesExistingPins":
		sources["example/plugin"] = sourcePinFixture{
			Revision: revision, URL: "https://example.org/plugin", SHA256: checksum,
		}
	case "UpdatesReviewedCommit":
		sources["catppuccin/bat"] = sourcePinFixture{
			Revision: revision, URL: "https://example.org/bat", SHA256: checksum,
		}
		sources["example/plugin"] = sourcePinFixture{
			Revision: revision, URL: "https://example.org/plugin", SHA256: checksum,
		}
		revision = reviewedBatRevision
		endpoint = "repos/catppuccin/bat/commits/" + revision
	case "InvalidRevision":
		revision = "not-a-commit"
	}
	path := filepath.Join(assets, "sources.lock.json")
	writeJSON(t, path, map[string]any{"sources": sources})
	writeJSON(t, filepath.Join(api, endpoint), map[string]string{"sha": revision})
	return path
}

func prepareBinaryLock(t *testing.T, assets, api, scenario, checksum string) string {
	t.Helper()
	path := filepath.Join(assets, "binaries.lock.json")
	writeFile(t, path, "previous reviewed lock\n", 0o644)
	var existing struct {
		Artifacts []struct {
			Repository string `json:"repository"`
			Version    string `json:"version"`
			URL        string `json:"url"`
		} `json:"artifacts"`
	}
	content := readFile(t, filepath.Join(repository(t), "files/system/usr/share/hephaestus/binaries.lock.json"))
	if err := json.Unmarshal([]byte(content), &existing); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range existing.Artifacts {
		digest := "sha256:" + checksum
		name := filepath.Base(artifact.URL)
		switch scenario {
		case "ChecksumMismatch":
			digest = "sha256:" + strings.Repeat("0", 64)
		case "NoPublisherDigest":
			digest = ""
		case "MissingAsset":
			name = "wrong-asset"
		}
		asset := map[string]string{
			"name": name, "browser_download_url": artifact.URL, "digest": digest,
		}
		release := map[string]any{"assets": []map[string]string{asset}}
		endpoint := "repos/" + artifact.Repository + "/releases/tags/" + artifact.Version
		writeJSON(t, filepath.Join(api, endpoint), release)
	}
	return path
}

func TestLockTools(t *testing.T) {
	for _, tool := range []string{"lock-profile", "lock-binaries"} {
		t.Run(tool, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), tool)
			build := exec.Command("go", "build", "-o", binary, "./tools/"+tool)
			build.Dir = repository(t)
			if output, ok := runCommand(t, build); !ok {
				t.Fatal(output)
			}
			scenarios := []lockScenario{
				{name: "Success"},
				{name: "APIFailure", expectedError: "GitHub request failed"},
				{name: "DownloadFailure", expectedError: "Download failed"},
			}
			if tool == "lock-profile" {
				scenarios = append(scenarios,
					lockScenario{name: "ReusesExistingPins"},
					lockScenario{name: "InvalidRevision", expectedError: "invalid commit"},
					lockScenario{name: "UpdatesReviewedCommit"},
				)
			} else {
				scenarios = append(scenarios,
					lockScenario{name: "ChecksumMismatch", expectedError: "publisher checksum mismatch"},
					lockScenario{name: "MissingAsset", expectedError: "missing asset"},
					lockScenario{name: "NoPublisherDigest"},
				)
			}
			for _, scenario := range scenarios {
				t.Run(scenario.name, func(t *testing.T) {
					checkLockScenario(t, binary, tool, scenario)
				})
			}
		})
	}
}

func checkLockScenario(t *testing.T, binary, tool string, scenario lockScenario) {
	t.Helper()
	root := t.TempDir()
	assets := filepath.Join(root, "files/system/usr/share/hephaestus")
	mkdir(t, assets)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n", 0o644)
	binDirectory := filepath.Join(root, "bin")
	mkdir(t, binDirectory)
	stub := filepath.Join(binDirectory, "stub")
	writeFile(t, stub, lockToolStub, 0o755)
	for _, command := range []string{"gh", "curl"} {
		if err := os.Symlink(stub, filepath.Join(binDirectory, command)); err != nil {
			t.Fatal(err)
		}
	}
	api := filepath.Join(root, "api")
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte("archive fixture")))
	var lockPath string
	if tool == "lock-profile" {
		lockPath = prepareProfileLock(t, assets, api, scenario.name, checksum)
	} else {
		lockPath = prepareBinaryLock(t, assets, api, scenario.name, checksum)
	}
	original := readFile(t, lockPath)
	log := filepath.Join(root, "calls")
	writeFile(t, log, "", 0o644)
	command := exec.Command(binary)
	command.Dir = binDirectory // Also exercise repository discovery from a subdirectory.
	command.Env = append(os.Environ(),
		"PATH="+binDirectory+string(os.PathListSeparator)+os.Getenv("PATH"),
		"CALL_LOG="+log, "API_FIXTURES="+api, "FAIL_API=", "FAIL_DOWNLOAD=",
	)
	switch scenario.name {
	case "APIFailure":
		command.Env = append(command.Env, "FAIL_API=1")
	case "DownloadFailure":
		command.Env = append(command.Env, "FAIL_DOWNLOAD=1")
	}
	output, success := runCommand(t, command)
	if scenario.expectedError != "" {
		if success {
			t.Fatal("lock update unexpectedly succeeded")
		}
		if !strings.Contains(output, scenario.expectedError) {
			t.Fatalf("expected %q error:\n%s", scenario.expectedError, output)
		}
		if readFile(t, lockPath) != original {
			t.Fatal("failed update changed the previous lock")
		}
		return
	}
	if !success {
		t.Fatal(output)
	}
	content := readFile(t, lockPath)
	if !json.Valid([]byte(content)) {
		t.Fatalf("generated lock is not valid JSON:\n%s", content)
	}
	if !strings.Contains(content, checksum) {
		t.Fatal("generated lock is missing the archive checksum")
	}
	if tool == "lock-profile" {
		if !strings.Contains(content, `"nix_config_revision": "5ea74ed40e60c520daae0a94cc50e05857abcb63"`) {
			t.Fatal("source revision changed")
		}
	} else {
		if strings.Count(content, `"repository"`) != 10 {
			t.Fatal("binary artifact count changed")
		}
		if !strings.Contains(content, `"architecture": "x86_64"`) {
			t.Fatal("binary architecture changed")
		}
	}
	if scenario.name == "ReusesExistingPins" && readFile(t, log) != "" {
		t.Fatal("existing reviewed pins caused network calls")
	}
}
