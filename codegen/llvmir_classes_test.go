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

func TestGenerateEmitsStableEntryWrapperABI(t *testing.T) {
	module := &ir.Module{
		Functions: []ir.Function{
			{Name: "main", Line: 2, Column: 1, Params: []ir.Parameter{{Name: "args", Kind: ir.ValueDynamic}}, Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.NumberLiteral{Value: 0}}}},
		},
	}

	out, err := NewLLVMIRGenerator().Generate(module, "x86_64-unknown-linux-gnu")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "target triple = \"x86_64-unknown-linux-gnu\"") {
		t.Fatalf("expected target triple in emitted IR, got:\n%s", text)
	}
	if strings.Contains(text, "target datalayout = ") {
		t.Fatalf("expected backend to rely on target triple instead of explicit datalayout, got:\n%s", text)
	}
	if !strings.Contains(text, "define double @jayess_user_main(ptr %args)") {
		t.Fatalf("expected source main to lower to jayess_user_main, got:\n%s", text)
	}
	for _, fragment := range []string{
		"; source function main at 2:1",
		"; debug frame main (2:1)",
		"; lowered symbol @jayess_user_main",
		"; native entry wrapper for main (2:1)",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected debug/source-location fragment %q, got:\n%s", fragment, text)
		}
	}
	if !strings.Contains(text, "define i32 @main(i32 %argc, ptr %argv)") {
		t.Fatalf("expected native entry wrapper signature, got:\n%s", text)
	}
	for _, fragment := range []string{
		"call void @jayess_init_globals()",
		"call ptr @jayess_make_args(i32 %argc, ptr %argv)",
		"call double @jayess_user_main(ptr %args)",
		"call void @jayess_run_microtasks()",
		"call void @jayess_runtime_shutdown()",
		"call i1 @jayess_has_exception()",
		"call void @jayess_report_uncaught_exception()",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected entry-wrapper fragment %q, got:\n%s", fragment, text)
		}
	}
}

func TestGenerateMatchesEntryWrapperSnapshot(t *testing.T) {
	module := &ir.Module{
		Functions: []ir.Function{
			{Name: "main", Line: 2, Column: 1, Body: []ir.Statement{&ir.ReturnStatement{Value: &ir.NumberLiteral{Value: 0}}}},
		},
	}

	out, err := NewLLVMIRGenerator().Generate(module, "x86_64-unknown-linux-gnu")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	text := string(out)
	start := strings.Index(text, "define double @jayess_user_main()")
	if start < 0 {
		t.Fatalf("expected jayess_user_main in emitted IR, got:\n%s", text)
	}
	got := text[start:]
	want := `define double @jayess_user_main() {
entry:
  call void @jayess_push_call_frame(ptr @.str.8)
  call void @jayess_pop_call_frame()
  ret double 0.000000
throw.uncaught.0:
  call void @jayess_pop_call_frame()
  ret double 0.000000
}

; native entry wrapper for main (2:1)
define i32 @main(i32 %argc, ptr %argv) {
entry:
  call void @jayess_init_globals()
  %result = call double @jayess_user_main()
  call void @jayess_run_microtasks()
  call void @jayess_runtime_shutdown()
  %thrown = call i1 @jayess_has_exception()
  br i1 %thrown, label %uncaught, label %exit.ok
uncaught:
  call void @jayess_report_uncaught_exception()
  ret i32 1
exit.ok:
  %exit = fptosi double %result to i32
  ret i32 %exit
}
`
	if got != want {
		t.Fatalf("unexpected IR snapshot\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestGenerateEmitsFunctionSourceLocationComments(t *testing.T) {
	module := &ir.Module{
		Functions: []ir.Function{
			{
				Name:   "helper",
				Line:   7,
				Column: 3,
				Body:   []ir.Statement{&ir.ReturnStatement{Value: &ir.UndefinedLiteral{}}},
			},
			{
				Name:   "main",
				Line:   11,
				Column: 1,
				Body:   []ir.Statement{&ir.ReturnStatement{Value: &ir.NumberLiteral{Value: 0}}},
			},
		},
	}

	out, err := NewLLVMIRGenerator().Generate(module, "x86_64-unknown-linux-gnu")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	text := string(out)
	for _, fragment := range []string{
		"; source function helper at 7:3",
		"; debug frame helper (7:3)",
		"; source function main at 11:1",
		"; debug frame main (11:1)",
		"; lowered symbol @jayess_user_main",
		"; native entry wrapper for main (11:1)",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected emitted IR to contain %q, got:\n%s", fragment, text)
		}
	}
}
