package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsTransitiveDependencyCountMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	counts, err := graph.TransitiveDependencyCountMap()
	if err != nil {
		t.Fatalf("TransitiveDependencyCountMap returned error: %v", err)
	}
	expected := map[string]int{
		"app.js":    1,
		"config.js": 0,
		"main.js":   3,
		"model.js":  0,
		"worker.js": 1,
	}
	if !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected transitive dependency count map %#v, got %#v", expected, counts)
	}
}

func TestResolverModuleGraphTransitiveDependencyCountMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencyCountMap()
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphTransitiveDependencyCountMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	counts, err := graph.TransitiveDependencyCountMap()
	if err != nil {
		t.Fatalf("TransitiveDependencyCountMap returned error: %v", err)
	}
	if counts != nil {
		t.Fatalf("expected nil count map for empty graph, got %#v", counts)
	}
}
