package test

import (
	"strings"
	"testing"

	"jayess-go/picohttpparser"
)

func TestPicoHTTPParserIncrementalRequestParsing(t *testing.T) {
	var parser picohttpparser.IncrementalParser
	if parser.Feed("GET /stream HTTP/1.1\r\nHost: ex") {
		t.Fatal("expected incomplete parser after first chunk")
	}
	if !parser.Feed("ample.test\r\n\r\n") {
		t.Fatal("expected complete parser after header terminator")
	}

	request, err := parser.Request()
	if err != nil {
		t.Fatalf("expected incremental request to parse, got %v", err)
	}
	if request.Path != "/stream" || len(request.Headers) != 1 {
		t.Fatalf("unexpected incremental request: %#v", request)
	}
}

func TestPicoHTTPParserDecodesChunkedBody(t *testing.T) {
	body, err := picohttpparser.DecodeChunked("4\r\nWiki\r\n5\r\npedia\r\n0\r\n\r\n")
	if err != nil {
		t.Fatalf("expected chunked body to decode, got %v", err)
	}
	if body != "Wikipedia" {
		t.Fatalf("unexpected decoded body %q", body)
	}
}

func TestPicoHTTPParserChunkedMalformedInput(t *testing.T) {
	_, err := picohttpparser.DecodeChunked("z\r\nnope\r\n0\r\n\r\n")
	if err == nil {
		t.Fatal("expected malformed chunk size error")
	}
	if !strings.Contains(err.Error(), string(picohttpparser.MalformedInput)) {
		t.Fatalf("expected malformed input diagnostic, got %v", err)
	}
}
