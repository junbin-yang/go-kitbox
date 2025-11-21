package fillp

import (
	"net"
	"testing"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/congestion"
)

func TestCongestionIntegration(t *testing.T) {
	tests := []struct {
		name      string
		algorithm congestion.AlgorithmType
	}{
		{"Default (Built-in)", ""},
		{"CUBIC", congestion.AlgorithmCubic},
		{"BBR", congestion.AlgorithmBBR},
		{"Reno", congestion.AlgorithmReno},
		{"Vegas", congestion.AlgorithmVegas},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConnectionConfig{
				CongestionAlgorithm: tt.algorithm,
			}

			localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
			remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

			conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
			if err != nil {
				t.Fatalf("Failed to create connection: %v", err)
			}
			defer conn.Close()

			// 验证拥塞控制设置
			if tt.algorithm == "" {
				if conn.UseExternalCC() {
					t.Error("Expected internal CC, got external")
				}
			} else {
				if !conn.UseExternalCC() {
					t.Error("Expected external CC, got internal")
				}

				// 验证统计信息可用
				stats := conn.GetCongestionStats()
				if stats == nil {
					t.Error("Expected non-nil congestion stats")
				}
			}
		})
	}
}

func TestCubicCustomConfig(t *testing.T) {
	cubicConfig := congestion.CubicConfig{
		Beta: 0.8,
		C:    0.5,
	}

	config := ConnectionConfig{
		CongestionAlgorithm: congestion.AlgorithmCubic,
		CongestionConfig:    cubicConfig,
	}

	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}
	defer conn.Close()

	if !conn.UseExternalCC() {
		t.Error("Expected external CC")
	}
}

func TestQuickConstructors(t *testing.T) {
	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	tests := []struct {
		name string
		fn   func(net.Addr, net.Addr) (*Connection, error)
	}{
		{"CUBIC", NewConnectionWithCUBIC},
		{"BBR", NewConnectionWithBBR},
		{"Vegas", NewConnectionWithVegas},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := tt.fn(localAddr, remoteAddr)
			if err != nil {
				t.Fatalf("Failed to create connection: %v", err)
			}
			defer conn.Close()

			if !conn.UseExternalCC() {
				t.Error("Expected external CC")
			}
		})
	}
}

func TestCongestionAdapter(t *testing.T) {
	config := ConnectionConfig{
		CongestionAlgorithm: congestion.AlgorithmCubic,
	}

	localAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}

	conn, err := NewConnectionWithConfig(localAddr, remoteAddr, config)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}
	defer conn.Close()

	// 模拟数据包发送和ACK
	conn.onPacketSent(1400)
	conn.onAckReceived(1400, 50*time.Millisecond)

	// 获取统计信息
	stats := conn.GetCongestionStats()
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.PacketsSent != 1 {
		t.Errorf("Expected 1 packet sent, got %d", stats.PacketsSent)
	}

	// 模拟丢包
	conn.onPacketLost()
	stats = conn.GetCongestionStats()
	if stats.PacketsLost != 1 {
		t.Errorf("Expected 1 packet lost, got %d", stats.PacketsLost)
	}
}
