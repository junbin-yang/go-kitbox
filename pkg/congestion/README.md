# Congestion - 拥塞控制算法模块

提供多种经典网络拥塞控制算法实现，用于动态调整数据发送策略，避免网络拥塞。

## 特性

- 支持4种经典拥塞控制算法：CUBIC、BBR、Reno、Vegas
- 统一的接口设计，易于切换和扩展
- 完整的RTT估算和统计信息
- **并发安全**：所有操作使用读写锁保护
- **窗口保护**：自动限制最小/最大拥塞窗口
- **可配置**：CUBIC支持自定义参数
- **精确估算**：BBR使用滑动窗口估算带宽
- 适用于自定义传输协议（如FILLP）

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/congestion
```

## 快速开始

```go
package main

import (
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/congestion"
)

func main() {
    // 创建CUBIC拥塞控制器
    ctrl, err := congestion.NewController(
        congestion.AlgorithmCubic,
        2800,  // 初始拥塞窗口（字节）
        65536, // 最大拥塞窗口（字节）
        1400,  // 数据包大小（字节）
    )
    if err != nil {
        panic(err)
    }

    // 发送数据包
    ctrl.OnPacketSent(1400)

    // 收到ACK确认
    ctrl.OnAckReceived(1400, 50*time.Millisecond)

    // 检测到丢包
    ctrl.OnPacketLost()

    // 获取当前拥塞窗口
    cwnd := ctrl.GetCongestionWindow()
    fmt.Printf("当前拥塞窗口: %d 字节\n", cwnd)

    // 获取统计信息
    stats := ctrl.GetStatistics()
    fmt.Printf("RTT: %v, 丢包率: %.2f%%\n", stats.RTT, stats.LossRate*100)
}
```

## 支持的算法

### 1. CUBIC（推荐）

基于时间的拥塞窗口增长算法，高带宽场景下性能优于传统Reno。

**特点：**
- 使用三次函数调整拥塞窗口
- 丢包后窗口缩减至70%（beta=0.7）
- 适合高带宽长延迟网络（BDP较大）
- Linux内核默认算法

**使用场景：**
- 数据中心网络
- 广域网文件传输
- 高带宽应用

```go
ctrl := congestion.NewCubicController(2800, 65536, 1400)
```

### 2. BBR（Google开发）

基于带宽和RTT的拥塞控制，不依赖丢包检测。

**特点：**
- 主动探测瓶颈带宽和最小RTT
- 对丢包不敏感（仅减少10%窗口）
- 状态机设计：STARTUP → DRAIN → PROBE_BW → PROBE_RTT
- 适合高丢包率网络

**使用场景：**
- 无线网络（WiFi/4G/5G）
- 卫星链路
- 跨国网络

```go
ctrl := congestion.NewBBRController(2800, 65536, 1400)
```

### 3. Reno（经典算法）

传统TCP Reno算法，包含慢启动、拥塞避免、快速重传和快速恢复。

**特点：**
- 基于丢包检测拥塞
- 3次重复ACK触发快速重传
- 丢包后窗口减半
- 实现简单，行为可预测

**使用场景：**
- 低延迟网络
- 稳定网络环境
- 教学和研究

```go
ctrl := congestion.NewRenoController(2800, 65536, 1400)
```

### 4. Vegas（基于延迟）

通过比较预期吞吐量和实际吞吐量检测拥塞，避免等到丢包才反应。

**特点：**
- 主动拥塞检测（延迟增加时减速）
- 对丢包更保守
- 适合低延迟敏感应用
- 需要准确的RTT测量

**使用场景：**
- 实时音视频
- 在线游戏
- 低延迟交易系统

```go
ctrl := congestion.NewVegasController(2800, 65536, 1400)
```

## 接口说明

### Controller接口

```go
type Controller interface {
    // 发送数据包时调用
    OnPacketSent(packetSize int)

    // 收到ACK确认时调用
    OnAckReceived(ackSize int, rtt time.Duration)

    // 检测到丢包时调用
    OnPacketLost()

    // 获取当前拥塞窗口大小（字节）
    GetCongestionWindow() int

    // 获取建议的发送速率（字节/秒）
    GetSendRate() int

    // 获取统计信息
    GetStatistics() CongestionStats
}
```

### 统计信息

```go
type CongestionStats struct {
    CongestionWindow   int           // 当前拥塞窗口（字节）
    Ssthresh           int           // 慢启动阈值（字节）
    RTT                time.Duration // 平均往返时间
    MinRTT             time.Duration // 最小往返时间
    LossRate           float64       // 丢包率（0-1）
    SendRate           int           // 发送速率（字节/秒）
    InFlight           int           // 飞行中字节数
    PacketsSent        int64         // 总发送包数
    PacketsLost        int64         // 总丢失包数
    FastRetransmits    int64         // 快速重传次数
    TimeoutRetransmits int64         // 超时重传次数
    CurrentState       string        // 当前状态（BBR专用）
}
```

## 高级用法

### 与FILLP协议集成

```go
// 在FILLP连接中使用拥塞控制
type Connection struct {
    congestionCtrl congestion.Controller
    // ...
}

func (c *Connection) sendPacket(data []byte) error {
    // 检查拥塞窗口
    cwnd := c.congestionCtrl.GetCongestionWindow()
    if c.inFlight >= cwnd {
        return ErrCongestionWindowFull
    }

    // 发送数据包
    c.congestionCtrl.OnPacketSent(len(data))
    // ... 实际发送逻辑

    return nil
}

func (c *Connection) onAckReceived(ackSize int, rtt time.Duration) {
    c.congestionCtrl.OnAckReceived(ackSize, rtt)
}

func (c *Connection) onPacketLost() {
    c.congestionCtrl.OnPacketLost()
}
```

### 动态切换算法

```go
func switchAlgorithm(oldCtrl congestion.Controller, newAlgo congestion.AlgorithmType) (congestion.Controller, error) {
    stats := oldCtrl.GetStatistics()

    // 使用旧控制器的状态创建新控制器
    return congestion.NewController(
        newAlgo,
        stats.CongestionWindow,
        65536,
        1400,
    )
}
```

### 监控和调优

```go
func monitorCongestion(ctrl congestion.Controller) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        stats := ctrl.GetStatistics()
        fmt.Printf("拥塞窗口: %d, RTT: %v, 丢包率: %.2f%%, 速率: %d B/s\n",
            stats.CongestionWindow,
            stats.RTT,
            stats.LossRate*100,
            stats.SendRate,
        )
    }
}
```

## 算法对比

| 算法 | 拥塞检测 | 丢包敏感度 | 适用场景 | 复杂度 |
|------|---------|-----------|---------|--------|
| CUBIC | 丢包 | 中等（70%） | 高带宽网络 | 中 |
| BBR | 带宽/RTT | 低（90%） | 高丢包网络 | 高 |
| Reno | 丢包 | 高（50%） | 稳定网络 | 低 |
| Vegas | 延迟 | 高（50%） | 低延迟应用 | 中 |

**选择建议：**
- 通用场景：CUBIC（平衡性能和稳定性）
- 无线网络：BBR（抗丢包）
- 低延迟：Vegas（主动避免拥塞）
- 简单可靠：Reno（行为可预测）

## 性能优化建议

1. **合理设置初始窗口**：根据网络带宽和RTT估算BDP（Bandwidth-Delay Product）
   ```go
   // BDP = 带宽(B/s) × RTT(s)
   // 例如：10Mbps × 50ms = 62500字节
   initialCWnd := 62500
   ```

2. **限制最大窗口**：避免占用过多内存
   ```go
   maxCWnd := 1024 * 1024 // 1MB
   ```

3. **准确测量RTT**：使用高精度时间戳
   ```go
   start := time.Now()
   // ... 发送数据包
   rtt := time.Since(start)
   ctrl.OnAckReceived(size, rtt)
   ```

4. **及时处理丢包**：快速检测并通知控制器
   ```go
   if isPacketLost() {
       ctrl.OnPacketLost()
   }
   ```

## 优化特性

### 并发安全
所有算法实现都使用 `sync.RWMutex` 保护，支持多goroutine并发调用：
```go
// 安全的并发调用
go func() {
    ctrl.OnPacketSent(1400)
}()
go func() {
    stats := ctrl.GetStatistics()
}()
```

### 窗口保护
自动限制拥塞窗口范围，防止窗口过小或过大：
- 最小窗口：2 × MSS（防止窗口降至0）
- 最大窗口：用户指定的 `maxCWnd`

### CUBIC配置
支持自定义CUBIC参数：
```go
config := congestion.CubicConfig{
    Beta: 0.8,  // 丢包后保留80%窗口
    C:    0.5,  // 调整增长速度
}
ctrl := congestion.NewCubicControllerWithConfig(config, 2800, 65536, 1400)
```

### BBR带宽估算
使用滑动窗口（10个样本）估算瓶颈带宽，取最大值作为估计值，比单次测量更准确。

## 注意事项

1. **RTT测量精度**：Vegas和BBR算法依赖准确的RTT测量，建议使用纳秒级时间戳
2. **状态持久化**：重启后需重新初始化，无法恢复历史状态
3. **参数调优**：不同网络环境可能需要调整alpha、beta等参数

## 测试

```bash
# 运行所有测试
go test ./pkg/congestion/...

# 运行详细测试
go test -v ./pkg/congestion/...

# 运行基准测试
go test -bench=. ./pkg/congestion/...
```

## 许可证

MIT License
