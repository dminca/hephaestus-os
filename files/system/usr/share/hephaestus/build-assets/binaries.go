package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type binaryArtifact struct {
	Repository string   `json:"repository"`
	Version    string   `json:"version"`
	URL        string   `json:"url"`
	SHA256     string   `json:"sha256"`
	Binaries   []string `json:"binaries"`
	Bundle     string   `json:"bundle,omitempty"`
	StripRoot  bool     `json:"strip_root,omitempty"`
}

type binaryLock struct {
	Architecture string           `json:"architecture"`
	Artifacts    []binaryArtifact `json:"artifacts"`
}

func lockedBinaryVersion(lockPath, name string) (string, error) {
	var lock binaryLock
	if err := readLock(lockPath, &lock); err != nil {
		return "", err
	}
	if lock.Architecture != "x86_64" || len(lock.Artifacts) == 0 {
		return "", errors.New("binary lock requires x86_64 architecture and artifacts")
	}
	found := false
	var version string
	for _, artifact := range lock.Artifacts {
		for _, binary := range artifact.Binaries {
			if binary != name {
				continue
			}
			if found {
				return "", fmt.Errorf("binary lock has multiple entries for %s", name)
			}
			found = true
			version = strings.TrimPrefix(artifact.Version, "v")
			if version == "" || strings.ContainsAny(version, " \t\r\n") {
				return "", fmt.Errorf("binary lock has invalid version for %s", name)
			}
		}
	}
	if !found {
		return "", fmt.Errorf("binary lock has no entry for %s", name)
	}
	return version, nil
}

func installBinaries(lockPath, destination string, verifyOnly bool, output io.Writer) error {
	var lock binaryLock
	if err := readLock(lockPath, &lock); err != nil {
		return err
	}
	if lock.Architecture != "x86_64" || len(lock.Artifacts) == 0 {
		return errors.New("binary lock requires x86_64 architecture and artifacts")
	}
	if !verifyOnly && (runtime.GOOS != "linux" || runtime.GOARCH != "amd64") {
		return errors.New("these artifacts require Linux x86_64; use --verify-only on other hosts")
	}
	if !verifyOnly {
		if err := os.MkdirAll(destination, 0o755); err != nil {
			return err
		}
	}
	temporary, err := os.MkdirTemp("", "hephaestus-binaries-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	for _, artifact := range lock.Artifacts {
		if len(artifact.Binaries) == 0 {
			return fmt.Errorf("artifact %s has no executable names", artifact.Repository)
		}
		archive := filepath.Join(temporary, "download")
		if err := downloadVerified(artifact.URL, artifact.SHA256, archive); err != nil {
			return err
		}
		if artifact.Bundle != "" {
			if err := installBundle(archive, artifact, destination, verifyOnly); err != nil {
				return fmt.Errorf("install %s bundle: %w", artifact.Repository, err)
			}
			fmt.Fprintf(output, "Verified %s bundle (%s)\n", artifact.Repository, artifact.Version)
			continue
		}
		data, err := os.ReadFile(archive)
		if err != nil {
			return err
		}
		for _, name := range artifact.Binaries {
			if name == "." || name == ".." || name == "" || strings.ContainsAny(name, `/\`) {
				return fmt.Errorf("invalid executable name: %q", name)
			}
			content, err := extractExecutable(data, artifact.URL, name)
			if err != nil {
				return err
			}
			if err := checkExecutable(content, name); err != nil {
				return err
			}
			if !verifyOnly {
				if err := writeExecutable(filepath.Join(destination, name), content); err != nil {
					return err
				}
			}
			fmt.Fprintf(output, "Verified %s (%s)\n", name, artifact.Version)
		}
	}
	return nil
}

func extractExecutable(data []byte, url, name string) ([]byte, error) {
	var content []byte
	matches := 0
	if strings.HasSuffix(url, ".zip") {
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, member := range archive.File {
			if path.Base(member.Name) != name || !member.Mode().IsRegular() {
				continue
			}
			file, err := member.Open()
			if err != nil {
				return nil, err
			}
			content, err = io.ReadAll(file)
			closeError := file.Close()
			if err != nil {
				return nil, err
			}
			if closeError != nil {
				return nil, closeError
			}
			matches++
		}
	} else if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		compressed, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer compressed.Close()
		archive := tar.NewReader(compressed)
		for {
			member, err := archive.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if path.Base(member.Name) != name || member.Typeflag != tar.TypeReg {
				continue
			}
			content, err = io.ReadAll(archive)
			if err != nil {
				return nil, err
			}
			matches++
		}
	} else if strings.HasSuffix(url, ".gz") {
		compressed, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer compressed.Close()
		return io.ReadAll(compressed)
	} else {
		return data, nil
	}
	if matches != 1 {
		return nil, fmt.Errorf("expected one %s executable, found %d", name, matches)
	}
	return content, nil
}

func checkExecutable(content []byte, name string) error {
	validHeader := len(content) >= 20 && bytes.Equal(content[:4], []byte("\x7fELF"))
	if !validHeader || content[4] != 2 || content[5] != 1 || binary.LittleEndian.Uint16(content[18:20]) != 62 {
		return fmt.Errorf("not an x86_64 ELF executable: %s", name)
	}
	return nil
}

func writeExecutable(destination string, content []byte) error {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Chmod(0o755); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
