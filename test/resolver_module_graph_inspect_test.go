package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphInspectsDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})

	if !graph.HasModule("main.js") {
		t.Fatalf("expected main.js to be present")
	}
	if !graph.HasModule("config.js") {
		t.Fatalf("expected implicit leaf config.js to be present")
	}
	if graph.HasModule("missing.js") {
		t.Fatalf("did not expect missing.js to be present")
	}

	dependencies := graph.Dependencies("main.js")
	expected := []string{"config.js", "app.js"}
	if !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected dependencies %#v, got %#v", expected, dependencies)
	}
}

func TestResolverModuleGraphListsModulesDeterministically(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("app.js", nil)

	expected := []string{"app.js", "main.js", "shared.js"}
	if modules := graph.Modules(); !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected modules %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphListsRootModulesDeterministically(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	expected := []string{"main.js", "worker.js"}
	if roots := graph.RootModules(); !reflect.DeepEqual(roots, expected) {
		t.Fatalf("expected root modules %#v, got %#v", expected, roots)
	}
}

func TestResolverModuleGraphListsLeafModulesDeterministically(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	expected := []string{"config.js", "shared.js"}
	if leaves := graph.LeafModules(); !reflect.DeepEqual(leaves, expected) {
		t.Fatalf("expected leaf modules %#v, got %#v", expected, leaves)
	}
}

func TestResolverModuleGraphDependenciesReturnsCopy(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"dep.js"})

	dependencies := graph.Dependencies("main.js")
	dependencies[0] = "changed.js"

	expected := []string{"dep.js"}
	if got := graph.Dependencies("main.js"); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected stored dependencies %#v, got %#v", expected, got)
	}
}

func TestResolverModuleGraphChecksDirectDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js", "app.js"})

	if !graph.DependsOn("main.js", "app.js") {
		t.Fatalf("expected main.js to depend on app.js")
	}
	if graph.DependsOn("app.js", "main.js") {
		t.Fatalf("did not expect app.js to depend on main.js")
	}
	if graph.DependsOn("missing.js", "app.js") {
		t.Fatalf("did not expect missing module to have dependencies")
	}
}

func TestResolverModuleGraphListsDependentsDeterministically(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	expected := []string{"main.js", "worker.js"}
	if dependents := graph.Dependents("shared.js"); !reflect.DeepEqual(dependents, expected) {
		t.Fatalf("expected dependents %#v, got %#v", expected, dependents)
	}
}

func TestResolverModuleGraphMissingDependentsAreEmpty(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"dep.js"})

	if dependents := graph.Dependents("missing.js"); len(dependents) != 0 {
		t.Fatalf("expected no dependents for missing module, got %#v", dependents)
	}
}

func TestResolverModuleGraphMissingDependenciesAreNil(t *testing.T) {
	graph := resolver.NewModuleGraph()

	if dependencies := graph.Dependencies("missing.js"); dependencies != nil {
		t.Fatalf("expected nil dependencies for missing module, got %#v", dependencies)
	}
}
