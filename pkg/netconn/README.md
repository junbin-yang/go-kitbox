# NetConn - 统一网络连接库

一个强大的 Go 网络通信库，提供统一的 API 同时支持 TCP 和 UDP（基于 FILLP）协议。

## 特性

-   ✅ **多协议支持**：统一 API 支持 TCP 和 UDP（基于 FILLP）
-   ✅ **统一回调接口**：客户端与服务端使用相同的事件回调
-   ✅ **虚拟 FD 管理**：全局自增 FD，支持海量连接
-   ✅ **连接管理器**：集中管理所有连接，支持查询和统计
-   ✅ **灵活配置**：支持超时、KeepAlive 等配置
-   ✅ **自动缓冲**：自动处理数据包边界
-   ✅ **线程安全**：所有操作并发安全

## 快速开始

### TCP 服务端

```go
package main

import (
	"fmt"
	"github.com/junbin-yang/go-kitbox/pkg/netconn"
)

func main() {
	server := netconn.NewBaseServer(nil)

	callback := &netconn.BaseListenerCallback{
		OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
			fmt.Printf("客户端连接 fd=%d\n", fd)
		},
		OnDisconnected: func(fd int, connType netconn.ConnectionType) {
			fmt.Printf("客户端断开 fd=%d\n", fd)
		},
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			fmt.Printf("收到数据: %s\n", string(buf[:used]))
			server.SendBytes(fd, buf[:used]) // 回显
			return used
		},
	}

	opt := &netconn.ServerOption{
		Protocol: netconn.ProtocolTCP,
		Addr:     "0.0.0.0",
		Port:     8080,
	}

	server.StartBaseListener(opt, callback)
	defer server.StopBaseListener()

	fmt.Println("TCP 服务器启动: 0.0.0.0:8080")
	select {}
}
```

### TCP 客户端

```go
package main

import (
	"fmt"
	"github.com/junbin-yang/go-kitbox/pkg/netconn"
)

func main() {
	callback := &netconn.BaseListenerCallback{
		OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
			fmt.Printf("连接成功 fd=%d\n", fd)
		},
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			fmt.Printf("收到回显: %s\n", string(buf[:used]))
			return used
		},
	}

	client := netconn.NewBaseClient(nil, callback)
	fd, _ := client.ConnectSimple(netconn.ProtocolTCP, "127.0.0.1", 8080)
	defer client.Close()

	fmt.Printf("已连接 fd=%d\n", fd)
	client.SendBytes([]byte("Hello Server"))

	select {}
}
```

### UDP/FILLP 服务端

```go
opt := &netconn.ServerOption{
	Protocol: netconn.ProtocolUDP,  // 使用 UDP/FILLP
	Addr:     "0.0.0.0",
	Port:     8080,
}
server.StartBaseListener(opt, callback)
```

### UDP/FILLP 客户端

```go
fd, _ := client.ConnectSimple(netconn.ProtocolUDP, "127.0.0.1", 8080)
```

## API 文档

### 协议类型

```go
const (
	ProtocolTCP ProtocolType = iota
	ProtocolUDP  // 使用 FILLP
)
```

### 核心类型

#### ServerOption（服务端配置）

```go
type ServerOption struct {
	Protocol ProtocolType // TCP 或 UDP
	Addr     string       // 监听地址
	Port     int          // 监听端口
}
```

#### ClientOption（客户端配置）

```go
type ClientOption struct {
	Protocol        ProtocolType  // TCP 或 UDP
	RemoteIP        string        // 远程IP地址
	RemotePort      int           // 远程端口
	Timeout         time.Duration // 连接超时（0=使用默认5秒）
	KeepAlive       bool          // 是否启用长连接（仅TCP）
	KeepAlivePeriod time.Duration // 保活周期（仅TCP）
}
```

#### ConnectOption（连接信息）

```go
type ConnectOption struct {
	Protocol     ProtocolType
	LocalSocket  *SocketOption // 本地地址和端口
	RemoteSocket *SocketOption // 远程地址和端口
	NetConn      NetConnection // 网络连接
}
```

### 服务器（BaseServer）

#### 创建与管理

-   `NewBaseServer(connMgr *ConnectionManager) *BaseServer`
    创建服务器实例。`connMgr` 为 `nil` 时自动创建新的管理器

-   `StartBaseListener(opt *ServerOption, callback *BaseListenerCallback) error`
    启动服务器监听

-   `StopBaseListener() error`
    停止服务器并关闭所有连接

#### 数据操作

-   `SendBytes(fd int, data []byte) error`
    向指定虚拟 fd 发送数据

-   `GetConnInfo(fd int) *ConnectOption`
    获取连接的完整信息（包含本地和远程地址）

#### 信息查询

-   `GetPort() int`
    获取监听端口（-1 表示未启动）

-   `GetAddr() string`
    获取监听地址（空字符串表示未启动）

### 客户端（BaseClient）

#### 创建与连接

-   `NewBaseClient(connMgr *ConnectionManager, callback *BaseListenerCallback) *BaseClient`
    创建客户端实例

-   `Connect(opt *ClientOption) (int, error)`
    使用配置选项连接服务器，返回虚拟 fd

-   `ConnectSimple(protocol ProtocolType, remoteIP string, remotePort int) (int, error)`
    简化的连接方法（使用默认配置），返回虚拟 fd

-   `Close()`
    关闭连接

#### 数据操作

-   `SendBytes(data []byte) error`
    发送数据

-   `GetConnInfo() *ConnectOption`
    获取连接的完整信息（包含本地和远程地址）

#### 状态查询

-   `GetFd() int`
    获取虚拟 fd（-1 表示未连接）

-   `IsConnected() bool`
    检查是否已连接

### 回调接口

```go
type BaseListenerCallback struct {
	OnConnected    func(fd int, connType ConnectionType, connOpt *ConnectOption)
	OnDisconnected func(fd int, connType ConnectionType)
	OnDataReceived func(fd int, connType ConnectionType, buf []byte, used int) int
}
```

#### 回调机制说明

**ConnectionType 类型：**

```go
const (
	ConnectionTypeServer ConnectionType = iota  // 服务端
	ConnectionTypeClient                        // 客户端
)
```

**服务端回调时机：**

| 回调函数         | 触发时机       | fd 值 | connType               | 说明                 |
| ---------------- | -------------- | ----- | ---------------------- | -------------------- |
| `OnConnected`    | 监听启动成功   | `0`   | `ConnectionTypeServer` | 服务端开始监听       |
| `OnConnected`    | 客户端连接成功 | `>0`  | `ConnectionTypeClient` | 新客户端连接         |
| `OnDataReceived` | 收到客户端数据 | `>0`  | `ConnectionTypeClient` | 处理客户端发送的数据 |
| `OnDisconnected` | 客户端断开连接 | `>0`  | `ConnectionTypeClient` | 客户端主动或异常断开 |

**客户端回调时机：**

| 回调函数         | 触发时机       | fd 值 | connType               | 说明                 |
| ---------------- | -------------- | ----- | ---------------------- | -------------------- |
| `OnConnected`    | 连接服务端成功 | `>0`  | `ConnectionTypeClient` | 客户端连接建立       |
| `OnDataReceived` | 收到服务端数据 | `>0`  | `ConnectionTypeClient` | 处理服务端返回的数据 |
| `OnDisconnected` | 与服务端断开   | `>0`  | `ConnectionTypeClient` | 连接断开             |

**OnDataReceived 返回值：**

-   `> 0`：已处理的字节数，剩余数据保留在缓冲区
-   `0`：等待更多数据（暂不处理）
-   `-1`：处理失败，关闭连接

**示例：处理返回值**

```go
OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
    // 假设协议格式：2字节长度 + N字节数据
    if used < 2 {
        return 0 // 等待更多数据
    }

    length := int(buf[0])<<8 | int(buf[1])
    if used < 2+length {
        return 0 // 包不完整，等待
    }

    // 处理完整的数据包
    data := buf[2 : 2+length]
    fmt.Printf("收到完整数据包 (fd=%d, type=%v): %s\n", fd, connType, string(data))

    return 2 + length // 返回已处理的字节数
}
```

**示例：区分服务端监听和客户端连接**

```go
callback := &netconn.BaseListenerCallback{
	OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
		if connType == netconn.ConnectionTypeServer {
			// 服务端监听启动
			fmt.Printf("服务器监听: %s:%d\n", connOpt.LocalSocket.Addr, connOpt.LocalSocket.Port)
		} else {
			// 客户端连接
			fmt.Printf("客户端连接 fd=%d\n", fd)
		}
	},
	OnDisconnected: func(fd int, connType netconn.ConnectionType) {
		if connType == netconn.ConnectionTypeClient {
			fmt.Printf("客户端断开 fd=%d\n", fd)
		}
	},
}
```

### 连接管理器（ConnectionManager）

#### 创建

-   `NewConnectionManager() *ConnectionManager`
    创建新的连接管理器

#### 连接操作

-   `RegisterConn(conn NetConnection, connType ConnectionType) int`
    注册连接并分配虚拟 fd

-   `UnregisterConn(fd int)`
    注销连接

-   `GetConn(fd int) (NetConnection, bool)`
    获取连接

-   `CloseConn(fd int) error`
    关闭指定连接

-   `CloseAll()`
    关闭所有连接

#### 数据操作

-   `SendBytes(fd int, data []byte) error`
    通过虚拟 fd 发送数据

-   `GetConnInfo(fd int) *ConnectOption`
    获取连接的完整信息（包含本地和远程地址）

#### 查询

-   `GetConnType(fd int) (ConnectionType, bool)`
    获取连接类型（服务端/客户端）

-   `GetAllFds() []int`
    获取所有活跃的虚拟 fd

-   `GetConnCount() int`
    获取连接总数

-   `AllocateFd(connType ConnectionType) int`
    分配虚拟 fd（从 1000 开始全局自增）

## 高级用法

### 共享连接管理器

服务端和客户端可以共用同一个 `ConnectionManager`，实现统一的连接管理：

```go
// 创建共享的连接管理器
sharedMgr := netconn.NewConnectionManager()

// 创建服务端（使用共享管理器）
server := netconn.NewBaseServer(sharedMgr)
server.StartBaseListener(serverOpt, serverCallback)

// 创建客户端（使用共享管理器）
client := netconn.NewBaseClient(sharedMgr, clientCallback)
client.ConnectSimple(netconn.ProtocolTCP, "127.0.0.1", 8080)

// 通过共享管理器查看所有连接
fmt.Printf("当前总连接数: %d\n", sharedMgr.GetConnCount())
for _, fd := range sharedMgr.GetAllFds() {
	connType, _ := sharedMgr.GetConnType(fd)
	fmt.Printf("  fd=%d, type=%v\n", fd, connType)
}
```

### 共享回调函数

服务端和客户端可以共用同一个回调函数，通过 `ConnectionType` 参数区分连接类型：

```go
// 共享的回调函数
sharedCallback := &netconn.BaseListenerCallback{
	OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
		if connType == netconn.ConnectionTypeServer {
			fmt.Printf("[服务端] 监听启动或客户端连接 fd=%d\n", fd)
		} else {
			fmt.Printf("[客户端] 连接成功 fd=%d\n", fd)
		}
	},
	OnDisconnected: func(fd int, connType netconn.ConnectionType) {
		typeStr := map[netconn.ConnectionType]string{
			netconn.ConnectionTypeServer: "服务端",
			netconn.ConnectionTypeClient: "客户端",
		}[connType]
		fmt.Printf("[%s] fd=%d 断开\n", typeStr, fd)
	},
	OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
		fmt.Printf("[fd=%d] 收到数据: %s\n", fd, string(buf[:used]))
		return used
	},
}

// 服务端和客户端都使用同一个回调
server := netconn.NewBaseServer(nil)
server.StartBaseListener(serverOpt, sharedCallback)

client := netconn.NewBaseClient(nil, sharedCallback)
client.ConnectSimple(netconn.ProtocolTCP, "127.0.0.1", 8080)
```

## 示例程序

查看 `examples/netconn_example/` 目录：

-   `tcp_server.go` - TCP 服务端示例
-   `tcp_client.go` - TCP 客户端示例
-   `udp_server.go` - UDP/FILLP 服务端示例
-   `udp_client.go` - UDP/FILLP 客户端示例

## 测试

```bash
cd pkg/netconn
go test -v
```

## 许可证

MIT License
