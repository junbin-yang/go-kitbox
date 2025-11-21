package congestion

import (
	"math"
	"time"
)

// ------------------------------
// CUBIC拥塞控制算法实现（TCP CUBIC变种）
// 特点：基于时间的拥塞窗口增长，高带宽场景下性能优于传统Reno
// ------------------------------

type CubicConfig struct {
	Beta float64 // 丢包后窗口缩减系数（默认0.7）
	C    float64 // CUBIC系数（默认0.4）
}

func DefaultCubicConfig() CubicConfig {
	return CubicConfig{Beta: 0.7, C: 0.4}
}

type CubicController struct {
	*BaseController
	beta        float64
	c           float64
	lastMaxCWnd int
	epochStart  time.Time
	K           float64
}

func NewCubicController(initialCWnd, maxCWnd, packetSize int) *CubicController {
	return NewCubicControllerWithConfig(DefaultCubicConfig(), initialCWnd, maxCWnd, packetSize)
}

func NewCubicControllerWithConfig(config CubicConfig, initialCWnd, maxCWnd, packetSize int) *CubicController {
	return &CubicController{
		BaseController: NewBaseController(initialCWnd, maxCWnd, packetSize),
		beta:           config.Beta,
		c:              config.C,
		lastMaxCWnd:    initialCWnd,
		epochStart:     time.Now(),
	}
}

func (c *CubicController) OnPacketSent(packetSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight += packetSize
	c.sentBytes += int64(packetSize)
	c.packetsSent++
}

func (c *CubicController) OnAckReceived(ackSize int, rtt time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inFlight -= ackSize
	c.ackedBytes += int64(ackSize)
	c.updateRTT(rtt)

	if c.cwnd < c.ssthresh {
		c.cwnd += c.packetSize
	} else {
		elapsed := time.Since(c.epochStart).Seconds()
		targetCWnd := int(c.c*(elapsed-c.K)*(elapsed-c.K)*(elapsed-c.K)) + c.lastMaxCWnd
		if targetCWnd > c.cwnd {
			inc := targetCWnd - c.cwnd
			if inc > c.packetSize {
				inc = c.packetSize
			}
			c.cwnd += inc
		}
	}

	c.setCongestionWindow(c.cwnd)
	c.updateStats()
}

func (c *CubicController) OnPacketLost() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lostBytes += int64(c.packetSize)
	c.packetsLost++
	c.lastMaxCWnd = c.cwnd
	c.cwnd = int(float64(c.cwnd) * c.beta)
	c.ssthresh = c.cwnd
	c.epochStart = time.Now()
	c.K = math.Pow(float64(c.lastMaxCWnd-c.cwnd)/c.c, 1.0/3.0)
	c.setCongestionWindow(c.cwnd)
	c.updateStats()
}

func (c *CubicController) GetCongestionWindow() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cwnd
}

func (c *CubicController) GetSendRate() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.rtt == 0 {
		return 0
	}
	return int(float64(c.cwnd) / c.rtt.Seconds())
}

func (c *CubicController) GetStatistics() CongestionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}
