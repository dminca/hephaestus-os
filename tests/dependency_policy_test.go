package tests

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPythonInputMatchesCompiledLock(t *testing.T) {
	directory := filepath.Join(repository(t), "files/system/usr/share/hephaestus/python")
	input := readFile(t, filepath.Join(directory, "requirements.in"))
	lock := readFile(t, filepath.Join(directory, "requirements.txt"))
	var lockedRequirements []string
	for _, line := range strings.Split(lock, "\n") {
		requirement := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\\"))
		if requirement != "" && !strings.HasPrefix(requirement, "#") && !strings.HasPrefix(requirement, "--") {
			lockedRequirements = append(lockedRequirements, requirement)
		}
	}
	for _, line := range strings.Split(input, "\n") {
		requirement := strings.TrimSpace(line)
		if requirement == "" || strings.HasPrefix(requirement, "#") {
			continue
		}
		if !slices.Contains(lockedRequirements, requirement) {
			t.Errorf("Python source %q is missing from the compiled lock", requirement)
		}
	}
}
