package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBundleKeepsSupportFiles(t *testing.T) {
	executable := linuxExecutable()
	archive := tarArchive(t, []*tar.Header{
		{Name: "bin/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(executable))},
		{Name: "main.lua", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(executable))},
	}, executable)
	filename := filepath.Join(t.TempDir(), "bundle.tar.gz")
	if err := os.WriteFile(filename, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "bin")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := binaryArtifact{URL: "https://example.test/bundle.tar.gz", Bundle: "server", Binaries: []string{"tool"}}
	if err := installBundle(filename, artifact, destination, true); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(filepath.Dir(destination), "server")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("verify-only installed a bundle: %v", err)
	}
	if err := installBundle(filename, artifact, destination, false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bin/tool", "main.lua"} {
		content, err := os.ReadFile(filepath.Join(target, name))
		if err != nil || !bytes.Equal(content, executable) {
			t.Fatalf("bundle file %s was not preserved: %v", name, err)
		}
	}
	if err := installBundle(filename, artifact, destination, false); err == nil {
		t.Fatal("overwrote existing bundle")
	}
}

func TestBundleExtractionRejectsUnsafeMembers(t *testing.T) {
	fixtures := []struct {
		name   string
		header *tar.Header
	}{
		{name: "absolute", header: &tar.Header{Name: "/outside", Typeflag: tar.TypeReg}},
		{name: "traversal", header: &tar.Header{Name: "../outside", Typeflag: tar.TypeReg}},
		{name: "symlink", header: &tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "file"}},
		{name: "hard link", header: &tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "file"}},
		{name: "device", header: &tar.Header{Name: "device", Typeflag: tar.TypeChar}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := tar.NewWriter(&buffer)
			if err := writer.WriteHeader(fixture.header); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := extractBundleTar(&buffer, t.TempDir(), false); err == nil {
				t.Fatal("accepted unsafe bundle")
			}
		})
	}
}

func TestBundleExtractionStripsOneRoot(t *testing.T) {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	for _, header := range []*tar.Header{
		{Name: "release/", Typeflag: tar.TypeDir},
		{Name: "release/bin/tool", Typeflag: tar.TypeReg, Mode: 0o755, Size: 4},
	} {
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size != 0 {
			if _, err := writer.Write([]byte("tool")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if err := extractBundleTar(&buffer, destination, true); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "bin/tool"))
	if err != nil || string(content) != "tool" {
		t.Fatalf("root was not stripped correctly: %v", err)
	}
}

func TestBundleRejectsInvalidMetadata(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../server", "/server"} {
		artifact := binaryArtifact{Bundle: name}
		if err := installBundle("unused", artifact, t.TempDir(), true); err == nil {
			t.Fatalf("accepted bundle name %q", name)
		}
	}
}
