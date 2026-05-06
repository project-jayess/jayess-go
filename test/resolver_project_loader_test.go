package test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverLoadProjectParsesReachableModules(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.js")
	shared := filepath.Join(root, "shared.js")
	leaf := filepath.Join(root, "leaf.js")
	writeFile(t, entry, `
import { value } from "./shared.js";
export { leaf } from "./leaf.js";
const main = value;
`)
	writeFile(t, shared, `
import "./leaf.js";
export const value = 1;
`)
	writeFile(t, leaf, `export const leaf = 2;`)

	project, err := resolver.LoadProject(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", project.Diagnostics)
	}
	requireProjectModules(t, project, []string{entry, leaf, shared})
	requireGraphOrder(t, project, []string{leaf, shared, entry})
}

func TestResolverLoadProjectReportsImportContext(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.js")
	writeFile(t, entry, "\nimport { missing } from \"./missing.js\";\nconst value = 1;\n")

	project, err := resolver.LoadProject(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", project.Diagnostics)
	}
	diagnostic := project.Diagnostics[0]
	if diagnostic.Path != mustAbs(t, entry) || diagnostic.Line != 2 || diagnostic.Column != 1 {
		t.Fatalf("expected import source position, got %#v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, `resolve module dependency "./missing.js"`) {
		t.Fatalf("expected dependency context, got %q", diagnostic.Message)
	}
}

func TestResolverLoadProjectReportsParseContext(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.js")
	child := filepath.Join(root, "child.js")
	writeFile(t, entry, `import "./child.js";`)
	writeFile(t, child, `let = ;`)

	project, err := resolver.LoadProject(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", project.Diagnostics)
	}
	diagnostic := project.Diagnostics[0]
	if diagnostic.Path != mustAbs(t, child) || diagnostic.Line == 0 {
		t.Fatalf("expected child parse position, got %#v", diagnostic)
	}
}

func TestResolverLoadProjectHandlesCompilerScaleFixtures(t *testing.T) {
	root := t.TempDir()
	const moduleCount = 64
	for i := 0; i < moduleCount; i++ {
		path := filepath.Join(root, moduleNameForIndex(i))
		next := ""
		if i+1 < moduleCount {
			next = `import "./` + moduleNameForIndex(i+1) + `";`
		}
		writeFile(t, path, next+"\nexport const value = "+string(rune('0'+i%10))+";")
	}
	entry := filepath.Join(root, moduleNameForIndex(0))
	project, err := resolver.LoadProject(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", project.Diagnostics)
	}
	if len(project.Modules) != moduleCount {
		t.Fatalf("expected %d modules, got %d", moduleCount, len(project.Modules))
	}
	order, err := project.Graph.InitializationOrder(mustAbs(t, entry))
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != moduleCount {
		t.Fatalf("expected %d ordered modules, got %d", moduleCount, len(order))
	}
	if order[0] != mustAbs(t, filepath.Join(root, moduleNameForIndex(moduleCount-1))) {
		t.Fatalf("expected deepest module first, got %q", order[0])
	}
}

func requireProjectModules(t *testing.T, project resolver.Project, paths []string) {
	t.Helper()
	if len(project.Modules) != len(paths) {
		t.Fatalf("expected %d modules, got %d", len(paths), len(project.Modules))
	}
	seen := map[string]bool{}
	for _, module := range project.Modules {
		seen[module.Path] = true
	}
	for _, path := range paths {
		if !seen[mustAbs(t, path)] {
			t.Fatalf("expected module %q in %#v", mustAbs(t, path), project.Modules)
		}
	}
}

func requireGraphOrder(t *testing.T, project resolver.Project, paths []string) {
	t.Helper()
	order, err := project.Graph.InitializationOrder(project.Entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != len(paths) {
		t.Fatalf("expected %d ordered modules, got %d", len(paths), len(order))
	}
	for index, path := range paths {
		if order[index] != mustAbs(t, path) {
			t.Fatalf("order[%d]: expected %q, got %q", index, mustAbs(t, path), order[index])
		}
	}
}

func moduleNameForIndex(index int) string {
	return fmt.Sprintf("module_%02d.js", index)
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(absolute)
}
