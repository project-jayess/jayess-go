package test

import (
	"testing"

	"jayess-go/tooling"
)

func TestToolingCommandsAreDeclared(t *testing.T) {
	for _, name := range []string{"compile", "package", "run", "init", "test"} {
		if !tooling.HasCommand(name) {
			t.Fatalf("expected tooling command %s", name)
		}
	}
}

func TestToolingOptionsAreDeclared(t *testing.T) {
	for _, name := range []string{"target", "output", "executable", "emit", "debug-info"} {
		if !tooling.HasOption(name) {
			t.Fatalf("expected tooling option %s", name)
		}
	}
}

func TestToolingEmitKindsAreDeclared(t *testing.T) {
	if !tooling.HasEmitKind(tooling.EmitLLVMIR) {
		t.Fatal("expected LLVM IR emit kind")
	}
	if !tooling.HasEmitKind(tooling.EmitNative) {
		t.Fatal("expected native executable emit kind")
	}
	if !tooling.HasEmitKind(tooling.EmitDist) {
		t.Fatal("expected app dist emit kind")
	}
}

func TestToolingDiagnosticFormatIncludesDebuggableLocations(t *testing.T) {
	format := tooling.DefaultDiagnosticFormat()
	if !format.ShowsFile || !format.ShowsLine || !format.ShowsColumn {
		t.Fatalf("expected file/line/column diagnostic location, got %#v", format)
	}
	if !format.ShowsDetail {
		t.Fatalf("expected diagnostic detail support, got %#v", format)
	}
}
