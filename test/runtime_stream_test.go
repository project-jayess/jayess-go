package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeStreamCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"readable",
		"writable",
		"duplex",
		"transform",
		"pipe",
		"awaitDrain",
	}
	for _, name := range expected {
		if !jayessruntime.HasStreamCapability(name) {
			t.Fatalf("expected stream runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsStreamSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(source, sink) {
			const input = stream.readable(source);
			const output = stream.writable(sink);
			const pair = stream.duplex(input, output);
			const upper = stream.transform((chunk) => chunk);
			stream.pipe(input, upper);
			stream.pipe(upper, output);
			const ready = stream.awaitDrain(output);
			return pair || ready;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeStreamCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.StreamCapabilities() {
		if capability.Name == "" {
			t.Fatalf("stream capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("stream capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("stream capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelStreamRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var stream = {};`)
	requireSemanticError(t, err, "duplicate declaration stream")
}
