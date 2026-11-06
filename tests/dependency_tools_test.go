//go:build integration

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func copyGoToolModule(t *testing.T) string {
	t.Helper()
	source := filepath.Join(repository(t), "files/system/usr/share/hephaestus/go-tools")
	directory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum"} {
		writeFile(t, filepath.Join(directory, name), readFile(t, filepath.Join(source, name)), 0o644)
	}
	return directory
}

func buildGoTool(directory string) *exec.Cmd {
	command := exec.Command("go", "build", "-mod=readonly", "-o",
		directory+string(os.PathSeparator), "golang.org/x/tools/cmd/goimports", "mvdan.cc/sh/v3/cmd/shfmt")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	return command
}

func TestGoToolReadOnlyBuild(t *testing.T) {
	directory := copyGoToolModule(t)
	before := readFile(t, filepath.Join(directory, "go.sum"))
	if output, err := buildGoTool(directory).CombinedOutput(); err != nil {
		t.Fatalf("build locked Go tools: %v\n%s", err, output)
	}
	command := exec.Command(filepath.Join(directory, "shfmt"))
	command.Stdin = strings.NewReader("echo   hello\n")
	if output, err := command.CombinedOutput(); err != nil || string(output) != "echo hello\n" {
		t.Fatalf("locked shfmt failed to format shell input: %v\n%s", err, output)
	}
	if after := readFile(t, filepath.Join(directory, "go.sum")); before != after {
		t.Fatal("read-only build modified go.sum")
	}
}

func TestGoToolRejectsMissingChecksums(t *testing.T) {
	directory := copyGoToolModule(t)
	if err := os.Remove(filepath.Join(directory, "go.sum")); err != nil {
		t.Fatal(err)
	}
	output, err := buildGoTool(directory).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "missing go.sum entry") {
		t.Fatalf("expected missing checksum rejection: %v\n%s", err, output)
	}
}

func TestGoToolRejectsTamperedChecksums(t *testing.T) {
	directory := copyGoToolModule(t)
	filename := filepath.Join(directory, "go.sum")
	lines := strings.Split(readFile(t, filename), "\n")
	for index, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "golang.org/x/tools" && !strings.HasSuffix(fields[1], "/go.mod") {
			fields[2] = "h1:" + strings.Repeat("A", 43) + "="
			lines[index] = strings.Join(fields, " ")
		}
	}
	writeFile(t, filename, strings.Join(lines, "\n"), 0o644)
	output, err := buildGoTool(directory).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch: %v\n%s", err, output)
	}
}

func TestGoToolRejectsUnsupportedCompiler(t *testing.T) {
	directory := copyGoToolModule(t)
	manifest := readFile(t, filepath.Join(directory, "go.mod"))
	lines := strings.Split(manifest, "\n")
	for index, line := range lines {
		if strings.HasPrefix(line, "go ") {
			lines[index] = "go 999.0.0"
		}
	}
	writeFile(t, filepath.Join(directory, "go.mod"), strings.Join(lines, "\n"), 0o644)
	output, err := buildGoTool(directory).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "requires go >= 999.0.0") ||
		!strings.Contains(string(output), "GOTOOLCHAIN=local") {
		t.Fatalf("expected local compiler rejection: %v\n%s", err, output)
	}
}

func rustProject(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "Cargo.toml"),
		"[package]\nname = \"lock-fixture\"\nversion = \"0.1.0\"\nedition = \"2021\"\n", 0o644)
	mkdir(t, filepath.Join(directory, "src"))
	writeFile(t, filepath.Join(directory, "src/main.rs"), "fn main() {}\n", 0o644)
	lock := exec.Command("cargo", "generate-lockfile", "--offline")
	lock.Dir = directory
	if output, err := lock.CombinedOutput(); err != nil {
		t.Fatalf("generate fixture lock: %v\n%s", err, output)
	}
	return directory
}

func buildRustProject(directory string) *exec.Cmd {
	command := exec.Command("cargo", "build", "--locked", "--offline")
	command.Dir = directory
	return command
}

func TestRustLockedBuild(t *testing.T) {
	directory := rustProject(t)
	if output, err := buildRustProject(directory).CombinedOutput(); err != nil {
		t.Fatalf("build locked Rust fixture: %v\n%s", err, output)
	}
}

func TestRustRejectsMissingLock(t *testing.T) {
	directory := rustProject(t)
	if err := os.Remove(filepath.Join(directory, "Cargo.lock")); err != nil {
		t.Fatal(err)
	}
	if output, err := buildRustProject(directory).CombinedOutput(); err == nil || !strings.Contains(string(output), "--locked") {
		t.Fatalf("expected missing lock rejection: %v\n%s", err, output)
	}
}

func TestRustRejectsUnsupportedCompiler(t *testing.T) {
	directory := rustProject(t)
	manifest := readFile(t, filepath.Join(directory, "Cargo.toml"))
	writeFile(t, filepath.Join(directory, "Cargo.toml"), manifest+"rust-version = \"1.999.0\"\n", 0o644)
	lock := exec.Command("cargo", "generate-lockfile", "--offline")
	lock.Dir = directory
	if output, err := lock.CombinedOutput(); err != nil {
		t.Fatalf("generate unsupported compiler fixture lock: %v\n%s", err, output)
	}
	if output, err := buildRustProject(directory).CombinedOutput(); err == nil || !strings.Contains(string(output), "requires rustc 1.999.0") {
		t.Fatalf("expected Rust compiler rejection: %v\n%s", err, output)
	}
}
