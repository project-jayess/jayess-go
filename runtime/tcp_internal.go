package runtime

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type TCPClient struct {
	Timeout time.Duration
	LastErr error
}

type TCPServer struct {
	Listener net.Listener
	Timeout  time.Duration
	LastErr  error
}

type TCPSocket struct {
	Conn    net.Conn
	Stream  *IOStream
	Timeout time.Duration
	LastErr error
}

func NewTCPClient() *TCPClient {
	return &TCPClient{}
}

func NewTCPServer() *TCPServer {
	return &TCPServer{}
}

func TCPConnect(client *TCPClient, address string) (*TCPSocket, error) {
	if address == "" {
		return nil, fmt.Errorf("TCP address is required")
	}
	timeout := time.Duration(0)
	if client != nil {
		timeout = client.Timeout
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		if client != nil {
			client.LastErr = err
		}
		return nil, err
	}
	return newTCPSocket(conn, timeout), nil
}

func TCPListen(server *TCPServer, address string) error {
	if server == nil {
		return fmt.Errorf("TCP server is required")
	}
	if address == "" {
		address = "127.0.0.1:0"
	}
	listener, err := listenTCP(address)
	if err != nil {
		server.LastErr = err
		return err
	}
	server.Listener = listener
	return nil
}

func TCPAccept(server *TCPServer) (*TCPSocket, error) {
	if server == nil || server.Listener == nil {
		return nil, fmt.Errorf("TCP server is not listening")
	}
	conn, err := server.Listener.Accept()
	if err != nil {
		server.LastErr = err
		return nil, err
	}
	return newTCPSocket(conn, server.Timeout), nil
}

func TCPRead(socket *TCPSocket, size int) ([]byte, error) {
	if socket == nil || socket.Conn == nil {
		return nil, fmt.Errorf("TCP socket is not connected")
	}
	if size <= 0 {
		size = 32 * 1024
	}
	if socket.Timeout > 0 {
		_ = socket.Conn.SetReadDeadline(time.Now().Add(socket.Timeout))
	}
	buffer := make([]byte, size)
	n, err := socket.Conn.Read(buffer)
	if err != nil {
		socket.LastErr = err
		return nil, err
	}
	return buffer[:n], nil
}

func TCPWrite(socket *TCPSocket, data []byte) (int, error) {
	if socket == nil || socket.Conn == nil {
		return 0, fmt.Errorf("TCP socket is not connected")
	}
	if socket.Timeout > 0 {
		_ = socket.Conn.SetWriteDeadline(time.Now().Add(socket.Timeout))
	}
	n, err := socket.Conn.Write(data)
	if err != nil {
		socket.LastErr = err
	}
	return n, err
}

func TCPClose(value any) error {
	switch resource := value.(type) {
	case *TCPSocket:
		if resource == nil || resource.Conn == nil {
			return nil
		}
		return resource.Conn.Close()
	case *TCPServer:
		if resource == nil || resource.Listener == nil {
			return nil
		}
		return resource.Listener.Close()
	default:
		return nil
	}
}

func TCPWithTimeout[T interface {
	*TCPClient | *TCPServer | *TCPSocket
}](value T, timeout time.Duration) T {
	switch resource := any(value).(type) {
	case *TCPClient:
		resource.Timeout = timeout
	case *TCPServer:
		resource.Timeout = timeout
	case *TCPSocket:
		resource.Timeout = timeout
	}
	return value
}

func TCPLastError(value any) error {
	switch resource := value.(type) {
	case *TCPClient:
		if resource != nil {
			return resource.LastErr
		}
	case *TCPServer:
		if resource != nil {
			return resource.LastErr
		}
	case *TCPSocket:
		if resource != nil {
			return resource.LastErr
		}
	}
	return nil
}

func TCPAwaitDrain(socket *TCPSocket) error {
	if socket == nil || socket.Conn == nil {
		return errors.New("TCP socket is not connected")
	}
	return nil
}

func newTCPSocket(conn net.Conn, timeout time.Duration) *TCPSocket {
	socket := &TCPSocket{Conn: conn, Timeout: timeout}
	socket.Stream = NewDuplexStream("tcp", conn, conn)
	socket.Stream.closer = conn
	return socket
}
