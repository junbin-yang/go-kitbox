# FILLP - Fast Intelligent Lossless Link Protocol

基于 UDP 的可靠数据传输协议实现，提供类似 TCP 的可靠性保证，同时保持 UDP 的低延迟特性。

## 特性

-   基于 UDP 的可靠传输
-   滑动窗口流量控制
-   拥塞控制（慢启动 + 拥塞避免）
-   超时重传机制（指数退避）
-   快速重传（3 次重复 ACK）
-   **智能延迟 ACK**（RFC 1122 标准，自动适配场景）
-   RTT 估算和动态 RTO 调整
-   连接管理（SYN/FIN 握手）
-   保活机制
-   并发安全

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/fillp
```

## 核心组件

### 1. RingBuffer（环形缓冲区）

线程安全的环形缓冲区，用于临时存储待发送/接收的字节数据。

**特点：**

-   高效的读写操作
-   自动循环利用空间
-   支持 Peek（查看但不移除）
-   并发安全

**使用示例：**

```go
// 创建1024字节的缓冲区
buffer := fillp.NewRingBuffer(1024)

// 写入数据
data := []byte("Hello FILLP")
err := buffer.Write(data)

// 查看数据（不移除）
peeked, _ := buffer.Peek(5)
fmt.Println(string(peeked)) // "Hello"

// 读取数据（移除）
read, _ := buffer.Read(11)
fmt.Println(string(read)) // "Hello FILLP"

// 查询状态
fmt.Printf("已用: %d, 可用: %d\n", buffer.Used(), buffer.Available())

// 清空缓冲区
buffer.Clear()

// 关闭缓冲区
buffer.Close()
```

### 2. RetransmissionQueue（重传队列）

管理待确认数据包，实现超时检测与指数退避重传。

**特点：**

-   按序列号跟踪数据包
-   自动超时检测
-   指数退避重传
-   累计确认裁切

**使用示例：**

```go
// 创建重传队列
rq := fillp.NewRetransmissionQueue()

// 添加数据包（序列号、数据、时间戳）
now := time.Now().UnixMilli()
rq.Add(1001, []byte("packet data"), now)

// 检查超时数据包
expired := rq.GetExpired(time.Now().UnixMilli())
for _, pkt := range expired {
    // 重传数据包
    fmt.Printf("重传序列号 %d\n", pkt.Sequence)
}

// 确认数据包（移除）
rq.Remove(1001)

// 累计确认裁切
rq.TrimUpTo(1005) // 移除序列号 < 1005 的所有数据包

// 清空队列
rq.Clear()
```

### 3. Connection（FILLP 连接）

封装完整的 FILLP 协议连接生命周期。

**特点：**

-   三次握手建立连接
-   可靠数据传输
-   流量控制和拥塞控制
-   自动重传和快速重传
-   保活机制
-   优雅关闭

## 快速开始

### 服务端示例

```go
package main

import (
    "fmt"
    "net"
    "github.com/junbin-yang/go-kitbox/pkg/fillp"
)

func main() {
    // 创建服务端连接
    serverAddr := &net.UDPAddr{
        IP:   net.ParseIP("127.0.0.1"),
        Port: 8080,
    }

    server, err := fillp.NewConnection(serverAddr, nil)
    if err != nil {
        panic(err)
    }
    defer server.Close()

    // 监听连接
    fmt.Println("服务端监听 127.0.0.1:8080")
    if err := server.Listen(); err != nil {
        panic(err)
    }

    fmt.Println("客户端已连接")

    // 接收数据
    for {
        data, err := server.ReceiveWithTimeout(30 * time.Second)
        if err != nil {
            fmt.Println("接收错误:", err)
            break
        }
        fmt.Printf("收到数据: %s\n", string(data))

        // 回显数据
        if err := server.Send(data); err != nil {
            fmt.Println("发送错误:", err)
            break
        }
    }
}
```

### 客户端示例

```go
package main

import (
    "fmt"
    "net"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/fillp"
)

func main() {
    // 创建客户端连接
    serverAddr := &net.UDPAddr{
        IP:   net.ParseIP("127.0.0.1"),
        Port: 8080,
    }

    client, err := fillp.NewConnection(nil, serverAddr)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 连接服务端
    fmt.Println("连接服务端...")
    if err := client.Connect(); err != nil {
        panic(err)
    }

    fmt.Println("连接成功")

    // 发送数据
    message := []byte("Hello FILLP Server!")
    if err := client.Send(message); err != nil {
        panic(err)
    }
    fmt.Printf("发送数据: %s\n", string(message))

    // 接收回显
    response, err := client.ReceiveWithTimeout(5 * time.Second)
    if err != nil {
        panic(err)
    }
    fmt.Printf("收到回显: %s\n", string(response))

    // 获取统计信息
    stats := client.GetStatistics()
    fmt.Printf("统计: 发送 %d 字节, 接收 %d 字节, RTT %v\n",
        stats.BytesSent, stats.BytesReceived, stats.RTT)
}
```

## 高级用法

### 大数据传输

FILLP 自动处理数据分片，支持传输超过 MTU 的数据：

```go
// 发送大数据（自动分片）
largeData := make([]byte, 100000) // 100KB
if err := client.Send(largeData); err != nil {
    panic(err)
}

// 接收大数据（可能需要多次接收）
var received []byte
for len(received) < 100000 {
    data, err := server.ReceiveWithTimeout(5 * time.Second)
    if err != nil {
        break
    }
    received = append(received, data...)
}
```

### 流量控制

```go
// 设置发送窗口大小（字节）
conn.SetFlowControl(32768) // 32KB
```

### 获取连接信息

```go
// 获取本地地址
localAddr := conn.LocalAddr()
fmt.Println("本地地址:", localAddr)

// 获取远程地址
remoteAddr := conn.RemoteAddr()
fmt.Println("远程地址:", remoteAddr)

// 获取统计信息
stats := conn.GetStatistics()
fmt.Printf("发送: %d 包, %d 字节\n", stats.PacketsSent, stats.BytesSent)
fmt.Printf("接收: %d 包, %d 字节\n", stats.PacketsReceived, stats.BytesReceived)
fmt.Printf("重传: %d 次\n", stats.Retransmissions)
fmt.Printf("RTT: %v\n", stats.RTT)
```

## 协议设计

### 数据包格式

```
+--------+--------+----------+----------+----------+-----------+----------+--------+
|  Type  | Flags  | Sequence |   Ack    |  Window  | Timestamp | Checksum |  Data  |
| (1B)   | (1B)   |  (4B)    |  (4B)    |  (4B)    |   (4B)    |  (4B)    | (变长) |
+--------+--------+----------+----------+----------+-----------+----------+--------+
```

**包类型：**

-   `PacketTypeData` - 数据包
-   `PacketTypeAck` - 确认包
-   `PacketTypeSyn` - 同步包（建立连接）
-   `PacketTypeFin` - 结束包（关闭连接）
-   `PacketTypeKeepAlive` - 保活包
-   `PacketTypeWindowUpdate` - 窗口更新包

### 连接状态

-   `StateIdle` - 空闲状态
-   `StateConnecting` - 连接中
-   `StateConnected` - 已连接
-   `StateListening` - 监听中（服务端）
-   `StateClosing` - 关闭中
-   `StateClosed` - 已关闭

### 可靠性机制

1. **序列号机制**：每个数据包都有唯一序列号，接收方按序接收
2. **确认机制**：接收方发送 ACK 确认已接收数据
3. **超时重传**：未收到 ACK 的数据包在超时后重传
4. **快速重传**：收到 3 次重复 ACK 立即重传
5. **累计确认**：ACK 确认所有小于等于该序列号的数据
6. **指数退避**：重传间隔按 2 的幂次增长
7. **智能延迟 ACK**：RFC 1122 标准实现，自动优化不同场景

### 智能延迟 ACK 优化

FILLP 实现了符合 RFC 1122 标准的智能延迟 ACK 机制，根据不同使用场景自动优化：

**三种工作模式：**

1. **请求-响应模式**（零延迟）

    - 检测到有响应数据待发送时，立即捎带 ACK
    - 适用场景：RPC 调用、HTTP 请求、数据库查询
    - 性能：亚毫秒级延迟（实测 271µs）

2. **批量传输模式**（减少 50% ACK）

    - 每收到 2 个数据包发送 1 个 ACK
    - 适用场景：文件传输、视频流、大数据同步
    - 性能：ACK 包减少 45-50%（实测 54.55%）

3. **单向流模式**（超时保证）
    - 40ms 超时自动发送 ACK
    - 适用场景：日志推送、监控数据上报
    - 性能：可靠性保证，符合 RFC 标准

**性能测试结果：**

| 场景      | ACK 包数量 | 延迟  | 优化效果 |
| --------- | ---------- | ----- | -------- |
| 批量传输  | 54.55%     | -     | ↓ 45%    |
| 请求-响应 | -          | 271µs | 无增加   |
| 单向流    | 超时触发   | 40ms  | 可靠保证 |

### 流量控制

-   滑动窗口机制
-   接收方通告可用窗口大小
-   发送方限制未确认数据量

### 拥塞控制

FILLP 提供两种拥塞控制方案：

#### 1. 内置算法（默认，推荐 80% 场景）

基于 TCP Reno 的经典实现：

-   **慢启动**：指数增长拥塞窗口
-   **拥塞避免**：线性增长拥塞窗口
-   **快速恢复**：检测到丢包时减半窗口

**适用场景：**

-   局域网传输（RTT < 10ms）
-   低延迟网络（RTT < 50ms）
-   带宽 < 100Mbps
-   简单网络环境

**优势：**

-   零配置，开箱即用
-   性能开销极小（3.59ns/操作）
-   稳定可靠

#### 2. 高级算法（可选，特殊场景）

集成 [congestion](../congestion/) 包的高级算法，适用于极端网络环境：

| 算法      | 适用场景                         | 性能提升      | 使用方式                    |
| --------- | -------------------------------- | ------------- | --------------------------- |
| **CUBIC** | 高带宽长延迟（数据中心、广域网） | +50% ~ +100%  | `NewConnectionWithCUBIC()`  |
| **BBR**   | 高丢包环境（无线网络、卫星链路） | +100% ~ +300% | `NewConnectionWithBBR()`    |
| **Vegas** | 低延迟敏感（实时音视频、游戏）   | 延迟 -40%     | `NewConnectionWithVegas()`  |
| **Reno**  | 经典 TCP 算法                    | 与内置相似    | `NewConnectionWithConfig()` |

**性能开销对比：**

| 算法  | 平均耗时 | 相对内置 | 内存分配 | 实际影响 |
| ----- | -------- | -------- | -------- | -------- |
| 内置  | 3.59 ns  | 1.0x     | 0 B      | 基准     |
| CUBIC | 69.22 ns | 19.3x    | 0 B      | 可忽略   |
| BBR   | 75.71 ns | 21.1x    | 15 B     | 可忽略   |
| Vegas | 67.41 ns | 18.8x    | 0 B      | 可忽略   |

> **注意**：虽然外部算法比内置慢 17-21 倍，但绝对值仅 65ns，对网络传输（毫秒级）无影响。即使高并发（1000 连接），CPU 开销增加 < 1%。

#### 使用示例

**默认使用（推荐）：**

```go
// 80% 场景使用默认配置即可
conn, _ := fillp.NewConnection(localAddr, remoteAddr)
```

**使用 CUBIC（高带宽长延迟）：**

```go
// 方式1：快捷函数
conn, _ := fillp.NewConnectionWithCUBIC(localAddr, remoteAddr)

// 方式2：自定义配置
cubicConfig := congestion.CubicConfig{
    Beta: 0.8,  // 丢包后保留 80% 窗口
    C:    0.5,  // 更快的增长速度
}
config := fillp.ConnectionConfig{
    CongestionAlgorithm: congestion.AlgorithmCubic,
    CongestionConfig:    cubicConfig,
}
conn, _ := fillp.NewConnectionWithConfig(localAddr, remoteAddr, config)
```

**使用 BBR（高丢包环境）：**

```go
conn, _ := fillp.NewConnectionWithBBR(localAddr, remoteAddr)
```

**使用 Vegas（低延迟敏感）：**

```go
conn, _ := fillp.NewConnectionWithVegas(localAddr, remoteAddr)
```

**获取拥塞控制统计信息：**

```go
// 检查是否使用外部算法
if conn.UseExternalCC() {
    stats := conn.GetCongestionStats()
    fmt.Printf("拥塞窗口: %d\n", stats.CongestionWindow)
    fmt.Printf("RTT: %v\n", stats.RTT)
    fmt.Printf("丢包率: %.2f%%\n", stats.LossRate*100)
    fmt.Printf("当前状态: %s\n", stats.CurrentState) // BBR 专用
}
```

#### 场景选择决策表

```
默认使用内置算法（80% 场景）
  ↓
遇到性能瓶颈？
  ├─ 高带宽长延迟（RTT > 100ms，带宽 > 100Mbps）→ CUBIC
  ├─ 高丢包率（> 5%）→ BBR
  ├─ 低延迟敏感（实时应用）→ Vegas
  └─ 其他 → 保持内置
```

**详细文档：** [拥塞控制使用指南](congestion-usage-guide.md)

## 配置参数

| 参数                 | 默认值 | 说明                 |
| -------------------- | ------ | -------------------- |
| `DefaultMTU`         | 1400   | 最大传输单元（字节） |
| `DefaultWindowSize`  | 65536  | 默认窗口大小（字节） |
| `DefaultTimeout`     | 30s    | 连接超时时间         |
| `DefaultKeepAlive`   | 10s    | 保活间隔             |
| `MaxRetransmissions` | 5      | 最大重传次数         |
| `InitialRTO`         | 200ms  | 初始重传超时         |
| `MinRTO`             | 50ms   | 最小重传超时         |
| `MaxRTO`             | 10s    | 最大重传超时         |

## 性能优化建议

1. **缓冲区大小**：根据网络带宽和延迟调整窗口大小

    ```go
    conn.SetFlowControl(128 * 1024) // 128KB for high bandwidth
    ```

2. **批量发送**：合并小数据包减少开销

    ```go
    // 不推荐：频繁发送小包
    for _, msg := range messages {
        conn.Send(msg)
    }

    // 推荐：批量发送
    batch := bytes.Join(messages, []byte("\n"))
    conn.Send(batch)
    ```

3. **接收缓冲**：及时读取接收缓冲区避免阻塞
    ```go
    go func() {
        for {
            data, err := conn.ReceiveWithTimeout(1 * time.Second)
            if err != nil {
                continue
            }
            processData(data)
        }
    }()
    ```

## 注意事项

1. **网络环境**：FILLP 适用于不可靠网络，但在极端丢包环境下性能会下降
2. **MTU 设置**：根据网络环境调整 MTU，避免 IP 分片
3. **超时配置**：本地回环可使用较小的 RTO，广域网需要更大的 RTO
4. **资源清理**：使用 `defer conn.Close()` 确保连接正确关闭
5. **并发安全**：所有公开方法都是并发安全的

## 应用场景

-   游戏服务器通信
-   实时音视频传输
-   物联网设备通信
-   文件传输
-   RPC 框架底层传输

## 许可证

MIT License
