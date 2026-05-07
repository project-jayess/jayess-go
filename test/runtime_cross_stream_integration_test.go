package test

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeCrossPackageStreams(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.txt")
	if err := jayessruntime.WriteFile(path, "cross stream"); err != nil {
		t.Fatalf("write file: %v", err)
	}
	fileStream, err := jayessruntime.CreateReadStream(path)
	if err != nil {
		t.Fatalf("file read stream: %v", err)
	}
	compressed, err := jayessruntime.CompressionCreateCompressStream("gzip", fileStream)
	if err != nil {
		t.Fatalf("compression stream: %v", err)
	}
	httpResponse := jayessruntime.NewHTTPResponse()
	if data, err := compressed.ReadAll(); err != nil {
		t.Fatalf("read compressed: %v", err)
	} else {
		jayessruntime.HTTPWriteBody(httpResponse, string(data))
	}
	httpBody := jayessruntime.HTTPStreamBody(httpResponse)
	buffer := jayessruntime.BufferCreate(0)
	bufferSink := jayessruntime.BufferCreateWriteStream(buffer)
	if _, err := jayessruntime.StreamPipe(httpBody, bufferSink); err != nil {
		t.Fatalf("http to buffer pipe: %v", err)
	}
	if len(buffer.Data) == 0 {
		t.Fatal("expected buffer data after stream pipeline")
	}
}

func TestRuntimeCrossProcessAndSocketStreams(t *testing.T) {
	result, err := jayessruntime.ExecProcess("sh", []string{"-c", "printf child"}, nil, "")
	if err != nil {
		t.Fatalf("exec child process: %v", err)
	}
	childStream := jayessruntime.StreamReadable([]byte(result.Stdout))
	var childSink bytes.Buffer
	if _, err := jayessruntime.StreamPipe(childStream, jayessruntime.NewWritableStream("child", &childSink)); err != nil {
		t.Fatalf("child stream pipe: %v", err)
	}
	if childSink.String() != "child" {
		t.Fatalf("unexpected child stream %q", childSink.String())
	}

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
		_, err = jayessruntime.StreamPipe(peer.Stream, jayessruntime.NewWritableStream("tcp", &bytes.Buffer{}))
		done <- err
	}()
	client, err := jayessruntime.TCPConnect(jayessruntime.NewTCPClient(), server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("connect TCP: %v", err)
	}
	if _, err := jayessruntime.TCPWrite(client, []byte("tcp")); err != nil {
		t.Fatalf("write TCP: %v", err)
	}
	_ = jayessruntime.TCPClose(client)
	if err := <-done; err != nil {
		t.Fatalf("pipe TCP stream: %v", err)
	}
}

func TestRuntimeCrossUDPStreamBridge(t *testing.T) {
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
	if _, err := jayessruntime.UDPSend(sender, []byte("udp"), receiver.LocalAddress()); err != nil {
		t.Fatalf("send UDP: %v", err)
	}
	packet, err := jayessruntime.UDPReceive(receiver, 16)
	if err != nil {
		t.Fatalf("receive UDP: %v", err)
	}
	stream := jayessruntime.StreamReadable(packet.Data)
	data, err := stream.ReadAll()
	if err != nil || string(data) != "udp" {
		t.Fatalf("udp stream got data=%q err=%v", data, err)
	}
}
