package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependentSetForModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	set := graph.TransitiveDependentSetFor([]string{"model.js", "config.js"})
	expected := map[string]bool{
		"app.js":    true,
		"main.js":   true,
		"worker.js": true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected transitive dependent set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphTransitiveDependentSetForNoModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	if set := graph.TransitiveDependentSetFor(nil); set != nil {
		t.Fatalf("expected nil dependent set for no modules, got %#v", set)
	}
}

func TestResolverModuleGraphTransitiveDependentSetForModulesSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	set := graph.TransitiveDependentSetFor([]string{"a.js", "b.js"})
	expected := map[string]bool{
		"a.js":    true,
		"b.js":    true,
		"main.js": true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected transitive dependent set %#v, got %#v", expected, set)
	}
}
