package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dminca/hephaestus-os/internal/lockfile"
)

const nixConfigRevision = "5ea74ed40e60c520daae0a94cc50e05857abcb63"

var reviewedRefs = map[string]string{
	"folke/lazy.nvim":                 "stable",
	"ThePrimeagen/harpoon":            "harpoon2",
	"saghen/blink.cmp":                "v1.7.0",
	"niksingh710/leetcode.nvim":       "cab77da208b0abca7070ef4213bc4cea1a339ada",
	"catppuccin/bat":                  "6810349b28055dce54076712fc05fc68da4b8ec0",
	"nvim-treesitter/nvim-treesitter": "c6dd314086f7b471bf6c9110092a94ce1c06d220",
}

type sourcePin struct {
	Revision string `json:"revision"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

type sourceLock struct {
	NixConfigRevision string               `json:"nix_config_revision"`
	Sources           map[string]sourcePin `json:"sources"`
}

func main() {
	if err := updateLock(); err != nil {
		fmt.Fprintln(os.Stderr, "Profile locking failed:", err)
		os.Exit(1)
	}
}

func updateLock() error {
	root, err := lockfile.RepositoryRoot()
	if err != nil {
		return err
	}
	assets := filepath.Join(root, "files/system/usr/share/hephaestus")
	config, err := os.ReadFile(filepath.Join(assets, "profile/config/nvim/lua/hephaestus/plugins.lua"))
	if err != nil {
		return err
	}
	repositories := map[string]bool{"folke/lazy.nvim": true, "catppuccin/bat": true}
	pattern := regexp.MustCompile(`\{\s*"([\w.-]+/[\w.-]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(string(config), -1) {
		repositories[match[1]] = true
	}
	path := filepath.Join(assets, "sources.lock.json")
	existing := sourceLock{}
	content, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(content, &existing); err != nil {
			return fmt.Errorf("read existing source lock: %w", err)
		}
		if existing.Sources == nil {
			return errors.New("existing source lock is missing sources")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	names := make([]string, 0, len(repositories))
	for repository := range repositories {
		names = append(names, repository)
	}
	sort.Strings(names)
	commitPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	checksumPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for _, repository := range names {
		pin, exists := existing.Sources[repository]
		if !exists {
			continue
		}
		validRevision := commitPattern.MatchString(pin.Revision)
		validURL := strings.HasPrefix(pin.URL, "https://")
		validChecksum := checksumPattern.MatchString(pin.SHA256)
		if !validRevision || !validURL || !validChecksum {
			return fmt.Errorf("existing source lock has incomplete or invalid metadata for %s", repository)
		}
	}
	sources := make(map[string]sourcePin, len(names))
	for _, repository := range names {
		ref := reviewedRefs[repository]
		pin, exists := existing.Sources[repository]
		reviewedCommitChanged := commitPattern.MatchString(ref) && pin.Revision != ref
		// Reuse reviewed pins; remove an entry to deliberately update it.
		if exists && !reviewedCommitChanged {
			sources[repository] = pin
			continue
		}
		if ref == "" {
			ref = "HEAD"
		}
		var commit struct {
			SHA string `json:"sha"`
		}
		if err := lockfile.GitHubJSON("repos/"+repository+"/commits/"+ref, &commit); err != nil {
			return err
		}
		if !commitPattern.MatchString(commit.SHA) {
			return fmt.Errorf("invalid commit for %s: %q", repository, commit.SHA)
		}
		url := "https://codeload.github.com/" + repository + "/tar.gz/" + commit.SHA
		checksum, err := lockfile.DownloadChecksum(url)
		if err != nil {
			return err
		}
		sources[repository] = sourcePin{Revision: commit.SHA, URL: url, SHA256: checksum}
		fmt.Printf("Locked %s at %s\n", repository, commit.SHA)
	}
	return lockfile.WriteJSON(path, sourceLock{NixConfigRevision: nixConfigRevision, Sources: sources})
}
