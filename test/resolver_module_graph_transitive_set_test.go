package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependencySet(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	set, err := graph.TransitiveDependencySet("main.js")
	if err != nil {
		t.Fatalf("TransitiveDependencySet returned error: %v", err)
	}
	expected := map[string]bool{
		"app.js":    true,
		"config.js": true,
		"model.js":  true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected transitive dependency set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphTransitiveDependencySetForLeafIsNil(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	set, err := graph.TransitiveDependencySet("main.js")
	if err != nil {
		t.Fatalf("TransitiveDependencySet returned error: %v", err)
	}
	if set != nil {
		t.Fatalf("expected nil dependency set for leaf module, got %#v", set)
	}
}

func TestResolverModuleGraphTransitiveDependencySetDetectsCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencySet("main.js")
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
