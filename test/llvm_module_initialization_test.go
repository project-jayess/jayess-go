package test

import (
	"strings"
	"testing"

	"jayess-go/llvmbackend"
	"jayess-go/resolver"
)

func TestLLVMBackendPlansModuleInitializationFromResolverOrder(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"math.js", "names.js"})
	graph.AddModule("math.js", []string{"constants.js"})
	graph.AddModule("names.js", nil)
	graph.AddModule("constants.js", nil)

	plan, err := llvmbackend.PlanModuleInitialization(graph, []string{"main.js"})
	if err != nil {
		t.Fatalf("plan module initialization: %v", err)
	}

	modules := initializerModules(plan)
	expected := []string{"constants.js", "math.js", "names.js", "main.js"}
	if strings.Join(modules, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected modules %#v, got %#v", expected, modules)
	}
}

func TestLLVMBackendLowersModuleInitializationCallsBeforeReturn(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"dep.js"})
	graph.AddModule("dep.js", nil)
	plan, err := llvmbackend.PlanModuleInitialization(graph, []string{"main.js"})
	if err != nil {
		t.Fatalf("plan module initialization: %v", err)
	}

	module := llvmbackend.LowerJayessProgram(llvmbackend.JayessProgram{
		Name:                 "app",
		ReturnCode:           0,
		ModuleInitialization: plan,
	})
	ir := llvmbackend.EmitLLVMIR(module)

	depCall := strings.Index(ir, "call void @__jayess_init_module_dep_js()")
	mainCall := strings.Index(ir, "call void @__jayess_init_module_main_js()")
	ret := strings.Index(ir, "ret i32 0")
	if depCall < 0 || mainCall < 0 || ret < 0 {
		t.Fatalf("expected module init calls and return in IR:\n%s", ir)
	}
	if !(depCall < mainCall && mainCall < ret) {
		t.Fatalf("expected dependency-first init calls before return:\n%s", ir)
	}
	if !strings.Contains(ir, "declare void () @__jayess_init_module_dep_js") {
		t.Fatalf("expected module init declaration in IR:\n%s", ir)
	}
}

func initializerModules(plan llvmbackend.ModuleInitializationPlan) []string {
	modules := make([]string, 0, len(plan.Initializers))
	for _, initializer := range plan.Initializers {
		modules = append(modules, initializer.Module)
	}
	return modules
}
