package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphAddCompactModuleDeduplicatesImports(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddCompactModule("main.js", []string{"setup.js", "math.js", "setup.js", "math.js"})

	expected := []string{"setup.js", "math.js"}
	if actual := graph.Dependencies("main.js"); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected compact imports %#v, got %#v", expected, actual)
	}
}

func TestResolverModuleGraphAddCompactModuleKeepsImportedModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddCompactModule("main.js", []string{"setup.js", "setup.js"})

	for _, module := range []string{"main.js", "setup.js"} {
		if !graph.HasModule(module) {
			t.Fatalf("expected graph to contain %q", module)
		}
	}
}
