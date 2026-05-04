package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingProgramDetectsDefaultBindExport(t *testing.T) {
	program := parseProgram(t, `
		import { bind } from "ffi";

		const f = () => {};
		export const add = f;

		export default bind({
			sources: ["./src/mylib.c"],
			includeDirs: ["./include"],
			cflags: [],
			ldflags: [],
			exports: {
				add: { symbol: "mylib_add", type: "function" }
			}
		});
	`)

	match := binding.BindingExport(program)
	if !match.Found {
		t.Fatalf("expected binding default export")
	}
	if match.ImportName != "bind" {
		t.Fatalf("expected bind import name, got %q", match.ImportName)
	}
	if kind := binding.ClassifyModule("./native/math.js", program); kind != binding.NativeBindingModule {
		t.Fatalf("expected native binding module, got %s", kind)
	}
}

func TestBindingProgramDetectsAliasedBindExport(t *testing.T) {
	program := parseProgram(t, `
		import { bind as nativeBind } from "ffi";
		export default nativeBind({ exports: { add: { symbol: "mylib_add", type: "function" } } });
	`)

	match := binding.BindingExport(program)
	if !match.Found || match.ImportName != "nativeBind" {
		t.Fatalf("expected aliased binding export, got %#v", match)
	}
}

func TestBindingProgramRequiresFfiBindImport(t *testing.T) {
	program := parseProgram(t, `
		function bind(value) {
			return value;
		}
		export default bind({ exports: { add: { symbol: "mylib_add", type: "function" } } });
	`)

	if binding.IsBindingProgram(program) {
		t.Fatalf("local bind function must not classify a module as native binding")
	}
	if kind := binding.ClassifyModule("./native/math.js", program); kind != binding.SourceModule {
		t.Fatalf("expected source module, got %s", kind)
	}
}
