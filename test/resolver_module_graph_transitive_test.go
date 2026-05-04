package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	dependencies, err := graph.TransitiveDependencies("main.js")
	if err != nil {
		t.Fatalf("TransitiveDependencies returned error: %v", err)
	}
	expected := []string{"config.js", "model.js", "app.js"}
	if !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected transitive dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphTransitiveDependenciesDetectsCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencies("main.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
	expected := []string{"a.js", "b.js", "a.js"}
	if !reflect.DeepEqual(cycleErr.Cycle, expected) {
		t.Fatalf("expected cycle %#v, got %#v", expected, cycleErr.Cycle)
	}
}

func TestResolverModuleGraphTransitiveDependenciesForLeafAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	dependencies, err := graph.TransitiveDependencies("main.js")
	if err != nil {
		t.Fatalf("TransitiveDependencies returned error: %v", err)
	}
	if len(dependencies) != 0 {
		t.Fatalf("expected no transitive dependencies, got %#v", dependencies)
	}
}
