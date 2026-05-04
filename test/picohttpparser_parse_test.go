package test

import (
	"strings"
	"testing"

	"jayess-go/picohttpparser"
)

func TestPicoHTTPParserParsesRequestLineAndHeaders(t *testing.T) {
	request, err := picohttpparser.ParseRequest("GET /hello HTTP/1.1\r\nHost: example.test\r\nAccept: */*\r\n\r\n")
	if err != nil {
		t.Fatalf("expected request to parse, got %v", err)
	}
	if request.Method != "GET" || request.Path != "/hello" || request.Version != "HTTP/1.1" {
		t.Fatalf("unexpected request line: %#v", request)
	}
	if len(request.Headers) != 2 || request.Headers[0].Name != "Host" || request.Headers[0].Value != "example.test" {
		t.Fatalf("unexpected request headers: %#v", request.Headers)
	}
}

func TestPicoHTTPParserParsesResponseLineAndHeaders(t *testing.T) {
	response, err := picohttpparser.ParseResponse("HTTP/1.1 204 No Content\r\nServer: jayess\r\n\r\n")
	if err != nil {
		t.Fatalf("expected response to parse, got %v", err)
	}
	if response.Version != "HTTP/1.1" || response.StatusCode != 204 || response.Reason != "No Content" {
		t.Fatalf("unexpected response line: %#v", response)
	}
	if len(response.Headers) != 1 || response.Headers[0].Name != "Server" || response.Headers[0].Value != "jayess" {
		t.Fatalf("unexpected response headers: %#v", response.Headers)
	}
}

func TestPicoHTTPParserSurfacesMalformedInput(t *testing.T) {
	_, err := picohttpparser.ParseRequest("GET /missing-version\r\nHost: example.test\r\n\r\n")
	if err == nil {
		t.Fatal("expected malformed request error")
	}
	if !strings.Contains(err.Error(), string(picohttpparser.MalformedInput)) {
		t.Fatalf("expected malformed input diagnostic, got %v", err)
	}

	_, err = picohttpparser.ParseResponse("HTTP/1.1 ok\r\nServer: jayess\r\n\r\n")
	if err == nil {
		t.Fatal("expected malformed response status error")
	}
	if !strings.Contains(err.Error(), string(picohttpparser.MalformedInput)) {
		t.Fatalf("expected malformed input diagnostic, got %v", err)
	}
}

func TestPicoHTTPParserSurfacesIncompleteInput(t *testing.T) {
	_, err := picohttpparser.ParseRequest("GET / HTTP/1.1\r\nHost: example.test\r\n")
	if err == nil {
		t.Fatal("expected incomplete request error")
	}
	if !strings.Contains(err.Error(), string(picohttpparser.IncompleteData)) {
		t.Fatalf("expected incomplete data diagnostic, got %v", err)
	}
}
