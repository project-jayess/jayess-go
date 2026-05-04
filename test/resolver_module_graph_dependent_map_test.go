package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsDependentMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})

	expected := map[string][]string{
		"config.js": []string{"main.js"},
		"main.js":   nil,
		"shared.js": []string{"main.js", "worker.js"},
		"worker.js": nil,
	}
	if dependents := graph.DependentMap(); !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected dependent map %#v, got %#v", expected, dependents)
	}
}

func TestResolverModuleGraphDependentMapDoesNotShareSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})

	dependents := graph.DependentMap()
	dependents["shared.js"][0] = "changed.js"

	expected := []string{"main.js"}
	if actual := graph.Dependents("shared.js"); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected original dependents %#v, got %#v", expected, actual)
	}
}
