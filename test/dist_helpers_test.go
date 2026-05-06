package test

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeLLVMLicenses(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "LICENSE.TXT"), "llvm project license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "llvm", "LICENSE.TXT"), "llvm license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "clang", "LICENSE.TXT"), "clang license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "lld", "LICENSE.TXT"), "lld license")
	writeFile(t, filepath.Join(root, "refs", "llvm-project", "third-party", "README.md"), "third party notices")
}

func writeFakeLLVMTools(t *testing.T, root string, tools []string) {
	t.Helper()
	for _, tool := range tools {
		writeFile(t, filepath.Join(root, "refs", "llvm-project", "build", "bin", tool), "fake "+tool)
	}
}

func writeFakeLLVMBuildTools(t *testing.T, buildDir string, tools []string) {
	t.Helper()
	for _, tool := range tools {
		writeFile(t, filepath.Join(buildDir, "bin", tool), "fake "+tool)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}
}

func tarGzEntries(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	entries := map[string]struct{}{}
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		entries[header.Name] = struct{}{}
	}
	return entries
}

func requireArchiveEntry(t *testing.T, entries map[string]struct{}, name string) {
	t.Helper()
	if _, ok := entries[name]; !ok {
		t.Fatalf("expected archive entry %q, got %#v", name, entries)
	}
}

func archiveEntries(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	if strings.HasSuffix(path, ".zip") {
		return zipEntries(t, path)
	}
	return tarGzEntries(t, path)
}

func zipEntries(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := map[string]struct{}{}
	for _, file := range reader.File {
		entries[file.Name] = struct{}{}
	}
	return entries
}
