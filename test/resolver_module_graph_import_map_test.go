package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsImportMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})

	expected := map[string][]string{
		"config.js": nil,
		"main.js":   []string{"shared.js", "config.js"},
		"shared.js": nil,
	}
	if imports := graph.ImportMap(); !reflect.DeepEqual(imports, expected) {
		t.Fatalf("expected import map %#v, got %#v", expected, imports)
	}
}

func TestResolverModuleGraphImportMapDoesNotShareImportSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})

	imports := graph.ImportMap()
	imports["main.js"][0] = "changed.js"

	expected := []string{"shared.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected original dependencies %#v, got %#v", expected, dependencies)
	}
}
