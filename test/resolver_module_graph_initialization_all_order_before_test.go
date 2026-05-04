package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksFullInitializationOrderBefore(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("shared.js", nil)

	before, err := graph.InitializesBeforeAll("config.js", "worker.js")
	if err != nil {
		t.Fatalf("InitializesBeforeAll returned error: %v", err)
	}
	if !before {
		t.Fatal("expected config.js to initialize before worker.js")
	}

	before, err = graph.InitializesBeforeAll("worker.js", "config.js")
	if err != nil {
		t.Fatalf("InitializesBeforeAll returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect worker.js to initialize before config.js")
	}
}

func TestResolverModuleGraphFullInitializationOrderBeforeMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("config.js", nil)

	before, err := graph.InitializesBeforeAll("config.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesBeforeAll returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect missing module to participate in initialization order comparison")
	}
}

func TestResolverModuleGraphFullInitializationOrderBeforeSameModuleIsFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	before, err := graph.InitializesBeforeAll("main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesBeforeAll returned error: %v", err)
	}
	if before {
		t.Fatal("did not expect a module to initialize before itself")
	}
}

func TestResolverModuleGraphFullInitializationOrderBeforeReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesBeforeAll("a.js", "main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
