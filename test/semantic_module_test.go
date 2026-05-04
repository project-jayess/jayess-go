package test

import "testing"

func TestSemanticDeclaresNamedImports(t *testing.T) {
	err := analyzeSource(t, `
		import { add, twice as double } from "./math.js";
		const value = add(1, double(2));
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticDeclaresDefaultNamedImportSpecifier(t *testing.T) {
	err := analyzeSource(t, `
		import { default as main } from "./main.js";
		main();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticDeclaresStringNamedImportSpecifierAlias(t *testing.T) {
	err := analyzeSource(t, `
		import { "kebab-name" as kebabName } from "./main.js";
		kebabName();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticDeclaresKeywordNamedImportSpecifierAlias(t *testing.T) {
	err := analyzeSource(t, `
		import { class as className } from "./main.js";
		className();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsDuplicateImportLocal(t *testing.T) {
	err := analyzeSource(t, `
		const add = 1;
		import { add } from "./math.js";
	`)
	requireSemanticError(t, err, "duplicate declaration add")
}

func TestSemanticAllowsParentDirectoryImportSource(t *testing.T) {
	err := analyzeSource(t, `
		import { add } from "../lib/math.js";
		add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsSideEffectRelativeImport(t *testing.T) {
	err := analyzeSource(t, `import "./setup.js";`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsBarePackageImportSource(t *testing.T) {
	err := analyzeSource(t, `
		import { add } from "math";
		add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsPackageSubpathImportSource(t *testing.T) {
	err := analyzeSource(t, `
		import { add } from "math/utils";
		add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsPackageNameWithDotImportSource(t *testing.T) {
	err := analyzeSource(t, `
		import { add } from "math.tools/utils.js";
		add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsScopedPackageImportSource(t *testing.T) {
	err := analyzeSource(t, `
		import { add } from "@scope/math";
		add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsScopedPackageSubpathImportSource(t *testing.T) {
	err := analyzeSource(t, `
		import { add } from "@scope/math/utils";
		add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsAbsoluteImportSource(t *testing.T) {
	err := analyzeSource(t, `import { add } from "/math.js";`)
	requireSemanticError(t, err, `unsupported module source "/math.js"`)
}

func TestSemanticAnalyzesExportedDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		export function add(a, b) {
			return a + b;
		}
		const value = add(1, 2);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticValidatesNamedExportList(t *testing.T) {
	err := analyzeSource(t, `
		const add = 1;
		export { add };
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticValidatesDefaultNamedExportSpecifier(t *testing.T) {
	err := analyzeSource(t, `
		const add = 1;
		export { add as default };
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticValidatesStringNamedExportSpecifier(t *testing.T) {
	err := analyzeSource(t, `
		const add = 1;
		export { add as "kebab-name" };
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticValidatesKeywordNamedExportSpecifier(t *testing.T) {
	err := analyzeSource(t, `
		const add = 1;
		export { add as class };
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownNamedExport(t *testing.T) {
	err := analyzeSource(t, `export { missing };`)
	requireSemanticError(t, err, "export of missing before declaration")
}
