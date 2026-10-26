package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type sourcePin struct {
	Revision string `json:"revision"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

type sourceLock struct {
	Sources map[string]sourcePin `json:"sources"`
}

func installSources(lockPath, destination string, output io.Writer) error {
	var lock sourceLock
	if err := readLock(lockPath, &lock); err != nil {
		return err
	}
	if len(lock.Sources) == 0 {
		return errors.New("source lock is missing sources")
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	repositories := make([]string, 0, len(lock.Sources))
	for repository := range lock.Sources {
		parts := strings.Split(repository, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" ||
			parts[1] == "." || parts[1] == ".." || strings.Contains(repository, `\`) {
			return fmt.Errorf("invalid source repository: %q", repository)
		}
		repositories = append(repositories, repository)
	}
	sort.Strings(repositories)
	for _, repository := range repositories {
		pin := lock.Sources[repository]
		target := filepath.Join(destination, path.Base(repository))
		if err := installSource(pin, target); err != nil {
			return fmt.Errorf("install %s: %w", repository, err)
		}
		fmt.Fprintf(output, "Installed %s at %s\n", repository, pin.Revision)
	}
	return nil
}

func installSource(pin sourcePin, destination string) error {
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("refusing to overwrite existing sources: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temporary, err := os.MkdirTemp("", "hephaestus-source-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	archive := filepath.Join(temporary, "source.tar.gz")
	if err := downloadVerified(pin.URL, pin.SHA256, archive); err != nil {
		return err
	}
	if err := validateSourceArchive(archive); err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		return err
	}
	command := exec.Command("tar", "-xzf", archive, "--strip-components=1", "-C", destination)
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("extract sources: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func validateSourceArchive(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	members := make(map[string]byte)
	links := make(map[string]string)
	root := ""
	for {
		member, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		// GitHub source archives include the commit as global PAX metadata.
		if member.Typeflag == tar.TypeXGlobalHeader {
			for key := range member.PAXRecords {
				if key != "comment" {
					return fmt.Errorf("unsupported global archive metadata: %s", key)
				}
			}
			continue
		}
		name := path.Clean(member.Name)
		if path.IsAbs(member.Name) || name == "." || containsParent(member.Name) {
			return fmt.Errorf("unsafe archive member: %s", member.Name)
		}
		top := strings.SplitN(name, "/", 2)[0]
		if root == "" {
			root = top
		}
		if top != root || (name == root && member.Typeflag != tar.TypeDir) {
			return fmt.Errorf("archive member is outside its top-level directory: %s", member.Name)
		}
		switch member.Typeflag {
		case tar.TypeReg, tar.TypeDir:
		case tar.TypeSymlink, tar.TypeLink:
			target := member.Linkname
			if path.IsAbs(target) {
				return fmt.Errorf("unsafe archive link: %s", member.Name)
			}
			if member.Typeflag == tar.TypeSymlink {
				target = path.Join(path.Dir(name), target)
			}
			target = path.Clean(target)
			if !strings.HasPrefix(target, root+"/") {
				return fmt.Errorf("unsafe archive link: %s", member.Name)
			}
			links[name] = target
		default:
			return fmt.Errorf("unsupported archive member: %s", member.Name)
		}
		if _, exists := members[name]; exists {
			return fmt.Errorf("duplicate archive member: %s", member.Name)
		}
		members[name] = member.Typeflag
	}
	if len(members) == 0 {
		return errors.New("source archive is empty")
	}
	// Reject writes through archive-created links, regardless of member order.
	for name := range members {
		if err := rejectLinkParents(name, links); err != nil {
			return err
		}
	}
	for name, target := range links {
		if err := rejectLinkParents(target, links); err != nil {
			return err
		}
		if members[name] == tar.TypeLink {
			if kind, exists := members[target]; !exists || kind != tar.TypeReg {
				return fmt.Errorf("hard link %s must target a regular archive file", name)
			}
		}
	}
	return nil
}

func containsParent(name string) bool {
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func rejectLinkParents(name string, links map[string]string) error {
	for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
		if _, exists := links[parent]; exists {
			return fmt.Errorf("archive path traverses link %s: %s", parent, name)
		}
	}
	return nil
}
