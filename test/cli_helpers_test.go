package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func cliRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if filepath.Base(wd) == "test" {
		return filepath.Dir(wd)
	}
	return wd
}

func cliTempDir(t *testing.T, root string, pattern string) string {
	t.Helper()
	tempRoot := filepath.Join(root, "temp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		t.Fatalf("create temp root: %v", err)
	}
	dir, err := os.MkdirTemp(tempRoot, pattern)
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func runJayessCLI(t *testing.T, root string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"run", "./cmd/jayess"}, args...)
	command := exec.Command("go", commandArgs...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("jayess CLI failed: %v\n%s", err, string(output))
	}
	return string(output)
}

func installFakeSharedToolchain(t *testing.T, root string) string {
	t.Helper()
	binDir := cliTempDir(t, root, "fake-toolchain-*")
	installFakeSharedToolchainAt(t, binDir)
	return binDir
}

func installFakeSharedToolchainAt(t *testing.T, binDir string) {
	t.Helper()
	writeFakeTool(t, filepath.Join(binDir, "clang"), `#!/bin/sh
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    out="$arg"
  fi
  prev="$arg"
done
test -n "$out" || exit 2
mkdir -p "$(dirname "$out")"
printf 'fake shared library\n%s\n' "$*" > "$out"
`)
}

func writeFakeTool(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake tool %s: %v", path, err)
	}
}
