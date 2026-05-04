package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphRemovesModuleAndIncomingEdges(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js", "shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	if !graph.RemoveModule("shared.js") {
		t.Fatalf("expected shared.js to be removed")
	}
	if graph.HasModule("shared.js") {
		t.Fatalf("did not expect shared.js to remain in graph")
	}
	expected := []string{"config.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected main.js dependencies %#v, got %#v", expected, dependencies)
	}
	if dependencies := graph.Dependencies("worker.js"); len(dependencies) != 0 {
		t.Fatalf("expected worker.js dependencies to be empty, got %#v", dependencies)
	}
}

func TestResolverModuleGraphRemoveModuleReportsMissingModule(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	if graph.RemoveModule("missing.js") {
		t.Fatalf("did not expect missing module to be removed")
	}
	if !graph.HasModule("main.js") {
		t.Fatalf("expected existing module to remain")
	}
}
