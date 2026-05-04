package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependencySetForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)
	graph.AddModule("unused.js", nil)

	set, err := graph.TransitiveDependencySetFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("TransitiveDependencySetFor returned error: %v", err)
	}
	expected := map[string]bool{
		"shared.js":     true,
		"worker_dep.js": true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected transitive dependency set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphTransitiveDependencySetForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	set, err := graph.TransitiveDependencySetFor(nil)
	if err != nil {
		t.Fatalf("TransitiveDependencySetFor returned error: %v", err)
	}
	if set != nil {
		t.Fatalf("expected nil dependency set for no entries, got %#v", set)
	}
}

func TestResolverModuleGraphTransitiveDependencySetForEntriesDetectsCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencySetFor([]string{"main.js", "a.js"})
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
