package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsTransitiveDependencyMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	dependencies, err := graph.TransitiveDependencyMap()
	if err != nil {
		t.Fatalf("TransitiveDependencyMap returned error: %v", err)
	}
	expected := map[string][]string{
		"app.js":    []string{"model.js"},
		"config.js": []string{},
		"main.js":   []string{"model.js", "app.js", "config.js"},
		"model.js":  []string{},
	}
	if !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected transitive dependency map %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphTransitiveDependencyMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencyMap()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphExportsTransitiveDependentMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	expected := map[string][]string{
		"app.js":    []string{"main.js"},
		"config.js": []string{"main.js"},
		"main.js":   nil,
		"model.js":  []string{"app.js", "main.js", "worker.js"},
		"worker.js": nil,
	}
	if dependents := graph.TransitiveDependentMap(); !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected transitive dependent map %#v, got %#v", expected, dependents)
	}
}

func TestResolverModuleGraphTransitiveDependentMapSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	dependents := graph.TransitiveDependentMap()
	expected := []string{"b.js", "main.js"}
	if !reflect.DeepEqual(dependents["a.js"], expected) {
		t.Fatalf("expected a.js transitive dependents %#v, got %#v", expected, dependents["a.js"])
	}
}
