package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksMultiInitializationOrderBefore(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)

	before, err := graph.InitializesBeforeFor([]string{"main.js", "worker.js"}, "shared.js", "worker.js")
	if err != nil {
		t.Fatalf("InitializesBeforeFor returned error: %v", err)
	}
	if !before {
		t.Fatal("expected shared.js to initialize before worker.js")
	}

	before, err = graph.InitializesBeforeFor([]string{"main.js", "worker.js"}, "worker.js", "shared.js")
	if err != nil {
		t.Fatalf("InitializesBeforeFor returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect worker.js to initialize before shared.js")
	}
}

func TestResolverModuleGraphMultiInitializationOrderBeforeMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	before, err := graph.InitializesBeforeFor([]string{"main.js"}, "shared.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesBeforeFor returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect missing module to participate in initialization order comparison")
	}
}

func TestResolverModuleGraphMultiInitializationOrderBeforeSameModuleIsFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	before, err := graph.InitializesBeforeFor([]string{"main.js"}, "main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesBeforeFor returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect a module to initialize before itself")
	}
}

func TestResolverModuleGraphMultiInitializationOrderBeforeReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesBeforeFor([]string{"main.js", "a.js"}, "a.js", "main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
