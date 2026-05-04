package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeWorkerCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"thread",
		"postMessage",
		"onMessage",
		"sharedMemory",
		"atomicLoad",
		"atomicStore",
	}
	for _, name := range expected {
		if !jayessruntime.HasWorkerCapability(name) {
			t.Fatalf("expected worker runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsWorkerSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(source) {
			const thread = worker.thread(source);
			worker.postMessage(thread, { ready: true });
			worker.onMessage(thread, (message) => message);
			const memory = worker.sharedMemory(1024);
			worker.atomicStore(memory, 0, 1);
			const value = worker.atomicLoad(memory, 0);
			return thread || value;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeWorkerCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.WorkerCapabilities() {
		if capability.Name == "" {
			t.Fatalf("worker capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("worker capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("worker capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelWorkerRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var worker = {};`)
	requireSemanticError(t, err, "duplicate declaration worker")
}
