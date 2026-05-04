package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeTCPCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"client",
		"server",
		"connect",
		"listen",
		"accept",
		"read",
		"write",
		"close",
		"lastError",
		"withTimeout",
		"awaitDrain",
	}
	for _, name := range expected {
		if !jayessruntime.HasTCPCapability(name) {
			t.Fatalf("expected TCP runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsTCPSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(host, port, payload) {
			const client = tcp.client();
			const socket = tcp.connect(client, host, port);
			const server = tcp.server();
			tcp.listen(server, port);
			const peer = tcp.accept(server);
			const input = tcp.read(peer);
			tcp.write(socket, payload);
			const timed = tcp.withTimeout(socket, 1000);
			const ready = tcp.awaitDrain(timed);
			const error = tcp.lastError(timed);
			tcp.close(peer);
			return input || ready || error;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeTCPCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.TCPCapabilities() {
		if capability.Name == "" {
			t.Fatalf("TCP capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("TCP capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("TCP capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelTCPRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var tcp = {};`)
	requireSemanticError(t, err, "duplicate declaration tcp")
}
