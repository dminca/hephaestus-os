package lockfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubJSONRejectsMalformedResponse(t *testing.T) {
	directory := t.TempDir()
	stub := "#!/bin/sh\nprintf '%s' 'not JSON'\n"
	if err := os.WriteFile(filepath.Join(directory, "gh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	var response struct {
		SHA string `json:"sha"`
	}
	err := GitHubJSON("repos/example/plugin/commits/HEAD", &response)
	if err == nil || !strings.Contains(err.Error(), "decode GitHub response") {
		t.Fatalf("expected JSON decoding error, got %v", err)
	}
}

func TestWriteJSON(t *testing.T) {
	t.Run("EncodingFailurePreservesPreviousLock", func(t *testing.T) {
		directory := t.TempDir()
		path := filepath.Join(directory, "sources.lock.json")
		const previous = "previous reviewed lock\n"
		if err := os.WriteFile(path, []byte(previous), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := WriteJSON(path, make(chan int)); err == nil {
			t.Fatal("expected unsupported JSON value to fail")
		}
		content, err := os.ReadFile(path)
		if err != nil || string(content) != previous {
			t.Fatalf("previous lock changed: %q (%v)", content, err)
		}
		assertNoTemporaryLocks(t, directory)
	})

	t.Run("RenameFailurePreservesDestinationAndCleansTemporaryFile", func(t *testing.T) {
		directory := t.TempDir()
		path := filepath.Join(directory, "sources.lock.json")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(path, "existing")
		if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := WriteJSON(path, map[string]string{"revision": "new"}); err == nil {
			t.Fatal("expected replacing a directory with a file to fail")
		}
		content, err := os.ReadFile(marker)
		if err != nil || string(content) != "keep" {
			t.Fatalf("destination changed: %q (%v)", content, err)
		}
		assertNoTemporaryLocks(t, directory)
	})

	t.Run("SuccessfulReplacementCleansTemporaryFile", func(t *testing.T) {
		directory := t.TempDir()
		path := filepath.Join(directory, "sources.lock.json")
		if err := WriteJSON(path, map[string]string{"revision": "reviewed"}); err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil || string(content) != "{\n  \"revision\": \"reviewed\"\n}\n" {
			t.Fatalf("unexpected lock contents: %q (%v)", content, err)
		}
		assertNoTemporaryLocks(t, directory)
	})
}

func assertNoTemporaryLocks(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".lock-*.json"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary lockfiles remain: %v (%v)", matches, err)
	}
}
