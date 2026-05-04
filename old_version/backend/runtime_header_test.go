package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func normalizeRuntimeHeaderSurface(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var lines []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || line == `extern "C" {` || line == "}" {
			continue
		}
		lines = append(lines, strings.Join(strings.Fields(line), " "))
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestRuntimeHeaderPublicAPISnapshot(t *testing.T) {
	runtimeDir, err := runtimeIncludePath()
	if err != nil {
		t.Fatalf("resolve runtime include path: %v", err)
	}

	got := normalizeRuntimeHeaderSurface(t, filepath.Join(runtimeDir, "jayess_runtime_types.h")) +
		normalizeRuntimeHeaderSurface(t, filepath.Join(runtimeDir, "jayess_runtime.h"))

	goldenPath := filepath.Join("testdata", "runtime_public_api.golden")
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read %s: %v", goldenPath, err)
	}
	want := string(wantBytes)

	if got != want {
		t.Fatalf("runtime public API surface changed unexpectedly\nwant:\n%s\ngot:\n%s", want, got)
	}
}
