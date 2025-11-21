package netconn

import (
	"net"
	"time"
)

// ProtocolType 协议类型
type ProtocolType int

const (
	ProtocolTCP ProtocolType = iota
	ProtocolUDP // 使用FILLP
)

// ConnectionType 连接类型
type ConnectionType int

const (
	ConnectionTypeServer ConnectionType = iota
	ConnectionTypeClient
)

// NetConnection 统一的网络连接接口
type NetConnection interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// SocketOption 套接字选项
type SocketOption struct {
	Addr string
	Port int
}

// ConnectOption 连接信息
type ConnectOption struct {
	Protocol     ProtocolType
	LocalSocket  *SocketOption
	RemoteSocket *SocketOption
	NetConn      NetConnection
}

// ServerOption 服务器选项
type ServerOption struct {
	Protocol ProtocolType
	Addr     string
	Port     int
}

// ClientOption 客户端选项
type ClientOption struct {
	Protocol        ProtocolType
	RemoteIP        string
	RemotePort      int
	Timeout         time.Duration
	KeepAlive       bool
	KeepAlivePeriod time.Duration
}

// BaseListenerCallback 统一的回调接口
type BaseListenerCallback struct {
	OnConnected    func(fd int, connType ConnectionType, connOpt *ConnectOption)
	OnDisconnected func(fd int, connType ConnectionType)
	OnDataReceived func(fd int, connType ConnectionType, buf []byte, used int) int
}

// 常量定义
const (
	DefaultBufSize        = 1536
	DefaultConnectTimeout = 5 * time.Second
)
