package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksFullInitializationOrderAfter(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("shared.js", nil)

	after, err := graph.InitializesAfterAll("worker.js", "config.js")
	if err != nil {
		t.Fatalf("InitializesAfterAll returned error: %v", err)
	}
	if !after {
		t.Fatal("expected worker.js to initialize after config.js")
	}

	after, err = graph.InitializesAfterAll("config.js", "worker.js")
	if err != nil {
		t.Fatalf("InitializesAfterAll returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect config.js to initialize after worker.js")
	}
}

func TestResolverModuleGraphFullInitializationOrderAfterMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("config.js", nil)

	after, err := graph.InitializesAfterAll("config.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesAfterAll returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect missing module to participate in initialization order comparison")
	}
}

func TestResolverModuleGraphFullInitializationOrderAfterSameModuleIsFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	after, err := graph.InitializesAfterAll("main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesAfterAll returned error: %v", err)
	}
	if after {
		t.Fatal("did not expect a module to initialize after itself")
	}
}

func TestResolverModuleGraphFullInitializationOrderAfterReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesAfterAll("main.js", "a.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
