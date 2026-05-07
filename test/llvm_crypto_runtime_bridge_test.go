package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectCryptoRuntimeCalls(t *testing.T) {
	source := `
		const nonce = crypto.randomBytes(16);
		const digest = crypto.hash("sha256", "message");
		const mac = crypto.hmac("sha256", "key", "message");
		const key = crypto.generateKey("ed25519");
		const sealed = crypto.encrypt("aes-256-gcm", "12345678901234567890123456789012", "123456789012", "message");
		crypto.decrypt("aes-256-gcm", "12345678901234567890123456789012", "123456789012", sealed);
		const publicText = crypto.publicEncrypt(key, "message");
		crypto.privateDecrypt(key, publicText);
		const sig = crypto.sign(key, "message");
		crypto.verify(key, "message", sig);
		const same = crypto.secureCompare(digest, mac);
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
		Name:       "crypto-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_crypto_random_bytes",
		"@jayess_crypto_hash",
		"@jayess_crypto_hmac",
		"@jayess_crypto_generate_key",
		"@jayess_crypto_encrypt",
		"@jayess_crypto_decrypt",
		"@jayess_crypto_public_encrypt",
		"@jayess_crypto_private_decrypt",
		"@jayess_crypto_sign",
		"@jayess_crypto_verify",
		"@jayess_crypto_secure_compare",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected crypto runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
