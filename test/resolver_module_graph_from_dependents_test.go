package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphBuildsFromDependentMap(t *testing.T) {
	graph := resolver.NewModuleGraphFromDependents(map[string][]string{
		"config.js": []string{"main.js"},
		"main.js":   nil,
		"shared.js": []string{"main.js", "worker.js"},
		"worker.js": nil,
	})

	expected := []string{"config.js", "shared.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected main.js dependencies %#v, got %#v", expected, dependencies)
	}
	if !graph.HasModule("worker.js") {
		t.Fatalf("expected dependent-only module to be present")
	}
}

func TestResolverModuleGraphBuildsEmptyGraphFromNoDependents(t *testing.T) {
	graph := resolver.NewModuleGraphFromDependents(nil)

	if count := graph.ModuleCount(); count != 0 {
		t.Fatalf("expected no modules, got %d", count)
	}
}
