package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsTransitiveDependentSetMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	expected := map[string]map[string]bool{
		"app.js":    {"main.js": true},
		"config.js": {"main.js": true},
		"main.js":   nil,
		"model.js":  {"app.js": true, "main.js": true, "worker.js": true},
		"worker.js": nil,
	}
	if sets := graph.TransitiveDependentSetMap(); !reflect.DeepEqual(sets, expected) {
		t.Fatalf("expected transitive dependent set map %#v, got %#v", expected, sets)
	}
}

func TestResolverModuleGraphTransitiveDependentSetMapSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	sets := graph.TransitiveDependentSetMap()
	expected := map[string]map[string]bool{
		"a.js":    {"b.js": true, "main.js": true},
		"b.js":    {"a.js": true, "main.js": true},
		"main.js": nil,
	}
	if !reflect.DeepEqual(sets, expected) {
		t.Fatalf("expected transitive dependent set map %#v, got %#v", expected, sets)
	}
}

func TestResolverModuleGraphTransitiveDependentSetMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	if sets := graph.TransitiveDependentSetMap(); sets != nil {
		t.Fatalf("expected nil set map for empty graph, got %#v", sets)
	}
}
