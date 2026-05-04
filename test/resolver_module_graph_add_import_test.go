package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphAddsSingleImportEdge(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddImport("main.js", "shared.js")
	graph.AddImport("main.js", "config.js")

	expected := []string{"shared.js", "config.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphAddImportKeepsImportedModule(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddImport("main.js", "shared.js")

	if !graph.HasModule("shared.js") {
		t.Fatalf("expected imported module to be present")
	}
	if !graph.IsLeafModule("shared.js") {
		t.Fatalf("expected imported module to be a leaf")
	}
}
