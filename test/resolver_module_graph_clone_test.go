package test

import (
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCloneCopiesModulesAndImports(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})

	clone := graph.Clone()

	if modules := clone.Modules(); !reflect.DeepEqual(modules, graph.Modules()) {
		t.Fatalf("expected cloned modules %#v, got %#v", graph.Modules(), modules)
	}
	if dependencies := clone.Dependencies("main.js"); !reflect.DeepEqual(dependencies, graph.Dependencies("main.js")) {
		t.Fatalf("expected cloned dependencies %#v, got %#v", graph.Dependencies("main.js"), dependencies)
	}
}

func TestResolverModuleGraphCloneDoesNotShareImportSlices(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	clone := graph.Clone()

	clone.ReplaceImports("main.js", []string{"config.js"})

	expected := []string{"shared.js"}
	if dependencies := graph.Dependencies("main.js"); !reflect.DeepEqual(dependencies, expected) {
		t.Fatalf("expected original dependencies %#v, got %#v", expected, dependencies)
	}
}
