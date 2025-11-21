package netconn

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/junbin-yang/go-kitbox/pkg/fillp"
)

// BaseServer 统一的服务器
type BaseServer struct {
	connMgr      *ConnectionManager
	callback     *BaseListenerCallback
	protocol     ProtocolType
	listener     net.Listener // TCP监听器
	addr         string
	port         int
	running      bool
	mu           sync.RWMutex
	stopChan     chan struct{}
	connHandlers map[int]chan struct{} // 每个连接的停止通道
	handlerMu    sync.Mutex
}

// NewBaseServer 创建服务器实例
func NewBaseServer(connMgr *ConnectionManager) *BaseServer {
	if connMgr == nil {
		connMgr = NewConnectionManager()
	}
	return &BaseServer{
		connMgr:      connMgr,
		connHandlers: make(map[int]chan struct{}),
	}
}

// StartBaseListener 启动服务器监听
func (s *BaseServer) StartBaseListener(opt *ServerOption, callback *BaseListenerCallback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("server already running")
	}

	s.callback = callback
	s.protocol = opt.Protocol
	s.addr = opt.Addr
	s.port = opt.Port
	s.stopChan = make(chan struct{})

	var err error
	if opt.Protocol == ProtocolTCP {
		err = s.startTCPListener(opt.Addr, opt.Port)
	} else {
		err = s.startUDPListener(opt.Addr, opt.Port)
	}

	if err != nil {
		return err
	}

	s.running = true
	return nil
}

// startTCPListener 启动TCP监听
func (s *BaseServer) startTCPListener(addr string, port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}

	s.listener = listener

	// 触发服务端监听回调
	if s.callback != nil && s.callback.OnConnected != nil {
		connOpt := &ConnectOption{
			Protocol:     ProtocolTCP,
			LocalSocket:  &SocketOption{Addr: addr, Port: port},
			RemoteSocket: &SocketOption{},
		}
		s.callback.OnConnected(0, ConnectionTypeServer, connOpt)
	}

	go s.acceptTCPLoop()
	return nil
}

// startUDPListener 启动UDP/FILLP监听
func (s *BaseServer) startUDPListener(addr string, port int) error {
	localAddr := &net.UDPAddr{
		IP:   net.ParseIP(addr),
		Port: port,
	}

	// 触发服务端监听回调
	if s.callback != nil && s.callback.OnConnected != nil {
		connOpt := &ConnectOption{
			Protocol:     ProtocolUDP,
			LocalSocket:  &SocketOption{Addr: addr, Port: port},
			RemoteSocket: &SocketOption{},
		}
		s.callback.OnConnected(0, ConnectionTypeServer, connOpt)
	}

	// 启动接受连接循环
	go s.acceptUDPLoop(localAddr)
	return nil
}

// acceptUDPLoop UDP/FILLP接受连接循环
func (s *BaseServer) acceptUDPLoop(localAddr *net.UDPAddr) {
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		conn, err := fillp.NewConnection(localAddr, nil)
		if err != nil {
			continue
		}

		if err := conn.Listen(); err != nil {
			conn.Close()
			continue
		}

		go s.handleUDPConnection(conn)
	}
}

// acceptTCPLoop TCP接受连接循环
func (s *BaseServer) acceptTCPLoop() {
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				continue
			}
		}

		go s.handleTCPConnection(conn)
	}
}

// handleTCPConnection 处理TCP连接
func (s *BaseServer) handleTCPConnection(conn net.Conn) {
	tcpConn := NewTCPConnection(conn)
	fd := s.connMgr.RegisterConn(tcpConn, ConnectionTypeClient)
	defer s.connMgr.UnregisterConn(fd)

	// 触发客户端连接回调
	if s.callback != nil && s.callback.OnConnected != nil {
		connOpt := s.connMgr.GetConnInfo(fd)
		s.callback.OnConnected(fd, ConnectionTypeClient, connOpt)
	}

	// 创建停止通道
	stopChan := make(chan struct{})
	s.handlerMu.Lock()
	s.connHandlers[fd] = stopChan
	s.handlerMu.Unlock()

	defer func() {
		s.handlerMu.Lock()
		delete(s.connHandlers, fd)
		s.handlerMu.Unlock()

		if s.callback != nil && s.callback.OnDisconnected != nil {
			s.callback.OnDisconnected(fd, ConnectionTypeClient)
		}
	}()

	// 接收数据循环
	buf := make([]byte, DefaultBufSize)
	offset := 0

	for {
		select {
		case <-stopChan:
			return
		default:
		}

		n, err := conn.Read(buf[offset:])
		if err != nil {
			return
		}

		offset += n
		if s.callback != nil && s.callback.OnDataReceived != nil {
			processed := s.callback.OnDataReceived(fd, ConnectionTypeServer, buf, offset)
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

// handleUDPConnection 处理UDP/FILLP连接
func (s *BaseServer) handleUDPConnection(conn *fillp.Connection) {
	udpConn := NewUDPConnection(conn)
	fd := s.connMgr.RegisterConn(udpConn, ConnectionTypeClient)
	defer s.connMgr.UnregisterConn(fd)

	// 触发客户端连接回调
	if s.callback != nil && s.callback.OnConnected != nil {
		connOpt := s.connMgr.GetConnInfo(fd)
		s.callback.OnConnected(fd, ConnectionTypeClient, connOpt)
	}

	// 创建停止通道
	stopChan := make(chan struct{})
	s.handlerMu.Lock()
	s.connHandlers[fd] = stopChan
	s.handlerMu.Unlock()

	defer func() {
		s.handlerMu.Lock()
		delete(s.connHandlers, fd)
		s.handlerMu.Unlock()

		if s.callback != nil && s.callback.OnDisconnected != nil {
			s.callback.OnDisconnected(fd, ConnectionTypeClient)
		}
	}()

	buf := make([]byte, DefaultBufSize)
	offset := 0

	for {
		select {
		case <-stopChan:
			return
		default:
		}

		data, err := conn.Receive()
		if err != nil {
			return
		}

		copy(buf[offset:], data)
		offset += len(data)

		if s.callback != nil && s.callback.OnDataReceived != nil {
			processed := s.callback.OnDataReceived(fd, ConnectionTypeServer, buf, offset)
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

// StopBaseListener 停止服务器
func (s *BaseServer) StopBaseListener() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.stopChan)

	// 停止所有连接处理
	s.handlerMu.Lock()
	for _, stopChan := range s.connHandlers {
		close(stopChan)
	}
	s.connHandlers = make(map[int]chan struct{})
	s.handlerMu.Unlock()

	if s.protocol == ProtocolTCP && s.listener != nil {
		s.listener.Close()
	}

	s.running = false
	return nil
}

// SendBytes 向指定fd发送数据
func (s *BaseServer) SendBytes(fd int, data []byte) error {
	return s.connMgr.SendBytes(fd, data)
}

// GetConnInfo 获取连接信息
func (s *BaseServer) GetConnInfo(fd int) *ConnectOption {
	return s.connMgr.GetConnInfo(fd)
}

// GetPort 获取监听端口
func (s *BaseServer) GetPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.running {
		return -1
	}
	return s.port
}

// GetAddr 获取监听地址
func (s *BaseServer) GetAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.running {
		return ""
	}
	return s.addr
}
