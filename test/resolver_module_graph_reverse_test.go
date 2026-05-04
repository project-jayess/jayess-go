package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependents(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	dependents := graph.TransitiveDependents("model.js")
	expected := []string{"app.js", "main.js", "worker.js"}
	if !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected transitive dependents %#v, got %#v", expected, dependents)
	}
}

func TestResolverModuleGraphTransitiveDependentsSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	dependents := graph.TransitiveDependents("a.js")
	expected := []string{"b.js", "main.js"}
	if !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected transitive dependents %#v, got %#v", expected, dependents)
	}
}

func TestResolverModuleGraphTransitiveDependentsForMissingModuleAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"dep.js"})

	if dependents := graph.TransitiveDependents("missing.js"); len(dependents) != 0 {
		t.Fatalf("expected no transitive dependents, got %#v", dependents)
	}
}
