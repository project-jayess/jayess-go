package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectTLSHTTPSRuntimeCalls(t *testing.T) {
	source := `
		const cert = tls.certificate("cert.pem", "key.pem");
		const client = tls.client("socket", { host: "localhost" });
		const server = tls.server("socket", cert);
		const withProtocols = tls.withALPN(client, ["h2", "http/1.1"]);
		tls.verifyHostname(withProtocols, "localhost");
		const httpsServer = https.createServer({ cert: cert }, (req, res) => res);
		const request = https.request("https://localhost", https.secureDefaults());
		const trust = https.trustStore("ca.pem");
		https.verifyCertificate(request, trust);
		httpsServer || server;
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
		Name:       "tls-https-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_tls_certificate",
		"@jayess_tls_client",
		"@jayess_tls_server",
		"@jayess_tls_with_alpn",
		"@jayess_tls_verify_hostname",
		"@jayess_https_create_server",
		"@jayess_https_request",
		"@jayess_https_secure_defaults",
		"@jayess_https_trust_store",
		"@jayess_https_verify_certificate",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected TLS/HTTPS runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
