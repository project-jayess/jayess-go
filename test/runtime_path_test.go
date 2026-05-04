package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimePathCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"join",
		"resolve",
		"normalize",
		"basename",
		"dirname",
		"extname",
		"relative",
	}
	for _, name := range expected {
		if !jayessruntime.HasPathCapability(name) {
			t.Fatalf("expected path runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsPathUtilitySurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(root, file) {
			const joined = path.join(root, "src", file);
			const resolved = path.resolve(joined);
			const clean = path.normalize(resolved);
			const base = path.basename(clean);
			const dir = path.dirname(clean);
			const ext = path.extname(base);
			const rel = path.relative(root, clean);
			return dir || ext || rel;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimePathCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.PathCapabilities() {
		if capability.Name == "" {
			t.Fatalf("path capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("path capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("path capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelPathRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var path = {};`)
	requireSemanticError(t, err, "duplicate declaration path")
}
