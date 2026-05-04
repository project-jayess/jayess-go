package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksMultiInitializationOrderAfter(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)

	after, err := graph.InitializesAfterFor([]string{"main.js", "worker.js"}, "worker.js", "shared.js")
	if err != nil {
		t.Fatalf("InitializesAfterFor returned error: %v", err)
	}
	if !after {
		t.Fatal("expected worker.js to initialize after shared.js")
	}

	after, err = graph.InitializesAfterFor([]string{"main.js", "worker.js"}, "shared.js", "worker.js")
	if err != nil {
		t.Fatalf("InitializesAfterFor returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect shared.js to initialize after worker.js")
	}
}

func TestResolverModuleGraphMultiInitializationOrderAfterMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	after, err := graph.InitializesAfterFor([]string{"main.js"}, "shared.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesAfterFor returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect missing module to participate in initialization order comparison")
	}
}

func TestResolverModuleGraphMultiInitializationOrderAfterSameModuleIsFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	after, err := graph.InitializesAfterFor([]string{"main.js"}, "main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesAfterFor returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect a module to initialize after itself")
	}
}

func TestResolverModuleGraphMultiInitializationOrderAfterReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesAfterFor([]string{"main.js", "a.js"}, "main.js", "a.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
