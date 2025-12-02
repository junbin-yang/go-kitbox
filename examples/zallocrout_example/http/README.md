# HTTP 集成测试说明

本目录包含两种类型的测试：

## 1. 单元测试 (`main_test.go`)

使用 `net/http/httptest` 包进行模拟 HTTP 请求测试，无需启动真实服务器。

**特点**：
- ✅ 快速执行（无网络 I/O）
- ✅ 隔离测试（每个测试独立）
- ✅ 适合 CI/CD 环境

**运行方式**：
```bash
go test -v -run TestHTTP
```

## 2. 集成测试 (`integration_test.go`)

**服务端和客户端都使用 `pkg/netconn` 包进行真实网络连接的集成测试**。

**特点**：
- ✅ 真实网络连接（TCP）
- ✅ 完整的 HTTP 协议处理
- ✅ 端到端验证（服务端和客户端都使用netconn）
- ✅ 并发场景测试
- ✅ 验证路由器在真实网络环境下的表现

**测试内容**：

### TestHTTP_Integration_WithNetConn
测试基本的 HTTP 请求场景：
- `GET /` - 首页
- `GET /users/123` - 参数路由
- `GET /users/123/posts/456` - 多参数路由
- `GET /nonexistent` - 404 错误

### TestHTTP_Integration_ConcurrentRequests
测试并发请求场景：
- 10 个并发客户端
- 同时发送请求到服务器
- 验证响应正确性

### BenchmarkHTTP_Integration_WithNetConn
性能基准测试：
- 复用 TCP 连接
- 测量真实网络延迟
- 包含内存分配统计

**运行方式**：
```bash
# 运行所有集成测试
go test -v -run Integration

# 运行单个集成测试
go test -v -run TestHTTP_Integration_WithNetConn

# 运行并发测试
go test -v -run TestHTTP_Integration_ConcurrentRequests

# 运行基准测试
go test -bench=BenchmarkHTTP_Integration_WithNetConn -benchmem
```

## 测试结果示例

### 基本测试
```
=== RUN   TestHTTP_Integration_WithNetConn
=== RUN   TestHTTP_Integration_WithNetConn/GET_/
    [Server] Listening on 0.0.0.0:18080
    Connected to localhost:18080
    [Server] Client connected, fd=1001
    [Server] Request: GET /
    Request succeeded: GET / -> 200
--- PASS: TestHTTP_Integration_WithNetConn (0.12s)
```

### 并发测试
```
=== RUN   TestHTTP_Integration_ConcurrentRequests
    Successfully completed 10 concurrent requests
--- PASS: TestHTTP_Integration_ConcurrentRequests (0.12s)
```

### 基准测试
```
BenchmarkHTTP_Integration_WithNetConn-8
    17126    69674 ns/op    7744 B/op    52 allocs/op
```

## 技术细节

### 服务端实现（使用 netconn 包）

**架构设计**：
1. **使用 netconn.BaseServer** 创建 TCP 服务器
2. **HTTP 请求解析**：在 OnDataReceived 回调中解析完整的 HTTP 请求
3. **路由匹配**：使用 zallocrout.Router 匹配路由并执行handler
4. **响应捕获**：通过 responseCapture 实现 http.ResponseWriter 接口，捕获handler的响应
5. **HTTP 响应构建**：从 responseCapture 构建符合 HTTP/1.1 规范的响应
6. **网络发送**：通过 server.SendBytes 发送响应给客户端

**关键代码片段**：

```go
// 启动 netconn 服务器
server := netconn.NewBaseServer(nil)

callback := &netconn.BaseListenerCallback{
    OnDataReceived: func(fd int, connType netconn.ConnectionType,
                        buf []byte, used int) int {
        // 1. 缓存数据直到收到完整HTTP请求
        buffer.Write(buf[:used])

        // 2. 解析HTTP请求
        req, err := http.ReadRequest(reader)

        // 3. 路由匹配
        ctx, handler, middlewares, ok := router.Match(
            req.Method, req.URL.Path, context.Background())

        // 4. 创建响应捕获器
        respCapture := newResponseCapture()
        zallocrout.SetValue(ctx, "http.ResponseWriter", respCapture)
        zallocrout.SetValue(ctx, "http.Request", req)

        // 5. 执行handler
        zallocrout.ExecuteHandler(ctx, handler, middlewares)

        // 6. 构建并发送HTTP响应
        response := respCapture.buildHTTPResponseFromCapture()
        server.SendBytes(fd, response)

        return used
    },
}

server.StartBaseListener(&netconn.ServerOption{
    Protocol: netconn.ProtocolTCP,
    Addr:     "0.0.0.0",
    Port:     18080,
}, callback)
```

**responseCapture 实现**：
```go
// 实现 http.ResponseWriter 接口
type responseCapture struct {
    statusCode int
    header     http.Header
    body       *bytes.Buffer
}

func (rc *responseCapture) Header() http.Header {
    return rc.header
}

func (rc *responseCapture) Write(data []byte) (int, error) {
    return rc.body.Write(data)
}

func (rc *responseCapture) WriteHeader(statusCode int) {
    rc.statusCode = statusCode
}
```

### 客户端实现（使用 netconn 包）

1. **创建 netconn 客户端**
   ```go
   connMgr := netconn.NewConnectionManager()
   client := netconn.NewBaseClient(connMgr, callback)
   client.Connect(&netconn.ClientOption{
       Protocol:   netconn.ProtocolTCP,
       RemoteIP:   "localhost",
       RemotePort: 18080,
   })
   ```

2. **发送 HTTP 请求**
   ```go
   request := "GET /users/123 HTTP/1.1\r\n" +
             "Host: localhost:18080\r\n" +
             "Connection: close\r\n\r\n"
   client.SendBytes([]byte(request))
   ```

3. **接收并解析响应**
   ```go
   callback.OnDataReceived = func(fd int, connType ConnectionType,
                                   buf []byte, used int) int {
       // 累积接收数据
       receivedData = append(receivedData, buf[:used]...)

       // 检查响应完整性
       if isCompleteHTTPResponse(receivedData) {
           // 解析HTTP响应
           resp, _ := http.ReadResponse(reader, nil)
           // 验证状态码和响应体
       }
   }
   ```

### 响应完整性检测

集成测试实现了智能的响应完整性检测：

1. **Content-Length 模式**：解析 Content-Length 头，等待指定字节数
2. **Chunked 模式**：检测 Transfer-Encoding: chunked，等待 `0\r\n\r\n` 结束标记
3. **Connection: close 模式**：等待连接关闭

### 端口分配

为避免端口冲突，不同测试使用不同端口：
- `TestHTTP_Integration_WithNetConn`: 18080
- `TestHTTP_Integration_ConcurrentRequests`: 18081
- `BenchmarkHTTP_Integration_WithNetConn`: 18082

## 注意事项

1. **端口占用**：确保测试端口（18080-18082）未被占用
2. **超时设置**：每个测试有 5 秒超时，避免挂起
3. **并发安全**：使用 `sync.Mutex` 保护共享数据
4. **资源清理**：使用 `defer` 确保连接和服务器正确关闭
5. **缓冲区管理**：每个连接维护独立的接收缓冲区
6. **响应构建**：通过 responseCapture 实现完整的 HTTP 响应协议

## 与 httptest 的对比

| 特性 | httptest | netconn 集成测试 |
|------|----------|------------------|
| 网络层 | 模拟 | 真实 TCP |
| 服务端实现 | net/http.Server | netconn.BaseServer |
| 客户端实现 | httptest.NewRequest | netconn.BaseClient |
| 执行速度 | 极快 | 较快 |
| 覆盖范围 | 应用层 | 传输层 + 应用层 |
| 并发测试 | 简单 | 完整 |
| HTTP协议处理 | 自动 | 手动（完整控制） |
| 适用场景 | 单元测试 | 集成测试 |

## 优势分析

### netconn 集成测试的优势

1. **端到端验证**：
   - 服务端使用 netconn 处理 TCP 连接和 HTTP 协议
   - 客户端使用 netconn 发送请求
   - 完整验证整个网络通信链路

2. **真实场景模拟**：
   - 真实的 TCP 连接建立和断开
   - 真实的网络 I/O 操作
   - 真实的并发连接处理

3. **协议完整性**：
   - 测试 HTTP 请求解析的正确性
   - 测试 HTTP 响应构建的正确性
   - 测试请求/响应的完整性检测

4. **性能验证**：
   - 测量真实网络延迟
   - 验证并发处理能力
   - 发现实际部署中可能出现的性能问题

## 总结

- **单元测试** (`main_test.go`)：日常开发和 CI 使用，快速验证路由逻辑
- **集成测试** (`integration_test.go`)：发布前验证和性能测试，端到端验证网络通信

**两种测试互补，确保路由器在各种场景下的正确性和性能。**

**集成测试的独特价值**：通过 netconn 包实现服务端和客户端，完整验证了路由器在真实网络环境下与底层 TCP 通信的集成能力，这是 httptest 无法提供的。
