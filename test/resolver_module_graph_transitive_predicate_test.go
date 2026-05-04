package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksTransitiveDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)

	depends, err := graph.TransitivelyDependsOn("main.js", "model.js")
	if err != nil {
		t.Fatalf("TransitivelyDependsOn returned error: %v", err)
	}
	if !depends {
		t.Fatalf("expected main.js to transitively depend on model.js")
	}

	depends, err = graph.TransitivelyDependsOn("app.js", "main.js")
	if err != nil {
		t.Fatalf("TransitivelyDependsOn returned error: %v", err)
	}
	if depends {
		t.Fatalf("did not expect app.js to transitively depend on main.js")
	}
}

func TestResolverModuleGraphTransitiveDependencyPredicateReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitivelyDependsOn("main.js", "b.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
