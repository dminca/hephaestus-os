package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const assetsDirectory = "/usr/share/hephaestus"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "Asset installation failed:", err)
		os.Exit(1)
	}
}

func run(arguments []string, output io.Writer) error {
	if len(arguments) == 0 {
		return errors.New("usage: hephaestus-assets <sources|binaries> [options]")
	}
	switch arguments[0] {
	case "sources":
		flags := flag.NewFlagSet("sources", flag.ContinueOnError)
		lock := flags.String("lock", filepath.Join(assetsDirectory, "sources.lock.json"), "source lock file")
		destination := flags.String("destination", filepath.Join(assetsDirectory, "nvim/plugins"), "plugin directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("sources does not accept positional arguments")
		}
		return installSources(*lock, *destination, output)
	case "binaries":
		flags := flag.NewFlagSet("binaries", flag.ContinueOnError)
		lock := flags.String("lock", filepath.Join(assetsDirectory, "binaries.lock.json"), "binary lock file")
		destination := flags.String("destination", "/usr/bin", "executable directory")
		verifyOnly := flags.Bool("verify-only", false, "verify downloads without installing or requiring a Linux host")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("binaries does not accept positional arguments")
		}
		return installBinaries(*lock, *destination, *verifyOnly, output)
	default:
		return fmt.Errorf("unknown asset command: %s", arguments[0])
	}
}

func readLock(filename string, result any) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, result); err != nil {
		return fmt.Errorf("read lock %s: %w", filename, err)
	}
	return nil
}

func downloadVerified(url, checksum, destination string) error {
	expected, err := hex.DecodeString(checksum)
	if err != nil || len(expected) != sha256.Size {
		return errors.New("lock entry requires a SHA256 checksum")
	}
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("download URL must use HTTPS: %s", url)
	}
	command := exec.Command("curl", "-fsSL", "--retry", "3",
		"--proto", "=https", "--proto-redir", "=https", url, "-o", destination)
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("download %s: %w: %s", url, err, strings.TrimSpace(stderr.String()))
	}
	file, err := os.Open(destination)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), checksum) {
		return fmt.Errorf("checksum mismatch: %s", url)
	}
	return nil
}
