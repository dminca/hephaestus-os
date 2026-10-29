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
	"strings"
)

// LuaLS and HLS need their support files, not just a standalone executable.
func installBundle(archive string, artifact binaryArtifact, destination string, verifyOnly bool) error {
	if !safeFilename(artifact.Bundle) {
		return fmt.Errorf("invalid bundle name: %q", artifact.Bundle)
	}
	parent := filepath.Dir(destination)
	if verifyOnly {
		parent = os.TempDir()
	}
	temporary, err := os.MkdirTemp(parent, ".hephaestus-bundle-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if err := unpackBundle(archive, artifact, temporary); err != nil {
		return err
	}
	for _, name := range artifact.Binaries {
		if !safeFilename(name) {
			return fmt.Errorf("invalid executable name: %q", name)
		}
		file, err := os.Open(filepath.Join(temporary, "bin", name))
		if err != nil {
			return err
		}
		header := make([]byte, 20)
		_, readError := io.ReadFull(file, header)
		file.Close()
		if readError != nil {
			return readError
		}
		if err := checkExecutable(header, name); err != nil {
			return err
		}
	}
	if verifyOnly {
		return nil
	}
	target := filepath.Join(parent, artifact.Bundle)
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("refusing to overwrite existing bundle: %s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Chmod(temporary, 0o755); err != nil {
		return err
	}
	return os.Rename(temporary, target)
}

func unpackBundle(filename string, artifact binaryArtifact, destination string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	if strings.HasSuffix(artifact.URL, ".tar.gz") {
		compressed, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer compressed.Close()
		return extractBundleTar(compressed, destination, artifact.StripRoot)
	}
	if !strings.HasSuffix(artifact.URL, ".tar.xz") {
		return fmt.Errorf("unsupported bundle archive: %s", artifact.URL)
	}
	command := exec.Command("xz", "-dc")
	command.Stdin = file
	var stderr strings.Builder
	command.Stderr = &stderr
	pipe, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	extractError := extractBundleTar(pipe, destination, artifact.StripRoot)
	// Drain the decompressor even after invalid input so Wait cannot deadlock.
	_, drainError := io.Copy(io.Discard, pipe)
	commandError := command.Wait()
	if extractError != nil {
		return extractError
	}
	if drainError != nil {
		return drainError
	}
	if commandError != nil {
		return fmt.Errorf("decompress bundle: %w: %s", commandError, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func extractBundleTar(reader io.Reader, destination string, stripRoot bool) error {
	archive := tar.NewReader(reader)
	root := ""
	for {
		member, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if path.IsAbs(member.Name) || containsParent(member.Name) || strings.Contains(member.Name, `\`) {
			return fmt.Errorf("unsafe bundle member: %s", member.Name)
		}
		name := path.Clean(member.Name)
		if stripRoot {
			parts := strings.SplitN(name, "/", 2)
			if root == "" {
				root = parts[0]
			}
			if parts[0] != root {
				return errors.New("bundle has multiple top-level directories")
			}
			if len(parts) == 1 {
				if member.Typeflag != tar.TypeDir {
					return fmt.Errorf("invalid bundle root: %s", name)
				}
				continue
			}
			name = parts[1]
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		switch member.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if member.Mode&0o111 != 0 {
				mode = 0o755
			}
			file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, copyError := io.Copy(file, archive)
			closeError := file.Close()
			if copyError != nil {
				return copyError
			}
			if closeError != nil {
				return closeError
			}
		default:
			return fmt.Errorf("unsupported bundle member: %s", member.Name)
		}
	}
}

func safeFilename(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
}
