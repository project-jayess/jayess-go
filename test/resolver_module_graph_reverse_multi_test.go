package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependentsForModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	dependents := graph.TransitiveDependentsFor([]string{"model.js", "config.js"})
	expected := []string{"app.js", "main.js", "worker.js"}
	if !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected transitive dependents %#v, got %#v", expected, dependents)
	}
}

func TestResolverModuleGraphTransitiveDependentsForNoModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	if dependents := graph.TransitiveDependentsFor(nil); dependents != nil {
		t.Fatalf("expected nil dependents for no modules, got %#v", dependents)
	}
}

func TestResolverModuleGraphTransitiveDependentsForModulesSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	dependents := graph.TransitiveDependentsFor([]string{"a.js", "b.js"})
	expected := []string{"a.js", "b.js", "main.js"}
	if !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected transitive dependents %#v, got %#v", expected, dependents)
	}
}
