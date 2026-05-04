package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsImportEdgesDeterministically(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})

	expected := []resolver.ModuleImportEdge{
		{From: "main.js", To: "config.js"},
		{From: "main.js", To: "shared.js"},
		{From: "worker.js", To: "shared.js"},
	}
	if edges := graph.ImportEdges(); !reflect.DeepEqual(edges, expected) {
		t.Fatalf("expected import edges %#v, got %#v", expected, edges)
	}
}

func TestResolverModuleGraphImportEdgesForEmptyGraphAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()

	if edges := graph.ImportEdges(); len(edges) != 0 {
		t.Fatalf("expected no import edges, got %#v", edges)
	}
}
