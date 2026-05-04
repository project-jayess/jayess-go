package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphBuildsFromImportEdges(t *testing.T) {
	graph := resolver.NewModuleGraphFromEdges([]resolver.ModuleImportEdge{
		{From: "main.js", To: "shared.js"},
		{From: "main.js", To: "config.js"},
		{From: "worker.js", To: "shared.js"},
	})

	expected := []string{"shared.js", "config.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected main.js dependencies %#v, got %#v", expected, dependencies)
	}
	if !graph.HasModule("shared.js") {
		t.Fatalf("expected imported leaf module to be present")
	}
}

func TestResolverModuleGraphBuildsEmptyGraphFromNoImportEdges(t *testing.T) {
	graph := resolver.NewModuleGraphFromEdges(nil)

	if count := graph.ModuleCount(); count != 0 {
		t.Fatalf("expected no modules, got %d", count)
	}
}
