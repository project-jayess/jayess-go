package lowering

import (
	"testing"

	"jayess-go/ast"
)

func TestLowerClassesPreservesClassMetadata(t *testing.T) {
	program := &ast.Program{
		Classes: []*ast.ClassDecl{
			{
				Name:       "Child",
				SuperClass: "Base",
				Members: []ast.ClassMember{
					&ast.ClassFieldDecl{Name: "name", Initializer: &ast.StringLiteral{Value: "jay"}},
					&ast.ClassFieldDecl{Name: "count", Static: true},
					&ast.ClassMethodDecl{Name: "constructor", IsConstructor: true, Params: []ast.Parameter{{Name: "value"}}},
					&ast.ClassMethodDecl{Name: "read", Params: []ast.Parameter{{Name: "extra"}}},
					&ast.ClassMethodDecl{Name: "hidden", Private: true},
				},
			},
		},
	}

	classes := LowerClasses(program)
	if len(classes) != 1 {
		t.Fatalf("expected 1 lowered class, got %d", len(classes))
	}
	classDecl := classes[0]
	if classDecl.Name != "Child" || classDecl.SuperClass != "Base" {
		t.Fatalf("unexpected class metadata: %+v", classDecl)
	}
	if len(classDecl.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(classDecl.Fields))
	}
	if len(classDecl.Methods) != 3 {
		t.Fatalf("expected 3 methods, got %d", len(classDecl.Methods))
	}
	if !classDecl.Fields[0].HasInitializer {
		t.Fatalf("expected instance field initializer to be preserved")
	}
	if !classDecl.Fields[1].Static {
		t.Fatalf("expected static field metadata")
	}
	if !classDecl.Methods[0].IsConstructor || classDecl.Methods[0].ParamCount != 1 {
		t.Fatalf("expected constructor metadata, got %+v", classDecl.Methods[0])
	}
	if !classDecl.Methods[2].Private {
		t.Fatalf("expected private method metadata")
	}
}
