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

type sourceSelection struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
	Revision   string `json:"revision"`
	Type       string `json:"type"`
}

type sourceManifest struct {
	Sources []sourceSelection `json:"sources"`
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
	manifestPath := filepath.Join(assets, "sources.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest sourceManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return fmt.Errorf("read source manifest: %w", err)
	}
	if len(manifest.Sources) == 0 {
		return errors.New("source manifest is empty")
	}

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

	commitPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	checksumPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	selections := make(map[string]sourceSelection, len(manifest.Sources))
	for _, selection := range manifest.Sources {
		if !repositories[selection.Repository] {
			return fmt.Errorf("source manifest includes unused repository %s", selection.Repository)
		}
		if _, exists := selections[selection.Repository]; exists {
			return fmt.Errorf("source manifest has duplicate repository %s", selection.Repository)
		}
		if selection.Ref == "" {
			return fmt.Errorf("source manifest has empty ref for %s", selection.Repository)
		}
		switch selection.Type {
		case "branch":
			if !commitPattern.MatchString(selection.Revision) {
				return fmt.Errorf("invalid branch digest for %s: %q", selection.Repository, selection.Revision)
			}
		case "release":
			if selection.Revision != "" {
				return fmt.Errorf("release source %s must not pin a separate revision", selection.Repository)
			}
		case "held":
			if !commitPattern.MatchString(selection.Ref) || selection.Ref != selection.Revision {
				return fmt.Errorf("held source %s must use the same full commit for ref and revision", selection.Repository)
			}
		default:
			return fmt.Errorf("unknown source policy %q for %s", selection.Type, selection.Repository)
		}
		selections[selection.Repository] = selection
	}
	if len(selections) != len(repositories) {
		for repository := range repositories {
			if _, exists := selections[repository]; !exists {
				return fmt.Errorf("source manifest is missing repository %s", repository)
			}
		}
	}
	names := make([]string, 0, len(selections))
	for repository := range selections {
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
		selection := selections[repository]
		revision := selection.Revision
		if selection.Type == "release" {
			var commit struct {
				SHA string `json:"sha"`
			}
			if err := lockfile.GitHubJSON("repos/"+repository+"/commits/"+selection.Ref, &commit); err != nil {
				return err
			}
			if !commitPattern.MatchString(commit.SHA) {
				return fmt.Errorf("invalid commit for %s: %q", repository, commit.SHA)
			}
			revision = commit.SHA
		}
		if selection.Type == "held" {
			revision = selection.Ref
		}
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
