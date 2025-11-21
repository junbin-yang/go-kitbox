package netconn

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// ConnectionManager 统一的连接管理器
type ConnectionManager struct {
	connMap     map[int]NetConnection
	connTypeMap map[int]ConnectionType
	mu          sync.RWMutex
	nextFd      int64 // 全局自增FD，从1000开始
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connMap:     make(map[int]NetConnection),
		connTypeMap: make(map[int]ConnectionType),
		nextFd:      1000,
	}
}

// AllocateFd 分配虚拟文件描述符（全局自增）
func (m *ConnectionManager) AllocateFd(connType ConnectionType) int {
	fd := int(atomic.AddInt64(&m.nextFd, 1))
	return fd
}

// RegisterConn 注册连接并分配虚拟文件描述符
func (m *ConnectionManager) RegisterConn(conn NetConnection, connType ConnectionType) int {
	fd := m.AllocateFd(connType)
	m.mu.Lock()
	m.connMap[fd] = conn
	m.connTypeMap[fd] = connType
	m.mu.Unlock()
	return fd
}

// UnregisterConn 注销连接
func (m *ConnectionManager) UnregisterConn(fd int) {
	m.mu.Lock()
	if conn, ok := m.connMap[fd]; ok {
		conn.Close()
		delete(m.connMap, fd)
		delete(m.connTypeMap, fd)
	}
	m.mu.Unlock()
}

// GetConn 通过虚拟文件描述符获取连接
func (m *ConnectionManager) GetConn(fd int) (NetConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connMap[fd]
	return conn, ok
}

// GetConnType 获取连接类型
func (m *ConnectionManager) GetConnType(fd int) (ConnectionType, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	connType, ok := m.connTypeMap[fd]
	return connType, ok
}

// CloseConn 关闭指定连接
func (m *ConnectionManager) CloseConn(fd int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connMap[fd]
	if !ok {
		return errors.New("connection not found")
	}

	err := conn.Close()
	delete(m.connMap, fd)
	delete(m.connTypeMap, fd)
	return err
}

// CloseAll 关闭所有连接
func (m *ConnectionManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connMap {
		conn.Close()
	}
	m.connMap = make(map[int]NetConnection)
	m.connTypeMap = make(map[int]ConnectionType)
}

// SendBytes 通过虚拟fd发送数据
func (m *ConnectionManager) SendBytes(fd int, data []byte) error {
	conn, ok := m.GetConn(fd)
	if !ok {
		return errors.New("connection not found")
	}

	_, err := conn.Write(data)
	return err
}

// GetConnInfo 获取连接的完整信息
func (m *ConnectionManager) GetConnInfo(fd int) *ConnectOption {
	conn, ok := m.GetConn(fd)
	if !ok {
		return nil
	}

	localAddr := conn.LocalAddr()
	remoteAddr := conn.RemoteAddr()

	connType, _ := m.GetConnType(fd)
	protocol := ProtocolTCP
	if connType == ConnectionTypeClient {
		if _, ok := conn.(*UDPConnection); ok {
			protocol = ProtocolUDP
		}
	}

	return &ConnectOption{
		Protocol:     protocol,
		LocalSocket:  addrToSocketOption(localAddr),
		RemoteSocket: addrToSocketOption(remoteAddr),
		NetConn:      conn,
	}
}

// GetAllFds 获取所有活跃的虚拟fd
func (m *ConnectionManager) GetAllFds() []int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fds := make([]int, 0, len(m.connMap))
	for fd := range m.connMap {
		fds = append(fds, fd)
	}
	return fds
}

// GetConnCount 获取连接总数
func (m *ConnectionManager) GetConnCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connMap)
}

// 辅助函数：将net.Addr转换为SocketOption
func addrToSocketOption(addr net.Addr) *SocketOption {
	if addr == nil {
		return &SocketOption{}
	}

	switch a := addr.(type) {
	case *net.TCPAddr:
		return &SocketOption{
			Addr: a.IP.String(),
			Port: a.Port,
		}
	case *net.UDPAddr:
		return &SocketOption{
			Addr: a.IP.String(),
			Port: a.Port,
		}
	default:
		host, port := parseAddr(addr.String())
		return &SocketOption{
			Addr: host,
			Port: port,
		}
	}
}

// 解析地址字符串
func parseAddr(addr string) (string, int) {
	var host string
	var port int
	fmt.Sscanf(addr, "%s:%d", &host, &port)
	return host, port
}
