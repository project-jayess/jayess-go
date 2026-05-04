package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphReplacesImports(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"old.js", "shared.js"})

	graph.ReplaceImports("main.js", []string{"new.js", "shared.js"})

	expected := []string{"new.js", "shared.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected dependencies %#v, got %#v", expected, dependencies)
	}
	if !graph.HasModule("new.js") {
		t.Fatalf("expected new imported module to be present")
	}
}

func TestResolverModuleGraphReplaceImportsCanClearImports(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"old.js"})

	graph.ReplaceImports("main.js", nil)

	if dependencies := graph.Dependencies("main.js"); len(dependencies) != 0 {
		t.Fatalf("expected no dependencies, got %#v", dependencies)
	}
	if !graph.HasModule("main.js") {
		t.Fatalf("expected main.js to remain in graph")
	}
}
