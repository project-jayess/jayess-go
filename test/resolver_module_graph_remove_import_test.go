package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphRemovesSingleImportEdge(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js", "shared.js"})

	if !graph.RemoveImport("main.js", "shared.js") {
		t.Fatalf("expected shared.js import to be removed")
	}
	expected := []string{"config.js", "shared.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphRemoveImportReportsMissingEdge(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})

	if graph.RemoveImport("main.js", "config.js") {
		t.Fatalf("did not expect missing import edge to be removed")
	}
	if graph.RemoveImport("missing.js", "shared.js") {
		t.Fatalf("did not expect missing module import edge to be removed")
	}
}

func TestResolverModuleGraphRemovesAllMatchingImportEdges(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js", "shared.js"})

	if removed := graph.RemoveAllImports("main.js", "shared.js"); removed != 2 {
		t.Fatalf("expected 2 removed import edges, got %d", removed)
	}
	expected := []string{"config.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphRemoveAllImportsReportsMissingEdges(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})

	if removed := graph.RemoveAllImports("main.js", "config.js"); removed != 0 {
		t.Fatalf("expected no removed import edges, got %d", removed)
	}
	if removed := graph.RemoveAllImports("missing.js", "shared.js"); removed != 0 {
		t.Fatalf("expected no removed import edges for missing module, got %d", removed)
	}
}
