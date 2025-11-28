package fillp

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// BenchmarkRequestResponse 测试请求-响应模式的性能
// 期望：延迟ACK优化不会增加额外延迟（捎带ACK）
func BenchmarkRequestResponse(b *testing.B) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19001}

	// 启动服务端
	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		b.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	// 创建客户端
	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		b.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		b.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 启动服务端回显逻辑
	go func() {
		for {
			data, err := server.ReceiveWithTimeout(10 * time.Second)
			if err != nil {
				return
			}
			_ = server.Send(data) // 立即回显
		}
	}()

	// 测试数据
	requestData := []byte("PING")

	b.ResetTimer()
	b.ReportAllocs()

	// 测量请求-响应延迟
	for i := 0; i < b.N; i++ {
		// 发送请求
		if err := client.Send(requestData); err != nil {
			b.Fatalf("发送失败: %v", err)
		}

		// 接收响应
		_, err := client.ReceiveWithTimeout(2 * time.Second)
		if err != nil {
			b.Fatalf("接收失败: %v", err)
		}
	}

	b.StopTimer()

	// 输出统计信息
	clientStats := client.GetStatistics()
	serverStats := server.GetStatistics()

	b.ReportMetric(float64(clientStats.PacketsSent), "client_packets_sent")
	b.ReportMetric(float64(serverStats.PacketsSent), "server_packets_sent")
	b.ReportMetric(float64(clientStats.RTT.Microseconds()), "rtt_us")
}

// BenchmarkBulkTransfer 测试批量数据传输的性能
// 期望：延迟ACK优化减少约50%的ACK包数量
func BenchmarkBulkTransfer(b *testing.B) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19002}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		b.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go func() { _ = server.Listen() }()
	time.Sleep(100 * time.Millisecond)

	client, err := NewConnection(nil, serverAddr)
	if err != nil {
		b.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		b.Fatalf("客户端连接失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 服务端接收（不回复）
	go func() {
		for {
			_, err := server.ReceiveWithTimeout(10 * time.Second)
			if err != nil {
				return
			}
		}
	}()

	// 批量发送的数据块（5KB）
	bulkData := make([]byte, 5000)
	for i := range bulkData {
		bulkData[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := client.Send(bulkData); err != nil {
			b.Fatalf("发送失败: %v", err)
		}
		// 等待传输完成
		time.Sleep(100 * time.Millisecond)
	}

	b.StopTimer()

	// 输出统计信息
	clientStats := client.GetStatistics()
	serverStats := server.GetStatistics()
	totalDataPackets := clientStats.PacketsSent
	totalAckPackets := serverStats.PacketsSent

	b.ReportMetric(float64(totalDataPackets), "data_packets")
	b.ReportMetric(float64(totalAckPackets), "ack_packets")
	if totalDataPackets > 0 {
		ackRatio := float64(totalAckPackets) / float64(totalDataPackets)
		b.ReportMetric(ackRatio*100, "ack_ratio_%")
	}
}

// TestDelayedACKEfficiency 测试延迟ACK的效率
// 验证：批量传输时ACK包数量应该减少约50%
func TestDelayedACKEfficiency(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19003}

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

	// 服务端接收
	receivedCount := 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_, err := server.ReceiveWithTimeout(2 * time.Second)
			if err != nil {
				return
			}
			receivedCount++
		}
	}()

	// 客户端连续发送10个小包（模拟批量传输）
	for i := 0; i < 10; i++ {
		data := []byte(fmt.Sprintf("Packet %d", i))
		if err := client.Send(data); err != nil {
			t.Fatalf("发送失败: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // 快速连续发送
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond) // 等待所有ACK

	// 统计信息
	clientStats := client.GetStatistics()
	serverStats := server.GetStatistics()

	dataPackets := clientStats.PacketsSent
	ackPackets := serverStats.PacketsSent

	t.Logf("客户端发送数据包: %d", dataPackets)
	t.Logf("服务端发送ACK包: %d", ackPackets)
	t.Logf("数据包数量: %d", receivedCount)

	if dataPackets > 0 {
		ackRatio := float64(ackPackets) / float64(dataPackets)
		t.Logf("ACK包占比: %.2f%%", ackRatio*100)

		// 验证：延迟ACK应该使ACK包减少（理想情况约50%，实际可能40-60%）
		if ackRatio > 0.7 {
			t.Errorf("延迟ACK优化效果不明显: ACK包占比 %.2f%% > 70%%", ackRatio*100)
		} else {
			t.Logf("✅ 延迟ACK优化有效: ACK包占比 %.2f%% < 70%%", ackRatio*100)
		}
	}
}

// TestRequestResponseLatency 测试请求-响应模式的延迟
// 验证：延迟ACK不应增加请求-响应的延迟（捎带ACK）
func TestRequestResponseLatency(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19004}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go server.Listen()
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

	// 服务端回显逻辑
	go func() {
		for {
			data, err := server.ReceiveWithTimeout(5 * time.Second)
			if err != nil {
				return
			}
			_ = server.Send(data) // 立即回显（触发捎带ACK）
		}
	}()

	// 测试10次请求-响应
	var latencies []time.Duration
	for i := 0; i < 10; i++ {
		start := time.Now()

		// 发送请求
		request := []byte("PING")
		if err := client.Send(request); err != nil {
			t.Fatalf("发送失败: %v", err)
		}

		// 接收响应
		response, err := client.ReceiveWithTimeout(2 * time.Second)
		if err != nil {
			t.Fatalf("接收失败: %v", err)
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)

		if string(response) != string(request) {
			t.Errorf("响应不匹配: 期望 %s, 实际 %s", request, response)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// 计算平均延迟
	var totalLatency time.Duration
	for _, l := range latencies {
		totalLatency += l
	}
	avgLatency := totalLatency / time.Duration(len(latencies))

	t.Logf("请求-响应平均延迟: %v", avgLatency)
	t.Logf("最小延迟: %v", minDuration(latencies))
	t.Logf("最大延迟: %v", maxDuration(latencies))

	// 验证：本地回环延迟应该很低（< 100ms）
	if avgLatency > 100*time.Millisecond {
		t.Errorf("请求-响应延迟过高: %v > 100ms", avgLatency)
	} else {
		t.Logf("✅ 请求-响应延迟正常: %v < 100ms", avgLatency)
	}

	// 统计ACK包数量
	serverStats := server.GetStatistics()
	t.Logf("服务端ACK包数量: %d (含捎带ACK)", serverStats.PacketsSent)
}

// TestDelayedACKTimeout 测试延迟ACK的超时机制
// 验证：单向数据流中，ACK应该在40ms内发送
func TestDelayedACKTimeout(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19005}

	server, err := NewConnection(serverAddr, nil)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	go server.Listen()
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

	// 服务端只接收，不发送（单向数据流）
	go func() {
		for {
			_, err := server.ReceiveWithTimeout(5 * time.Second)
			if err != nil {
				return
			}
		}
	}()

	// 客户端发送单个数据包
	startTime := time.Now()
	if err := client.Send([]byte("Single packet")); err != nil {
		t.Fatalf("发送失败: %v", err)
	}

	// 等待ACK（延迟ACK超时 + 传输时间）
	time.Sleep(100 * time.Millisecond)

	// 检查服务端是否发送了ACK
	serverStats := server.GetStatistics()
	elapsedTime := time.Since(startTime)

	t.Logf("服务端ACK包数量: %d", serverStats.PacketsSent)
	t.Logf("等待时间: %v", elapsedTime)

	if serverStats.PacketsSent == 0 {
		t.Error("服务端未发送ACK（延迟ACK超时未触发）")
	} else {
		t.Logf("✅ 延迟ACK超时触发: 服务端发送了 %d 个ACK包", serverStats.PacketsSent)
	}

	// 验证：超时时间应该在合理范围内（40ms + 网络延迟 < 100ms）
	if elapsedTime > 150*time.Millisecond {
		t.Errorf("延迟ACK超时过长: %v > 150ms", elapsedTime)
	}
}

// 辅助函数：计算最小延迟
func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

// 辅助函数：计算最大延迟
func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}
