package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMStorageRuntimeBridgeEmitsRuntimeCalls(t *testing.T) {
	source := `
		const db = storage.open("./app.store.json");
		storage.put(db, "key", "value");
		const value = storage.get(db, "key");
		const rows = storage.scan(db, "");
		storage.delete(db, "key");
		storage.close(db);
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
		Name:       "storage-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_storage_open",
		"@jayess_storage_put",
		"@jayess_storage_get",
		"@jayess_storage_scan",
		"@jayess_storage_delete",
		"@jayess_storage_close",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected storage runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
