package test

import (
	"os"
	"path/filepath"
	"testing"

	"jayess-go/resolver"
	jayessruntime "jayess-go/runtime"
)

func TestRuntimeTerminalCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{"isTTY", "size", "supportsColor"}
	for _, name := range expected {
		if !jayessruntime.HasTerminalCapability(name) {
			t.Fatalf("expected terminal runtime capability %s", name)
		}
	}
}

func TestDetectTerminalUsesFileAndEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-terminal.txt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create returned error: %v", err)
	}
	defer file.Close()

	info := jayessruntime.DetectTerminal(file, map[string]string{
		"COLUMNS": "120",
		"LINES":   "40",
		"TERM":    "xterm-256color",
	})
	if info.IsTerminal || info.Columns != 120 || info.Rows != 40 || !info.SupportsColor {
		t.Fatalf("unexpected terminal info: %#v", info)
	}

	noColor := jayessruntime.DetectTerminal(file, map[string]string{
		"TERM":     "xterm-256color",
		"NO_COLOR": "1",
	})
	if noColor.SupportsColor {
		t.Fatalf("NO_COLOR should disable color support")
	}
}

func TestSemanticAllowsTerminalSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			const tty = terminal.isTTY(process.stdout);
			const size = terminal.size(process.stdout);
			const color = terminal.supportsColor(process.stdout);
			return tty || size || color;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestResolverAllowsTerminalStdlibImport(t *testing.T) {
	path, err := resolver.ResolveStdlibImport("terminal")
	if err != nil {
		t.Fatalf("ResolveStdlibImport returned error: %v", err)
	}
	if path != "jayess:stdlib/terminal" {
		t.Fatalf("unexpected terminal stdlib path %q", path)
	}
}
