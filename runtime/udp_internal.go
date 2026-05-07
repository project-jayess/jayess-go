package runtime

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type UDPSocket struct {
	Conn      *net.UDPConn
	Timeout   time.Duration
	Broadcast bool
	LastErr   error
}

type UDPPacket struct {
	Data    []byte
	Address string
}

func NewUDPSocket() *UDPSocket {
	return &UDPSocket{}
}

func UDPBind(socket *UDPSocket, address string) error {
	if socket == nil {
		return fmt.Errorf("UDP socket is required")
	}
	if address == "" {
		address = "127.0.0.1:0"
	}
	udpAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		socket.LastErr = err
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		socket.LastErr = err
		return err
	}
	socket.Conn = conn
	return nil
}

func UDPSend(socket *UDPSocket, data []byte, address string) (int, error) {
	if socket == nil || socket.Conn == nil {
		return 0, fmt.Errorf("UDP socket is not bound")
	}
	udpAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		socket.LastErr = err
		return 0, err
	}
	if socket.Timeout > 0 {
		_ = socket.Conn.SetWriteDeadline(time.Now().Add(socket.Timeout))
	}
	n, err := socket.Conn.WriteToUDP(data, udpAddress)
	if err != nil {
		socket.LastErr = err
	}
	return n, err
}

func UDPReceive(socket *UDPSocket, size int) (UDPPacket, error) {
	if socket == nil || socket.Conn == nil {
		return UDPPacket{}, fmt.Errorf("UDP socket is not bound")
	}
	if size <= 0 {
		size = 64 * 1024
	}
	if socket.Timeout > 0 {
		_ = socket.Conn.SetReadDeadline(time.Now().Add(socket.Timeout))
	}
	buffer := make([]byte, size)
	n, address, err := socket.Conn.ReadFromUDP(buffer)
	if err != nil {
		socket.LastErr = err
		return UDPPacket{}, err
	}
	return UDPPacket{Data: buffer[:n], Address: address.String()}, nil
}

func UDPSetBroadcast(socket *UDPSocket, enabled bool) error {
	if socket == nil {
		return fmt.Errorf("UDP socket is required")
	}
	socket.Broadcast = enabled
	return nil
}

func UDPJoinMulticast(socket *UDPSocket, group string) error {
	if socket != nil {
		socket.LastErr = ErrUnsupportedUDPFeature
	}
	return fmt.Errorf("%w: multicast group %s", ErrUnsupportedUDPFeature, group)
}

func UDPWithTimeout(socket *UDPSocket, timeout time.Duration) *UDPSocket {
	if socket != nil {
		socket.Timeout = timeout
	}
	return socket
}

func UDPClose(socket *UDPSocket) error {
	if socket == nil || socket.Conn == nil {
		return nil
	}
	return socket.Conn.Close()
}

func (socket *UDPSocket) LocalAddress() string {
	if socket == nil || socket.Conn == nil {
		return ""
	}
	return socket.Conn.LocalAddr().String()
}

var ErrUnsupportedUDPFeature = errors.New("unsupported UDP feature")
