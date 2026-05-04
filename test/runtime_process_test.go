package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeProcessCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"argv",
		"env",
		"cwd",
		"exit",
		"stdin",
		"stdout",
		"stderr",
		"pid",
		"platform",
		"hrtime",
		"on",
	}
	for _, name := range expected {
		if !jayessruntime.HasProcessCapability(name) {
			t.Fatalf("expected process runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsProcessEnvironmentSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			const args = process.argv;
			const home = process.env.HOME;
			const cwd = process.cwd();
			process.stdout.write(cwd);
			process.stderr.write(home);
			process.stdin.read();
			const pid = process.pid;
			const platform = process.platform;
			const stamp = process.hrtime();
			process.on("SIGINT", () => process.exit(130));
			return args[0] || pid || platform || stamp;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeProcessCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.ProcessCapabilities() {
		if capability.Name == "" {
			t.Fatalf("process capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("process capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "property" && capability.Kind != "function" {
			t.Fatalf("process capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelProcessRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var process = {};`)
	requireSemanticError(t, err, "duplicate declaration process")
}
