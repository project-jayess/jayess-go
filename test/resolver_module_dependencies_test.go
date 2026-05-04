package test

import (
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverResolvesASTModuleDependenciesInSourceOrder(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":                           ``,
		"src/setup.js":                          `export const value = 1;`,
		"src/math.js":                           `export const add = 1;`,
		"node_modules/@scope/tools/index.js":    `export const tool = 1;`,
		"node_modules/@scope/more/package.json": `{"jayess":"src/index.js"}`,
		"node_modules/@scope/more/src/index.js": `export const more = 1;`,
	})
	program := parseProgram(t, `
		import "./setup.js";
		import { add } from "./math";
		export * as tools from "@scope/tools";
		export { more } from "@scope/more";
	`)

	dependencies, err := resolver.ResolveModuleDependencies(filepath.Join(root, "src", "main.js"), program)
	if err != nil {
		t.Fatalf("ResolveModuleDependencies returned error: %v", err)
	}
	if len(dependencies) != 4 {
		t.Fatalf("expected four dependencies, got %#v", dependencies)
	}
	requireResolvedDependency(t, dependencies[0], "./setup.js", filepath.Join("src", "setup.js"), false, true)
	requireResolvedDependency(t, dependencies[1], "./math", filepath.Join("src", "math.js"), false, false)
	requireResolvedDependency(t, dependencies[2], "@scope/tools", filepath.Join("node_modules", "@scope", "tools", "index.js"), true, false)
	requireResolvedDependency(t, dependencies[3], "@scope/more", filepath.Join("node_modules", "@scope", "more", "src", "index.js"), true, false)
}

func TestResolverModuleDependenciesWrapsResolutionErrors(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})
	program := parseProgram(t, `import { missing } from "missing";`)

	_, err := resolver.ResolveModuleDependencies(filepath.Join(root, "src", "main.js"), program)
	if err == nil {
		t.Fatalf("expected dependency resolution error")
	}
	if !strings.Contains(err.Error(), `resolve module dependency "missing"`) {
		t.Fatalf("expected dependency source in diagnostic, got %v", err)
	}
}

func TestResolverResolvesCompactASTModuleDependencies(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":  ``,
		"src/setup.js": `export const value = 1;`,
		"src/math.js":  `export const add = 1;`,
	})
	program := parseProgram(t, `
		import "./setup.js";
		import { add } from "./math";
		export * from "./setup.js";
		import "./math.js";
	`)

	dependencies, err := resolver.ResolveCompactModuleDependencies(filepath.Join(root, "src", "main.js"), program)
	if err != nil {
		t.Fatalf("ResolveCompactModuleDependencies returned error: %v", err)
	}
	if len(dependencies) != 2 {
		t.Fatalf("expected two compact dependencies, got %#v", dependencies)
	}
	requireResolvedDependency(t, dependencies[0], "./setup.js", filepath.Join("src", "setup.js"), true, true)
	requireResolvedDependency(t, dependencies[1], "./math", filepath.Join("src", "math.js"), false, true)
}

func TestResolverCompactsResolvedModuleDependenciesByPath(t *testing.T) {
	dependencies := []resolver.ResolvedModuleDependency{
		{Source: "./setup", Path: "setup.js", SideEffect: true},
		{Source: "./math", Path: "math.js"},
		{Source: "./setup.js", Path: "setup.js", ReExport: true},
		{Source: "./math.js", Path: "math.js", SideEffect: true},
	}

	compacted := resolver.CompactResolvedModuleDependencies(dependencies)
	if len(compacted) != 2 {
		t.Fatalf("expected two compact resolved dependencies, got %#v", compacted)
	}
	requireResolvedDependency(t, compacted[0], "./setup", "setup.js", true, true)
	requireResolvedDependency(t, compacted[1], "./math", "math.js", false, true)
}

func TestResolverModuleDependenciesHandlesProgramWithoutDependencies(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})
	program := parseProgram(t, `const local = 1; export { local };`)

	dependencies, err := resolver.ResolveModuleDependencies(filepath.Join(root, "src", "main.js"), program)
	if err != nil {
		t.Fatalf("ResolveModuleDependencies returned error: %v", err)
	}
	if dependencies != nil {
		t.Fatalf("expected nil dependencies, got %#v", dependencies)
	}
}

func requireResolvedDependency(t *testing.T, dependency resolver.ResolvedModuleDependency, source string, suffix string, reExport bool, sideEffect bool) {
	t.Helper()
	if dependency.Source != source {
		t.Fatalf("expected source %q, got %#v", source, dependency)
	}
	requireResolvedSuffix(t, dependency.Path, suffix)
	if dependency.ReExport != reExport || dependency.SideEffect != sideEffect {
		t.Fatalf("unexpected dependency flags: %#v", dependency)
	}
}
