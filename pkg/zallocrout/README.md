# Zallocrout - 零分配通用路由器

零分配、高性能通用路由器，基于压缩 Trie 树和无锁设计，实现亚微秒级延迟。支持 HTTP、RPC、CLI 等多种场景。

## 特性

-   **零内存分配**：路由匹配核心流程实现 0 allocs/op，彻底规避 GC 压力，保障高并发场景下的性能稳定性
-   **无锁并发设计**：静态路由采用无锁查找机制，并发扩展能力优异，高负载下性能波动控制在 2% 以内
-   **极致性能**：单参数路由匹配低至 35.36 ns/op，多参数场景下性能显著优于同类主流框架，展现出强劲的处理效率
-   **热点缓存**：搭载 16 分片缓存架构，结合无锁读取设计，进一步提升高频路由的匹配速度
-   **通用设计**：基于 context.Context 标准化设计，无缝适配 HTTP、RPC、CLI 等多种业务场景
-   **生产就绪**：内置完善的监控指标、路由合法性验证及优雅降级机制，满足企业级应用的稳定性要求

## 设计理念

zallocrout 通过以下方式实现极致性能：

1. **零分配设计**：消除所有不必要的内存分配，无 GC 压力
2. **无锁并发架构**：
    - 静态路由：完全无锁哈希查找（90%+ 请求）
    - 参数路由：原子指针 + 细粒度锁
    - 热点缓存：Copy-on-Write 实现无锁读取
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
-   极致性能（35.36 ns/op）

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

2. **路由优先级**

    - 静态路由 vs 参数路由优先级
    - 参数路由 vs 通配符路由优先级
    - 混合路由类型场景

3. **边界情况**

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

-   ✅ 单元测试：使用 `httptest` 快速验证路由逻辑
-   ✅ 集成测试：使用 `pkg/netconn` 进行真实 TCP 连接测试
-   ✅ 端到端验证：服务端和客户端都使用 netconn 实现
-   ✅ 并发场景：验证 10 个并发客户端

详细文档：[examples/zallocrout_example/http/README.md](../../examples/zallocrout_example/http/README.md)

#### RPC 集成测试

```bash
# 运行 RPC 测试
cd examples/zallocrout_example/rpc
go test -v
```

测试内容：

-   JSON-RPC 2.0 协议实现
-   方法路由和参数解析
-   错误处理（Method not found）

#### CLI 集成测试

```bash
# 运行 CLI 测试
cd examples/zallocrout_example/cli
go test -v
```

测试内容：

-   命令行参数解析
-   子命令路由
-   未知命令处理

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

### 框架对比基准测试

与主流 Go HTTP 路由框架的性能对比（Intel i7-12700 @ 3.30GHz）。

<details>
<summary>点击展开查看完整测试命令和原始输出</summary>

**测试命令**：

```bash
cd internal/go-http-routing-benchmark
go test -bench="(Gin|HttpRouter|Echo|Zallocrout)_(Param|Param20|GithubAll|StaticAll)$" -benchmem -benchtime=1s
```

**原始输出**：

```
#GithubAPI Routes: 203
   Echo: 97576 Bytes
   Gin: 58280 Bytes
   HttpRouter: 37072 Bytes
   Zallocrout: 190376 Bytes

#Static Routes: 157
   Echo: 78120 Bytes
   Gin: 34488 Bytes
   HttpRouter: 21680 Bytes
   Zallocrout: 133856 Bytes

BenchmarkEcho_Param           	43272403	        27.57 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_Param            	31027795	        40.91 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_Param     	29167202	        43.39 ns/op	      32 B/op	       1 allocs/op
BenchmarkZallocrout_Param     	34413535	        35.36 ns/op	       0 B/op	       0 allocs/op

BenchmarkEcho_Param20         	 5506965	       223.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_Param20          	 5825694	       187.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_Param20   	 2970740	       413.7 ns/op	     704 B/op	       1 allocs/op
BenchmarkZallocrout_Param20   	24945846	        42.62 ns/op	       0 B/op	       0 allocs/op

BenchmarkEcho_GithubAll       	   78099	     15977 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_GithubAll        	   72324	     15446 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubAll 	   60429	     19237 ns/op	   13792 B/op	     167 allocs/op
BenchmarkZallocrout_GithubAll 	  117910	     10302 ns/op	       0 B/op	       0 allocs/op

BenchmarkEcho_StaticAll       	  118599	      9946 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_StaticAll        	  125265	      9408 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_StaticAll 	  238275	      5177 ns/op	       0 B/op	       0 allocs/op
BenchmarkZallocrout_StaticAll 	  167029	      7155 ns/op	       0 B/op	       0 allocs/op
```

</details>

**性能对比总结**：

#### 单参数路由 (Param)

| 框架           | 性能            | 内存分配                | 对比 Zallocrout |
| -------------- | --------------- | ----------------------- | --------------- |
| Echo           | **27.57 ns/op** | 0 B/op, 0 allocs/op     | **1.28x 快**    |
| **Zallocrout** | **35.36 ns/op** | **0 B/op, 0 allocs/op** | **基准**        |
| Gin            | 40.91 ns/op     | 0 B/op, 0 allocs/op     | 1.16x 慢        |
| HttpRouter     | 43.39 ns/op     | 32 B/op, 1 allocs/op    | 1.23x 慢        |

#### 20 参数路由 (Param20)

| 框架           | 性能            | 内存分配                | 对比 Zallocrout |
| -------------- | --------------- | ----------------------- | --------------- |
| **Zallocrout** | **42.62 ns/op** | **0 B/op, 0 allocs/op** | **基准**        |
| Gin            | 187.9 ns/op     | 0 B/op, 0 allocs/op     | **4.41x 慢** ⚡ |
| Echo           | 223.2 ns/op     | 0 B/op, 0 allocs/op     | **5.24x 慢** ⚡ |
| HttpRouter     | 413.7 ns/op     | 704 B/op, 1 allocs/op   | **9.71x 慢** ⚡ |

#### GitHub API (203 路由)

| 框架           | 性能             | 内存分配                   | 对比 Zallocrout |
| -------------- | ---------------- | -------------------------- | --------------- |
| **Zallocrout** | **10,302 ns/op** | **0 B/op, 0 allocs/op**    | **基准**        |
| Gin            | 15,446 ns/op     | 0 B/op, 0 allocs/op        | 1.50x 慢        |
| Echo           | 15,977 ns/op     | 0 B/op, 0 allocs/op        | 1.55x 慢        |
| HttpRouter     | 19,237 ns/op     | 13,792 B/op, 167 allocs/op | 1.87x 慢        |

#### 静态路由 (StaticAll - 157 路由)

| 框架           | 性能            | 内存分配                | 对比 Zallocrout |
| -------------- | --------------- | ----------------------- | --------------- |
| HttpRouter     | **5,177 ns/op** | 0 B/op, 0 allocs/op     | **1.38x 快**    |
| **Zallocrout** | **7,155 ns/op** | **0 B/op, 0 allocs/op** | **基准**        |
| Gin            | 9,408 ns/op     | 0 B/op, 0 allocs/op     | 1.31x 慢        |
| Echo           | 9,946 ns/op     | 0 B/op, 0 allocs/op     | 1.39x 慢        |

**关键优势**：

-   ⚡ **多参数场景领先**：在 Param20 场景下比 Gin 快 **4.41 倍**，比 HttpRouter 快 **9.71 倍**
-   🚀 **复杂路由优势明显**：GitHub API 场景比 Gin 快 50%，比 HttpRouter 快 87%
-   💎 **零内存分配**：所有场景保持 0 allocs/op，无 GC 压力
-   🎯 **综合性能优秀**：在多参数和复杂路由场景下全面领先

**内存占用对比**：

| API               | Zallocrout | Gin      | HttpRouter | Echo     |
| ----------------- | ---------- | -------- | ---------- | -------- |
| GitHub (203 路由) | 190,376 B  | 58,280 B | 37,072 B   | 97,576 B |
| Static (157 路由) | 133,856 B  | 34,488 B | 21,680 B   | 78,120 B |

_注：Zallocrout 内存占用较高是因为包含热点缓存、性能指标等生产级特性_

---

### 并发扩展性测试

测试 Zallocrout 在不同并发级别下的性能表现（Intel i7-12700 @ 3.30GHz）。

<details>
<summary>点击展开查看完整测试命令和原始输出</summary>

**测试命令**：

```bash
cd internal/go-http-routing-benchmark
go test -bench="(Gin|HttpRouter|Echo|Zallocrout)_(Param20|GithubAll)$" -benchmem -benchtime=3s -cpu=1,8,16
```

**原始输出**：

```
BenchmarkEcho_Param20               	16499728	       220.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkEcho_Param20-8             	16496673	       218.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkEcho_Param20-16            	16390389	       219.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_Param20                	17309340	       204.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_Param20-8              	17600047	       183.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_Param20-16             	17668728	       205.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_Param20         	 8813630	       401.0 ns/op	     704 B/op	       1 allocs/op
BenchmarkHttpRouter_Param20-8       	 8409320	       414.5 ns/op	     704 B/op	       1 allocs/op
BenchmarkHttpRouter_Param20-16      	 8652339	       449.0 ns/op	     704 B/op	       1 allocs/op
BenchmarkZallocrout_Param20         	84818840	        41.87 ns/op	       0 B/op	       0 allocs/op
BenchmarkZallocrout_Param20-8       	84569575	        42.17 ns/op	       0 B/op	       0 allocs/op
BenchmarkZallocrout_Param20-16      	84060691	        41.52 ns/op	       0 B/op	       0 allocs/op
BenchmarkEcho_GithubAll             	  235902	     15433 ns/op	       0 B/op	       0 allocs/op
BenchmarkEcho_GithubAll-8           	  234062	     15487 ns/op	       0 B/op	       0 allocs/op
BenchmarkEcho_GithubAll-16          	  225049	     15487 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_GithubAll              	  223654	     14959 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_GithubAll-8            	  242713	     31111 ns/op	       0 B/op	       0 allocs/op
BenchmarkGin_GithubAll-16           	  238107	     14671 ns/op	       0 B/op	       0 allocs/op
BenchmarkHttpRouter_GithubAll       	  201024	     17647 ns/op	   13792 B/op	     167 allocs/op
BenchmarkHttpRouter_GithubAll-8     	  232092	     15693 ns/op	   13792 B/op	     167 allocs/op
BenchmarkHttpRouter_GithubAll-16    	  238790	     16094 ns/op	   13792 B/op	     167 allocs/op
BenchmarkZallocrout_GithubAll       	  379326	      9596 ns/op	       0 B/op	       0 allocs/op
BenchmarkZallocrout_GithubAll-8     	  357009	      9763 ns/op	       0 B/op	       0 allocs/op
BenchmarkZallocrout_GithubAll-16    	  374456	      9719 ns/op	       0 B/op	       0 allocs/op
```

</details>

**并发扩展性分析**：

#### Param20 场景（20 个参数）

| 框架           | 1 CPU        | 8 CPU        | 16 CPU       | 性能波动    |
| -------------- | ------------ | ------------ | ------------ | ----------- |
| **Zallocrout** | **41.87 ns** | **42.17 ns** | **41.52 ns** | **< 2%** ⚡ |
| Gin            | 204.4 ns     | 183.0 ns     | 205.7 ns     | 12%         |
| Echo           | 220.5 ns     | 218.4 ns     | 219.0 ns     | < 1%        |
| HttpRouter     | 401.0 ns     | 414.5 ns     | 449.0 ns     | 12% ⚠️      |

#### GithubAll 场景（203 个路由）

| 框架           | 1 CPU        | 8 CPU         | 16 CPU       | 性能波动    |
| -------------- | ------------ | ------------- | ------------ | ----------- |
| **Zallocrout** | **9,596 ns** | **9,763 ns**  | **9,719 ns** | **< 2%** ⚡ |
| Gin            | 14,959 ns    | **31,111 ns** | 14,671 ns    | **108%** ⚠️ |
| Echo           | 15,433 ns    | 15,487 ns     | 15,487 ns    | < 1%        |
| HttpRouter     | 17,647 ns    | 15,693 ns     | 16,094 ns    | 11%         |

**并发总结**：

-   🎯 **Zallocrout 并发扩展性优秀**：在 1-16 CPU 下性能波动 < 2%，无锁设计效果显著
-   ⚠️ **Gin 高并发性能问题**：在 8 CPU 时 GithubAll 性能下降 **2 倍**（14,959ns → 31,111ns）
-   ⚠️ **HttpRouter 并发性能下降**：16 CPU 时 Param20 性能下降 12%（401ns → 449ns）
-   ✅ **零内存分配优势**：Zallocrout 在高并发下保持 0 allocs/op，无 GC 压力

---

### 内部基准测试

内部组件性能测试结果（Intel i7-12700 @ 3.30GHz）：

```
Context 操作（零分配）：
BenchmarkRouteContext_GetParam-8       764767176     1.562 ns/op    0 B/op    0 allocs/op
BenchmarkRouteContext_SetValue-8        96654154    12.67 ns/op    0 B/op    0 allocs/op
BenchmarkRouteContext_Value-8          517993138     2.333 ns/op    0 B/op    0 allocs/op
BenchmarkContextPool-8                 100000000    11.37 ns/op    0 B/op    0 allocs/op
BenchmarkContextPool_Parallel-8        582329784     1.948 ns/op    0 B/op    0 allocs/op

核心组件（零分配）：
BenchmarkRouteNode_FindStaticChild-8   211180455     5.712 ns/op    0 B/op    0 allocs/op
BenchmarkRouteNode_FindParamChild-8   1000000000     0.2415 ns/op   0 B/op    0 allocs/op
BenchmarkNormalizePathBytes-8           37844979    33.18 ns/op    0 B/op    0 allocs/op
BenchmarkSplitPathToCompressedSegs-8    47397669    23.13 ns/op    0 B/op    0 allocs/op
BenchmarkUnsafeString-8               1000000000     0.2960 ns/op   0 B/op    0 allocs/op
```

**性能说明**：

-   ✅ **零分配保证**：所有核心操作均为 0 allocs/op
-   ✅ **极速参数访问**：GetParam 仅需 1.56 ns/op
-   ✅ **高并发性能**：并行 Context 池化操作仅 1.95 ns/op
-   ✅ **无锁查找**：静态子节点查找 5.71 ns/op，参数子节点查找 0.24 ns/op

## 实现细节

### 零分配 Context 设计

-   固定数组存储参数：`[MaxParams]paramPair`（栈分配）
-   固定数组存储自定义值：`[MaxValues]valuePair`（栈分配）
-   Context 池化：复用 routeContext 结构
-   完全零堆内存分配，无 GC 压力

### 无锁并发架构

**静态路由（90%+ 请求）**：

-   静态子节点存储在只读 map 中
-   完全无锁哈希查找，O(1) 复杂度
-   并发读取零竞争

**参数路由**：

-   使用 `atomic.Pointer[RouteNode]` 实现无锁读取
-   写入时使用细粒度锁 + 双重检查
-   最小化锁竞争范围

**热点缓存**：

-   Copy-on-Write 策略：读取完全无锁
-   16 个分片降低写入竞争
-   WyHash 快速哈希分布
-   每分片 LRU 淘汰（满时淘汰 10%）

**性能优势**：

-   并发扩展性优秀：1-16 CPU 性能波动 < 2%
-   避免锁竞争导致的性能下降
-   高并发场景下性能稳定

## 限制

-   每个路由最多 32 个参数（覆盖 99.9% 场景）
-   每个 context 最多 6 个自定义值（可通过标准 context.WithValue 扩展）
-   缓存限制为 16,000 条目（每分片 1000 条）
-   通配符路由不会被缓存

## 许可证

MIT License
