package test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphAddsParsedProgramDependencies(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":  ``,
		"src/setup.js": `export const value = 1;`,
		"src/app.js":   `export const app = 1;`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `
		import "./setup.js";
		import { app } from "./app";
	`)
	graph := resolver.NewModuleGraph()

	dependencies, err := graph.AddProgramModule(mainPath, program)
	if err != nil {
		t.Fatalf("AddProgramModule returned error: %v", err)
	}
	if len(dependencies) != 2 {
		t.Fatalf("expected two dependencies, got %#v", dependencies)
	}
	order, err := graph.InitializationOrder(mainPath)
	if err != nil {
		t.Fatalf("InitializationOrder returned error: %v", err)
	}
	expected := []string{dependencies[0].Path, dependencies[1].Path, mainPath}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphProgramDependencyResolutionErrorIncludesSource(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import { missing } from "missing";`)
	graph := resolver.NewModuleGraph()

	_, err := graph.AddProgramModule(mainPath, program)
	if err == nil {
		t.Fatalf("expected dependency resolution error")
	}
	if !strings.Contains(err.Error(), `resolve module dependency "missing"`) {
		t.Fatalf("expected dependency source in diagnostic, got %v", err)
	}
}

func TestResolverModuleGraphAddsCompactParsedProgramDependencies(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":  ``,
		"src/setup.js": `export const value = 1;`,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `
		import "./setup.js";
		export * from "./setup.js";
	`)
	graph := resolver.NewModuleGraph()

	dependencies, err := graph.AddCompactProgramModule(mainPath, program)
	if err != nil {
		t.Fatalf("AddCompactProgramModule returned error: %v", err)
	}
	if len(dependencies) != 1 {
		t.Fatalf("expected one compact dependency, got %#v", dependencies)
	}
	if !dependencies[0].ReExport || !dependencies[0].SideEffect {
		t.Fatalf("expected merged dependency flags, got %#v", dependencies[0])
	}
	expectedImports := []string{dependencies[0].Path}
	if imports := graph.Dependencies(mainPath); !reflect.DeepEqual(imports, expectedImports) {
		t.Fatalf("expected compact graph imports %#v, got %#v", expectedImports, imports)
	}
}

func TestResolverModuleGraphAddsProgramWithoutDependencies(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `const value = 1; export { value };`)
	graph := resolver.NewModuleGraph()

	dependencies, err := graph.AddProgramModule(mainPath, program)
	if err != nil {
		t.Fatalf("AddProgramModule returned error: %v", err)
	}
	if dependencies != nil {
		t.Fatalf("expected nil dependencies, got %#v", dependencies)
	}
	order, err := graph.InitializationOrder(mainPath)
	if err != nil {
		t.Fatalf("InitializationOrder returned error: %v", err)
	}
	expected := []string{mainPath}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}
