package test

import (
	"errors"
	"testing"
	"time"

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

func TestRuntimeUDPBindSendReceiveClose(t *testing.T) {
	receiver := jayessruntime.UDPWithTimeout(jayessruntime.NewUDPSocket(), time.Second)
	if err := jayessruntime.UDPBind(receiver, "127.0.0.1:0"); err != nil {
		t.Fatalf("bind UDP receiver: %v", err)
	}
	defer jayessruntime.UDPClose(receiver)

	sender := jayessruntime.UDPWithTimeout(jayessruntime.NewUDPSocket(), time.Second)
	if err := jayessruntime.UDPBind(sender, "127.0.0.1:0"); err != nil {
		t.Fatalf("bind UDP sender: %v", err)
	}
	defer jayessruntime.UDPClose(sender)

	if _, err := jayessruntime.UDPSend(sender, []byte("hello"), receiver.LocalAddress()); err != nil {
		t.Fatalf("send UDP: %v", err)
	}
	packet, err := jayessruntime.UDPReceive(receiver, 64)
	if err != nil {
		t.Fatalf("receive UDP: %v", err)
	}
	if string(packet.Data) != "hello" || packet.Address == "" {
		t.Fatalf("unexpected UDP packet: %#v", packet)
	}
}

func TestRuntimeUDPBroadcastAndUnsupportedMulticast(t *testing.T) {
	socket := jayessruntime.NewUDPSocket()
	if err := jayessruntime.UDPSetBroadcast(socket, true); err != nil {
		t.Fatalf("set UDP broadcast: %v", err)
	}
	if !socket.Broadcast {
		t.Fatal("expected UDP broadcast flag")
	}
	if err := jayessruntime.UDPJoinMulticast(socket, "224.0.0.1"); !errors.Is(err, jayessruntime.ErrUnsupportedUDPFeature) {
		t.Fatalf("expected unsupported multicast error, got %v", err)
	}
}

func TestSemanticRejectsTopLevelUDPRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var udp = {};`)
	requireSemanticError(t, err, "duplicate declaration udp")
}
