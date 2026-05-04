package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksRootModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	if !graph.IsRootModule("main.js") {
		t.Fatalf("expected main.js to be a root module")
	}
	if graph.IsRootModule("shared.js") {
		t.Fatalf("did not expect shared.js to be a root module")
	}
	if graph.IsRootModule("missing.js") {
		t.Fatalf("did not expect missing module to be a root module")
	}
}

func TestResolverModuleGraphChecksLeafModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})

	if !graph.IsLeafModule("shared.js") {
		t.Fatalf("expected shared.js to be a leaf module")
	}
	if graph.IsLeafModule("main.js") {
		t.Fatalf("did not expect main.js to be a leaf module")
	}
	if graph.IsLeafModule("missing.js") {
		t.Fatalf("did not expect missing module to be a leaf module")
	}
}

func TestResolverModuleGraphChecksIsolatedModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("standalone.js", nil)

	if !graph.IsIsolatedModule("standalone.js") {
		t.Fatalf("expected standalone.js to be an isolated module")
	}
	if graph.IsIsolatedModule("main.js") {
		t.Fatalf("did not expect importing module to be isolated")
	}
	if graph.IsIsolatedModule("shared.js") {
		t.Fatalf("did not expect imported module to be isolated")
	}
	if graph.IsIsolatedModule("missing.js") {
		t.Fatalf("did not expect missing module to be isolated")
	}
}

func TestResolverModuleGraphListsIsolatedModulesDeterministically(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("standalone-b.js", nil)
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("standalone-a.js", nil)

	expected := []string{"standalone-a.js", "standalone-b.js"}
	if isolated := graph.IsolatedModules(); !reflect.DeepEqual(isolated, expected) {
		t.Fatalf("expected isolated modules %#v, got %#v", expected, isolated)
	}
}
