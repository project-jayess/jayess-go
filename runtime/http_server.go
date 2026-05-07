package runtime

import (
	"bytes"
	"context"
	"io"
	"net"
	nethttp "net/http"
	"strings"
	"time"
)

type HTTPHandler func(*HTTPRequest, *HTTPResponse)

type HTTPServer struct {
	server   *nethttp.Server
	listener chan string
	errs     chan error
	events   *httpServerEvents
}

type HTTPRequest struct {
	Method  string
	Path    string
	URL     string
	Headers map[string]string
	Body    []byte
}

type HTTPResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       bytes.Buffer
}

type HTTPClientRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
	Timeout time.Duration
}

type HTTPClientResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

func CreateHTTPServer(handler HTTPHandler) *HTTPServer {
	mux := nethttp.NewServeMux()
	runtimeServer := &HTTPServer{
		listener: make(chan string, 1),
		errs:     make(chan error, 1),
		events:   newHTTPServerEvents(),
	}
	if handler != nil {
		runtimeServer.On(HTTPEventRequest, func(event HTTPServerEvent) {
			handler(event.Request, event.Response)
		})
	}
	runtimeServer.server = &nethttp.Server{Handler: mux}
	runtimeServer.server.ConnState = func(connection net.Conn, state nethttp.ConnState) {
		if state == nethttp.StateNew {
			runtimeServer.Emit(HTTPServerEvent{Name: HTTPEventConnection, Address: connection.RemoteAddr().String()})
		}
	}
	mux.HandleFunc("/", func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		runtimeRequest, err := NewHTTPRequest(request)
		if err != nil {
			runtimeServer.emitError(HTTPEventClientError, err)
			nethttp.Error(writer, err.Error(), nethttp.StatusBadRequest)
			return
		}
		runtimeResponse := NewHTTPResponse()
		runtimeServer.emitProtocolEvents(runtimeRequest, runtimeResponse)
		if !runtimeServer.Emit(HTTPServerEvent{Name: HTTPEventRequest, Request: runtimeRequest, Response: runtimeResponse}) {
			runtimeResponse.StatusCode = nethttp.StatusNotFound
		}
		writeNetHTTPResponse(writer, runtimeResponse)
	})
	return runtimeServer
}

func (server *HTTPServer) Listen(address string) error {
	if server == nil || server.server == nil {
		return nil
	}
	if address == "" {
		address = "127.0.0.1:0"
	}
	listener, err := listenTCP(address)
	if err != nil {
		server.emitError(HTTPEventError, err)
		return err
	}
	server.listener <- listener.Addr().String()
	server.Emit(HTTPServerEvent{Name: HTTPEventListening, Address: listener.Addr().String()})
	go func() {
		if err := server.server.Serve(listener); err != nil && err != nethttp.ErrServerClosed {
			server.emitError(HTTPEventError, err)
			server.errs <- err
		}
	}()
	return nil
}

func (server *HTTPServer) Addr() string {
	if server == nil || server.listener == nil {
		return ""
	}
	select {
	case address := <-server.listener:
		server.listener <- address
		return address
	default:
		return ""
	}
}

func (server *HTTPServer) Close(ctx context.Context) error {
	if server == nil || server.server == nil {
		return nil
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		defer cancel()
	}
	err := server.server.Shutdown(ctx)
	if err != nil {
		server.emitError(HTTPEventError, err)
		return err
	}
	server.Emit(HTTPServerEvent{Name: HTTPEventClose})
	return nil
}

func (server *HTTPServer) LastError() error {
	if server == nil || server.errs == nil {
		return nil
	}
	select {
	case err := <-server.errs:
		return err
	default:
		return nil
	}
}

func NewHTTPRequest(request *nethttp.Request) (*HTTPRequest, error) {
	if request == nil {
		return &HTTPRequest{Headers: map[string]string{}}, nil
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	return &HTTPRequest{
		Method:  request.Method,
		Path:    request.URL.Path,
		URL:     request.URL.String(),
		Headers: flattenHeaders(request.Header),
		Body:    body,
	}, nil
}

func NewHTTPResponse() *HTTPResponse {
	return &HTTPResponse{StatusCode: nethttp.StatusOK, Headers: map[string]string{}}
}

func HTTPHeaders(request *HTTPRequest) map[string]string {
	if request == nil {
		return map[string]string{}
	}
	return copyStringMap(request.Headers)
}

func HTTPStatus(response *HTTPResponse, status int) {
	if response == nil {
		return
	}
	response.StatusCode = status
}

func HTTPReadBody(request *HTTPRequest) string {
	if request == nil {
		return ""
	}
	return string(request.Body)
}

func HTTPWriteBody(response *HTTPResponse, body string) {
	if response == nil {
		return
	}
	response.Body.WriteString(body)
}

func HTTPStreamBody(response *HTTPResponse) *IOStream {
	if response == nil {
		return NewReadableStream("http-body", strings.NewReader(""))
	}
	return NewReadableStream("http-body", bytes.NewReader(response.Body.Bytes()))
}

func HTTPKeepAlive(request HTTPClientRequest) HTTPClientRequest {
	request.Headers = copyStringMap(request.Headers)
	request.Headers["Connection"] = "keep-alive"
	return request
}

func HTTPWithTimeout(request HTTPClientRequest, timeout time.Duration) HTTPClientRequest {
	request.Timeout = timeout
	return request
}

func HTTPRequestObject(request *HTTPRequest) *HTTPRequest {
	return request
}

func HTTPResponseObject(response *HTTPResponse) *HTTPResponse {
	return response
}

func HTTPDoRequest(request HTTPClientRequest) (HTTPClientResponse, error) {
	method := request.Method
	if method == "" {
		method = nethttp.MethodGet
	}
	body := bytes.NewReader(request.Body)
	netRequest, err := nethttp.NewRequest(method, request.URL, body)
	if err != nil {
		return HTTPClientResponse{}, err
	}
	for key, value := range request.Headers {
		netRequest.Header.Set(key, value)
	}
	client := &nethttp.Client{}
	if request.Timeout > 0 {
		client.Timeout = request.Timeout
	}
	netResponse, err := client.Do(netRequest)
	if err != nil {
		return HTTPClientResponse{}, err
	}
	defer netResponse.Body.Close()
	return netHTTPResponseToRuntime(netResponse)
}

func netHTTPResponseToRuntime(netResponse *nethttp.Response) (HTTPClientResponse, error) {
	if netResponse == nil {
		return HTTPClientResponse{Headers: map[string]string{}}, nil
	}
	responseBody, err := io.ReadAll(netResponse.Body)
	if err != nil {
		return HTTPClientResponse{}, err
	}
	return HTTPClientResponse{
		StatusCode: netResponse.StatusCode,
		Headers:    flattenHeaders(netResponse.Header),
		Body:       responseBody,
	}, nil
}

func writeNetHTTPResponse(writer nethttp.ResponseWriter, response *HTTPResponse) {
	if response == nil {
		writer.WriteHeader(nethttp.StatusNoContent)
		return
	}
	for key, value := range response.Headers {
		writer.Header().Set(key, value)
	}
	if response.StatusCode == 0 {
		response.StatusCode = nethttp.StatusOK
	}
	writer.WriteHeader(response.StatusCode)
	_, _ = writer.Write(response.Body.Bytes())
}

func flattenHeaders(headers nethttp.Header) map[string]string {
	values := map[string]string{}
	for key, headerValues := range headers {
		values[key] = strings.Join(headerValues, ", ")
	}
	return values
}

func (server *HTTPServer) emitProtocolEvents(request *HTTPRequest, response *HTTPResponse) {
	expect := request.Headers["Expect"]
	if strings.EqualFold(expect, "100-continue") {
		server.Emit(HTTPServerEvent{Name: HTTPEventCheckContinue, Request: request, Response: response})
	} else if expect != "" {
		server.Emit(HTTPServerEvent{Name: HTTPEventCheckExpectation, Request: request, Response: response})
	}
	if strings.EqualFold(request.Method, nethttp.MethodConnect) {
		server.Emit(HTTPServerEvent{Name: HTTPEventConnect, Request: request, Response: response})
	}
	if request.Headers["Upgrade"] != "" {
		server.Emit(HTTPServerEvent{Name: HTTPEventUpgrade, Request: request, Response: response})
	}
}
