package test

import (
	"testing"

	"jayess-go/lowering"
	jayessruntime "jayess-go/runtime"
)

func TestLoweringModuleBindingPlanCollectsImportsAndExports(t *testing.T) {
	program := parseProgram(t, `
		import main, { add as sum } from "./math.js";
		import * as names from "./names.js";
		import "./setup.js";
		export const local = sum;
		export { local as value };
		export { default as otherMain } from "./other.js";
		export default function run() { return local; }
		export * as tools from "./tools.js";
	`)

	plan := lowering.LowerModuleBindingPlan("app.js", program)
	if plan.Module != "app.js" {
		t.Fatalf("expected app.js module, got %q", plan.Module)
	}
	if len(plan.Imports) != 4 {
		t.Fatalf("expected four imports, got %#v", plan.Imports)
	}
	if plan.Imports[0].Imported != "default" || plan.Imports[0].Local != "main" || !plan.Imports[0].Default {
		t.Fatalf("unexpected default import: %#v", plan.Imports[0])
	}
	if plan.Imports[1].Imported != "add" || plan.Imports[1].Local != "sum" {
		t.Fatalf("unexpected named import: %#v", plan.Imports[1])
	}
	if !plan.Imports[2].Namespace || plan.Imports[2].Local != "names" {
		t.Fatalf("unexpected namespace import: %#v", plan.Imports[2])
	}
	if !plan.Imports[3].SideEffect || plan.Imports[3].Source != "./setup.js" {
		t.Fatalf("unexpected side-effect import: %#v", plan.Imports[3])
	}

	if len(plan.Exports) != 5 {
		t.Fatalf("expected five exports, got %#v", plan.Exports)
	}
	if plan.Exports[0].Local != "local" || plan.Exports[0].Exported != "local" {
		t.Fatalf("unexpected declaration export: %#v", plan.Exports[0])
	}
	if plan.Exports[1].Local != "local" || plan.Exports[1].Exported != "value" {
		t.Fatalf("unexpected local export specifier: %#v", plan.Exports[1])
	}
	if plan.Exports[2].Source != "./other.js" || plan.Exports[2].Local != "default" || plan.Exports[2].Exported != "otherMain" {
		t.Fatalf("unexpected re-export: %#v", plan.Exports[2])
	}
	if !plan.Exports[3].Default || plan.Exports[3].Local != "run" || plan.Exports[3].Exported != "default" {
		t.Fatalf("unexpected default export: %#v", plan.Exports[3])
	}
	if plan.Exports[4].Source != "./tools.js" || plan.Exports[4].Namespace != "tools" || plan.Exports[4].Exported != "tools" {
		t.Fatalf("unexpected namespace re-export: %#v", plan.Exports[4])
	}
}

func TestRuntimeModuleBindingsExposeImportedAndExportedValues(t *testing.T) {
	source := jayessruntime.NewModuleBindings()
	exported := source.DefineLocal("count", jayessruntime.NewNumber(1))
	if !source.BindExport("count", exported) {
		t.Fatal("expected source export binding")
	}

	consumer := jayessruntime.NewModuleBindings()
	if !consumer.BindImport("count", exported) {
		t.Fatal("expected consumer import binding")
	}
	if !consumer.BindExport("seen", exported) {
		t.Fatal("expected consumer export binding")
	}

	exported.Set(jayessruntime.NewNumber(2))
	imported, ok := consumer.Import("count")
	if !ok {
		t.Fatal("expected imported binding")
	}
	importedValue, ok := imported.Value()
	if !ok || importedValue.Number() != 2 {
		t.Fatalf("expected live imported value 2, got %#v ok=%v", importedValue, ok)
	}

	namespace := consumer.NamespaceObject()
	seen, ok := namespace.GetNamedProperty("seen")
	if !ok || seen.Number() != 2 {
		t.Fatalf("expected namespace exported value 2, got %#v ok=%v", seen, ok)
	}
}
