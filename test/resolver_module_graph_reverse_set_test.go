package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependentSet(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	set := graph.TransitiveDependentSet("model.js")
	expected := map[string]bool{
		"app.js":    true,
		"main.js":   true,
		"worker.js": true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected transitive dependent set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphTransitiveDependentSetForLeafIsNil(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	if set := graph.TransitiveDependentSet("main.js"); set != nil {
		t.Fatalf("expected nil dependent set for leaf module, got %#v", set)
	}
}

func TestResolverModuleGraphTransitiveDependentSetSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	set := graph.TransitiveDependentSet("a.js")
	expected := map[string]bool{
		"b.js":    true,
		"main.js": true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected transitive dependent set %#v, got %#v", expected, set)
	}
}
