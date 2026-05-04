package test

import (
	"path/filepath"
	"testing"

	"jayess-go/resolver"
)

func TestResolverBindingModulesExtractsImportedBindingManifest(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
		"src/native/math.js": `
			import { bind } from "ffi";
			const f = () => {};
			export const add = f;
			export default bind({
				sources: ["./math.c"],
				includeDirs: ["./include"],
				cflags: ["-DMATH=1"],
				ldflags: ["-lm"],
				exports: {
					add: { symbol: "mylib_add", type: "function" }
				}
			});
		`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import { add } from "./native/math.js";`)

	modules, diagnostics, err := resolver.ResolveBindingModules(mainPath, program)
	if err != nil {
		t.Fatalf("ResolveBindingModules returned error: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected clean binding module resolution, got %#v", diagnostics)
	}
	if len(modules) != 1 {
		t.Fatalf("expected one binding module, got %#v", modules)
	}
	requireResolvedSuffix(t, modules[0].Path, filepath.Join("src", "native", "math.js"))
	requireStringSlice(t, modules[0].Manifest.Sources, []string{"./math.c"})
	requireStringSlice(t, modules[0].Manifest.IncludeDirs, []string{"./include"})
	requireStringSlice(t, modules[0].Manifest.CFlags, []string{"-DMATH=1"})
	requireStringSlice(t, modules[0].Manifest.LDFlags, []string{"-lm"})
}

func TestResolverBindingBuildPlanUsesExtractedManifest(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
		"src/native/math.js": `
			import { bind } from "ffi";
			const f = () => {};
			export const add = f;
			export default bind({
				sources: ["./math.c"],
				includeDirs: ["./include"],
				ldflags: ["-lm"],
				exports: {
					add: { symbol: "mylib_add", type: "function" }
				}
			});
		`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import { add } from "./native/math.js";`)

	plan, err := resolver.ResolveBindingBuildPlan(mainPath, program, "linux", "./runtime")
	if err != nil {
		t.Fatalf("ResolveBindingBuildPlan returned error: %v", err)
	}
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one compile unit, got %#v", plan.CompileUnits)
	}
	if plan.CompileUnits[0].Source != "./math.c" {
		t.Fatalf("expected extracted source in compile unit, got %#v", plan.CompileUnits[0])
	}
	includeDir, err := filepath.Abs(filepath.Join(root, "src", "native", "include"))
	if err != nil {
		t.Fatalf("resolve expected include dir: %v", err)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{includeDir, "./runtime"})
	requireStringSlice(t, plan.LDFlags, []string{"-lm"})
}

func TestResolverBindingModulesReportsMissingExport(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
		"src/native/math.js": `
			import { bind } from "ffi";
			export default bind({
				sources: ["./math.c"],
				exports: {
					add: { symbol: "mylib_add", type: "function" }
				}
			});
		`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import { missing } from "./native/math.js";`)

	_, diagnostics, err := resolver.ResolveBindingModules(mainPath, program)
	if err != nil {
		t.Fatalf("ResolveBindingModules returned error: %v", err)
	}
	requireDiagnostic(t, diagnostics, "binding export missing was not declared")
}

func TestResolverBindingModulesRejectsUnsupportedBindingImportForm(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
		"src/native/math.js": `
			import { bind } from "ffi";
			export default bind({
				sources: ["./math.c"],
				exports: {
					add: { symbol: "mylib_add", type: "function" }
				}
			});
		`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import math from "./native/math.js";`)

	_, diagnostics, err := resolver.ResolveBindingModules(mainPath, program)
	if err != nil {
		t.Fatalf("ResolveBindingModules returned error: %v", err)
	}
	requireDiagnostic(t, diagnostics, "binding modules only support named imports")
}

func TestResolverBindingModulesIgnoresNormalSourceModulesAndStdlib(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
		"src/math.js": `export const add = 1;`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `
		import { add } from "./math.js";
		import { readFile } from "fs";
	`)

	modules, diagnostics, err := resolver.ResolveBindingModules(mainPath, program)
	if err != nil {
		t.Fatalf("ResolveBindingModules returned error: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
	if len(modules) != 0 {
		t.Fatalf("expected no binding modules, got %#v", modules)
	}
}
