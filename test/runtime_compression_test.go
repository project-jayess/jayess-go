package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeCompressionCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"gzip",
		"gunzip",
		"deflate",
		"inflate",
		"brotliCompress",
		"brotliDecompress",
		"createCompressStream",
		"createDecompressStream",
	}
	for _, name := range expected {
		if !jayessruntime.HasCompressionCapability(name) {
			t.Fatalf("expected compression runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsCompressionSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(data) {
			const gz = compression.gzip(data);
			const plain = compression.gunzip(gz);
			const deflated = compression.deflate(plain);
			const inflated = compression.inflate(deflated);
			const brotli = compression.brotliCompress(inflated);
			const restored = compression.brotliDecompress(brotli);
			const input = compression.createCompressStream("gzip");
			const output = compression.createDecompressStream("gzip");
			return restored || input || output;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeCompressionCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.CompressionCapabilities() {
		if capability.Name == "" {
			t.Fatalf("compression capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("compression capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("compression capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelCompressionRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var compression = {};`)
	requireSemanticError(t, err, "duplicate declaration compression")
}
