package fillp

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// 场景1：创建连接
func TestScenario1_NewConnection(t *testing.T) {
	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8001}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8002}

	conn, err := NewConnection(localAddr, remoteAddr)
	if err != nil {
		t.Fatalf("创建连接失败: %v", err)
	}

	if conn.localAddr.String() != localAddr.String() {
		t.Errorf("本地地址不匹配")
	}

	if conn.remoteAddr.String() != remoteAddr.String() {
		t.Errorf("远程地址不匹配")
	}
}

// 场景2：客户端-服务端连接建立
func TestScenario2_ClientServerConnect(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9001}
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}

	// 创建服务端
	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	// 启动服务端监听
	serverReady := make(chan error, 1)
	go func() {
		serverReady <- server.Listen()
	}()

	// 等待服务端启动
	time.Sleep(100 * time.Millisecond)

	// 创建客户端
	client, err := NewConnection(clientAddr, serverAddr)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 客户端连接
	if err := client.Connect(); err != nil {
		t.Fatalf("客户端连接失败: %v", err)
	}

	// 等待服务端接受连接
	select {
	case err := <-serverReady:
		if err != nil {
			t.Fatalf("服务端监听失败: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("服务端连接超时")
	}
}

// 场景3：数据发送和接收
func TestScenario3_SendReceive(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9002}

	// 创建服务端
	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	// 启动服务端
	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	// 创建客户端
	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 客户端连接
	if err := client.Connect(); err != nil {
		t.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 客户端发送数据
	testData := []byte("Hello FILLP")
	if err := client.Send(testData); err != nil {
		t.Fatalf("发送数据失败: %v", err)
	}

	// 服务端接收数据
	received, err := server.ReceiveWithTimeout(2 * time.Second)
	if err != nil {
		t.Fatalf("接收数据失败: %v", err)
	}

	if string(received) != string(testData) {
		t.Errorf("接收数据不匹配，期望 %s，实际 %s", testData, received)
	}
}

// 场景4：大数据传输（分片）
func TestScenario4_LargeDataTransfer(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9003}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 发送大数据（超过MTU）
	largeData := make([]byte, 5000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	if err := client.Send(largeData); err != nil {
		t.Fatalf("发送大数据失败: %v", err)
	}

	// 接收所有数据
	var received []byte
	timeout := time.After(5 * time.Second)
	for len(received) < len(largeData) {
		select {
		case <-timeout:
			t.Fatalf("接收超时，期望 %d 字节，实际 %d 字节", len(largeData), len(received))
		default:
			data, err := server.ReceiveWithTimeout(1 * time.Second)
			if err != nil {
				continue
			}
			received = append(received, data...)
		}
	}

	if len(received) != len(largeData) {
		t.Errorf("数据长度不匹配，期望 %d，实际 %d", len(largeData), len(received))
	}
}

// 场景5：连接关闭
func TestScenario5_CloseConnection(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9004}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 关闭客户端
	if err := client.Close(); err != nil {
		t.Errorf("关闭客户端失败: %v", err)
	}

	// 验证状态
	if client.state != StateClosed {
		t.Errorf("客户端状态应为已关闭")
	}

	server.Close()
}

// 场景6：获取统计信息
func TestScenario6_GetStatistics(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9005}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 发送数据
	testData := []byte("Statistics test")
	_ = client.Send(testData)
	time.Sleep(500 * time.Millisecond)

	// 获取统计信息
	stats := client.GetStatistics()
	if stats.PacketsSent == 0 {
		t.Error("应该有发送的数据包")
	}
}

// 场景7：流量控制
func TestScenario7_FlowControl(t *testing.T) {
	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9006}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9007}

	conn, err := NewConnection(localAddr, remoteAddr)
	if err != nil {
		t.Fatalf("创建连接失败: %v", err)
	}
	defer conn.Close()

	// 设置流量控制窗口
	newWindow := uint32(32768)
	conn.SetFlowControl(newWindow)

	if conn.sendWindow != newWindow {
		t.Errorf("流量控制窗口设置失败，期望 %d，实际 %d", newWindow, conn.sendWindow)
	}
}

// 场景8：RingBuffer 边界测试
func TestScenario8_RingBufferBoundary(t *testing.T) {
	buffer := NewRingBuffer(10)

	// 写入满缓冲区
	data := []byte("1234567890")
	if err := buffer.Write(data); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	// 尝试写入超出容量
	if err := buffer.Write([]byte("X")); err == nil {
		t.Error("应该返回空间不足错误")
	}

	// 读取部分数据
	read, _ := buffer.Read(5)
	if string(read) != "12345" {
		t.Errorf("读取数据不匹配")
	}

	// 再次写入
	if err := buffer.Write([]byte("ABCDE")); err != nil {
		t.Errorf("写入失败: %v", err)
	}
}

// 场景9：RetransmissionQueue 超时测试
func TestScenario9_RetransmissionTimeout(t *testing.T) {
	rq := NewRetransmissionQueue()

	now := time.Now().UnixMilli()
	rq.Add(1, []byte("test"), now)

	// 立即检查，不应超时
	expired := rq.GetExpired(now)
	if len(expired) != 0 {
		t.Error("不应有超时数据包")
	}

	// 等待超时
	time.Sleep(250 * time.Millisecond)
	expired = rq.GetExpired(time.Now().UnixMilli())
	if len(expired) != 1 {
		t.Errorf("应有1个超时数据包，实际 %d", len(expired))
	}
}

// 场景10：并发发送接收
func TestScenario10_ConcurrentSendReceive(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9008}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 并发发送多条消息
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(n int) {
			data := []byte(fmt.Sprintf("Message %d", n))
			if err := client.Send(data); err != nil {
				t.Errorf("发送失败: %v", err)
			}
			done <- true
		}(i)
	}

	// 等待所有发送完成
	for i := 0; i < 5; i++ {
		<-done
	}

	time.Sleep(1 * time.Second)
}
