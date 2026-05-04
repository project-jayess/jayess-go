package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeUDPCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"socket",
		"send",
		"receive",
		"bind",
		"joinMulticast",
		"setBroadcast",
	}
	for _, name := range expected {
		if !jayessruntime.HasUDPCapability(name) {
			t.Fatalf("expected UDP runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsUDPSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(host, port, group, payload) {
			const socket = udp.socket();
			udp.bind(socket, host, port);
			udp.send(socket, payload, host, port);
			const packet = udp.receive(socket);
			udp.joinMulticast(socket, group);
			udp.setBroadcast(socket, true);
			return packet;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeUDPCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.UDPCapabilities() {
		if capability.Name == "" {
			t.Fatalf("UDP capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("UDP capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("UDP capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelUDPRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var udp = {};`)
	requireSemanticError(t, err, "duplicate declaration udp")
}
