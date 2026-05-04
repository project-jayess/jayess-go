package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependenciesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)
	graph.AddModule("unused.js", nil)

	dependencies, err := graph.TransitiveDependenciesFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("TransitiveDependenciesFor returned error: %v", err)
	}
	expected := []string{"shared.js", "worker_dep.js"}
	if !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected transitive dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphTransitiveDependenciesForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	dependencies, err := graph.TransitiveDependenciesFor(nil)
	if err != nil {
		t.Fatalf("TransitiveDependenciesFor returned error: %v", err)
	}
	if dependencies != nil {
		t.Fatalf("expected nil dependencies for no entries, got %#v", dependencies)
	}
}

func TestResolverModuleGraphTransitiveDependenciesForEntriesDetectsCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependenciesFor([]string{"main.js", "a.js"})
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
