package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectTCPUDPRuntimeCalls(t *testing.T) {
	source := `
		const client = tcp.client();
		const socket = tcp.connect(client, "127.0.0.1:9000");
		const server = tcp.server();
		tcp.listen(server, "127.0.0.1:9001");
		const peer = tcp.accept(server);
		tcp.read(peer);
		tcp.write(socket, "payload");
		const timed = tcp.withTimeout(socket, 1000);
		tcp.awaitDrain(timed);
		tcp.lastError(timed);
		tcp.close(peer);

		const datagram = udp.socket();
		udp.bind(datagram, "127.0.0.1:0");
		udp.send(datagram, "payload", "127.0.0.1:9002");
		udp.receive(datagram);
		udp.joinMulticast(datagram, "224.0.0.1");
		udp.setBroadcast(datagram, true);
	`
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module, err := llvmbackend.LowerJayessStatementProgram(llvmbackend.JayessStatementProgram{
		Name:       "tcp-udp-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_tcp_client",
		"@jayess_tcp_connect",
		"@jayess_tcp_server",
		"@jayess_tcp_listen",
		"@jayess_tcp_accept",
		"@jayess_tcp_read",
		"@jayess_tcp_write",
		"@jayess_tcp_with_timeout",
		"@jayess_tcp_await_drain",
		"@jayess_tcp_last_error",
		"@jayess_tcp_close",
		"@jayess_udp_socket",
		"@jayess_udp_bind",
		"@jayess_udp_send",
		"@jayess_udp_receive",
		"@jayess_udp_join_multicast",
		"@jayess_udp_set_broadcast",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected TCP/UDP runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
