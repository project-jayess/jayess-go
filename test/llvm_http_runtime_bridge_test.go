package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectHTTPRuntimeCalls(t *testing.T) {
	source := `
		const server = http.createServer((req, res) => res);
		const client = http.request("http://127.0.0.1");
		const request = http.requestObject(client);
		const response = http.responseObject(client);
		http.status(response, 200);
		http.writeBody(response, http.readBody(request));
		http.headers(request);
		const kept = http.keepAlive(http.withTimeout(client, 1000));
		http.streamBody(kept);
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
		Name:       "http-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_http_create_server",
		"@jayess_http_request_object",
		"@jayess_http_response_object",
		"@jayess_http_status",
		"@jayess_http_write_body",
		"@jayess_http_read_body",
		"@jayess_http_headers",
		"@jayess_http_request",
		"@jayess_http_with_timeout",
		"@jayess_http_keep_alive",
		"@jayess_http_stream_body",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected HTTP runtime IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestHTTPServerExampleCompilesToRuntimeCalls(t *testing.T) {
	root := cliRepoRoot(t)
	outputDir := cliTempDir(t, root, "http-example-*")
	output := filepath.Join(outputDir, "17-http-server.ll")
	runJayessCLI(t, root, "compile", "--target=linux-x64", "--emit=llvm", "-o", output, filepath.Join(root, "examples", "17-http-server.js"))

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read LLVM output: %v", err)
	}
	ir := string(content)
	for _, want := range []string{
		"@jayess_http_create_server",
		"@jayess_http_status",
		"@jayess_http_write_body",
		"@jayess_http_request",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected compiled HTTP example to contain %q:\n%s", want, ir)
		}
	}
}
