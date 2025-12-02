# Zallocrout Context 迁移计划

## 概述

将 zallocrout 从 HTTP 专用路由器改造为通用路由器，使用 `context.Context` 替代 `http.ResponseWriter` 和 `*http.Request`，提升通用性。

## 当前限制

当前 HandlerFunc 签名：
```go
type HandlerFunc func(http.ResponseWriter, *http.Request, map[string]string)
```

**问题**：
- 强依赖 `net/http` 包
- 只能用于 HTTP 路由
- 无法用于 RPC、消息队列、CLI 等场景

## 目标设计

### 新的 HandlerFunc 签名

```go
type HandlerFunc func(context.Context) error
type Middleware func(HandlerFunc) HandlerFunc
```

**优势**：
- 完全解耦 HTTP 依赖
- 可用于任意场景（HTTP、gRPC、消息队列、CLI 等）
- 通过 context 传递任意数据
- 标准 Go 错误处理

### Context 池化设计（零分配优化）

**核心思路**：使用自定义 context 实现，避免 `context.WithValue()` 的内存分配。

```go
// 自定义 context 实现（池化）
type routeContext struct {
    context.Context                    // 嵌入父 context
    params          map[string]string  // 路由参数（从池中获取）
    values          map[interface{}]interface{} // 用户自定义值（从池中获取）
}

// Context 池
type contextPool struct {
    ctxPool    sync.Pool
    paramsPool sync.Pool
    valuesPool sync.Pool
}

var globalContextPool = &contextPool{
    ctxPool: sync.Pool{
        New: func() interface{} {
            return &routeContext{}
        },
    },
    paramsPool: sync.Pool{
        New: func() interface{} {
            return make(map[string]string, 8)
        },
    },
    valuesPool: sync.Pool{
        New: func() interface{} {
            return make(map[interface{}]interface{}, 4)
        },
    },
}

// 从池中获取 context
func (cp *contextPool) acquire(parent context.Context, params map[string]string) *routeContext {
    ctx := cp.ctxPool.Get().(*routeContext)
    ctx.Context = parent
    ctx.params = params
    ctx.values = cp.valuesPool.Get().(map[interface{}]interface{})
    return ctx
}

// 释放 context 到池
func (cp *contextPool) release(ctx *routeContext) {
    // 清空 params
    for k := range ctx.params {
        delete(ctx.params, k)
    }
    cp.paramsPool.Put(ctx.params)

    // 清空 values
    for k := range ctx.values {
        delete(ctx.values, k)
    }
    cp.valuesPool.Put(ctx.values)

    // 重置 context
    ctx.Context = nil
    ctx.params = nil
    ctx.values = nil
    cp.ctxPool.Put(ctx)
}

// 实现 context.Context 接口
func (c *routeContext) Value(key interface{}) interface{} {
    // 优先查找自定义值
    if v, ok := c.values[key]; ok {
        return v
    }
    // 回退到父 context
    return c.Context.Value(key)
}

// 设置自定义值（零分配）
func (c *routeContext) SetValue(key, value interface{}) {
    c.values[key] = value
}

// 获取路由参数
func (c *routeContext) GetParam(key string) (string, bool) {
    val, ok := c.params[key]
    return val, ok
}

func (c *routeContext) GetParams() map[string]string {
    return c.params
}
```

**使用示例**：

```go
// Match 返回池化的 context
func (r *Router) Match(method, path string) (context.Context, bool) {
    // ...匹配逻辑...

    // 从池中获取参数 map
    params := r.resourceMgr.acquireParamMap()
    for i := 0; i < paramCount; i++ {
        params[paramPairs[i].key] = paramPairs[i].value
    }

    // 从池中获取 context
    ctx := globalContextPool.acquire(context.Background(), params)

    return ctx, true
}

// 使用完毕后释放
defer func() {
    if rctx, ok := ctx.(*routeContext); ok {
        globalContextPool.release(rctx)
    }
}()
```

## 需要修改的文件

### 1. context.go（新增文件）

**新增内容**：
```go
package zallocrout

import (
    "context"
    "sync"
    "time"
)

// 自定义 context 实现（池化）
type routeContext struct {
    context.Context
    params map[string]string
    values map[interface{}]interface{}
}

// Context 池
type contextPool struct {
    ctxPool    sync.Pool
    valuesPool sync.Pool
}

var globalContextPool = &contextPool{
    ctxPool: sync.Pool{
        New: func() interface{} { return &routeContext{} },
    },
    valuesPool: sync.Pool{
        New: func() interface{} { return make(map[interface{}]interface{}, 4) },
    },
}

func (cp *contextPool) acquire(parent context.Context, params map[string]string) *routeContext {
    ctx := cp.ctxPool.Get().(*routeContext)
    ctx.Context = parent
    ctx.params = params
    ctx.values = cp.valuesPool.Get().(map[interface{}]interface{})
    return ctx
}

func (cp *contextPool) release(ctx *routeContext) {
    for k := range ctx.values {
        delete(ctx.values, k)
    }
    cp.valuesPool.Put(ctx.values)
    ctx.Context = nil
    ctx.params = nil
    ctx.values = nil
    cp.ctxPool.Put(ctx)
}

func (c *routeContext) Deadline() (deadline time.Time, ok bool) {
    return c.Context.Deadline()
}

func (c *routeContext) Done() <-chan struct{} {
    return c.Context.Done()
}

func (c *routeContext) Err() error {
    return c.Context.Err()
}

func (c *routeContext) Value(key interface{}) interface{} {
    if v, ok := c.values[key]; ok {
        return v
    }
    return c.Context.Value(key)
}

func (c *routeContext) SetValue(key, value interface{}) {
    c.values[key] = value
}

func (c *routeContext) GetParam(key string) (string, bool) {
    val, ok := c.params[key]
    return val, ok
}

func (c *routeContext) GetParams() map[string]string {
    return c.params
}

// 辅助函数
func GetParam(ctx context.Context, key string) (string, bool) {
    if rctx, ok := ctx.(*routeContext); ok {
        return rctx.GetParam(key)
    }
    return "", false
}

func GetParams(ctx context.Context) map[string]string {
    if rctx, ok := ctx.(*routeContext); ok {
        return rctx.GetParams()
    }
    return nil
}

func SetValue(ctx context.Context, key, value interface{}) {
    if rctx, ok := ctx.(*routeContext); ok {
        rctx.SetValue(key, value)
    }
}
```

### 2. node.go（核心修改）

**修改内容**：
```go
// 旧签名
type HandlerFunc func(http.ResponseWriter, *http.Request, map[string]string)

// 新签名
type HandlerFunc func(context.Context) error

// 移除 http 包导入
// import "net/http" // 删除

// 添加 context 包导入
import "context"
```

**影响**：
- `RouteNode.handler` 类型变更
- 移除 `MatchResult` 结构（直接返回 context）

### 3. router.go（核心修改）

**修改内容**：
```go
// Match 返回池化的 context（零分配）
func (r *Router) Match(method, path string, parent context.Context) (context.Context, HandlerFunc, []Middleware, bool) {
    // ...匹配逻辑...

    // 构建参数 map（从池中获取）
    params := r.resourceMgr.acquireParamMap()
    for i := 0; i < paramCount; i++ {
        params[paramPairs[i].key] = paramPairs[i].value
    }

    // 从池中获取 context（零分配）
    ctx := globalContextPool.acquire(parent, params)

    return ctx, handler, middlewares, true
}

// 释放 context（必须调用）
func ReleaseContext(ctx context.Context) {
    if rctx, ok := ctx.(*routeContext); ok {
        // 释放参数 map
        globalResourceManager.releaseParamMap(rctx.params)
        // 释放 context
        globalContextPool.release(rctx)
    }
}
```

**影响**：
- Match 返回 context 而非 MatchResult
- 用户必须调用 ReleaseContext 释放资源
- 完全零分配（池命中时）

### 4. cache.go（缓存层修改）

**修改内容**：
```go
type cacheEntry struct {
    handler       HandlerFunc       // 类型变更
    middlewares   []Middleware      // 类型不变
    paramTemplate map[string]string // 不变
    timestamp     int64
    hitCount      uint64
}
```

**影响**：
- `cacheEntry.handler` 类型变更
- 缓存逻辑不变

### 4. resource.go（资源管理修改）

**修改内容**：
```go
// 参数 map 池已存在，无需修改
func (rm *resourceManager) acquireParamMap() map[string]string
func (rm *resourceManager) releaseParamMap(params map[string]string)
```

**影响**：
- 无需修改，已有参数 map 池

### 5. 测试文件修改

需要修改的测试文件：
- `router_test.go`
- `node_test.go`
- `cache_test.go`

**修改内容**：
```go
// 旧的测试 handler
handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {
    // ...
}

// 新的测试 handler
handler := func(ctx context.Context) error {
    params := GetParams(ctx)
    // ...
    return nil
}
```

### 6. 示例程序修改

**examples/zallocrout_example/main.go**

**修改内容**：
```go
// 旧的 HTTP 适配器
type routerHandler struct {
    router *zallocrout.Router
}

func (h *routerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    result, ok := h.router.Match(r.Method, r.URL.Path)
    if !ok {
        http.NotFound(w, r)
        return
    }
    defer result.Release()

    // 执行处理函数
    result.Handler(w, r, nil)
}

// 新的 HTTP 适配器
type routerHandler struct {
    router *zallocrout.Router
}

func (h *routerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    result, ok := h.router.Match(r.Method, r.URL.Path)
    if !ok {
        http.NotFound(w, r)
        return
    }
    defer result.Release()

    // 构建 context（包含参数和 HTTP 上下文）
    ctx := r.Context()
    ctx = zallocrout.WithParams(ctx, result.Params)
    ctx = context.WithValue(ctx, "http.ResponseWriter", w)
    ctx = context.WithValue(ctx, "http.Request", r)

    // 执行中间件链
    handler := result.Handler
    for i := len(result.Middlewares) - 1; i >= 0; i-- {
        handler = result.Middlewares[i](handler)
    }

    // 执行处理函数
    if err := handler(ctx); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// 新的业务 handler
func getUserHandler(ctx context.Context) error {
    w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
    userID, _ := zallocrout.GetParam(ctx, "id")

    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"user_id":"%s"}`, userID)
    return nil
}

// 新的中间件
func authMiddleware(next zallocrout.HandlerFunc) zallocrout.HandlerFunc {
    return func(ctx context.Context) error {
        r := ctx.Value("http.Request").(*http.Request)
        if r.Header.Get("Authorization") == "" {
            w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
            w.WriteHeader(http.StatusUnauthorized)
            fmt.Fprintf(w, `{"error":"未授权"}`)
            return nil
        }
        return next(ctx)
    }
}
```

### 7. README.md 更新

**修改内容**：
- 更新特性描述：从"HTTP 路由器"改为"通用路由器"
- 更新使用示例：展示 context 用法
- 添加 HTTP 适配器示例
- 添加其他场景示例（gRPC、CLI 等）

## 性能影响评估

### 零分配保证

**影响分析**：
1. **参数传递**：从固定数组改为 map（从池中获取）
   - 固定数组：栈分配，0 allocs
   - map 池：堆分配，但可复用，实际 0 allocs（池命中时）

2. **context 创建**：
   - `context.WithValue()` 会分配新的 context 节点
   - 每次调用 1 alloc（无法避免）

3. **总体影响**：
   - 匹配阶段：仍然 0 allocs（Match 方法）
   - 执行阶段：1-2 allocs（context 创建）
   - **结论**：核心匹配路径仍然零分配，执行阶段有少量分配

### 性能测试预期

**使用 context 池化后**：

```
// 匹配阶段（零分配，池命中时）
BenchmarkRouter_Match-20    8746125    133.8 ns/op    0 B/op    0 allocs/op

// 完整执行（含 context 创建和释放，池命中时）
BenchmarkRouter_Execute-20  8000000    150 ns/op      0 B/op    0 allocs/op
```

**关键优化**：
- context 池化：0 allocs（池命中时）
- 参数 map 池化：0 allocs（池命中时）
- values map 池化：0 allocs（池命中时）
- **总体**：完全零分配（池预热后）

## 迁移策略

### 方案 1：破坏性升级（推荐）

**优势**：
- 代码简洁，无历史包袱
- 性能最优
- 维护成本低

**劣势**：
- 破坏向后兼容性
- 需要用户修改代码

**实施**：
1. 直接修改当前代码
2. 发布 v2.0.0 版本
3. 提供迁移指南

### 方案 2：新包并存

**优势**：
- 保持向后兼容
- 用户可逐步迁移

**劣势**：
- 维护两套代码
- 代码重复

**实施**：
1. 创建新包 `zallocrout/v2`
2. 保留旧包 `zallocrout`
3. 文档说明旧包已废弃

### 方案 3：适配器模式

**优势**：
- 同时支持两种用法
- 向后兼容

**劣势**：
- 代码复杂
- 性能略有损失

**实施**：
1. 核心使用 context 设计
2. 提供 HTTP 适配器
3. 提供旧版兼容层

## 推荐方案

**选择方案 1：破坏性升级**

**理由**：
1. 当前包还未广泛使用（新包）
2. 设计更清晰，性能更好
3. 避免技术债务
4. 符合 Go 语义化版本规范（v2 可破坏兼容性）

## 实施步骤

### 阶段 1：核心代码修改（1-2 天）

1. 修改 `node.go`：更新 HandlerFunc 和 Middleware 签名
2. 修改 `router.go`：更新 Match 返回值，使用参数 map
3. 修改 `cache.go`：更新 cacheEntry 类型
4. 添加 `context.go`：实现 context 辅助函数

### 阶段 2：测试修改（1 天）

1. 修改 `router_test.go`
2. 修改 `node_test.go`
3. 修改 `cache_test.go`
4. 运行所有测试，确保通过

### 阶段 3：示例和文档（1 天）

1. 修改 `examples/zallocrout_example/main.go`
2. 更新 `README.md`
3. 添加迁移指南
4. 添加其他场景示例（gRPC、CLI）

### 阶段 4：性能测试（0.5 天）

1. 运行基准测试
2. 验证零分配保证
3. 对比性能数据

### 阶段 5：发布（0.5 天）

1. 更新 CHANGELOG
2. 打 v2.0.0 标签
3. 发布 Release Notes

**总计：3-5 天**

## 风险评估

### 高风险

- **破坏兼容性**：所有现有用户需要修改代码
  - **缓解**：提供详细迁移指南和示例

### 中风险

- **性能回退**：context 创建可能引入分配
  - **缓解**：核心匹配路径仍然零分配，影响可控

### 低风险

- **测试覆盖不足**：可能遗漏边界情况
  - **缓解**：保持现有测试覆盖率，逐个迁移测试

## 迁移指南（用户视角）

### 旧版代码

```go
router := zallocrout.NewRouter()

// 注册路由
router.AddRoute("GET", "/users/:id", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
    userID := params["id"]
    fmt.Fprintf(w, "User: %s", userID)
})

// 匹配路由
result, ok := router.Match("GET", "/users/123")
if ok {
    result.Handler(w, r, nil)
}
```

### 新版代码（使用 context 池化）

```go
router := zallocrout.NewRouter()

// 注册路由
router.AddRoute("GET", "/users/:id", func(ctx context.Context) error {
    userID, _ := zallocrout.GetParam(ctx, "id")

    // 从 context 获取 HTTP 对象
    w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
    fmt.Fprintf(w, "User: %s", userID)
    return nil
})

// 匹配路由（零分配）
ctx, handler, middlewares, ok := router.Match("GET", "/users/123", r.Context())
if ok {
    defer zallocrout.ReleaseContext(ctx) // 释放到池

    // 设置 HTTP 对象到 context
    zallocrout.SetValue(ctx, "http.ResponseWriter", w)
    zallocrout.SetValue(ctx, "http.Request", r)

    // 执行中间件链
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }

    if err := handler(ctx); err != nil {
        // 处理错误
    }
}
```

### HTTP 适配器（简化用法，零分配）

```go
// HTTP 适配器（零分配实现）
type HTTPAdapter struct {
    router *zallocrout.Router
}

func NewHTTPAdapter(router *zallocrout.Router) *HTTPAdapter {
    return &HTTPAdapter{router: router}
}

func (h *HTTPAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, handler, middlewares, ok := h.router.Match(r.Method, r.URL.Path, r.Context())
    if !ok {
        http.NotFound(w, r)
        return
    }
    defer zallocrout.ReleaseContext(ctx)

    // 设置 HTTP 对象（零分配）
    zallocrout.SetValue(ctx, "http.ResponseWriter", w)
    zallocrout.SetValue(ctx, "http.Request", r)

    // 执行中间件链
    for i := len(middlewares) - 1; i >= 0; i-- {
        handler = middlewares[i](handler)
    }

    if err := handler(ctx); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// 使用示例
router := zallocrout.NewRouter()
router.AddRoute("GET", "/users/:id", func(ctx context.Context) error {
    w := ctx.Value("http.ResponseWriter").(http.ResponseWriter)
    userID, _ := zallocrout.GetParam(ctx, "id")
    fmt.Fprintf(w, "User: %s", userID)
    return nil
})

http.ListenAndServe(":8080", NewHTTPAdapter(router))
```

## 总结

将 zallocrout 改造为 context 设计可以显著提升通用性，使其不仅限于 HTTP 路由，还可用于 gRPC、消息队列、CLI 等场景。虽然会破坏向后兼容性，但考虑到当前包还未广泛使用，这是最佳时机。通过提供详细的迁移指南和 HTTP 适配器，可以降低用户迁移成本。

**核心优势**：
- 完全解耦 HTTP 依赖
- 适用于任意场景
- **完全零分配**：通过 context 池化实现（池预热后）
- 符合 Go 标准库设计理念

**关键技术点**：
1. **自定义 context 实现**：避免 `context.WithValue()` 的分配
2. **三级池化**：context 池 + params map 池 + values map 池
3. **显式资源管理**：用户必须调用 `ReleaseContext()` 释放资源
4. **零分配保证**：池预热后，整个请求处理过程 0 allocs/op
