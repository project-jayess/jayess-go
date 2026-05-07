package runtime

import (
	"bytes"
	"crypto/tls"
	"net"
	nethttp "net/http"
)

type HTTPSServer struct {
	HTTP *HTTPServer
	TLS  TLSRuntimeConfig
}

type HTTPSClient struct {
	TLS TLSRuntimeConfig
}

func CreateHTTPSServer(config TLSRuntimeConfig, handler HTTPHandler) *HTTPSServer {
	return &HTTPSServer{HTTP: CreateHTTPServer(handler), TLS: config}
}

func (server *HTTPSServer) Listen(address string) error {
	if server == nil || server.HTTP == nil || server.HTTP.server == nil {
		return nil
	}
	if address == "" {
		address = "127.0.0.1:0"
	}
	listener, err := listenTCP(address)
	if err != nil {
		server.HTTP.emitError(HTTPEventError, err)
		return err
	}
	tlsListener := tls.NewListener(listener, TLSServerConfig(server.TLS))
	server.HTTP.listener <- tlsListener.Addr().String()
	server.HTTP.Emit(HTTPServerEvent{Name: HTTPEventListening, Address: tlsListener.Addr().String()})
	go func() {
		if err := server.HTTP.server.Serve(tlsListener); err != nil && err != nethttp.ErrServerClosed {
			server.HTTP.emitError(HTTPEventError, err)
			server.HTTP.errs <- err
		}
	}()
	return nil
}

func HTTPSRequestObject(url string, config TLSRuntimeConfig) HTTPClientRequest {
	return HTTPClientRequest{Method: nethttp.MethodGet, URL: url, Headers: map[string]string{}}
}

func HTTPSClientConfig(config TLSRuntimeConfig) *nethttp.Client {
	transport := &nethttp.Transport{TLSClientConfig: TLSClientConfig(config)}
	return &nethttp.Client{Transport: transport}
}

func HTTPSDoRequest(request HTTPClientRequest, config TLSRuntimeConfig) (HTTPClientResponse, error) {
	method := request.Method
	if method == "" {
		method = nethttp.MethodGet
	}
	netRequest, err := nethttp.NewRequest(method, request.URL, bytes.NewReader(request.Body))
	if err != nil {
		return HTTPClientResponse{}, err
	}
	for key, value := range request.Headers {
		netRequest.Header.Set(key, value)
	}
	client := HTTPSClientConfig(config)
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

func HTTPSListener(address string, config TLSRuntimeConfig) (net.Listener, error) {
	if address == "" {
		address = "127.0.0.1:0"
	}
	listener, err := listenTCP(address)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(listener, TLSServerConfig(config)), nil
}
