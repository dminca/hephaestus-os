package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type profileLink struct {
	source      string
	destination string
	active      bool
	conflict    bool
}

func main() {
	source := flag.String("source", "/usr/share/hephaestus/profile", "Profile asset directory")
	replace := flag.Bool("replace", false, "Back up conflicting files first")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Unexpected arguments:", strings.Join(flag.Args(), " "))
		os.Exit(2)
	}
	if err := activateProfile(*source, *replace); err != nil {
		fmt.Fprintln(os.Stderr, "Profile activation failed:", err)
		os.Exit(1)
	}
}

func activateProfile(source string, replace bool) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("Incomplete profile: %w", err)
	}
	for _, required := range []string{"zshrc", "config/nvim/init.lua"} {
		path := filepath.Join(source, required)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("Incomplete profile: %w", err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Incomplete profile: %s is not a regular file", path)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	config, configIsSet := os.LookupEnv("XDG_CONFIG_HOME")
	if !configIsSet {
		config = filepath.Join(home, ".config")
	}
	if !filepath.IsAbs(config) {
		return errors.New("XDG_CONFIG_HOME must be absolute")
	}
	info, err := os.Lstat(config)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Refusing symlinked configuration directory: %s", config)
	}

	links := []profileLink{{
		source: filepath.Join(source, "zshrc"), destination: filepath.Join(home, ".zshrc"),
	}}
	entries, err := os.ReadDir(filepath.Join(source, "config"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		links = append(links, profileLink{
			source:      filepath.Join(source, "config", entry.Name()),
			destination: filepath.Join(config, entry.Name()),
		})
	}

	// Check every destination before changing any user files.
	var conflicts []string
	for index := range links {
		link := &links[index]
		destinationInfo, err := os.Lstat(link.destination)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if destinationInfo.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(link.destination)
			if err == nil && resolved == link.source {
				link.active = true
				continue
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		link.conflict = true
		conflicts = append(conflicts, link.destination)
	}
	if len(conflicts) > 0 && !replace {
		return fmt.Errorf("Existing files were not changed:\n%s\nRun hephaestus-profile --replace to keep backups and activate.",
			strings.Join(conflicts, "\n"))
	}
	if err := os.MkdirAll(config, 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	for _, link := range links {
		if link.active {
			continue
		}
		if link.conflict {
			backup := link.destination + ".hephaestus-backup." + stamp
			if _, err := os.Lstat(backup); !errors.Is(err, os.ErrNotExist) {
				if err != nil {
					return err
				}
				return fmt.Errorf("Backup already exists: %s", backup)
			}
			if err := os.Rename(link.destination, backup); err != nil {
				return err
			}
			fmt.Printf("Backed up %s -> %s\n", link.destination, backup)
		}
		if err := os.Symlink(link.source, link.destination); err != nil {
			return err
		}
		fmt.Printf("Activated %s\n", link.destination)
	}
	fmt.Println("Profile active. Open a new Zsh session; no login shell or secrets were changed.")
	return nil
}
