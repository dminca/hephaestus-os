package tests

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRenovateBranchDigestExtraction(t *testing.T) {
	root := repository(t)
	var config struct {
		CustomManagers []struct {
			MatchStrings       []string `json:"matchStrings"`
			DatasourceTemplate string   `json:"datasourceTemplate"`
		} `json:"customManagers"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, "renovate.json5"))), &config); err != nil {
		t.Fatal(err)
	}
	var patterns []*regexp.Regexp
	for _, manager := range config.CustomManagers {
		if manager.DatasourceTemplate == "git-refs" {
			for _, pattern := range manager.MatchStrings {
				patterns = append(patterns, regexp.MustCompile(pattern))
			}
		}
	}
	if len(patterns) == 0 {
		t.Fatal("missing git-refs manager")
	}

	var catalog struct {
		Sources []sourceSelectionFixture `json:"sources"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(root, "files/system/usr/share/hephaestus/sources.json"))), &catalog); err != nil {
		t.Fatal(err)
	}
	// Cover explicit refs even if the catalog later stops using them.
	for _, ref := range []string{"HEAD", "main", "harpoon2", "HEAD-feature"} {
		catalog.Sources = append(catalog.Sources, sourceSelectionFixture{
			Repository: "example/plugin", Ref: ref, Revision: strings.Repeat("a", 40), Type: "branch",
		})
	}
	for _, source := range catalog.Sources {
		t.Run(source.Repository+"/"+source.Ref, func(t *testing.T) {
			entry, err := json.Marshal(source)
			if err != nil {
				t.Fatal(err)
			}
			matches := 0
			for _, pattern := range patterns {
				match := pattern.FindStringSubmatch(string(entry))
				if match == nil {
					continue
				}
				matches++
				if source.Type != "branch" {
					t.Fatal("git-refs manager must not match release or held pins")
				}
				expectedValue := source.Ref
				if expectedValue == "HEAD" {
					expectedValue = ""
				}
				for name, want := range map[string]string{
					"depName": source.Repository, "currentValue": expectedValue, "currentDigest": source.Revision,
				} {
					if index := pattern.SubexpIndex(name); index < 0 || match[index] != want {
						t.Errorf("capture %s: expected %q, got %v", name, want, match)
					}
				}
			}
			if source.Type == "branch" && matches != 1 {
				t.Errorf("expected exactly one branch match, got %d", matches)
			}
		})
	}
}
