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

type sourcePin struct {
	Revision string `json:"revision"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

type pluginPin struct {
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	Repository string `json:"repository"`
	Type       string `json:"type"`
}

type lazyLock map[string]pluginPin

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
	expected := map[string]string{
		"folke/lazy.nvim": "lazy.nvim",
		"catppuccin/bat":  "catppuccin-bat",
	}
	pattern := regexp.MustCompile(`\{\s*"([\w.-]+/[\w.-]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(string(config), -1) {
		expected[match[1]] = lazyPluginName(match[1])
	}

	lazyLockPath := filepath.Join(assets, "nvim/lazy-lock.json")
	content, err := os.ReadFile(lazyLockPath)
	if err != nil {
		return err
	}
	var pins lazyLock
	if err := json.Unmarshal(content, &pins); err != nil {
		return fmt.Errorf("read lazy.nvim lockfile: %w", err)
	}
	if len(pins) == 0 {
		return errors.New("lazy.nvim lockfile is empty")
	}

	commitPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	repositoryPattern := regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)
	repositories := make(map[string]pluginPin, len(pins))
	for name, pin := range pins {
		if !repositoryPattern.MatchString(pin.Repository) {
			return fmt.Errorf("invalid repository for lazy.nvim plugin %s: %q", name, pin.Repository)
		}
		if !commitPattern.MatchString(pin.Commit) {
			return fmt.Errorf("invalid commit for lazy.nvim plugin %s: %q", name, pin.Commit)
		}
		if pin.Branch == "" {
			return fmt.Errorf("lazy.nvim plugin %s has an empty branch or tag", name)
		}
		switch pin.Type {
		case "branch", "release":
		case "held":
			if pin.Branch != pin.Commit {
				return fmt.Errorf("held lazy.nvim plugin %s must use its commit as branch", name)
			}
		default:
			return fmt.Errorf("unknown pin policy %q for lazy.nvim plugin %s", pin.Type, name)
		}
		expectedName, exists := expected[pin.Repository]
		if !exists {
			return fmt.Errorf("lazy.nvim lockfile includes unused repository %s", pin.Repository)
		}
		if expectedName != name {
			return fmt.Errorf("lazy.nvim lock name for %s is %q, want %q", pin.Repository, name, expectedName)
		}
		if _, exists := repositories[pin.Repository]; exists {
			return fmt.Errorf("lazy.nvim lockfile has duplicate repository %s", pin.Repository)
		}
		repositories[pin.Repository] = pin
		delete(expected, pin.Repository)
	}
	if len(expected) != 0 {
		for repository, name := range expected {
			return fmt.Errorf("lazy.nvim lockfile is missing %s (%s)", repository, name)
		}
	}

	path := filepath.Join(assets, "sources.lock.json")
	existing := sourceLock{NixConfigRevision: nixConfigRevision}
	content, err = os.ReadFile(path)
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

	checksumPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	names := make([]string, 0, len(repositories))
	for repository := range repositories {
		names = append(names, repository)
	}
	sort.Strings(names)
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
		revision := repositories[repository].Commit
		pin, exists := existing.Sources[repository]
		if exists && pin.Revision == revision {
			sources[repository] = pin
			continue
		}
		url := "https://codeload.github.com/" + repository + "/tar.gz/" + revision
		checksum, err := lockfile.DownloadChecksum(url)
		if err != nil {
			return err
		}
		sources[repository] = sourcePin{Revision: revision, URL: url, SHA256: checksum}
		fmt.Printf("Locked %s at %s\n", repository, revision)
	}
	return lockfile.WriteJSON(path, sourceLock{NixConfigRevision: existing.NixConfigRevision, Sources: sources})
}

func lazyPluginName(repository string) string {
	if repository == "catppuccin/nvim" {
		return "catppuccin"
	}
	return filepath.Base(repository)
}
