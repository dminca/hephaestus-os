package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFedoraBaseAlignment(t *testing.T) {
	script := filepath.Join(repository(t), "files/system/usr/share/hephaestus/check-fedora-version.sh")
	for _, scenario := range []struct {
		name        string
		baseVersion string
		wantSuccess bool
	}{
		{name: "matching release", baseVersion: "44.20261029.0", wantSuccess: true},
		{name: "new major release", baseVersion: "45.20261029.0"},
		{name: "missing version label", baseVersion: "null"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			output, err := exec.Command("bash", script, "44", scenario.baseVersion).CombinedOutput()
			if scenario.wantSuccess {
				if err != nil {
					t.Fatalf("matching Fedora release rejected: %v\n%s", err, output)
				}
			} else if err == nil || !strings.Contains(string(output), "differs from parser Fedora 44") {
				t.Fatalf("expected release mismatch: %v\n%s", err, output)
			}
		})
	}
}

func TestPythonInputMatchesCompiledLock(t *testing.T) {
	directory := filepath.Join(repository(t), "files/system/usr/share/hephaestus/python")
	input := readFile(t, filepath.Join(directory, "requirements.in"))
	lock := readFile(t, filepath.Join(directory, "requirements.txt"))
	for _, line := range strings.Split(input, "\n") {
		requirement := strings.TrimSpace(line)
		if requirement == "" || strings.HasPrefix(requirement, "#") {
			continue
		}
		if !strings.Contains("\n"+lock, "\n"+requirement+" \\\n") {
			t.Errorf("Python source %q is missing from the compiled lock", requirement)
		}
	}
}
