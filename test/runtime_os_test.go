package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeOSCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"platform",
		"arch",
		"tmpdir",
		"hostname",
		"uptime",
		"cpus",
		"memory",
		"userInfo",
		"env",
	}
	for _, name := range expected {
		if !jayessruntime.HasOSCapability(name) {
			t.Fatalf("expected OS runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsOSSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			const platform = os.platform();
			const arch = os.arch();
			const temp = os.tmpdir();
			const host = os.hostname();
			const uptime = os.uptime();
			const cpus = os.cpus();
			const memory = os.memory();
			const user = os.userInfo();
			const env = os.env();
			return platform || arch || temp || host || uptime || cpus || memory || user || env;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeOSCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.OSCapabilities() {
		if capability.Name == "" {
			t.Fatalf("OS capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("OS capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("OS capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelOSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var os = {};`)
	requireSemanticError(t, err, "duplicate declaration os")
}
