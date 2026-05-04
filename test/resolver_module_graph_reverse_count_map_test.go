package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsTransitiveDependentCountMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	expected := map[string]int{
		"app.js":    1,
		"config.js": 1,
		"main.js":   0,
		"model.js":  3,
		"worker.js": 0,
	}
	if counts := graph.TransitiveDependentCountMap(); !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected transitive dependent count map %#v, got %#v", expected, counts)
	}
}

func TestResolverModuleGraphTransitiveDependentCountMapSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	counts := graph.TransitiveDependentCountMap()
	expected := map[string]int{
		"a.js":    2,
		"b.js":    2,
		"main.js": 0,
	}
	if !reflect.DeepEqual(counts, expected) {
		t.Fatalf("expected transitive dependent count map %#v, got %#v", expected, counts)
	}
}

func TestResolverModuleGraphTransitiveDependentCountMapForEmptyGraphIsNil(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	if counts := graph.TransitiveDependentCountMap(); counts != nil {
		t.Fatalf("expected nil count map for empty graph, got %#v", counts)
	}
}
