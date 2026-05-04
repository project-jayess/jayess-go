package test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphAddsResolvedModuleDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddResolvedModule("main.js", []resolver.ResolvedModuleDependency{
		{Source: "./setup.js", Path: "setup.js", SideEffect: true},
		{Source: "./app.js", Path: "app.js"},
	})
	graph.AddResolvedModule("app.js", []resolver.ResolvedModuleDependency{
		{Source: "./model.js", Path: "model.js"},
	})

	order, err := graph.InitializationOrder("main.js")
	if err != nil {
		t.Fatalf("InitializationOrder returned error: %v", err)
	}
	expected := []string{"setup.js", "model.js", "app.js", "main.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphAddsCompactResolvedModuleDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddCompactResolvedModule("main.js", []resolver.ResolvedModuleDependency{
		{Source: "./setup.js", Path: "setup.js", SideEffect: true},
		{Source: "./setup.js", Path: "setup.js", ReExport: true},
		{Source: "./app.js", Path: "app.js"},
	})

	expected := []string{"setup.js", "app.js"}
	if imports := graph.Dependencies("main.js"); !reflect.DeepEqual(imports, expected) {
		t.Fatalf("expected compact imports %#v, got %#v", expected, imports)
	}
}

func TestResolverModuleGraphDetectsCycleFromResolvedDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddResolvedModule("main.js", []resolver.ResolvedModuleDependency{{Source: "./a.js", Path: "a.js"}})
	graph.AddResolvedModule("a.js", []resolver.ResolvedModuleDependency{{Source: "./main.js", Path: "main.js"}})

	_, err := graph.InitializationOrder("main.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "main.js -> a.js -> main.js") {
		t.Fatalf("expected readable resolved dependency cycle, got %v", err)
	}
}
