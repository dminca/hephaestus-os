package tests

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"testing"
)

func TestRenovateLazyLockVersionExtraction(t *testing.T) {
	root := repository(t)
	var config struct {
		CustomManagers []struct {
			Description        string   `json:"description"`
			MatchStrings       []string `json:"matchStrings"`
			DatasourceTemplate string   `json:"datasourceTemplate"`
		} `json:"customManagers"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, "renovate.json5"))), &config); err != nil {
		t.Fatal(err)
	}
	managers := make(map[string][]*regexp.Regexp)
	for _, manager := range config.CustomManagers {
		if manager.Description == "Track branch commits in the lazy.nvim plugin lock" ||
			manager.Description == "Track release tags in the lazy.nvim plugin lock" {
			if manager.Description == "Track branch commits in the lazy.nvim plugin lock" &&
				manager.DatasourceTemplate != "git-refs" {
				t.Fatal("branch pins must use the git-refs datasource")
			}
			if manager.Description == "Track release tags in the lazy.nvim plugin lock" &&
				manager.DatasourceTemplate != "github-tags" {
				t.Fatal("release pins must use the github-tags datasource")
			}
			for _, pattern := range manager.MatchStrings {
				managers[manager.Description] = append(managers[manager.Description], regexp.MustCompile(pattern))
			}
		}
	}
	for _, name := range []string{
		"Track branch commits in the lazy.nvim plugin lock",
		"Track release tags in the lazy.nvim plugin lock",
	} {
		if len(managers[name]) == 0 {
			t.Fatalf("missing Renovate manager %q", name)
		}
	}

	var lock map[string]lazyPluginPinFixture
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, "files/system/usr/share/hephaestus/nvim/lazy-lock.json"))), &lock); err != nil {
		t.Fatal(err)
	}
	for name, pin := range lock {
		t.Run(name, func(t *testing.T) {
			entry, err := json.Marshal(pin)
			if err != nil {
				t.Fatal(err)
			}
			matches := 0
			description := ""
			switch pin.Type {
			case "branch":
				description = "Track branch commits in the lazy.nvim plugin lock"
			case "release":
				description = "Track release tags in the lazy.nvim plugin lock"
			case "held":
			default:
				t.Fatalf("unknown lazy lock type %q", pin.Type)
			}
			for _, pattern := range managers[description] {
				match := pattern.FindStringSubmatch(string(entry))
				if match == nil {
					continue
				}
				matches++
				expectedValue := pin.Branch
				if pin.Type == "branch" && expectedValue == "HEAD" {
					expectedValue = ""
				}
				for name, want := range map[string]string{
					"depName": pin.Repository, "currentValue": expectedValue, "currentDigest": pin.Commit,
				} {
					if index := pattern.SubexpIndex(name); index < 0 || match[index] != want {
						t.Errorf("capture %s: expected %q, got %v", name, want, match)
					}
				}
			}
			if pin.Type != "held" && matches != 1 {
				t.Errorf("expected exactly one Renovate match, got %d", matches)
			}
			if pin.Type == "held" && matches != 0 {
				t.Errorf("held pin must not be automatically updated, got %d matches", matches)
			}
		})
	}
}
