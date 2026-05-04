package test

import (
	"reflect"
	"testing"

	"jayess-go/ast"
)

func TestASTModuleDependenciesCollectsImportsAndReExportsInSourceOrder(t *testing.T) {
	program := parseProgram(t, `
		import "./setup.js";
		import { add } from "./math.js";
		const local = 1;
		export { add as sum } from "./more.js";
		export * as tools from "@scope/tools";
		export { local };
	`)

	dependencies := ast.ModuleDependencies(program)
	expected := []ast.ModuleDependency{
		{Source: "./setup.js", SideEffect: true},
		{Source: "./math.js"},
		{Source: "./more.js", ReExport: true},
		{Source: "@scope/tools", ReExport: true},
	}
	if !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected module dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestASTModuleDependenciesIgnoresLocalExports(t *testing.T) {
	program := parseProgram(t, `
		const local = 1;
		export { local };
		export default local;
	`)

	dependencies := ast.ModuleDependencies(program)
	if len(dependencies) != 0 {
		t.Fatalf("expected no module dependencies, got %#v", dependencies)
	}
}

func TestASTModuleDependenciesHandlesNilProgram(t *testing.T) {
	if dependencies := ast.ModuleDependencies(nil); dependencies != nil {
		t.Fatalf("expected nil dependencies for nil program, got %#v", dependencies)
	}
}

func TestASTCompactModuleDependenciesPreservesOrderAndMergesFlags(t *testing.T) {
	dependencies := []ast.ModuleDependency{
		{Source: "./setup.js", SideEffect: true},
		{Source: "./math.js"},
		{Source: "./setup.js", ReExport: true},
		{Source: "./math.js", SideEffect: true},
	}

	compacted := ast.CompactModuleDependencies(dependencies)
	expected := []ast.ModuleDependency{
		{Source: "./setup.js", ReExport: true, SideEffect: true},
		{Source: "./math.js", SideEffect: true},
	}
	if !reflect.DeepEqual(compacted, expected) {
		t.Fatalf("expected compacted dependencies %#v, got %#v", expected, compacted)
	}
}

func TestASTCompactModuleDependenciesHandlesEmptyInput(t *testing.T) {
	if dependencies := ast.CompactModuleDependencies(nil); dependencies != nil {
		t.Fatalf("expected nil dependencies for empty input, got %#v", dependencies)
	}
}
