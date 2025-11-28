package netconn

import (
	"testing"
	"time"
)

func TestTCPClientServer(t *testing.T) {
	// 创建服务器
	server := NewBaseServer(nil)
	serverCallback := &BaseListenerCallback{
		OnConnected: func(fd int, connType ConnectionType, connOpt *ConnectOption) {
			t.Logf("Server: client connected, fd=%d", fd)
		},
		OnDisconnected: func(fd int, connType ConnectionType) {
			t.Logf("Server: client disconnected, fd=%d", fd)
		},
		OnDataReceived: func(fd int, connType ConnectionType, buf []byte, used int) int {
			t.Logf("Server: received data from fd=%d: %s", fd, string(buf[:used]))
			_ = server.SendBytes(fd, buf[:used]) // 回显
			return used
		},
	}

	opt := &ServerOption{
		Protocol: ProtocolTCP,
		Addr:     "127.0.0.1",
		Port:     18080,
	}

	if err := server.StartBaseListener(opt, serverCallback); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() { _ = server.StopBaseListener() }()

	time.Sleep(100 * time.Millisecond)

	// 创建客户端
	clientCallback := &BaseListenerCallback{
		OnConnected: func(fd int, connType ConnectionType, connOpt *ConnectOption) {
			t.Logf("Client: connected, fd=%d", fd)
		},
		OnDisconnected: func(fd int, connType ConnectionType) {
			t.Logf("Client: disconnected, fd=%d", fd)
		},
		OnDataReceived: func(fd int, connType ConnectionType, buf []byte, used int) int {
			t.Logf("Client: received data: %s", string(buf[:used]))
			return used
		},
	}

	client := NewBaseClient(nil, clientCallback)
	fd, err := client.ConnectSimple(ProtocolTCP, "127.0.0.1", 18080)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	t.Logf("Client connected with fd=%d", fd)

	// 发送数据
	if err := client.SendBytes([]byte("Hello Server")); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
}

func TestGlobalFdAllocation(t *testing.T) {
	mgr := NewConnectionManager()

	// 分配多个FD，验证全局自增
	fd1 := mgr.AllocateFd(ConnectionTypeServer)
	fd2 := mgr.AllocateFd(ConnectionTypeClient)
	fd3 := mgr.AllocateFd(ConnectionTypeServer)
	fd4 := mgr.AllocateFd(ConnectionTypeClient)

	t.Logf("Allocated FDs: %d, %d, %d, %d", fd1, fd2, fd3, fd4)

	// 验证FD唯一且递增
	if fd1 >= fd2 || fd2 >= fd3 || fd3 >= fd4 {
		t.Errorf("FDs should be strictly increasing")
	}

	// 验证无冲突（即使分配1000+个）
	fds := make(map[int]bool)
	for i := 0; i < 2000; i++ {
		fd := mgr.AllocateFd(ConnectionTypeServer)
		if fds[fd] {
			t.Errorf("FD %d allocated twice", fd)
		}
		fds[fd] = true
	}

	t.Logf("Successfully allocated 2000+ unique FDs without conflict")
}

func TestClientGetters(t *testing.T) {
	server := NewBaseServer(nil)
	opt := &ServerOption{Protocol: ProtocolTCP, Addr: "127.0.0.1", Port: 18081}
	_ = server.StartBaseListener(opt, &BaseListenerCallback{})
	defer func() { _ = server.StopBaseListener() }()
	time.Sleep(50 * time.Millisecond)

	client := NewBaseClient(nil, &BaseListenerCallback{})
	fd, _ := client.ConnectSimple(ProtocolTCP, "127.0.0.1", 18081)
	defer client.Close()

	if client.GetFd() != fd {
		t.Errorf("GetFd() = %d, want %d", client.GetFd(), fd)
	}
	if !client.IsConnected() {
		t.Error("IsConnected() = false, want true")
	}
	if info := client.GetConnInfo(); info == nil {
		t.Error("GetConnInfo() = nil, want non-nil")
	}
}

func TestServerGetters(t *testing.T) {
	server := NewBaseServer(nil)
	opt := &ServerOption{Protocol: ProtocolTCP, Addr: "127.0.0.1", Port: 18082}
	_ = server.StartBaseListener(opt, &BaseListenerCallback{})
	defer func() { _ = server.StopBaseListener() }()

	if server.GetPort() != 18082 {
		t.Errorf("GetPort() = %d, want 18082", server.GetPort())
	}
	if server.GetAddr() != "127.0.0.1" {
		t.Errorf("GetAddr() = %s, want 127.0.0.1", server.GetAddr())
	}
}

func TestManagerMethods(t *testing.T) {
	mgr := NewConnectionManager()
	server := NewBaseServer(mgr)
	opt := &ServerOption{Protocol: ProtocolTCP, Addr: "127.0.0.1", Port: 18083}
	_ = server.StartBaseListener(opt, &BaseListenerCallback{})
	defer func() { _ = server.StopBaseListener() }()
	time.Sleep(50 * time.Millisecond)

	client := NewBaseClient(mgr, &BaseListenerCallback{})
	client.ConnectSimple(ProtocolTCP, "127.0.0.1", 18083)
	time.Sleep(100 * time.Millisecond)

	if mgr.GetConnCount() == 0 {
		t.Error("GetConnCount() = 0, want > 0")
	}
	fds := mgr.GetAllFds()
	if len(fds) == 0 {
		t.Error("GetAllFds() returned empty slice")
	}
	if info := server.GetConnInfo(fds[0]); info == nil {
		t.Error("GetConnInfo() = nil")
	}

	mgr.CloseAll()
	if mgr.GetConnCount() != 0 {
		t.Errorf("GetConnCount() after CloseAll = %d, want 0", mgr.GetConnCount())
	}
}
