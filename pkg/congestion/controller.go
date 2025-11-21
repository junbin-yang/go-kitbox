package congestion

import (
	"fmt"
	"sync"
	"time"
)

// Controller 拥塞控制接口，定义所有拥塞控制算法需实现的核心方法
type Controller interface {
	// OnPacketSent 当数据包被发送时调用，用于更新飞行中数据包数量等状态
	OnPacketSent(packetSize int)
	// OnAckReceived 当收到ACK确认时调用，用于调整拥塞窗口、更新RTT等
	OnAckReceived(ackSize int, rtt time.Duration)
	// OnPacketLost 当检测到数据包丢失时调用，用于触发拥塞避免逻辑
	OnPacketLost()
	// GetCongestionWindow 获取当前拥塞窗口大小（字节）
	GetCongestionWindow() int
	// GetSendRate 获取当前建议的发送速率（字节/秒）
	GetSendRate() int
	// GetStatistics 获取当前拥塞控制的统计信息
	GetStatistics() CongestionStats
}

// CongestionStats 拥塞控制统计信息，用于监控和分析算法性能
type CongestionStats struct {
	CongestionWindow   int           // 当前拥塞窗口大小（字节）
	Ssthresh           int           // 慢启动阈值（字节）
	RTT                time.Duration // 平均往返时间
	MinRTT             time.Duration // 最小往返时间（瓶颈链路RTT估计）
	LossRate           float64       // 丢包率（丢失包数/总发送包数）
	SendRate           int           // 实际发送速率（字节/秒）
	InFlight           int           // 飞行中数据包数量（未确认的字节数）
	PacketsSent        int64         // 总发送包数
	PacketsLost        int64         // 总丢失包数
	FastRetransmits    int64         // 快速重传次数
	TimeoutRetransmits int64         // 超时重传次数
	CurrentState       string        // 当前状态（如BBR的状态）
}

// BaseController 基础拥塞控制器，封装所有算法的通用属性和方法
type BaseController struct {
	mu                 sync.RWMutex // 并发安全保护
	cwnd               int          // 拥塞窗口（字节）
	ssthresh           int          // 慢启动阈值（字节）
	rtt                time.Duration
	minRTT             time.Duration
	inFlight           int
	maxCWnd            int
	packetSize         int
	stats              CongestionStats
	sentBytes          int64
	ackedBytes         int64
	lostBytes          int64
	lastUpdate         time.Time
	packetsSent        int64
	packetsLost        int64
	fastRetransmits    int64
	timeoutRetransmits int64
}

// 初始化基础控制器
func NewBaseController(initialCWnd, maxCWnd, packetSize int) *BaseController {
	return &BaseController{
		cwnd:       initialCWnd,
		ssthresh:   65536, // 默认慢启动阈值为64KB
		maxCWnd:    maxCWnd,
		packetSize: packetSize,
		lastUpdate: time.Now(),
	}
}

// 更新RTT（通用逻辑：平滑处理新RTT值）
func (b *BaseController) updateRTT(newRTT time.Duration) {
	if b.rtt == 0 {
		b.rtt = newRTT
		b.minRTT = newRTT
		return
	}
	// 平滑RTT计算（指数加权移动平均）
	b.rtt = time.Duration(0.875*float64(b.rtt) + 0.125*float64(newRTT))
	// 更新最小RTT（只保留更小的值）
	if newRTT < b.minRTT {
		b.minRTT = newRTT
	}
}

// 计算发送速率（字节/秒）
func (b *BaseController) calculateSendRate() int {
	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()
	if elapsed == 0 {
		return 0
	}
	// 基于最近确认的字节数计算速率
	rate := int(float64(b.ackedBytes) / elapsed)
	b.lastUpdate = now
	b.ackedBytes = 0 // 重置计数器
	return rate
}

// 更新统计信息
func (b *BaseController) updateStats() {
	b.stats.CongestionWindow = b.cwnd
	b.stats.Ssthresh = b.ssthresh
	b.stats.RTT = b.rtt
	b.stats.MinRTT = b.minRTT
	b.stats.InFlight = b.inFlight
	b.stats.SendRate = b.calculateSendRate()
	b.stats.PacketsSent = b.packetsSent
	b.stats.PacketsLost = b.packetsLost
	b.stats.FastRetransmits = b.fastRetransmits
	b.stats.TimeoutRetransmits = b.timeoutRetransmits
	if b.sentBytes > 0 {
		b.stats.LossRate = float64(b.lostBytes) / float64(b.sentBytes)
	}
}

// 设置拥塞窗口（带下限保护）
func (b *BaseController) setCongestionWindow(cwnd int) {
	minCWnd := 2 * b.packetSize
	if cwnd < minCWnd {
		cwnd = minCWnd
	}
	if cwnd > b.maxCWnd {
		cwnd = b.maxCWnd
	}
	b.cwnd = cwnd
}

type AlgorithmType string

const (
	AlgorithmCubic AlgorithmType = "cubic"
	AlgorithmBBR   AlgorithmType = "bbr"
	AlgorithmReno  AlgorithmType = "reno"
	AlgorithmVegas AlgorithmType = "vegas"
)

// 创建拥塞控制器实例（根据算法类型）
func NewController(algorithm AlgorithmType, initialCWnd, maxCWnd, packetSize int) (Controller, error) {
	switch algorithm {
	case AlgorithmCubic:
		return NewCubicController(initialCWnd, maxCWnd, packetSize), nil
	case AlgorithmBBR:
		return NewBBRController(initialCWnd, maxCWnd, packetSize), nil
	case AlgorithmReno:
		return NewRenoController(initialCWnd, maxCWnd, packetSize), nil
	case AlgorithmVegas:
		return NewVegasController(initialCWnd, maxCWnd, packetSize), nil
	default:
		return nil, fmt.Errorf("不支持的拥塞控制算法: %s（支持的算法：%v）", algorithm,
			[]AlgorithmType{AlgorithmCubic, AlgorithmBBR, AlgorithmReno, AlgorithmVegas})
	}
}
