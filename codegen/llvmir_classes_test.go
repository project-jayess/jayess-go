package codegen

import (
	"strings"
	"testing"

	"jayess-go/ir"
)

func TestGenerateEmitsClassMetadataComments(t *testing.T) {
	module := &ir.Module{
		Classes: []ir.ClassDecl{
			{
				Name: "Base",
			},
			{
				Name:       "Child",
				SuperClass: "Base",
				Fields: []ir.ClassField{
					{Name: "count", Static: true, HasInitializer: true},
				},
				Methods: []ir.ClassMethod{
					{Name: "constructor", IsConstructor: true, ParamCount: 1},
					{Name: "read", ParamCount: 1},
				},
			},
		},
		Globals: []ir.VariableDecl{
			{Name: "Child__count", Kind: ir.DeclarationVar, Value: &ir.NumberLiteral{Value: 1}},
		},
		Functions: []ir.Function{
			{Name: "Base", Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.UndefinedLiteral{}}}},
			{Name: "Child", Params: []ir.Parameter{{Name: "value", Kind: ir.ValueDynamic}}, Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.UndefinedLiteral{}}}},
			{Name: "Child__read", Params: []ir.Parameter{{Name: "__self", Kind: ir.ValueDynamic}, {Name: "extra", Kind: ir.ValueDynamic}}, Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.UndefinedLiteral{}}}},
			{Name: "main", Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.NumberLiteral{Value: 0}}}},
		},
	}

	out, err := NewLLVMIRGenerator().Generate(module, "x86_64-pc-windows-msvc")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "; class Child extends Base") {
		t.Fatalf("expected class metadata comment, got:\n%s", text)
	}
	if !strings.Contains(text, ";   field count [static init]") {
		t.Fatalf("expected field metadata comment, got:\n%s", text)
	}
	if !strings.Contains(text, ";   method read [instance params=1]") {
		t.Fatalf("expected method metadata comment, got:\n%s", text)
	}
}

func TestGenerateRejectsInvalidClassLayout(t *testing.T) {
	module := &ir.Module{
		Classes: []ir.ClassDecl{
			{
				Name: "Counter",
				Methods: []ir.ClassMethod{
					{Name: "tick", ParamCount: 0},
				},
			},
		},
		Functions: []ir.Function{
			{Name: "Counter", Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.UndefinedLiteral{}}}},
			{Name: "main", Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.NumberLiteral{Value: 0}}}},
		},
	}

	_, err := NewLLVMIRGenerator().Generate(module, "x86_64-pc-windows-msvc")
	if err == nil {
		t.Fatalf("expected Generate to reject invalid class layout")
	}
	if !strings.Contains(err.Error(), "missing lowered method tick") {
		t.Fatalf("unexpected error: %v", err)
	}
}
