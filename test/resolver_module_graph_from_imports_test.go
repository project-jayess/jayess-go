package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphBuildsFromImportMap(t *testing.T) {
	graph := resolver.NewModuleGraphFromImports(map[string][]string{
		"main.js":   []string{"shared.js", "config.js"},
		"worker.js": []string{"shared.js"},
	})

	expected := []string{"config.js", "main.js", "shared.js", "worker.js"}
	if modules := graph.Modules(); !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphFromImportMapCopiesImportSlices(t *testing.T) {
	imports := map[string][]string{
		"main.js": []string{"shared.js"},
	}
	graph := resolver.NewModuleGraphFromImports(imports)
	imports["main.js"][0] = "changed.js"

	expected := []string{"shared.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected copied dependencies %#v, got %#v", expected, dependencies)
	}
}
