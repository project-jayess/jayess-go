package test

import (
	"strings"
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMBackendLowersModuleImportsThroughRuntimeBindings(t *testing.T) {
	program := parseProgram(t, `
		import "./setup.js";
		import main, { add as sum } from "./math.js";
		import * as nativeLib from "./native.bind.js";
		return sum;
	`)

	fn, declarations, globals, err := llvmbackend.LowerRuntimeProgramFunction("main", program)
	if err != nil {
		t.Fatalf("lower module imports: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "module-imports",
		Functions:    []llvmbackend.Function{fn},
		Declarations: declarations,
		Globals:      globals,
	})

	requireIRContains(t, ir,
		"call void @jayess_module_initialize",
		"call %jayess.value @jayess_module_import_default",
		"call %jayess.value @jayess_module_import_binding",
		"call %jayess.value @jayess_native_binding_wrapper",
		"ret %jayess.value",
	)
}

func TestLLVMBackendLowersModuleExportsThroughRuntimeBindings(t *testing.T) {
	program := parseProgram(t, `
		const answer = 42;
		export { answer as default };
		export { add as sum } from "./math.js";
		export * from "./shared.js";
		export * as names from "./names.js";
		export default answer;
	`)

	fn, declarations, globals, err := llvmbackend.LowerRuntimeProgramFunction("main", program)
	if err != nil {
		t.Fatalf("lower module exports: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(llvmbackend.Module{
		Name:         "module-exports",
		Functions:    []llvmbackend.Function{fn},
		Declarations: declarations,
		Globals:      globals,
	})

	requireIRContains(t, ir,
		"call void @jayess_module_export_binding",
		"call void @jayess_module_reexport_binding",
		"call void @jayess_module_reexport_all",
		"call void @jayess_module_reexport_namespace",
		"call void @jayess_module_export_value",
	)
}

func requireIRContains(t *testing.T, ir string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(ir, needle) {
			t.Fatalf("expected IR to contain %q:\n%s", needle, ir)
		}
	}
}
