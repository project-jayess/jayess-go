package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectCompressionRuntimeCalls(t *testing.T) {
	source := `
		const gz = compression.gzip("data");
		const plain = compression.gunzip(gz);
		const deflated = compression.deflate(plain);
		const inflated = compression.inflate(deflated);
		const brotli = compression.brotliCompress(inflated);
		const restored = compression.brotliDecompress(brotli);
		const compressedStream = compression.createCompressStream("gzip", restored);
		compression.createDecompressStream("gzip", compressedStream);
	`
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module, err := llvmbackend.LowerJayessStatementProgram(llvmbackend.JayessStatementProgram{
		Name:       "compression-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_compression_gzip",
		"@jayess_compression_gunzip",
		"@jayess_compression_deflate",
		"@jayess_compression_inflate",
		"@jayess_compression_brotli_compress",
		"@jayess_compression_brotli_decompress",
		"@jayess_compression_create_compress_stream",
		"@jayess_compression_create_decompress_stream",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected compression runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
