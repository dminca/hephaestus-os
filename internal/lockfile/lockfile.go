package lockfile

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepositoryRoot lets the tools run from the repository root or a subdirectory.
func RepositoryRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("run this command inside the hephaestus-os repository")
		}
		directory = parent
	}
}

func GitHubJSON(endpoint string, result any) error {
	command := exec.Command("gh", "api", endpoint)
	var stderr strings.Builder
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("gh api %s: %w: %s", endpoint, err, strings.TrimSpace(stderr.String()))
	}
	if err := json.Unmarshal(output, result); err != nil {
		return fmt.Errorf("decode GitHub response for %s: %w", endpoint, err)
	}
	return nil
}

func DownloadChecksum(url string) (string, error) {
	hash := sha256.New()
	command := exec.Command("curl", "-fsSL", "--retry", "3", "--proto", "=https", "--proto-redir", "=https", url)
	command.Stdout = hash
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("download %s: %w: %s", url, err, strings.TrimSpace(stderr.String()))
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// WriteJSON replaces a lock only after all entries have been resolved successfully.
func WriteJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".lock-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(append(content, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
