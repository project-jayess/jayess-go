package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsTransitiveDependencySetMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	sets, err := graph.TransitiveDependencySetMap()
	if err != nil {
		t.Fatalf("TransitiveDependencySetMap returned error: %v", err)
	}
	expected := map[string]map[string]bool{
		"app.js":    {"model.js": true},
		"config.js": nil,
		"main.js":   {"app.js": true, "config.js": true, "model.js": true},
		"model.js":  nil,
		"worker.js": {"model.js": true},
	}
	if !reflect.DeepEqual(sets, expected) {
		t.Fatalf("expected transitive dependency set map %#v, got %#v", expected, sets)
	}
}

func TestResolverModuleGraphTransitiveDependencySetMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencySetMap()
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphTransitiveDependencySetMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	sets, err := graph.TransitiveDependencySetMap()
	if err != nil {
		t.Fatalf("TransitiveDependencySetMap returned error: %v", err)
	}
	if sets != nil {
		t.Fatalf("expected nil set map for empty graph, got %#v", sets)
	}
}
