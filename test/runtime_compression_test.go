package test

import (
	"bytes"
	"errors"
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

func TestRuntimeCompressionGzipAndDeflateRoundTrip(t *testing.T) {
	input := []byte("jayess compression data jayess compression data")
	gzipped, err := jayessruntime.CompressionGzip(input)
	if err != nil {
		t.Fatalf("gzip failed: %v", err)
	}
	plain, err := jayessruntime.CompressionGunzip(gzipped)
	if err != nil {
		t.Fatalf("gunzip failed: %v", err)
	}
	if !bytes.Equal(plain, input) {
		t.Fatalf("gzip round trip mismatch: %q", plain)
	}

	deflated, err := jayessruntime.CompressionDeflate(input)
	if err != nil {
		t.Fatalf("deflate failed: %v", err)
	}
	inflated, err := jayessruntime.CompressionInflate(deflated)
	if err != nil {
		t.Fatalf("inflate failed: %v", err)
	}
	if !bytes.Equal(inflated, input) {
		t.Fatalf("deflate round trip mismatch: %q", inflated)
	}
}

func TestRuntimeCompressionStreamsUseSharedIOStream(t *testing.T) {
	source := jayessruntime.NewReadableStream("plain", bytes.NewReader([]byte("stream me")))
	compressed, err := jayessruntime.CompressionCreateCompressStream("gzip", source)
	if err != nil {
		t.Fatalf("compress stream failed: %v", err)
	}
	if !compressed.CanRead() {
		t.Fatal("expected compressed stream to be readable")
	}
	restored, err := jayessruntime.CompressionCreateDecompressStream("gzip", compressed)
	if err != nil {
		t.Fatalf("decompress stream failed: %v", err)
	}
	output, err := restored.ReadAll()
	if err != nil {
		t.Fatalf("read restored stream: %v", err)
	}
	if string(output) != "stream me" {
		t.Fatalf("unexpected restored stream %q", output)
	}
}

func TestRuntimeCompressionBrotliIsExplicitlyUnsupported(t *testing.T) {
	if _, err := jayessruntime.CompressionBrotliCompress([]byte("data")); !errors.Is(err, jayessruntime.ErrUnsupportedCompressionFormat) {
		t.Fatalf("expected unsupported brotli error, got %v", err)
	}
	if _, err := jayessruntime.CompressionBrotliDecompress([]byte("data")); !errors.Is(err, jayessruntime.ErrUnsupportedCompressionFormat) {
		t.Fatalf("expected unsupported brotli error, got %v", err)
	}
}

func TestSemanticRejectsTopLevelCompressionRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var compression = {};`)
	requireSemanticError(t, err, "duplicate declaration compression")
}
