package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBackendExpressionSequenceEmitsLeftToRightAndReturnsLast(t *testing.T) {
	first, err := parser.New(lexer.New(`1`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse first expression: %v", err)
	}
	second, err := parser.New(lexer.New(`2`)).ParseExpression()
	if err != nil {
		t.Fatalf("parse second expression: %v", err)
	}
	emitter := llvmbackend.NewExpressionEmitter()
	value, err := emitter.EmitExpressionSequence(first, second)
	if err != nil {
		t.Fatalf("emit expression sequence: %v", err)
	}
	if value != "%v1" {
		t.Fatalf("expected sequence to return final value %%v1, got %q", value)
	}
	body := strings.Join(emitter.Body(), "\n")
	firstIndex := strings.Index(body, "double 1")
	secondIndex := strings.Index(body, "double 2")
	if firstIndex < 0 || secondIndex < 0 || firstIndex > secondIndex {
		t.Fatalf("expected sequence body to emit left-to-right:\n%s", body)
	}
}

func TestLLVMBackendExpressionSequenceRejectsEmptySequence(t *testing.T) {
	emitter := llvmbackend.NewExpressionEmitter()
	if _, err := emitter.EmitExpressionSequence(); err == nil {
		t.Fatal("expected empty sequence to be rejected")
	}
}
