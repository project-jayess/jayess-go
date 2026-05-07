package test

import (
	"bytes"
	"testing"
	"time"

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

func TestRuntimeTCPConnectReadWriteClose(t *testing.T) {
	server := jayessruntime.TCPWithTimeout(jayessruntime.NewTCPServer(), time.Second)
	if err := jayessruntime.TCPListen(server, "127.0.0.1:0"); err != nil {
		t.Fatalf("listen TCP: %v", err)
	}
	defer jayessruntime.TCPClose(server)

	done := make(chan error, 1)
	go func() {
		peer, err := jayessruntime.TCPAccept(server)
		if err != nil {
			done <- err
			return
		}
		defer jayessruntime.TCPClose(peer)
		data, err := jayessruntime.TCPRead(peer, 32)
		if err != nil {
			done <- err
			return
		}
		if !bytes.Equal(data, []byte("ping")) {
			done <- errUnexpected("unexpected TCP payload")
			return
		}
		_, err = jayessruntime.TCPWrite(peer, []byte("pong"))
		done <- err
	}()

	client := jayessruntime.TCPWithTimeout(jayessruntime.NewTCPClient(), time.Second)
	socket, err := jayessruntime.TCPConnect(client, server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("connect TCP: %v", err)
	}
	defer jayessruntime.TCPClose(socket)
	if socket.Stream == nil || !socket.Stream.CanRead() || !socket.Stream.CanWrite() {
		t.Fatal("expected TCP socket to expose duplex stream")
	}
	if _, err := jayessruntime.TCPWrite(socket, []byte("ping")); err != nil {
		t.Fatalf("write TCP: %v", err)
	}
	response, err := jayessruntime.TCPRead(socket, 32)
	if err != nil {
		t.Fatalf("read TCP: %v", err)
	}
	if string(response) != "pong" {
		t.Fatalf("unexpected TCP response %q", response)
	}
	if err := <-done; err != nil {
		t.Fatalf("server TCP flow: %v", err)
	}
	if err := jayessruntime.TCPAwaitDrain(socket); err != nil {
		t.Fatalf("await TCP drain: %v", err)
	}
}

func TestRuntimeTCPLastError(t *testing.T) {
	client := jayessruntime.TCPWithTimeout(jayessruntime.NewTCPClient(), time.Millisecond)
	_, err := jayessruntime.TCPConnect(client, "127.0.0.1:1")
	if err == nil {
		t.Fatal("expected TCP connect error")
	}
	if jayessruntime.TCPLastError(client) == nil {
		t.Fatal("expected TCP last error")
	}
}

func TestSemanticRejectsTopLevelTCPRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var tcp = {};`)
	requireSemanticError(t, err, "duplicate declaration tcp")
}

type errUnexpected string

func (err errUnexpected) Error() string {
	return string(err)
}
