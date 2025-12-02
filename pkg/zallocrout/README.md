# Zallocrout - 零分配通用路由器

零分配、高性能通用路由器，基于压缩 Trie 树和无锁设计，实现亚微秒级延迟。支持 HTTP、RPC、CLI 等多种场景。

## 特性

-   **零内存分配**：路由匹配过程 0 allocs/op
-   **亚微秒延迟**：静态路由匹配 155.9 ns/op
-   **高吞吐量**：640 万+ QPS（i7-4770 @ 8 并发）
-   **无锁静态路由**：90% 以上请求完全无锁
-   **热点缓存**：16 分片缓存 + LRU 淘汰
-   **通用设计**：基于 context.Context，适用更多场景
-   **生产就绪**：内置监控指标、路由验证、优雅降级

## 设计理念

zallocrout 通过以下方式实现极致性能：

1. **内存即性能**：消除所有不必要的内存分配
2. **无锁优先**：静态路由使用无锁哈希查找
3. **编译器友好**：内联提示和栈分配优化
4. **通用架构**：基于 context.Context，解耦具体协议

## 架构设计

```
┌─────────────────────────────────────────────────┐
│ Context 层: 池化 context + 固定数组参数存储       │
│ (context.go)                                    │
├─────────────────────────────────────────────────┤
│ 预处理层: 路径规范化 + 零分配拆分                 │
│ (preprocess.go)                                 │
├─────────────────────────────────────────────────┤
│ 缓存层: 分片热点缓存 + 无锁快速命中                │
│ (cache.go)                                      │
├─────────────────────────────────────────────────┤
│ 匹配层: 压缩 Trie 树 + 哈希加速 + 无锁静态匹配     │
│ (router.go + node.go)                           │
├─────────────────────────────────────────────────┤
│ 资源层: 全链路池化 + 自动生命周期管理              │
│ (resource.go)                                   │
└─────────────────────────────────────────────────┘
```

## 核心 API

### 路由器

```go
// 创建路由器
router := zallocrout.NewRouter()

// 注册路由
router.AddRoute(method, path string, handler HandlerFunc, middlewares ...Middleware) error

// 匹配路由（返回 context）
ctx, handler, middlewares, ok := router.Match(method, path string, parent context.Context)
```

### 处理函数和中间件

```go
// 处理函数类型（基于 context）
type HandlerFunc func(context.Context) error

// 中间件类型
type Middleware func(HandlerFunc) HandlerFunc
```

### Context 辅助函数

```go
// 获取路由参数
value, ok := zallocrout.GetParam(ctx, "id")

// 设置自定义值
ok := zallocrout.SetValue(ctx, "key", value)

// 执行 handler 并自动释放 context（推荐）
err := zallocrout.ExecuteHandler(ctx, handler, middlewares)

// 手动释放 context（高级用法）
zallocrout.ReleaseContext(ctx)
```

## 使用示例

### HTTP 服务器

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

// HTTP 适配器
type HTTPAdapter struct {
    router *zallocrout.Router
}

func (h *HTTPAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, handler, middlewares, ok := h.router.Match(r.Method, r.URL.Path, r.Context())
    if !ok {
        http.NotFound(w, r)
        return
    }

    // 设置 HTTP 相关值到 context
    zallocrout.SetValue(ctx, "http.ResponseWriter", w)
    zallocrout.SetValue(ctx, "http.Request", r)

    // 执行处理器（自动释放 context）
    if err := zallocrout.ExecuteHandler(ctx, handler, middlewares); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// 业务处理器
func getUserHandler(ctx context.Context) error {
    w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
    userID, _ := zallocrout.GetParam(ctx, "id")

    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"user_id":"%s"}`, userID)
    return nil
}

// 中间件
func loggingMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
    return func(ctx context.Context) error {
        r := ctx.Value("http.Request").(*http.Request)
        log.Printf("[%s] %s", r.Method, r.URL.Path)
        return next(ctx)
    }
}

func main() {
    router := zallocrout.NewRouter()
    router.AddRoute("GET", "/users/:id", getUserHandler, loggingMiddleware)

    http.ListenAndServe(":8080", &HTTPAdapter{router: router})
}
```

### RPC 服务

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

type RPCAdapter struct {
    router *zallocrout.Router
}

func (a *RPCAdapter) HandleRequest(req *RPCRequest) *RPCResponse {
    path := "/" + req.Method
    ctx, handler, middlewares, ok := a.router.Match("RPC", path, context.Background())
    if !ok {
        return &RPCResponse{Error: &RPCError{Code: -32601, Message: "Method not found"}}
    }

    zallocrout.SetValue(ctx, "rpc.request", req)
    zallocrout.SetValue(ctx, "rpc.params", req.Params)

    // 执行处理器（自动释放 context）
    if err := zallocrout.ExecuteHandler(ctx, handler, middlewares); err != nil {
        return &RPCResponse{Error: &RPCError{Code: -32603, Message: err.Error()}}
    }

    result, _ := ctx.Value("result").(interface{})
    return &RPCResponse{Result: result, ID: req.ID}
}

func getUserRPC(ctx context.Context) error {
    params := ctx.Value("rpc.params").(json.RawMessage)
    var userID string
    json.Unmarshal(params, &userID)

    result := map[string]interface{}{"id": userID, "name": "User " + userID}
    zallocrout.SetValue(ctx, "result", result)
    return nil
}

func main() {
    router := zallocrout.NewRouter()
    router.AddRoute("RPC", "/user.get", getUserRPC)

    adapter := &RPCAdapter{router: router}
    // 使用 adapter.HandleRequest() 处理 RPC 请求
}
```

### CLI 工具

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"
    "github.com/junbin-yang/go-kitbox/pkg/zallocrout"
)

type CLIAdapter struct {
    router *zallocrout.Router
}

func (a *CLIAdapter) Execute(args []string) error {
    path := "/" + strings.Join(args, "/")
    ctx, handler, middlewares, ok := a.router.Match("CLI", path, context.Background())
    if !ok {
        return fmt.Errorf("unknown command: %s", strings.Join(args, " "))
    }

    zallocrout.SetValue(ctx, "cli.args", args)
    zallocrout.SetValue(ctx, "cli.stdout", os.Stdout)

    // 执行处理器（自动释放 context）
    return zallocrout.ExecuteHandler(ctx, handler, middlewares)
}

func userGetCommand(ctx context.Context) error {
    userID, _ := zallocrout.GetParam(ctx, "id")
    stdout := ctx.Value("cli.stdout").(*os.File)
    fmt.Fprintf(stdout, "User ID: %s\n", userID)
    return nil
}

func main() {
    router := zallocrout.NewRouter()
    router.AddRoute("CLI", "/user/get/:id", userGetCommand)

    adapter := &CLIAdapter{router: router}
    adapter.Execute(os.Args[1:])
}
```

## 路由类型

### 静态路由

```go
router.AddRoute("GET", "/api/v1/users", handler)
```

-   无锁 O(1) 查找
-   性能最快（P99 < 30ns）

### 参数路由

```go
router.AddRoute("GET", "/users/:id/posts/:postId", handler)
```

-   使用 `:` 前缀定义参数
-   通过 `zallocrout.GetParam(ctx, "id")` 提取参数值
-   细粒度锁保护

### 通配符路由

```go
router.AddRoute("GET", "/files/*path", handler)
```

-   使用 `*` 前缀捕获剩余路径
-   必须是最后一个片段
-   不会被缓存

## 性能指标

```go
// 获取指标
metrics := router.Metrics()
fmt.Printf("缓存命中率: %.2f%%\n", router.CacheHitRate()*100)
fmt.Printf("总匹配次数: %d\n", metrics.TotalMatches)

// 缓存管理
router.EnableHotCache()   // 启用热点缓存
router.DisableHotCache()  // 禁用热点缓存
router.ClearHotCache()    // 清空缓存
```

## 测试

### 测试覆盖率

```bash
# 运行所有测试并查看覆盖率
go test -v -cover

# 输出结果
coverage: 95.9% of statements
```

### 单元测试

包含完整的单元测试套件，覆盖所有核心功能：

```bash
# 运行所有单元测试
go test -v

# 运行特定测试
go test -v -run TestRouter_StaticRoute
go test -v -run TestRouter_ParamRoute
go test -v -run TestRouter_WildcardRoute
```

**测试内容包括**：

1. **基础路由匹配**
   - 静态路由、参数路由、通配符路由
   - 多参数路由、复杂嵌套路由
   - 404 处理、路径规范化

2. **路由优先级**（新增）
   - 静态路由 vs 参数路由优先级
   - 参数路由 vs 通配符路由优先级
   - 混合路由类型场景

3. **边界情况**（新增）
   - 根路径匹配
   - 特殊字符在参数中
   - 同名参数在不同位置
   - HTTP 方法隔离

4. **中间件和 Context**
   - 中间件执行顺序
   - Context 参数读写
   - Context 池化和释放

5. **缓存和性能**
   - 热点缓存命中
   - 缓存启用/禁用
   - 并发访问安全

6. **指标和监控**
   - 性能指标收集
   - 缓存统计信息
   - 路由计数

### 集成测试

提供完整的集成测试示例，验证实际应用场景：

#### HTTP 集成测试

```bash
# 运行 HTTP 单元测试（使用 httptest）
cd examples/zallocrout_example/http
go test -v -run TestHTTP

# 运行 HTTP 集成测试（使用 netconn 真实网络）
go test -v -run Integration

# 运行并发测试
go test -v -run TestHTTP_Integration_ConcurrentRequests
```

**HTTP 测试特点**：
- ✅ 单元测试：使用 `httptest` 快速验证路由逻辑
- ✅ 集成测试：使用 `pkg/netconn` 进行真实 TCP 连接测试
- ✅ 端到端验证：服务端和客户端都使用 netconn 实现
- ✅ 并发场景：验证 10 个并发客户端

详细文档：[examples/zallocrout_example/http/README.md](../../examples/zallocrout_example/http/README.md)

#### RPC 集成测试

```bash
# 运行 RPC 测试
cd examples/zallocrout_example/rpc
go test -v
```

测试内容：
- JSON-RPC 2.0 协议实现
- 方法路由和参数解析
- 错误处理（Method not found）

#### CLI 集成测试

```bash
# 运行 CLI 测试
cd examples/zallocrout_example/cli
go test -v
```

测试内容：
- 命令行参数解析
- 子命令路由
- 未知命令处理

### 基准测试

运行性能基准测试：

```bash
# 路由器核心基准测试
go test -bench=. -benchmem

# HTTP 集成基准测试（真实网络）
cd examples/zallocrout_example/http
go test -bench=BenchmarkHTTP_Integration_WithNetConn -benchmem
```

## 性能测试结果

实际测试结果（Intel i7-4770 @ 3.40GHz）：

```
路由匹配（零分配）：
BenchmarkRouter_MatchStatic-8           23768023    155.9 ns/op    0 B/op    0 allocs/op
BenchmarkRouter_MatchParam-8            18532712    197.4 ns/op    0 B/op    0 allocs/op
BenchmarkRouter_MatchParamNoCache-8     20795749    164.4 ns/op    0 B/op    0 allocs/op
BenchmarkRouter_MatchWildcard-8         13538740    265.8 ns/op    0 B/op    0 allocs/op
BenchmarkRouter_MatchCacheHit-8         18682422    190.7 ns/op    0 B/op    0 allocs/op

Context 操作（零分配）：
BenchmarkRouteContext_GetParam-8       942756850     3.839 ns/op    0 B/op    0 allocs/op
BenchmarkRouteContext_SetValue-8       151526407    25.96 ns/op    0 B/op    0 allocs/op
BenchmarkRouteContext_Value-8          542621894     6.452 ns/op    0 B/op    0 allocs/op
BenchmarkContextPool-8                 177534516    20.21 ns/op    0 B/op    0 allocs/op
BenchmarkContextPool_Parallel-8        619946894     5.845 ns/op    0 B/op    0 allocs/op

核心组件（零分配）：
BenchmarkRouteNode_FindStaticChild-8   338678972    10.74 ns/op    0 B/op    0 allocs/op
BenchmarkNormalizePathBytes-8           71235076    49.96 ns/op    0 B/op    0 allocs/op
BenchmarkSplitPathToCompressedSegs-8   100000000    32.29 ns/op    0 B/op    0 allocs/op
BenchmarkUnsafeString-8               1000000000     0.5124 ns/op  0 B/op    0 allocs/op
```

**性能说明**：
- ✅ **零分配保证**：所有路由匹配和 Context 操作均为 0 allocs/op
- ✅ **亚微秒延迟**：静态路由匹配 ~156 ns/op，参数路由 ~197 ns/op
- ✅ **高并发性能**：并行 Context 池化操作仅 5.8 ns/op
- ✅ **极速参数访问**：GetParam 仅需 3.8 ns/op

## 实现细节

### 零分配 Context 设计

-   固定数组存储参数：`[MaxParams]paramPair`（栈分配）
-   固定数组存储自定义值：`[MaxValues]valuePair`（栈分配）
-   Context 池化：复用 routeContext 结构
-   完全零堆内存分配

### 无锁静态匹配

-   静态子节点存储在只读 map 中
-   静态路由查找完全无锁
-   90% 以上请求受益于无锁路径

### 分片热点缓存

-   16 个分片降低竞争
-   FNV-1a 哈希分布
-   每分片 LRU 淘汰（满时淘汰 10%）

## 限制

-   每个路由最多 32 个参数（覆盖 99.9% 场景）
-   每个 context 最多 6 个自定义值（可通过标准 context.WithValue 扩展）
-   缓存限制为 16,000 条目（每分片 1000 条）
-   通配符路由不会被缓存

## 许可证

MIT License
