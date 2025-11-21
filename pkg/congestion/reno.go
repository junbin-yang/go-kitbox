package congestion

import (
	"time"
)

// ------------------------------
// Reno拥塞控制算法实现（经典TCP Reno）
// 特点：基于丢包检测，包含慢启动、拥塞避免、快速重传和快速恢复
// ------------------------------

type RenoController struct {
	*BaseController
	dupAckCount int // 重复ACK计数器（用于检测丢包）
}

func NewRenoController(initialCWnd, maxCWnd, packetSize int) *RenoController {
	return &RenoController{
		BaseController: NewBaseController(initialCWnd, maxCWnd, packetSize),
	}
}

func (r *RenoController) OnPacketSent(packetSize int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inFlight += packetSize
	r.sentBytes += int64(packetSize)
	r.packetsSent++
}

func (r *RenoController) OnAckReceived(ackSize int, rtt time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inFlight -= ackSize
	r.ackedBytes += int64(ackSize)
	r.updateRTT(rtt)

	if ackSize == 0 {
		r.dupAckCount++
		if r.dupAckCount == 3 {
			r.fastRetransmits++
			r.ssthresh = r.cwnd / 2
			if r.ssthresh < 2*r.packetSize {
				r.ssthresh = 2 * r.packetSize
			}
			r.cwnd = r.ssthresh + 3*r.packetSize
		} else if r.dupAckCount > 3 {
			r.cwnd += r.packetSize
		}
	} else {
		r.dupAckCount = 0
		if r.cwnd < r.ssthresh {
			r.cwnd += r.packetSize
		} else {
			r.cwnd += r.packetSize * r.packetSize / r.cwnd
		}
	}

	r.setCongestionWindow(r.cwnd)
	r.updateStats()
}

func (r *RenoController) OnPacketLost() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lostBytes += int64(r.packetSize)
	r.packetsLost++
	r.timeoutRetransmits++
	r.ssthresh = r.cwnd / 2
	if r.ssthresh < 2*r.packetSize {
		r.ssthresh = 2 * r.packetSize
	}
	r.cwnd = r.ssthresh
	r.setCongestionWindow(r.cwnd)
	r.updateStats()
}

func (r *RenoController) GetCongestionWindow() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cwnd
}

func (r *RenoController) GetSendRate() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.rtt == 0 {
		return 0
	}
	return int(float64(r.cwnd) / r.rtt.Seconds())
}

func (r *RenoController) GetStatistics() CongestionStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}
