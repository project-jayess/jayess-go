package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestParserTryCatchStatement(t *testing.T) {
	program := parseProgram(t, `try { throw error; } catch (err) { return err; }`)
	stmt := requireType[*ast.TryStatement](t, program.Statements[0])
	if stmt.CatchName != "err" {
		t.Fatalf("expected catch binding err, got %q", stmt.CatchName)
	}
	if len(stmt.TryBody) != 1 || len(stmt.CatchBody) != 1 {
		t.Fatalf("expected try and catch bodies, got %#v", stmt)
	}
	requireType[*ast.ThrowStatement](t, stmt.TryBody[0])
	requireType[*ast.ReturnStatement](t, stmt.CatchBody[0])
}

func TestParserTryCatchObjectBinding(t *testing.T) {
	program := parseProgram(t, `try { work(); } catch ({ message }) { return message; }`)
	stmt := requireType[*ast.TryStatement](t, program.Statements[0])
	pattern := requireType[*ast.ObjectBindingPattern](t, stmt.CatchPattern)
	if len(pattern.Properties) != 1 || pattern.Properties[0].Key != "message" {
		t.Fatalf("unexpected catch binding pattern: %#v", pattern)
	}
}

func TestParserTryCatchArrayBinding(t *testing.T) {
	program := parseProgram(t, `try { work(); } catch ([code]) { return code; }`)
	stmt := requireType[*ast.TryStatement](t, program.Statements[0])
	pattern := requireType[*ast.ArrayBindingPattern](t, stmt.CatchPattern)
	if len(pattern.Elements) != 1 {
		t.Fatalf("unexpected catch binding pattern: %#v", pattern)
	}
}

func TestParserTryFinallyStatement(t *testing.T) {
	program := parseProgram(t, `try { work(); } finally { cleanup(); }`)
	stmt := requireType[*ast.TryStatement](t, program.Statements[0])
	if len(stmt.FinallyBody) != 1 {
		t.Fatalf("expected finally body, got %#v", stmt)
	}
	requireType[*ast.ExpressionStatement](t, stmt.FinallyBody[0])
}

func TestParserTryCatchFinallyStatement(t *testing.T) {
	program := parseProgram(t, `try { work(); } catch { recover(); } finally { cleanup(); }`)
	stmt := requireType[*ast.TryStatement](t, program.Statements[0])
	if len(stmt.CatchBody) != 1 || len(stmt.FinallyBody) != 1 {
		t.Fatalf("expected catch and finally bodies, got %#v", stmt)
	}
}

func TestParserRejectsBareTry(t *testing.T) {
	_, err := parser.New(lexer.New(`try { work(); }`)).ParseProgram()
	if err == nil {
		t.Fatalf("expected bare try error")
	}
}

func TestParserRejectsCatchBindingTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`try { work(); } catch (err: unknown) { return err; }`)
	if err == nil {
		t.Fatalf("expected unsupported catch binding type annotation error")
	}
	if !strings.Contains(err.Error(), "type annotations are not supported") {
		t.Fatalf("expected clear type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsDestructuredCatchBindingTypeAnnotationWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`try { work(); } catch ({ message }: Error) { return message; }`)
	if err == nil {
		t.Fatalf("expected unsupported destructured catch binding type annotation error")
	}
	if !strings.Contains(err.Error(), "type annotations are not supported") {
		t.Fatalf("expected clear type annotation diagnostic, got %v", err)
	}
}

func TestParserRejectsOptionalCatchBindingWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`try { work(); } catch (err?) { return err; }`)
	if err == nil {
		t.Fatalf("expected unsupported optional catch binding error")
	}
	if !strings.Contains(err.Error(), "optional bindings are not supported") {
		t.Fatalf("expected clear optional binding diagnostic, got %v", err)
	}
}

func TestParserRejectsOptionalDestructuredCatchBindingWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`try { work(); } catch ({ message }?) { return message; }`)
	if err == nil {
		t.Fatalf("expected unsupported optional destructured catch binding error")
	}
	if !strings.Contains(err.Error(), "optional bindings are not supported") {
		t.Fatalf("expected clear optional binding diagnostic, got %v", err)
	}
}

func TestParserRejectsCatchBindingDefiniteAssignmentWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`try { work(); } catch (err!) { return err; }`)
	if err == nil {
		t.Fatalf("expected unsupported catch binding definite assignment error")
	}
	if !strings.Contains(err.Error(), "definite assignment assertions are not supported") {
		t.Fatalf("expected clear definite assignment diagnostic, got %v", err)
	}
}

func TestParserRejectsDestructuredCatchBindingDefiniteAssignmentWithClearDiagnostic(t *testing.T) {
	_, err := parseProgramError(`try { work(); } catch ({ message }!) { return message; }`)
	if err == nil {
		t.Fatalf("expected unsupported destructured catch binding definite assignment error")
	}
	if !strings.Contains(err.Error(), "definite assignment assertions are not supported") {
		t.Fatalf("expected clear definite assignment diagnostic, got %v", err)
	}
}
