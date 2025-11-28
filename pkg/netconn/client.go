package netconn

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/junbin-yang/go-kitbox/pkg/fillp"
)

// BaseClient 统一的客户端
type BaseClient struct {
	connMgr  *ConnectionManager
	callback *BaseListenerCallback
	fd       int
	protocol ProtocolType
	running  bool
	mu       sync.RWMutex
	stopChan chan struct{}
}

// NewBaseClient 创建客户端实例
func NewBaseClient(connMgr *ConnectionManager, callback *BaseListenerCallback) *BaseClient {
	if connMgr == nil {
		connMgr = NewConnectionManager()
	}
	return &BaseClient{
		connMgr:  connMgr,
		callback: callback,
		fd:       -1,
	}
}

// Connect 使用配置选项连接服务器
func (c *BaseClient) Connect(opt *ClientOption) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return -1, errors.New("already connected")
	}

	c.protocol = opt.Protocol
	c.stopChan = make(chan struct{})

	var conn NetConnection
	var err error

	if opt.Protocol == ProtocolTCP {
		conn, err = c.connectTCP(opt)
	} else {
		conn, err = c.connectUDP(opt)
	}

	if err != nil {
		return -1, err
	}

	c.fd = c.connMgr.RegisterConn(conn, ConnectionTypeClient)
	c.running = true

	// 触发连接回调
	if c.callback != nil && c.callback.OnConnected != nil {
		connOpt := c.connMgr.GetConnInfo(c.fd)
		c.callback.OnConnected(c.fd, ConnectionTypeClient, connOpt)
	}

	// 启动接收循环
	go c.receiveLoop()

	return c.fd, nil
}

// connectTCP 连接TCP服务器
func (c *BaseClient) connectTCP(opt *ClientOption) (NetConnection, error) {
	timeout := opt.Timeout
	if timeout == 0 {
		timeout = DefaultConnectTimeout
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", opt.RemoteIP, opt.RemotePort), timeout)
	if err != nil {
		return nil, err
	}

	if opt.KeepAlive {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
			if opt.KeepAlivePeriod > 0 {
				tcpConn.SetKeepAlivePeriod(opt.KeepAlivePeriod)
			}
		}
	}

	return NewTCPConnection(conn), nil
}

// connectUDP 连接UDP/FILLP服务器
func (c *BaseClient) connectUDP(opt *ClientOption) (NetConnection, error) {
	remoteAddr := &net.UDPAddr{
		IP:   net.ParseIP(opt.RemoteIP),
		Port: opt.RemotePort,
	}

	conn, err := fillp.NewConnection(nil, remoteAddr)
	if err != nil {
		return nil, err
	}

	if err := conn.Connect(); err != nil {
		conn.Close()
		return nil, err
	}

	return NewUDPConnection(conn), nil
}

// ConnectSimple 简化的连接方法
func (c *BaseClient) ConnectSimple(protocol ProtocolType, remoteIP string, remotePort int) (int, error) {
	opt := &ClientOption{
		Protocol:   protocol,
		RemoteIP:   remoteIP,
		RemotePort: remotePort,
		Timeout:    DefaultConnectTimeout,
	}
	return c.Connect(opt)
}

// receiveLoop 接收数据循环
func (c *BaseClient) receiveLoop() {
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()

		if c.callback != nil && c.callback.OnDisconnected != nil {
			c.callback.OnDisconnected(c.fd, ConnectionTypeClient)
		}
	}()

	conn, ok := c.connMgr.GetConn(c.fd)
	if !ok {
		return
	}

	buf := make([]byte, DefaultBufSize)
	offset := 0

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		n, err := conn.Read(buf[offset:])
		if err != nil {
			return
		}

		offset += n
		if c.callback != nil && c.callback.OnDataReceived != nil {
			processed := c.callback.OnDataReceived(c.fd, ConnectionTypeClient, buf, offset)
			if processed < 0 {
				return
			}
			if processed > 0 {
				copy(buf, buf[processed:offset])
				offset -= processed
			}
		}
	}
}

// SendBytes 发送数据
func (c *BaseClient) SendBytes(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return errors.New("not connected")
	}

	return c.connMgr.SendBytes(c.fd, data)
}

// Close 关闭连接
func (c *BaseClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	close(c.stopChan)
	c.connMgr.UnregisterConn(c.fd)
	c.running = false
}

// GetFd 获取虚拟fd
func (c *BaseClient) GetFd() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fd
}

// IsConnected 检查是否已连接
func (c *BaseClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetConnInfo 获取连接信息
func (c *BaseClient) GetConnInfo() *ConnectOption {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return nil
	}

	return c.connMgr.GetConnInfo(c.fd)
}
