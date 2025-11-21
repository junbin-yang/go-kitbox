package netconn

import (
	"net"

	"github.com/junbin-yang/go-kitbox/pkg/fillp"
)

// TCPConnection TCP连接适配器
type TCPConnection struct {
	conn net.Conn
}

func NewTCPConnection(conn net.Conn) *TCPConnection {
	return &TCPConnection{conn: conn}
}

func (t *TCPConnection) Read(b []byte) (n int, err error) {
	return t.conn.Read(b)
}

func (t *TCPConnection) Write(b []byte) (n int, err error) {
	return t.conn.Write(b)
}

func (t *TCPConnection) Close() error {
	return t.conn.Close()
}

func (t *TCPConnection) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *TCPConnection) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

// UDPConnection UDP连接适配器（基于FILLP）
type UDPConnection struct {
	conn *fillp.Connection
}

func NewUDPConnection(conn *fillp.Connection) *UDPConnection {
	return &UDPConnection{conn: conn}
}

func (u *UDPConnection) Read(b []byte) (n int, err error) {
	data, err := u.conn.Receive()
	if err != nil {
		return 0, err
	}
	n = copy(b, data)
	return n, nil
}

func (u *UDPConnection) Write(b []byte) (n int, err error) {
	err = u.conn.Send(b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (u *UDPConnection) Close() error {
	return u.conn.Close()
}

func (u *UDPConnection) LocalAddr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *UDPConnection) RemoteAddr() net.Addr {
	return u.conn.RemoteAddr()
}
