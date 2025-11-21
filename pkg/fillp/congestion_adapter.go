package fillp

import (
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/congestion"
)

// 拥塞控制适配器 - 将Congestion包集成到FILLP

// initCongestionControl 初始化拥塞控制器
func (c *Connection) initCongestionControl(config ConnectionConfig) error {
	if config.CongestionAlgorithm == "" {
		// 使用FILLP内置算法
		c.useExternalCC = false
		return nil
	}

	// 使用Congestion包算法
	c.useExternalCC = true

	// 创建控制器
	initialCWnd := int(DefaultMTU * 2)
	maxCWnd := int(DefaultWindowSize)
	packetSize := int(DefaultMTU)

	// 如果是CUBIC且提供了配置
	if config.CongestionAlgorithm == congestion.AlgorithmCubic {
		if cubicCfg, ok := config.CongestionConfig.(congestion.CubicConfig); ok {
			c.ccController = congestion.NewCubicControllerWithConfig(cubicCfg, initialCWnd, maxCWnd, packetSize)
			return nil
		}
	}

	// 使用默认配置创建控制器
	ctrl, err := congestion.NewController(config.CongestionAlgorithm, initialCWnd, maxCWnd, packetSize)
	if err != nil {
		return err
	}
	c.ccController = ctrl
	return nil
}

// onPacketSent 数据包发送通知
func (c *Connection) onPacketSent(size int) {
	if c.useExternalCC && c.ccController != nil {
		c.ccController.OnPacketSent(size)
	}
}

// onAckReceived ACK接收通知
func (c *Connection) onAckReceived(size int, rtt time.Duration) {
	if c.useExternalCC && c.ccController != nil {
		c.ccController.OnAckReceived(size, rtt)
	} else {
		// 使用内置算法
		c.updateCongestionWindow(true)
	}
}

// onPacketLost 丢包通知
func (c *Connection) onPacketLost() {
	if c.useExternalCC && c.ccController != nil {
		c.ccController.OnPacketLost()
	} else {
		// 使用内置算法
		c.updateCongestionWindow(false)
	}
}

// getCongestionWindow 获取拥塞窗口
func (c *Connection) getCongestionWindow() uint32 {
	if c.useExternalCC && c.ccController != nil {
		return uint32(c.ccController.GetCongestionWindow())
	}
	return c.congestionWnd
}

// GetCongestionStats 获取拥塞控制统计信息（仅外部算法）
func (c *Connection) GetCongestionStats() *congestion.CongestionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.useExternalCC && c.ccController != nil {
		stats := c.ccController.GetStatistics()
		return &stats
	}
	return nil
}

// UseExternalCC 是否使用外部拥塞控制
func (c *Connection) UseExternalCC() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.useExternalCC
}
