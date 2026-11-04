package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
)

type profileLink struct {
	source      string
	destination string
	active      bool
	conflict    bool
	content     []byte
	mode        os.FileMode
}

func main() {
	source := flag.String("source", "/usr/share/hephaestus/profile", "Profile asset directory")
	replace := flag.Bool("replace", false, "Back up conflicting files first")
	wallet := flag.String("git-wallet", "", "Read Git identity from this KDE wallet (requires a user session)")
	folder := flag.String("git-wallet-folder", "Hephaestus", "Wallet folder containing Git identity")
	entry := flag.String("git-wallet-entry", "git-identity", "Wallet entry containing JSON name and email")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Unexpected arguments:", strings.Join(flag.Args(), " "))
		os.Exit(2)
	}
	var identity []byte
	if *wallet != "" {
		var err error
		identity, err = walletIdentity(*wallet, *folder, *entry)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Git identity retrieval failed:", err)
			os.Exit(1)
		}
	}
	if err := activateProfileWithIdentity(*source, *replace, identity); err != nil {
		fmt.Fprintln(os.Stderr, "Profile activation failed:", err)
		os.Exit(1)
	}
}

func activateProfile(source string, replace bool) error {
	return activateProfileWithIdentity(source, replace, nil)
}

func activateProfileWithIdentity(source string, replace bool, identity []byte) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		return fmt.Errorf("Incomplete profile: %w", err)
	}
	for _, required := range []string{"zshrc", "config/nvim/init.lua", "config/tmux/tmux.conf"} {
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
	links = append(links, profileLink{
		source: filepath.Join(source, "config/tmux/tmux.conf"), destination: filepath.Join(home, ".tmux.conf"),
	})
	private := filepath.Join(config, "hephaestus-private")
	if err := checkPrivateDirectory(private); err != nil {
		return err
	}
	identityPath := filepath.Join(private, "git-identity")
	if info, err := os.Lstat(identityPath); err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			return fmt.Errorf("Private identity must be a regular file with permissions 0600: %s", identityPath)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	links = append(links, profileLink{
		destination: filepath.Join(home, ".gitconfig"),
		content:     []byte("[include]\n\tpath = " + gitQuote(identityPath) + "\n"), mode: 0o644,
	})
	if identity != nil {
		links = append(links, profileLink{
			destination: identityPath, content: identity, mode: 0o600,
		})
	}
	entries, err := os.ReadDir(filepath.Join(source, "config"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == "hephaestus-private" || entry.Name() == "git" {
			return errors.New("Profile must not contain image-backed private or Git directories")
		}
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
		if link.mode == 0o600 && (!destinationInfo.Mode().IsRegular() || destinationInfo.Mode().Perm() != 0o600) {
			return fmt.Errorf("Private identity must be a regular file with permissions 0600: %s", link.destination)
		}
		if link.content != nil && destinationInfo.Mode().IsRegular() {
			content, err := os.ReadFile(link.destination)
			if err != nil {
				return err
			}
			if bytes.Equal(content, link.content) && destinationInfo.Mode().Perm() == link.mode {
				link.active = true
				continue
			}
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
	if identity != nil {
		if err := os.MkdirAll(private, 0o700); err != nil {
			return err
		}
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
			info, err := os.Lstat(link.destination)
			if err != nil {
				return err
			}
			if link.content != nil && info.Mode().IsRegular() {
				original, err := os.ReadFile(link.destination)
				if err != nil {
					return err
				}
				if err := publishFile(backup, original, info.Mode().Perm()); err != nil {
					return err
				}
			} else {
				if err := os.Rename(link.destination, backup); err != nil {
					return err
				}
			}
			fmt.Printf("Backed up %s -> %s\n", link.destination, backup)
		}
		if link.content != nil {
			if err := publishFile(link.destination, link.content, link.mode); err != nil {
				return err
			}
		} else {
			if err := os.Symlink(link.source, link.destination); err != nil {
				return err
			}
		}
		fmt.Printf("Activated %s\n", link.destination)
	}
	fmt.Println("Profile active. Open a new Zsh session. The login shell and wallet were not changed.")
	return nil
}

func gitQuote(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`, "\b", `\b`).Replace(value) + `"`
}

func walletIdentity(wallet, folder, entry string) ([]byte, error) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		return nil, errors.New("Run in your KDE user session with its D-Bus session bus")
	}
	var binary string
	for _, name := range []string{"kwallet-query", "kwallet6-query"} {
		if path, err := exec.LookPath(name); err == nil {
			binary = path
			break
		}
	}
	if binary == "" {
		return nil, errors.New("KDE wallet query executable is missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "--folder", folder, "--read-password", entry, "--", wallet)
	output, err := command.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("Wallet read timed out; unlock the wallet and retry")
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, fmt.Errorf("Wallet read failed (exit %d); check wallet/folder/entry and unlock authorization", exit.ExitCode())
		}
		return nil, errors.New("Wallet read failed or timed out; check wallet/folder/entry and unlock authorization")
	}
	return parseIdentity(output)
}

func parseIdentity(output []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(output, &fields); err != nil || len(fields) != 2 {
		return nil, errors.New("Wallet entry must be a JSON object containing only name and email")
	}
	values := make(map[string]string, 2)
	for _, key := range []string{"name", "email"} {
		var value string
		if err := json.Unmarshal(fields[key], &value); err != nil || strings.TrimSpace(value) == "" ||
			strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return nil, fmt.Errorf("Wallet identity has an invalid %s field", key)
		}
		values[key] = value
	}
	return []byte("[user]\n\tname = " + gitQuote(values["name"]) + "\n\temail = " + gitQuote(values["email"]) + "\n"), nil
}

func checkPrivateDirectory(path string) error {
	for _, current := range []string{path, filepath.Dir(path)} {
		info, err := os.Lstat(current)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("Refusing unsafe private configuration parent: %s", current)
			}
			if current == path && info.Mode().Perm() != 0o700 {
				return fmt.Errorf("Private configuration directory must have permissions 0700: %s", path)
			}
			if current == path {
				stat, ok := info.Sys().(*syscall.Stat_t)
				if !ok || stat.Uid != uint32(os.Getuid()) {
					return fmt.Errorf("Private configuration directory must belong to the current user: %s", path)
				}
			}
		}
	}
	return nil
}

func publishFile(path string, content []byte, mode os.FileMode) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".hephaestus-*")
	if err != nil {
		return err
	}
	defer func() {
		removeErr := os.Remove(file.Name())
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	if _, err = file.Write(content); err != nil {
		return errors.Join(err, file.Close())
	}
	if err = file.Chmod(mode); err != nil {
		return errors.Join(err, file.Close())
	}
	if err = file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
