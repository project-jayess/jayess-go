package test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeHTTPServerHandlesRequestsInternally(t *testing.T) {
	server := jayessruntime.CreateHTTPServer(func(request *jayessruntime.HTTPRequest, response *jayessruntime.HTTPResponse) {
		if request.Method != http.MethodPost || request.Path != "/echo" {
			jayessruntime.HTTPStatus(response, http.StatusNotFound)
			return
		}
		headers := jayessruntime.HTTPHeaders(request)
		response.Headers["X-Jayess"] = headers["X-Test"]
		jayessruntime.HTTPStatus(response, http.StatusCreated)
		jayessruntime.HTTPWriteBody(response, "echo:"+jayessruntime.HTTPReadBody(request))
	})
	if err := server.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	defer server.Close(nil)

	result, err := jayessruntime.HTTPDoRequest(jayessruntime.HTTPClientRequest{
		Method:  http.MethodPost,
		URL:     "http://" + server.Addr() + "/echo",
		Headers: map[string]string{"X-Test": "yes"},
		Body:    []byte("body"),
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("HTTPDoRequest returned error: %v", err)
	}
	if result.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", result.StatusCode)
	}
	if string(result.Body) != "echo:body" {
		t.Fatalf("unexpected body %q", string(result.Body))
	}
	if result.Headers["X-Jayess"] != "yes" {
		t.Fatalf("expected response header to round trip, got %#v", result.Headers)
	}
	if err := server.LastError(); err != nil {
		t.Fatalf("unexpected server error: %v", err)
	}
}

func TestRuntimeHTTPHelpersSupportTimeoutKeepAliveAndBodyStreams(t *testing.T) {
	request := jayessruntime.HTTPKeepAlive(jayessruntime.HTTPClientRequest{URL: "http://example.test"})
	request = jayessruntime.HTTPWithTimeout(request, 50*time.Millisecond)
	if request.Headers["Connection"] != "keep-alive" || request.Timeout != 50*time.Millisecond {
		t.Fatalf("unexpected helper request: %#v", request)
	}

	response := jayessruntime.NewHTTPResponse()
	jayessruntime.HTTPWriteBody(response, "streamed")
	stream := jayessruntime.HTTPStreamBody(response)
	data, err := stream.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if strings.TrimSpace(string(data)) != "streamed" {
		t.Fatalf("unexpected stream data %q", string(data))
	}
}
