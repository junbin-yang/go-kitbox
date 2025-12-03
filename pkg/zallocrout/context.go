package zallocrout

import (
	"context"
	"sync"
	"time"
)

// MaxValues 最大自定义值数量（固定6个）
const MaxValues = 6

// 自定义值键值对
type valuePair struct {
	key   string
	value interface{}
}

// routeContext 自定义 context 实现（池化，零分配）
type routeContext struct {
	context.Context
	paramPairs [MaxParams]paramPair // 值类型存储，避免指针逃逸
	paramCount int
	valuePairs [MaxValues]valuePair
	valueCount int
}

// globalContextPool Context 池（仅池化 context 结构本身）
var globalContextPool = sync.Pool{
	New: func() interface{} {
		ctx := &routeContext{}
		// 预分配容量，避免后续扩容
		ctx.paramCount = 0
		ctx.valueCount = 0
		return ctx
	},
}

// acquireContext 从池中获取 context（零分配，通过值复制避免逃逸）
func acquireContext(parent context.Context, paramPairs *[MaxParams]paramPair, paramCount int) *routeContext {
	ctx := globalContextPool.Get().(*routeContext)
	ctx.Context = parent
	ctx.paramCount = paramCount
	ctx.valueCount = 0
	// 从源数组复制参数（避免指针逃逸导致堆分配）
	if paramCount > 0 {
		copy(ctx.paramPairs[:paramCount], paramPairs[:paramCount])
	}
	return ctx
}

// releaseContext 释放 context 到池
func releaseContext(ctx *routeContext) {
	ctx.Context = nil
	ctx.paramCount = 0
	// 清空但保留底层数组容量
	if ctx.valueCount > 0 {
		for i := 0; i < ctx.valueCount; i++ {
			ctx.valuePairs[i] = valuePair{}
		}
		ctx.valueCount = 0
	}
	globalContextPool.Put(ctx)
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
	if strKey, ok := key.(string); ok {
		for i := 0; i < c.valueCount; i++ {
			if c.valuePairs[i].key == strKey {
				return c.valuePairs[i].value
			}
		}
	}
	return c.Context.Value(key)
}

// SetValue 设置自定义值（零分配）
func (c *routeContext) SetValue(key string, value interface{}) bool {
	if c.valueCount >= MaxValues {
		return false
	}
	c.valuePairs[c.valueCount] = valuePair{key, value}
	c.valueCount++
	return true
}

// GetValue 获取自定义值（直接访问，避免 interface{} 转换）
func (c *routeContext) GetValue(key string) (interface{}, bool) {
	for i := 0; i < c.valueCount; i++ {
		if c.valuePairs[i].key == key {
			return c.valuePairs[i].value, true
		}
	}
	return nil, false
}

// GetParam 获取路由参数（零分配）
func (c *routeContext) GetParam(key string) (string, bool) {
	for i := 0; i < c.paramCount; i++ {
		if c.paramPairs[i].key == key {
			return c.paramPairs[i].value, true
		}
	}
	return "", false
}

// GetParam 辅助函数：从 context 获取路由参数
func GetParam(ctx context.Context, key string) (string, bool) {
	if rctx, ok := ctx.(*routeContext); ok {
		return rctx.GetParam(key)
	}
	return "", false
}

// SetValue 辅助函数：设置自定义值
func SetValue(ctx context.Context, key string, value interface{}) bool {
	if rctx, ok := ctx.(*routeContext); ok {
		return rctx.SetValue(key, value)
	}
	return false
}

// ReleaseContext 辅助函数：释放 context
func ReleaseContext(ctx context.Context) {
	if rctx, ok := ctx.(*routeContext); ok {
		releaseContext(rctx)
	}
}

// ExecuteHandler 执行 handler 并自动释放 context（推荐使用）
// 这个函数会在 handler 执行完毕后自动调用 ReleaseContext，无需用户手动管理
func ExecuteHandler(ctx context.Context, handler HandlerFunc, middlewares []Middleware) error {
	// 确保 context 会被释放
	defer ReleaseContext(ctx)

	// 应用中间件链
	finalHandler := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}

	// 执行最终的 handler
	return finalHandler(ctx)
}
