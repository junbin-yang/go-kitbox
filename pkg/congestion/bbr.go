package congestion

import (
	"time"
)

// ------------------------------
// BBR拥塞控制算法实现（基于带宽和RTT）
// 特点：不依赖丢包检测，而是基于瓶颈链路带宽和最小RTT调整发送速率
// ------------------------------

type BBRController struct {
	*BaseController
	bwEstimate    float64
	bwSamples     []int
	maxSamples    int
	pacingGain    float64
	cwndGain      float64
	state         string
	probeRTTStart time.Time
	lastProbeRTT  time.Time
}

// BBR状态常量
const (
	BBRStartup  = "STARTUP"   // 启动阶段：快速提升带宽估计
	BBRDrain    = "DRAIN"     // 排水阶段：降低队列占用
	BBRProbeBW  = "PROBE_BW"  // 带宽探测阶段：稳定带宽利用
	BBRProbeRTT = "PROBE_RTT" // RTT探测阶段：定期测量最小RTT
)

func NewBBRController(initialCWnd, maxCWnd, packetSize int) *BBRController {
	return &BBRController{
		BaseController: NewBaseController(initialCWnd, maxCWnd, packetSize),
		bwSamples:      make([]int, 0, 10),
		maxSamples:     10,
		pacingGain:     2.0,
		cwndGain:       2.0,
		state:          BBRStartup,
		lastProbeRTT:   time.Now(),
	}
}

func (b *BBRController) OnPacketSent(packetSize int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inFlight += packetSize
	b.sentBytes += int64(packetSize)
	b.packetsSent++
}

func (b *BBRController) OnAckReceived(ackSize int, rtt time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inFlight -= ackSize
	b.ackedBytes += int64(ackSize)
	b.updateRTT(rtt)

	currentBW := int(float64(ackSize) / rtt.Seconds())
	b.updateBandwidth(currentBW)

	switch b.state {
	case BBRStartup:
		if float64(currentBW) < b.bwEstimate*0.9 {
			b.state = BBRDrain
			b.pacingGain = 0.5
		}
	case BBRDrain:
		if b.inFlight < int(b.bwEstimate*b.minRTT.Seconds()) {
			b.state = BBRProbeBW
			b.pacingGain = 1.25
		}
	case BBRProbeBW:
		if time.Since(b.lastProbeRTT) > 10*time.Second && b.state != BBRProbeRTT {
			b.state = BBRProbeRTT
			b.probeRTTStart = time.Now()
			b.lastProbeRTT = time.Now()
			b.cwnd = b.packetSize * 4
		}
	case BBRProbeRTT:
		if time.Since(b.probeRTTStart) > 200*time.Millisecond {
			b.state = BBRProbeBW
		}
	}

	targetCWnd := int(b.bwEstimate * b.minRTT.Seconds() * b.cwndGain)
	b.setCongestionWindow(targetCWnd)
	b.stats.CurrentState = b.state
	b.updateStats()
}

func (b *BBRController) updateBandwidth(rate int) {
	b.bwSamples = append(b.bwSamples, rate)
	if len(b.bwSamples) > b.maxSamples {
		b.bwSamples = b.bwSamples[1:]
	}
	maxBW := 0
	for _, bw := range b.bwSamples {
		if bw > maxBW {
			maxBW = bw
		}
	}
	b.bwEstimate = float64(maxBW)
}

func (b *BBRController) OnPacketLost() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lostBytes += int64(b.packetSize)
	b.packetsLost++
	b.cwnd = int(float64(b.cwnd) * 0.9)
	b.setCongestionWindow(b.cwnd)
	b.updateStats()
}

func (b *BBRController) GetCongestionWindow() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.cwnd
}

func (b *BBRController) GetSendRate() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return int(b.bwEstimate * b.pacingGain)
}

func (b *BBRController) GetStatistics() CongestionStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stats
}
