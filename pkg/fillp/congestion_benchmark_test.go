package fillp

import (
	"net"
	"testing"

	"github.com/junbin-yang/go-kitbox/pkg/congestion"
)

// BenchmarkCongestionAlgorithms 对比不同拥塞控制算法的性能
func BenchmarkCongestionAlgorithms(b *testing.B) {
	algorithms := []struct {
		name      string
		algorithm congestion.AlgorithmType
	}{
		{"Built-in", ""},
		{"CUBIC", congestion.AlgorithmCubic},
		{"BBR", congestion.AlgorithmBBR},
		{"Reno", congestion.AlgorithmReno},
		{"Vegas", congestion.AlgorithmVegas},
	}

	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	for _, algo := range algorithms {
		b.Run(algo.name, func(b *testing.B) {
			config := ConnectionConfig{
				CongestionAlgorithm: algo.algorithm,
			}

			conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
			if err != nil {
				b.Fatalf("Failed to create connection: %v", err)
			}
			defer conn.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// 模拟数据包发送
				conn.onPacketSent(1400)
				// 模拟ACK接收
				conn.onAckReceived(1400, 50*1000) // 50ms
				// 获取拥塞窗口
				_ = conn.getCongestionWindow()
			}
		})
	}
}

// BenchmarkCongestionWindowQuery 测试拥塞窗口查询性能
func BenchmarkCongestionWindowQuery(b *testing.B) {
	algorithms := []struct {
		name      string
		algorithm congestion.AlgorithmType
	}{
		{"Built-in", ""},
		{"CUBIC", congestion.AlgorithmCubic},
	}

	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	for _, algo := range algorithms {
		b.Run(algo.name, func(b *testing.B) {
			config := ConnectionConfig{
				CongestionAlgorithm: algo.algorithm,
			}

			conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
			if err != nil {
				b.Fatalf("Failed to create connection: %v", err)
			}
			defer conn.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = conn.getCongestionWindow()
			}
		})
	}
}

// BenchmarkPacketSent 测试数据包发送通知性能
func BenchmarkPacketSent(b *testing.B) {
	algorithms := []struct {
		name      string
		algorithm congestion.AlgorithmType
	}{
		{"Built-in", ""},
		{"CUBIC", congestion.AlgorithmCubic},
		{"BBR", congestion.AlgorithmBBR},
	}

	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	for _, algo := range algorithms {
		b.Run(algo.name, func(b *testing.B) {
			config := ConnectionConfig{
				CongestionAlgorithm: algo.algorithm,
			}

			conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
			if err != nil {
				b.Fatalf("Failed to create connection: %v", err)
			}
			defer conn.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn.onPacketSent(1400)
			}
		})
	}
}

// BenchmarkAckReceived 测试ACK接收处理性能
func BenchmarkAckReceived(b *testing.B) {
	algorithms := []struct {
		name      string
		algorithm congestion.AlgorithmType
	}{
		{"Built-in", ""},
		{"CUBIC", congestion.AlgorithmCubic},
		{"BBR", congestion.AlgorithmBBR},
	}

	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	for _, algo := range algorithms {
		b.Run(algo.name, func(b *testing.B) {
			config := ConnectionConfig{
				CongestionAlgorithm: algo.algorithm,
			}

			conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
			if err != nil {
				b.Fatalf("Failed to create connection: %v", err)
			}
			defer conn.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn.onAckReceived(1400, 50*1000)
			}
		})
	}
}
