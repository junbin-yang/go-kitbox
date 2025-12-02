package zallocrout

import (
	"context"
	"sync"
	"sync/atomic"
)

// 节点类型（优先级：静态 > 参数 > 通配符）
type NodeType uint8

const (
	StaticCompressed NodeType = iota // 静态压缩节点
	ParamNode                        // 参数节点（:id）
	WildcardNode                     // 通配符节点（*path）
)

// 处理函数类型
type HandlerFunc func(context.Context) error

// 中间件类型
type Middleware func(HandlerFunc) HandlerFunc

// 路由节点（内存对齐优化）
// 将高频访问字段放在结构体前部，确保在同一个 CPU 缓存行
type RouteNode struct {
	// 无锁静态匹配部分（只读，无需锁）
	staticChildren map[string]*RouteNode // 静态子节点哈希表（O(1)查找）
	seg            []byte                // 路径片段（字节切片）

	// 动态匹配部分（需锁）
	mu            sync.RWMutex // 节点级锁
	typ           NodeType     // 节点类型
	handler       HandlerFunc  // 处理函数
	middlewares   []Middleware // 预编译中间件链
	paramChild    *RouteNode // 参数子节点（单例）
	wildcardChild *RouteNode // 通配符子节点（单例）
	paramName     []byte     // 参数名称（如 :id → "id"）
	paramNames    [][]byte   // 此路由的参数名称列表（按顺序，字节切片避免转换开销）

	// 性能优化字段
	isWildcard bool   // 预计算通配符标记
	hitCount   uint32 // 命中计数（热点缓存用）
}

// 参数键值对（栈分配）
type paramPair struct {
	key   string
	value string
}

// MaxParams 最大参数数量（固定32个，覆盖99%场景）
const MaxParams = 32

// 插入子节点
func (n *RouteNode) insertChild(seg string, child *RouteNode) {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch child.typ {
	case StaticCompressed:
		n.staticChildren[seg] = child
	case ParamNode:
		n.paramChild = child
	case WildcardNode:
		n.wildcardChild = child
	}
}

// 查找子节点（静态）
func (n *RouteNode) findStaticChild(seg string) (*RouteNode, bool) {
	// 静态子节点无锁查找
	child, ok := n.staticChildren[seg]
	return child, ok
}

// 查找参数子节点
func (n *RouteNode) findParamChild() *RouteNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.paramChild
}

// 查找通配符子节点
func (n *RouteNode) findWildcardChild() *RouteNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.wildcardChild
}

// 设置处理器和中间件
func (n *RouteNode) setHandler(handler HandlerFunc, middlewares []Middleware) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handler = handler
	if len(middlewares) > 0 {
		n.middlewares = append(n.middlewares[:0], middlewares...)
	}
}

// 设置处理器和参数名称列表
func (n *RouteNode) setHandlerWithParams(handler HandlerFunc, middlewares []Middleware, paramNames [][]byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handler = handler
	if len(middlewares) > 0 {
		n.middlewares = append(n.middlewares[:0], middlewares...)
	}
	// 存储此路由的参数名称列表（字节切片）
	if len(paramNames) > 0 {
		n.paramNames = make([][]byte, len(paramNames))
		for i, name := range paramNames {
			n.paramNames[i] = make([]byte, len(name))
			copy(n.paramNames[i], name)
		}
	}
}

// 获取处理器
func (n *RouteNode) getHandler() (HandlerFunc, []Middleware, [][]byte) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.handler, n.middlewares, n.paramNames
}

// 增加命中计数（原子操作）
func (n *RouteNode) incrementHitCount() uint32 {
	return atomic.AddUint32(&n.hitCount, 1)
}

// 获取命中计数
func (n *RouteNode) getHitCount() uint32 {
	return atomic.LoadUint32(&n.hitCount)
}
