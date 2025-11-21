package congestion

import (
	"testing"
	"time"
)

func TestNewController(t *testing.T) {
	tests := []struct {
		name      string
		algorithm AlgorithmType
		wantErr   bool
	}{
		{"CUBIC", AlgorithmCubic, false},
		{"BBR", AlgorithmBBR, false},
		{"Reno", AlgorithmReno, false},
		{"Vegas", AlgorithmVegas, false},
		{"Invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, err := NewController(tt.algorithm, 1400, 65536, 1400)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewController() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && ctrl == nil {
				t.Error("Expected non-nil controller")
			}
		})
	}
}

func TestCubicController(t *testing.T) {
	ctrl := NewCubicController(2800, 65536, 1400)

	// 测试初始状态
	if ctrl.GetCongestionWindow() != 2800 {
		t.Errorf("Initial cwnd = %d, want 2800", ctrl.GetCongestionWindow())
	}

	// 模拟发送数据包
	ctrl.OnPacketSent(1400)
	if ctrl.inFlight != 1400 {
		t.Errorf("InFlight = %d, want 1400", ctrl.inFlight)
	}

	// 模拟收到ACK（慢启动阶段）
	ctrl.OnAckReceived(1400, 50*time.Millisecond)
	if ctrl.GetCongestionWindow() <= 2800 {
		t.Error("Expected cwnd to increase in slow start")
	}

	// 模拟丢包
	oldCwnd := ctrl.GetCongestionWindow()
	ctrl.OnPacketLost()
	if ctrl.GetCongestionWindow() >= oldCwnd {
		t.Error("Expected cwnd to decrease on packet loss")
	}
}

func TestBBRController(t *testing.T) {
	ctrl := NewBBRController(2800, 65536, 1400)

	// 测试初始状态
	if ctrl.state != BBRStartup {
		t.Errorf("Initial state = %s, want %s", ctrl.state, BBRStartup)
	}

	// 模拟发送和确认
	for i := 0; i < 10; i++ {
		ctrl.OnPacketSent(1400)
		ctrl.OnAckReceived(1400, 50*time.Millisecond)
	}

	// 验证带宽估计
	if ctrl.bwEstimate == 0 {
		t.Error("Expected non-zero bandwidth estimate")
	}

	// BBR对丢包不敏感
	oldCwnd := ctrl.GetCongestionWindow()
	ctrl.OnPacketLost()
	reduction := float64(oldCwnd-ctrl.GetCongestionWindow()) / float64(oldCwnd)
	if reduction > 0.2 {
		t.Errorf("BBR reduced cwnd by %.1f%%, expected < 20%%", reduction*100)
	}
}

func TestBaseControllerRTT(t *testing.T) {
	base := NewBaseController(2800, 65536, 1400)

	// 第一次RTT测量
	base.updateRTT(100 * time.Millisecond)
	if base.rtt != 100*time.Millisecond {
		t.Errorf("First RTT = %v, want 100ms", base.rtt)
	}
	if base.minRTT != 100*time.Millisecond {
		t.Errorf("MinRTT = %v, want 100ms", base.minRTT)
	}

	// 第二次RTT测量（更小）
	base.updateRTT(50 * time.Millisecond)
	if base.minRTT != 50*time.Millisecond {
		t.Errorf("MinRTT = %v, want 50ms", base.minRTT)
	}

	// 平滑RTT应该在50-100ms之间
	if base.rtt < 50*time.Millisecond || base.rtt > 100*time.Millisecond {
		t.Errorf("Smoothed RTT = %v, expected between 50-100ms", base.rtt)
	}
}

func TestCongestionStats(t *testing.T) {
	ctrl := NewCubicController(2800, 65536, 1400)

	for i := 0; i < 5; i++ {
		ctrl.OnPacketSent(1400)
		ctrl.OnAckReceived(1400, 50*time.Millisecond)
	}

	ctrl.OnPacketLost()

	stats := ctrl.GetStatistics()
	if stats.CongestionWindow == 0 {
		t.Error("Expected non-zero congestion window in stats")
	}
	if stats.RTT == 0 {
		t.Error("Expected non-zero RTT in stats")
	}
	if stats.LossRate == 0 {
		t.Error("Expected non-zero loss rate after packet loss")
	}
	if stats.PacketsSent != 5 {
		t.Errorf("Expected 5 packets sent, got %d", stats.PacketsSent)
	}
	if stats.PacketsLost != 1 {
		t.Errorf("Expected 1 packet lost, got %d", stats.PacketsLost)
	}
}

func TestCubicConfig(t *testing.T) {
	config := CubicConfig{Beta: 0.8, C: 0.5}
	ctrl := NewCubicControllerWithConfig(config, 2800, 65536, 1400)

	ctrl.OnPacketSent(1400)
	ctrl.OnAckReceived(1400, 50*time.Millisecond)

	oldCwnd := ctrl.GetCongestionWindow()
	ctrl.OnPacketLost()
	newCwnd := ctrl.GetCongestionWindow()

	reduction := float64(oldCwnd-newCwnd) / float64(oldCwnd)
	if reduction < 0.15 || reduction > 0.25 {
		t.Errorf("Expected ~20%% reduction (beta=0.8), got %.1f%%", reduction*100)
	}
}

func TestWindowLowerBound(t *testing.T) {
	ctrl := NewCubicController(2800, 65536, 1400)

	for i := 0; i < 10; i++ {
		ctrl.OnPacketLost()
	}

	cwnd := ctrl.GetCongestionWindow()
	minCwnd := 2 * 1400
	if cwnd < minCwnd {
		t.Errorf("Window %d below minimum %d", cwnd, minCwnd)
	}
}

func TestVegasBoundaryCondition(t *testing.T) {
	ctrl := NewVegasController(2800, 65536, 1400)

	ctrl.OnPacketSent(1400)
	ctrl.OnAckReceived(1400, 50*time.Millisecond)

	if ctrl.GetCongestionWindow() == 0 {
		t.Error("Vegas should handle minRTT=0 gracefully")
	}
}

func TestBBRBandwidthEstimation(t *testing.T) {
	ctrl := NewBBRController(2800, 65536, 1400)

	for i := 0; i < 15; i++ {
		ctrl.OnPacketSent(1400)
		ctrl.OnAckReceived(1400, 50*time.Millisecond)
	}

	stats := ctrl.GetStatistics()
	if stats.CurrentState == "" {
		t.Error("Expected BBR state to be set")
	}

	rate := ctrl.GetSendRate()
	if rate <= 0 {
		t.Errorf("Expected positive send rate, got %d", rate)
	}
}
