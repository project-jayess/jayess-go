package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectDNSRuntimeCalls(t *testing.T) {
	source := `
		const records = dns.lookup("localhost");
		const names = dns.reverse("127.0.0.1");
		const resolver = dns.resolver(["1.1.1.1"]);
		const family = dns.isIP("127.0.0.1");
		const parsed = dns.parseIP("::1");
		records || names || resolver || family || parsed;
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
		Name:       "dns-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_dns_lookup",
		"@jayess_dns_reverse",
		"@jayess_dns_resolver",
		"@jayess_dns_is_ip",
		"@jayess_dns_parse_ip",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected DNS runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
