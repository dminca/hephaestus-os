package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractExecutable(t *testing.T) {
	executable := linuxExecutable()
	fixtures := []struct {
		name string
		url  string
		data []byte
	}{
		{name: "raw", url: "https://example.test/tool", data: executable},
		{name: "gzip", url: "https://example.test/tool.gz", data: compress(t, executable)},
		{name: "zip", url: "https://example.test/tool.zip", data: zipArchive(t, []string{"release/tool"}, executable)},
		{name: "tar", url: "https://example.test/tool.tar.gz", data: tarArchive(t, []*tar.Header{
			{Name: "release/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(executable))},
		}, executable)},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			content, err := extractExecutable(fixture.data, fixture.url, "tool")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(content, executable) {
				t.Fatal("extracted executable differs from the archive")
			}
			if err := checkExecutable(content, "tool"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRejectMissingOrDuplicateExecutables(t *testing.T) {
	for _, names := range [][]string{{"other"}, {"first/tool", "second/tool"}} {
		data := zipArchive(t, names, linuxExecutable())
		if _, err := extractExecutable(data, "https://example.test/archive.zip", "tool"); err == nil {
			t.Fatalf("accepted archive entries %v", names)
		}
		headers := make([]*tar.Header, 0, len(names))
		for _, name := range names {
			headers = append(headers, &tar.Header{
				Name: name, Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(linuxExecutable())),
			})
		}
		data = tarArchive(t, headers, linuxExecutable())
		if _, err := extractExecutable(data, "https://example.test/archive.tar.gz", "tool"); err == nil {
			t.Fatalf("accepted archive entries %v", names)
		}
	}
}

func TestRejectInvalidExecutables(t *testing.T) {
	wrongArchitecture := linuxExecutable()
	wrongArchitecture[18] = 183
	wrongClass := linuxExecutable()
	wrongClass[4] = 1
	for _, content := range [][]byte{nil, []byte("not ELF"), wrongArchitecture, wrongClass} {
		if err := checkExecutable(content, "tool"); err == nil {
			t.Fatal("accepted invalid executable")
		}
	}
}

func TestWriteExecutableRefusesExistingPaths(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "tool")
	if err := writeExecutable(executable, linuxExecutable()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(executable)
	if err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("executable permissions not set: %v", err)
	}
	if err := writeExecutable(executable, []byte("replacement")); err == nil {
		t.Fatal("overwrote an existing executable")
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(executable, link); err != nil {
		t.Fatal(err)
	}
	if err := writeExecutable(link, []byte("replacement")); err == nil {
		t.Fatal("followed an existing symlink")
	}
	content, err := os.ReadFile(executable)
	if err != nil || !bytes.Equal(content, linuxExecutable()) {
		t.Fatalf("original executable changed: %v", err)
	}
}

func TestPinnedExecutableCoexistsWithDistributionBinary(t *testing.T) {
	root := t.TempDir()
	distributionDirectory := filepath.Join(root, "usr/bin")
	pinnedDirectory := filepath.Join(root, strings.TrimPrefix(binariesDirectory, "/"))
	for _, directory := range []string{distributionDirectory, pinnedDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	distributionBinary := filepath.Join(distributionDirectory, "tree-sitter")
	original := []byte("distribution-managed executable")
	if err := os.WriteFile(distributionBinary, original, 0o755); err != nil {
		t.Fatal(err)
	}
	pinnedBinary := filepath.Join(pinnedDirectory, "tree-sitter")
	if err := writeExecutable(pinnedBinary, linuxExecutable()); err != nil {
		t.Fatalf("pinned executable collided with distribution package: %v", err)
	}
	content, err := os.ReadFile(distributionBinary)
	if err != nil || !bytes.Equal(content, original) {
		t.Fatalf("distribution executable changed: %v", err)
	}
	if err := writeExecutable(pinnedBinary, linuxExecutable()); err == nil {
		t.Fatal("isolation must not disable overwrite protection")
	}
}

func TestBinaryVerificationCommand(t *testing.T) {
	download := compress(t, linuxExecutable())
	stubDownload(t, download)
	lock := binaryLock{Architecture: "x86_64", Artifacts: []binaryArtifact{{
		Repository: "owner/tool", Version: "v1", URL: "https://example.test/tool.gz",
		SHA256: checksum(download), Binaries: []string{"tool"},
	}}}
	lockPath := writeLockFixture(t, lock)
	destination := filepath.Join(t.TempDir(), "not-created")
	var output bytes.Buffer
	if err := run([]string{"binaries", "--lock", lockPath, "--destination", destination, "--verify-only"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Verified tool (v1)") {
		t.Fatalf("missing verification output: %s", output.String())
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("verify-only created the destination: %v", err)
	}
	lock.Artifacts[0].SHA256 = strings.Repeat("0", 64)
	lockPath = writeLockFixture(t, lock)
	if err := run([]string{"binaries", "--lock", lockPath, "--verify-only"}, io.Discard); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	lock.Artifacts[0].SHA256 = checksum(download)
	lock.Artifacts[0].Binaries = []string{"../escape"}
	lockPath = writeLockFixture(t, lock)
	if err := run([]string{"binaries", "--lock", lockPath, "--verify-only"}, io.Discard); err == nil || !strings.Contains(err.Error(), "invalid executable name") {
		t.Fatalf("expected unsafe name rejection, got %v", err)
	}
}

func TestBinaryVersionCommand(t *testing.T) {
	for _, version := range []string{"v9.8.7", "10.2.3"} {
		t.Run(version, func(t *testing.T) {
			lock := binaryLock{Architecture: "x86_64", Artifacts: []binaryArtifact{{
				Version: version, Binaries: []string{"tool", "alias"},
			}}}
			for _, name := range []string{"tool", "alias"} {
				var output bytes.Buffer
				err := run([]string{"binary-version", "--lock", writeLockFixture(t, lock), name}, &output)
				if err != nil || output.String() != strings.TrimPrefix(version, "v")+"\n" {
					t.Fatalf("unexpected version output %q: %v", output.String(), err)
				}
			}
		})
	}
	for name, lock := range map[string]binaryLock{
		"missing":       {Architecture: "x86_64", Artifacts: []binaryArtifact{{Version: "v1", Binaries: []string{"other"}}}},
		"empty version": {Architecture: "x86_64", Artifacts: []binaryArtifact{{Binaries: []string{"tool"}}}},
		"prefix only":   {Architecture: "x86_64", Artifacts: []binaryArtifact{{Version: "v", Binaries: []string{"tool"}}}},
		"whitespace":    {Architecture: "x86_64", Artifacts: []binaryArtifact{{Version: "v1\n", Binaries: []string{"tool"}}}},
		"duplicate": {Architecture: "x86_64", Artifacts: []binaryArtifact{
			{Version: "v1", Binaries: []string{"tool"}}, {Version: "v2", Binaries: []string{"tool"}},
		}},
		"duplicate alias": {Architecture: "x86_64", Artifacts: []binaryArtifact{{Version: "v1", Binaries: []string{"tool", "tool"}}}},
		"architecture":    {Architecture: "other", Artifacts: []binaryArtifact{{Version: "v1", Binaries: []string{"tool"}}}},
		"empty lock":      {},
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			if err := run([]string{"binary-version", "--lock", writeLockFixture(t, lock), "tool"}, &output); err == nil {
				t.Fatal("accepted invalid version lookup")
			}
			if output.Len() != 0 {
				t.Fatalf("failed lookup printed version: %q", output.String())
			}
		})
	}
}

func TestSourceInstallationCommand(t *testing.T) {
	data := []byte("plugin content\n")
	archive := tarArchive(t, []*tar.Header{
		{Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader, PAXRecords: map[string]string{"comment": "commit revision"}},
		{Name: "plugin-revision/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "plugin-revision/plugin.lua", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(data))},
		{Name: "plugin-revision/alias.lua", Typeflag: tar.TypeSymlink, Linkname: "plugin.lua", Mode: 0o777},
	}, data)
	stubDownload(t, archive)
	lock := sourceLock{Sources: map[string]sourcePin{
		"owner/plugin": {Revision: strings.Repeat("a", 40), URL: "https://example.test/plugin.tar.gz", SHA256: checksum(archive)},
	}}
	lockPath := writeLockFixture(t, lock)
	destination := t.TempDir()
	arguments := []string{"sources", "--lock", lockPath, "--destination", destination}
	var output bytes.Buffer
	if err := run(arguments, &output); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"plugin.lua", "alias.lua"} {
		content, err := os.ReadFile(filepath.Join(destination, "plugin", name))
		if err != nil || !bytes.Equal(content, data) {
			t.Fatalf("incorrect extracted %s: %v", name, err)
		}
	}
	if !strings.Contains(output.String(), "Installed owner/plugin") {
		t.Fatalf("missing installation output: %s", output.String())
	}
	if err := run(arguments, io.Discard); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected overwrite rejection, got %v", err)
	}
}

func TestUnsafeSourceArchives(t *testing.T) {
	fixtures := []struct {
		name    string
		headers []*tar.Header
	}{
		{name: "absolute path", headers: []*tar.Header{{Name: "/outside", Typeflag: tar.TypeReg}}},
		{name: "parent traversal", headers: []*tar.Header{{Name: "root/../outside", Typeflag: tar.TypeReg}}},
		{name: "device", headers: []*tar.Header{{Name: "root/device", Typeflag: tar.TypeChar}}},
		{name: "escaping symlink", headers: []*tar.Header{{Name: "root/link", Typeflag: tar.TypeSymlink, Linkname: "../../outside"}}},
		{name: "absolute symlink", headers: []*tar.Header{{Name: "root/link", Typeflag: tar.TypeSymlink, Linkname: "/outside"}}},
		{name: "escaping hard link", headers: []*tar.Header{{Name: "root/link", Typeflag: tar.TypeLink, Linkname: "../outside"}}},
		{name: "file under symlink", headers: []*tar.Header{
			{Name: "root/link", Typeflag: tar.TypeSymlink, Linkname: "directory"},
			{Name: "root/link/file", Typeflag: tar.TypeReg},
		}},
		{name: "link declared after file", headers: []*tar.Header{
			{Name: "root/link/file", Typeflag: tar.TypeReg},
			{Name: "root/link", Typeflag: tar.TypeSymlink, Linkname: "directory"},
		}},
		{name: "multiple roots", headers: []*tar.Header{
			{Name: "first/file", Typeflag: tar.TypeReg},
			{Name: "second/file", Typeflag: tar.TypeReg},
		}},
		{name: "duplicate path", headers: []*tar.Header{
			{Name: "root/file", Typeflag: tar.TypeReg},
			{Name: "root/file", Typeflag: tar.TypeSymlink, Linkname: "other"},
		}},
		{name: "empty archive"},
		{name: "global path override", headers: []*tar.Header{
			{Name: "pax_global_header", Typeflag: tar.TypeXGlobalHeader, PAXRecords: map[string]string{"path": "/outside"}},
			{Name: "root/file", Typeflag: tar.TypeReg},
		}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			archive := tarArchive(t, fixture.headers, nil)
			stubDownload(t, archive)
			parent := t.TempDir()
			destination := filepath.Join(parent, "plugin")
			err := installSource(sourcePin{URL: "https://example.test/source.tar.gz", SHA256: checksum(archive)}, destination)
			if err == nil {
				t.Fatal("accepted unsafe source archive")
			}
			if _, err := os.Lstat(destination); !os.IsNotExist(err) {
				t.Fatalf("created destination before archive validation: %v", err)
			}
		})
	}
}

func TestSourceInstallationPreservesExistingSymlink(t *testing.T) {
	parent := t.TempDir()
	destination := filepath.Join(parent, "plugin")
	missingTarget := filepath.Join(parent, "missing")
	if err := os.Symlink(missingTarget, destination); err != nil {
		t.Fatal(err)
	}
	if err := installSource(sourcePin{}, destination); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected existing symlink rejection, got %v", err)
	}
	target, err := os.Readlink(destination)
	if err != nil || target != missingTarget {
		t.Fatalf("existing symlink changed: %v", err)
	}
}

func TestAssetCommandErrors(t *testing.T) {
	for _, arguments := range [][]string{nil, {"unknown"}, {"sources", "extra"}, {"binaries", "extra"},
		{"binary-version"}, {"binary-version", "tool", "extra"}, {"binary-version", "--unknown"},
		{"binary-version", "--lock", filepath.Join(t.TempDir(), "missing"), "tool"}} {
		if err := run(arguments, io.Discard); err == nil {
			t.Fatalf("accepted arguments %v", arguments)
		}
	}
	for _, content := range []string{"{", "{}"} {
		filename := filepath.Join(t.TempDir(), "lock.json")
		if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"sources", "binaries", "binary-version"} {
			arguments := []string{command, "--lock", filename}
			if command == "binary-version" {
				arguments = append(arguments, "tool")
			}
			if err := run(arguments, io.Discard); err == nil {
				t.Fatalf("accepted invalid %s lock %q", command, content)
			}
		}
	}
}

func TestDownloadErrors(t *testing.T) {
	stubDownload(t, []byte("payload"))
	destination := filepath.Join(t.TempDir(), "download")
	for _, fixture := range []struct {
		url      string
		checksum string
	}{
		{url: "http://example.test/source", checksum: strings.Repeat("0", 64)},
		{url: "https://example.test/source", checksum: "invalid"},
	} {
		if err := downloadVerified(fixture.url, fixture.checksum, destination); err == nil {
			t.Fatal("accepted invalid download metadata")
		}
	}
	t.Setenv("HEPHAESTUS_TEST_DOWNLOAD", filepath.Join(t.TempDir(), "missing"))
	if err := downloadVerified("https://example.test/source", strings.Repeat("0", 64), destination); err == nil || !strings.Contains(err.Error(), "download https://") {
		t.Fatalf("download failure was not surfaced: %v", err)
	}
}

func linuxExecutable() []byte {
	header := make([]byte, 20)
	copy(header, "\x7fELF")
	header[4] = 2
	header[5] = 1
	header[18] = 62
	return header
}

func compress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func tarArchive(t *testing.T, headers []*tar.Header, content []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, header := range headers {
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := writer.Write(content); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return compress(t, buffer.Bytes())
}

func zipArchive(t *testing.T, names []string, content []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeLockFixture(t *testing.T, value any) string {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "lock.json")
	if err := os.WriteFile(filename, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return filename
}

func stubDownload(t *testing.T, content []byte) {
	t.Helper()
	directory := t.TempDir()
	fixture := filepath.Join(directory, "payload")
	if err := os.WriteFile(fixture, content, 0o600); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    cp "$HEPHAESTUS_TEST_DOWNLOAD" "$2"
    exit $?
  fi
  shift
done
echo "missing download destination" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(directory, "curl"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HEPHAESTUS_TEST_DOWNLOAD", fixture)
}
