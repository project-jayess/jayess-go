package test

import (
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksTransitiveDependentsForModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	if !graph.TransitivelyDependedOnByFor([]string{"model.js", "config.js"}, "main.js") {
		t.Fatalf("expected main.js to transitively depend on one target module")
	}
	if !graph.TransitivelyDependedOnByFor([]string{"model.js", "config.js"}, "worker.js") {
		t.Fatalf("expected worker.js to transitively depend on one target module")
	}
	if graph.TransitivelyDependedOnByFor([]string{"config.js"}, "worker.js") {
		t.Fatalf("did not expect worker.js to transitively depend on config.js")
	}
}

func TestResolverModuleGraphChecksTransitiveDependentsForNoModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	if graph.TransitivelyDependedOnByFor(nil, "main.js") {
		t.Fatalf("did not expect no modules to have a transitive dependent")
	}
}

func TestResolverModuleGraphChecksTransitiveDependentsForModulesSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	if !graph.TransitivelyDependedOnByFor([]string{"a.js", "b.js"}, "main.js") {
		t.Fatalf("expected main.js to transitively depend on one target module")
	}
	if !graph.TransitivelyDependedOnByFor([]string{"a.js", "b.js"}, "a.js") {
		t.Fatalf("expected cycle member a.js in transitive dependents")
	}
}
