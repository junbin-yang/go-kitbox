package congestion

import (
	"time"
)

// ------------------------------
// Vegas拥塞控制算法实现（基于延迟的拥塞控制）
// 特点：通过比较预期吞吐量和实际吞吐量检测拥塞，避免等到丢包才反应
// ------------------------------

type VegasController struct {
	*BaseController
	alpha        int     // 最小允许的吞吐量差异（字节）
	beta         int     // 最大允许的吞吐量差异（字节）
	expectedRate float64 // 预期吞吐量（基于最小RTT）
}

func NewVegasController(initialCWnd, maxCWnd, packetSize int) *VegasController {
	return &VegasController{
		BaseController: NewBaseController(initialCWnd, maxCWnd, packetSize),
		alpha:          3 * packetSize, // 通常为3个MSS
		beta:           6 * packetSize, // 通常为6个MSS
	}
}

func (v *VegasController) OnPacketSent(packetSize int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.inFlight += packetSize
	v.sentBytes += int64(packetSize)
	v.packetsSent++
}

func (v *VegasController) OnAckReceived(ackSize int, rtt time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.inFlight -= ackSize
	v.ackedBytes += int64(ackSize)
	v.updateRTT(rtt)

	if v.minRTT == 0 || rtt == 0 {
		return
	}

	actualRate := float64(ackSize) / rtt.Seconds()
	v.expectedRate = float64(v.cwnd) / v.minRTT.Seconds()
	throughputDiff := v.expectedRate - actualRate

	if throughputDiff < float64(v.alpha) {
		v.cwnd += v.packetSize
	} else if throughputDiff > float64(v.beta) {
		v.cwnd -= v.packetSize
	}

	v.setCongestionWindow(v.cwnd)
	v.updateStats()
}

func (v *VegasController) OnPacketLost() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lostBytes += int64(v.packetSize)
	v.packetsLost++
	v.ssthresh = v.cwnd / 2
	v.cwnd = v.ssthresh
	v.setCongestionWindow(v.cwnd)
	v.updateStats()
}

func (v *VegasController) GetCongestionWindow() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.cwnd
}

func (v *VegasController) GetSendRate() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return int(v.expectedRate)
}

func (v *VegasController) GetStatistics() CongestionStats {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.stats
}
