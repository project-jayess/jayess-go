package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeChildProcessCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"spawn",
		"exec",
		"pipe",
		"exitStatus",
		"signal",
		"cleanup",
	}
	for _, name := range expected {
		if !jayessruntime.HasChildProcessCapability(name) {
			t.Fatalf("expected child process runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsChildProcessSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(command, args) {
			const child = childProcess.spawn(command, args);
			const result = childProcess.exec(command);
			childProcess.pipe(child.stdin, process.stdin);
			childProcess.pipe(child.stdout, process.stdout);
			childProcess.pipe(child.stderr, process.stderr);
			const status = childProcess.exitStatus(child);
			childProcess.signal(child, "SIGTERM");
			childProcess.cleanup(child);
			return result || status;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeChildProcessCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.ChildProcessCapabilities() {
		if capability.Name == "" {
			t.Fatalf("child process capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("child process capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("child process capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelChildProcessRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var childProcess = {};`)
	requireSemanticError(t, err, "duplicate declaration childProcess")
}
